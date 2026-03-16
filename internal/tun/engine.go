package tun

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shuairongzeng/aether/internal/config"
	"github.com/shuairongzeng/aether/internal/dns"
	"github.com/shuairongzeng/aether/internal/proxy"
	"github.com/shuairongzeng/aether/internal/routing"
	"golang.zx2c4.com/wintun"
	"gvisor.dev/gvisor/pkg/buffer"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/link/channel"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
	"gvisor.dev/gvisor/pkg/waiter"
)

// #region agent log
func debugLog(location, message string, data map[string]interface{}) {
	f, err := os.OpenFile(debugSessionLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	entry := map[string]interface{}{"sessionId": debugSessionID, "location": location, "message": message, "data": data, "timestamp": time.Now().UnixMilli()}
	b, _ := json.Marshal(entry)
	f.Write(append(b, '\n'))
}

// #endregion

const (
	debugSessionLogPath   = "debug-724b6f.log"
	debugSessionID        = "724b6f"
	debugRunInvestigation = "investigation-1"
)

// #region agent log
func debugLogCurrent(runID, hypothesisID, location, message string, data map[string]interface{}) {
	f, err := os.OpenFile(debugSessionLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	entry := map[string]interface{}{
		"sessionId":    debugSessionID,
		"runId":        runID,
		"hypothesisId": hypothesisID,
		"location":     location,
		"message":      message,
		"data":         data,
		"timestamp":    time.Now().UnixMilli(),
	}
	b, _ := json.Marshal(entry)
	f.Write(append(b, '\n'))
}

// #endregion

func debugAddrString(addr net.Addr) string {
	if addr == nil {
		return ""
	}
	return addr.String()
}

func debugIPString(ip net.IP) string {
	if ip == nil {
		return ""
	}
	return ip.String()
}

type routeReleaseConn struct {
	net.Conn
	once    sync.Once
	release func()
}

func newRouteReleaseConn(conn net.Conn, release func()) net.Conn {
	if conn == nil || release == nil {
		return conn
	}
	return &routeReleaseConn{Conn: conn, release: release}
}

func (c *routeReleaseConn) Close() error {
	err := c.Conn.Close()
	c.once.Do(func() {
		if c.release != nil {
			c.release()
		}
	})
	return err
}

func (c *routeReleaseConn) CloseWrite() error {
	if cw, ok := c.Conn.(interface{ CloseWrite() error }); ok {
		return cw.CloseWrite()
	}
	return nil
}

func pickDirectIPv4(ips []string) string {
	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip != nil && ip.To4() != nil {
			return ip.String()
		}
	}
	if len(ips) > 0 {
		if ip := net.ParseIP(ips[0]); ip != nil {
			return ip.String()
		}
		return ips[0]
	}
	return ""
}

func currentProcessName() string {
	exePath, err := os.Executable()
	if err != nil {
		return ""
	}
	return strings.ToLower(filepath.Base(exePath))
}

const (
	defaultMTU            = 9000
	tcpQueueSize          = 512
	tcpMaxInFlight        = 1024
	tcpReceiveWnd         = 0 // 0 = gVisor 默认
	udpTimeout            = 60 * time.Second
	udpBufSize            = 65536
	defaultMaxUDPSessions = 2048 // 同时最多 UDP 会话数，防止 socket 耗尽
)

// udpSessionKey 标识一条 UDP 会话（四元组）
type udpSessionKey struct {
	srcAddr string // srcIP:srcPort
	dstAddr string // dstIP:dstPort
}

// udpSession 代表一个活跃的 UDP 中继会话
type udpSession struct {
	appConn  *gonet.UDPConn // gVisor 侧：与应用通信
	cancel   context.CancelFunc
	lastSeen time.Time
}

type Engine struct {
	cfg            *config.Config
	adapter        *wintun.Adapter
	session        wintun.Session
	sessionOK      bool
	netStack       *stack.Stack
	linkEP         *channel.Endpoint
	dnsServer      *dns.Server
	router         *routing.Engine
	socks5         *proxy.Socks5Client
	proxyProcName  string        // 自动检测到的代理进程名，出站流量强制 DIRECT
	selfProcName   string        // 当前 aether 进程名，用于检测自回流
	defaultIfIndex uint32        // 物理网卡索引
	defaultGateway string        // 物理网卡默认网关
	localIP        net.IP        // 物理网卡 IPv4 地址，用于 protectSocket 绑定
	dnsIP          net.IP        // TUN FakeIP DNS 监听 IP
	dnsPort        uint16        // TUN FakeIP DNS 监听端口
	directResolver *net.Resolver // 使用上游 DNS 直接解析，绕过 FakeIP
	directDNSHost  string        // 上游 DNS 主机（用于直连例外路由）
	ctx            context.Context
	cancel         context.CancelFunc

	udpMu             sync.Mutex
	udpSessions       map[udpSessionKey]*udpSession
	maxUDPSessions    int
	routeMu           sync.Mutex
	routeRefs         map[string]int
	dnsRouteRelease   func()
	resolverDiagOnce  sync.Once
	tcpDirectDiagOnce sync.Once
}

func (e *Engine) maybeRunDirectSocketDiagnostics(trigger, failingTarget string) {
	run := func() {
		go e.runDirectSocketDiagnostics(trigger, failingTarget)
	}
	switch trigger {
	case "resolver":
		e.resolverDiagOnce.Do(run)
	case "tcp-direct":
		e.tcpDirectDiagOnce.Do(run)
	default:
		e.resolverDiagOnce.Do(run)
	}
}

func New(cfg *config.Config, dnsServer *dns.Server, router *routing.Engine) *Engine {
	ctx, cancel := context.WithCancel(context.Background())

	// 自动检测监听代理端口的进程（如 xray.exe），启动时绑定一次
	proxyProc := detectProxyProcessName(uint16(cfg.Proxy.Port))
	selfProc := currentProcessName()
	if proxyProc != "" {
		log.Printf("[TUN] 自动绕过代理进程: %s", proxyProc)
	} else {
		log.Printf("[TUN] 未检测到代理进程（端口 %d），可在 config.json 手动添加 process 规则", cfg.Proxy.Port)
	}

	// 在 TUN 路由接管前，记录物理网卡索引用于出站连接绕过 TUN
	ifIndex := getDefaultInterfaceIndex()
	defaultGateway, _, gatewayErr := getDefaultGateway(ifIndex)
	localIP := getPhysicalInterfaceIP(ifIndex)
	if localIP != nil {
		log.Printf("[TUN] 默认出站网卡: index=%d ip=%s（DIRECT 连接绑定此地址）", ifIndex, localIP)
	} else {
		log.Printf("[TUN] 警告: 未能获取默认网卡 IP，DIRECT 连接可能形成回环")
	}
	if defaultGateway != "" {
		log.Printf("[TUN] 默认物理网关: %s", defaultGateway)
	} else if gatewayErr != nil {
		log.Printf("[TUN] 警告: 未能获取默认网关: %v", gatewayErr)
	}

	// 解析 FakeIP DNS 监听地址（用于在 UDP handler 内联处理 DNS 查询）
	dnsHost, dnsPortStr, _ := net.SplitHostPort(cfg.Tun.DNSListen)
	dnsPortNum, _ := strconv.Atoi(dnsPortStr)

	e := &Engine{
		cfg:            cfg,
		dnsServer:      dnsServer,
		router:         router,
		socks5:         proxy.NewSocks5Client(cfg.Proxy.Host, cfg.Proxy.Port),
		proxyProcName:  proxyProc,
		selfProcName:   selfProc,
		defaultIfIndex: ifIndex,
		defaultGateway: defaultGateway,
		localIP:        localIP,
		dnsIP:          net.ParseIP(dnsHost),
		dnsPort:        uint16(dnsPortNum),
		ctx:            ctx,
		cancel:         cancel,
		udpSessions:    make(map[udpSessionKey]*udpSession),
		maxUDPSessions: cfg.Tun.MaxUDPSessions,
		routeRefs:      make(map[string]int),
	}
	if e.maxUDPSessions <= 0 {
		e.maxUDPSessions = defaultMaxUDPSessions
	}

	upstream := dnsServer.Upstream()
	upstreamHost, _, err := net.SplitHostPort(upstream)
	if err != nil {
		upstreamHost = upstream
	}
	e.directDNSHost = upstreamHost
	e.directResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			// #region agent log
			debugLogCurrent(debugRunInvestigation, "H4", "engine.go:directResolver.Dial", "Resolver dial request", map[string]interface{}{"network": network, "resolverAddress": address, "forcedUpstream": upstream, "defaultIfIndex": e.defaultIfIndex, "defaultGateway": e.defaultGateway, "localIP": debugIPString(e.localIP)})
			// #endregion
			d := net.Dialer{}
			if e.localIP != nil {
				d.LocalAddr = &net.UDPAddr{IP: e.localIP}
			}
			conn, err := d.DialContext(ctx, "udp", upstream)
			localAddr := ""
			remoteAddr := ""
			if conn != nil {
				localAddr = debugAddrString(conn.LocalAddr())
				remoteAddr = debugAddrString(conn.RemoteAddr())
			}
			// #region agent log
			debugLogCurrent(debugRunInvestigation, "H2", "engine.go:directResolver.Dial", "Resolver dial result", map[string]interface{}{"network": network, "resolverAddress": address, "forcedUpstream": upstream, "localAddr": localAddr, "remoteAddr": remoteAddr, "err": fmt.Sprintf("%v", err)})
			// #endregion
			if err != nil {
				e.maybeRunDirectSocketDiagnostics("resolver", upstream)
			}
			if conn != nil {
				// #region agent log
				debugLogCurrent(debugRunInvestigation, "H2", "engine.go:directResolver.Dial", "Resolver dial connected", map[string]interface{}{"network": network, "resolverAddress": address, "forcedUpstream": upstream, "localAddr": debugAddrString(conn.LocalAddr()), "remoteAddr": debugAddrString(conn.RemoteAddr())})
				// #endregion
			}
			return conn, err
		},
	}

	// 将物理网卡 IP 传给 DNS Server，让出站 DNS 查询绑定物理网卡
	dnsServer.SetLocalIP(localIP)

	ifaceName := ""
	if ifIndex != 0 {
		if iface, err := net.InterfaceByIndex(int(ifIndex)); err == nil {
			ifaceName = iface.Name
		}
	}
	// #region agent log
	debugLogCurrent(debugRunInvestigation, "H3", "engine.go:New", "Engine direct path config", map[string]interface{}{"defaultIfIndex": ifIndex, "defaultIfName": ifaceName, "defaultGateway": defaultGateway, "localIP": debugIPString(localIP), "dnsListen": cfg.Tun.DNSListen, "dnsUpstream": upstream, "proxyProcName": proxyProc, "selfProcName": selfProc})
	// #endregion

	return e
}

func (e *Engine) Start() error {
	mtu := e.cfg.Tun.MTU
	if mtu == 0 {
		mtu = defaultMTU
	}

	// 1. 创建 wintun 适配器
	adapter, err := createAdapterSafe(e.cfg.Tun.AdapterName)
	if err != nil {
		return fmt.Errorf("创建 TUN 适配器失败: %w", err)
	}
	e.adapter = adapter

	session, err := adapter.StartSession(0x800000) // 8MB ring buffer
	if err != nil {
		adapter.Close()
		return fmt.Errorf("启动 wintun 会话失败: %w", err)
	}
	e.session = session
	e.sessionOK = true

	// 2. 配置 Windows 适配器 IP 和路由（需在 gVisor 初始化前完成）
	if err := e.configureWindowsAdapter(); err != nil {
		e.session.End()
		e.adapter.Close()
		return fmt.Errorf("配置网络适配器失败: %w", err)
	}
	if release, err := e.acquireDirectRoute(e.directDNSHost); err != nil {
		e.teardownWindowsAdapter()
		e.session.End()
		e.adapter.Close()
		return fmt.Errorf("配置直连 DNS 路由失败: %w", err)
	} else {
		e.dnsRouteRelease = release
	}

	// 3. 初始化 gVisor 网络栈
	if err := e.initStack(mtu); err != nil {
		if e.dnsRouteRelease != nil {
			e.dnsRouteRelease()
			e.dnsRouteRelease = nil
		}
		e.session.End()
		e.adapter.Close()
		return err
	}

	// 4. 注册 TCP forwarder（正确的连接级拦截方式）
	tcpFwd := tcp.NewForwarder(e.netStack, tcpReceiveWnd, tcpMaxInFlight, e.handleTCPConn)
	e.netStack.SetTransportProtocolHandler(tcp.ProtocolNumber, tcpFwd.HandlePacket)

	// 5. 注册 UDP forwarder
	udpFwd := udp.NewForwarder(e.netStack, e.handleUDPPacket)
	e.netStack.SetTransportProtocolHandler(udp.ProtocolNumber, udpFwd.HandlePacket)

	// 6. 启动 UDP session 过期清理
	go e.cleanupUDPSessions()

	// 7. 启动 wintun ↔ gVisor 数据泵
	go e.readFromTUN()
	go e.writeToTUN()

	log.Printf("[TUN] 启动成功: 适配器=%s MTU=%d 代理=%s:%d",
		e.cfg.Tun.AdapterName, mtu, e.cfg.Proxy.Host, e.cfg.Proxy.Port)
	return nil
}

func (e *Engine) Stop() {
	e.cancel()
	if e.dnsRouteRelease != nil {
		e.dnsRouteRelease()
		e.dnsRouteRelease = nil
	}
	e.teardownWindowsAdapter()
	if e.sessionOK {
		e.session.End()
		e.sessionOK = false
	}
	if e.adapter != nil {
		e.adapter.Close()
		e.adapter = nil
	}
	if e.netStack != nil {
		e.netStack.Close()
		e.netStack = nil
	}
	log.Println("[TUN] 已停止")
}

// initStack 初始化 gVisor TCP/IP 协议栈
func (e *Engine) initStack(mtu uint32) error {
	e.linkEP = channel.New(tcpQueueSize, mtu, "")

	s := stack.New(stack.Options{
		NetworkProtocols: []stack.NetworkProtocolFactory{
			ipv4.NewProtocol,
			ipv6.NewProtocol,
		},
		TransportProtocols: []stack.TransportProtocolFactory{
			tcp.NewProtocol,
			udp.NewProtocol,
		},
	})
	e.netStack = s

	const nicID tcpip.NICID = 1
	if tcpipErr := s.CreateNIC(nicID, e.linkEP); tcpipErr != nil {
		return fmt.Errorf("创建 NIC 失败: %v", tcpipErr)
	}

	// 绑定 TUN 接口 IP 地址
	prefix, err := netip.ParsePrefix(e.cfg.Tun.Address)
	if err != nil {
		return fmt.Errorf("解析 TUN 地址失败: %w", err)
	}
	as4 := prefix.Addr().As4()
	tcpipAddr := tcpip.AddrFromSlice(as4[:])
	if tcpipErr := s.AddProtocolAddress(nicID, tcpip.ProtocolAddress{
		Protocol: ipv4.ProtocolNumber,
		AddressWithPrefix: tcpip.AddressWithPrefix{
			Address:   tcpipAddr,
			PrefixLen: prefix.Bits(),
		},
	}, stack.AddressProperties{}); tcpipErr != nil {
		return fmt.Errorf("绑定 NIC 地址失败: %v", tcpipErr)
	}

	// 默认路由：所有 IP 包都走 TUN NIC
	s.SetRouteTable([]tcpip.Route{
		{Destination: header.IPv4EmptySubnet, NIC: nicID},
		{Destination: header.IPv6EmptySubnet, NIC: nicID},
	})

	// 混杂模式 + spoofing：接受所有目标 IP 的包（透明代理必须）
	if tcpipErr := s.SetPromiscuousMode(nicID, true); tcpipErr != nil {
		return fmt.Errorf("设置混杂模式失败: %v", tcpipErr)
	}
	if tcpipErr := s.SetSpoofing(nicID, true); tcpipErr != nil {
		return fmt.Errorf("设置 spoofing 失败: %v", tcpipErr)
	}

	return nil
}

// readFromTUN 从 wintun 读取原始 IP 包，注入 gVisor 协议栈
func (e *Engine) readFromTUN() {
	for {
		select {
		case <-e.ctx.Done():
			return
		default:
		}

		pkt, err := e.session.ReceivePacket()
		if err != nil {
			if e.ctx.Err() != nil {
				return
			}
			continue
		}

		buf := buffer.MakeWithData(pkt)
		pkb := stack.NewPacketBuffer(stack.PacketBufferOptions{Payload: buf})
		e.linkEP.InjectInbound(ipv4.ProtocolNumber, pkb)
		pkb.DecRef()
		e.session.ReleaseReceivePacket(pkt)
	}
}

// writeToTUN 从 gVisor 取出出站 IP 包，写回 wintun（发给应用）
func (e *Engine) writeToTUN() {
	for {
		pkb := e.linkEP.ReadContext(e.ctx)
		if pkb == nil {
			return // ctx cancelled
		}

		view := pkb.ToView()
		data := view.AsSlice()
		if len(data) > 0 {
			sendBuf, err := e.session.AllocateSendPacket(len(data))
			if err == nil {
				copy(sendBuf, data)
				e.session.SendPacket(sendBuf)
			}
		}
		view.Release()
		pkb.DecRef()
	}
}

func (e *Engine) resolveDirectHost(ctx context.Context, targetHost string) (string, []string, string, error) {
	if ip := net.ParseIP(targetHost); ip != nil {
		normalized := ip.String()
		return normalized, []string{normalized}, "literal", nil
	}
	ips, resolverMode, err := e.dnsServer.LookupIPv4(ctx, targetHost)
	// #region agent log
	debugLogCurrent(debugRunInvestigation, "H14", "engine.go:resolveDirectHost", "Direct host lookup result", map[string]interface{}{"targetHost": targetHost, "resolverMode": resolverMode, "resolvedIPs": ips, "err": fmt.Sprintf("%v", err)})
	// #endregion
	if err != nil {
		return "", ips, resolverMode, err
	}
	selected := pickDirectIPv4(ips)
	if selected == "" {
		return "", ips, resolverMode, fmt.Errorf("未解析到可用 IPv4 地址: %s", targetHost)
	}
	return selected, ips, resolverMode, nil
}

func (e *Engine) dialDirectTCP(ctx context.Context, resolvedTarget string, dstPort uint16) (net.Conn, error) {
	releaseRoute, err := e.acquireDirectRoute(resolvedTarget)
	if err != nil {
		return nil, err
	}
	addr := net.JoinHostPort(resolvedTarget, fmt.Sprintf("%d", dstPort))
	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		releaseRoute()
		return nil, err
	}
	return newRouteReleaseConn(conn, releaseRoute), nil
}

// handleTCPConn 是 tcp.Forwarder 的回调：每个新 TCP 连接调用一次
// 此时三次握手尚未完成，调用 r.CreateEndpoint 后握手才完成
func (e *Engine) handleTCPConn(r *tcp.ForwarderRequest) {
	id := r.ID()
	dstIP := net.IP(id.LocalAddress.AsSlice())
	dstPort := id.LocalPort

	// FakeIP → 还原真实域名
	targetHost := dstIP.String()
	if domain, ok := e.dnsServer.FakeIPMap().LookupDomain(dstIP); ok {
		targetHost = domain
	}

	// 查找源进程名（IPv4 only，用于 process 规则匹配）
	var procName string
	if id.RemoteAddress.Len() == 4 {
		procName = lookupTCPProcess(id.RemoteAddress.As4(), id.RemotePort)
	}

	// 路由决策：代理进程自身强制 DIRECT，防止环路
	var action routing.Action
	if e.proxyProcName != "" && strings.EqualFold(procName, e.proxyProcName) {
		action = routing.ActionDirect
	} else {
		action = e.router.Match(dstIP, targetHost, procName)
	}
	if strings.EqualFold(procName, e.selfProcName) || targetHost == "1.1.1.1" {
		// #region agent log
		debugLogCurrent(debugRunInvestigation, "H9", "engine.go:handleTCPConn", "TCP route decision", map[string]interface{}{"targetHost": targetHost, "dstPort": dstPort, "procName": procName, "selfProcName": e.selfProcName, "proxyProcName": e.proxyProcName, "action": fmt.Sprintf("%v", action), "srcIP": net.IP(id.RemoteAddress.AsSlice()).String(), "srcPort": id.RemotePort})
		// #endregion
	}

	if action == routing.ActionBlock {
		log.Printf("[TCP] BLOCK  %s:%d", targetHost, dstPort)
		r.Complete(true) // 发 RST，拒绝连接
		return
	}

	// 完成三次握手，建立 gVisor 端的 TCP 连接（gonet.TCPConn）
	wq := new(waiter.Queue)
	ep, tcpipErr := r.CreateEndpoint(wq)
	if tcpipErr != nil {
		log.Printf("[TCP] 建立端点失败 %s:%d: %v", targetHost, dstPort, tcpipErr)
		r.Complete(true)
		return
	}
	r.Complete(false)

	appConn := gonet.NewTCPConn(wq, ep)

	switch action {
	case routing.ActionDirect:
		log.Printf("[TCP] DIRECT %s:%d", targetHost, dstPort)
		go e.relayConcurrent(appConn, func() (net.Conn, error) {
			addr := net.JoinHostPort(targetHost, fmt.Sprintf("%d", dstPort))
			resolvedTarget, ips, _, resolveErr := e.resolveDirectHost(e.ctx, targetHost)
			// #region agent log
			debugLog("engine.go:TCP-DIRECT-dial", "TCP DIRECT resolution", map[string]interface{}{"hypothesisId": "H1", "targetHost": targetHost, "origDstIP": dstIP.String(), "resolvedIPs": ips, "selectedIP": resolvedTarget, "resolveErr": fmt.Sprintf("%v", resolveErr), "dialAddr": addr})
			// #endregion
			// #region agent log
			debugLogCurrent(debugRunInvestigation, "H1", "engine.go:TCP-DIRECT-dial", "TCP DIRECT resolution", map[string]interface{}{"targetHost": targetHost, "origDstIP": dstIP.String(), "resolvedIPs": ips, "selectedIP": resolvedTarget, "resolveErr": fmt.Sprintf("%v", resolveErr), "dialAddr": addr})
			// #endregion
			if resolveErr != nil {
				e.maybeRunDirectSocketDiagnostics("tcp-direct", addr)
				return nil, resolveErr
			}
			conn, err := e.dialDirectTCP(e.ctx, resolvedTarget, dstPort)
			localAddr := ""
			remoteAddr := ""
			if conn != nil {
				localAddr = debugAddrString(conn.LocalAddr())
				remoteAddr = debugAddrString(conn.RemoteAddr())
			}
			// #region agent log
			debugLogCurrent(debugRunInvestigation, "H1", "engine.go:TCP-DIRECT-dial", "TCP DIRECT dial result", map[string]interface{}{"targetHost": targetHost, "selectedIP": resolvedTarget, "dialAddr": addr, "localAddr": localAddr, "remoteAddr": remoteAddr, "err": fmt.Sprintf("%v", err)})
			// #endregion
			if err != nil {
				e.maybeRunDirectSocketDiagnostics("tcp-direct", net.JoinHostPort(resolvedTarget, fmt.Sprintf("%d", dstPort)))
			}
			if conn != nil {
				// #region agent log
				debugLogCurrent(debugRunInvestigation, "H1", "engine.go:TCP-DIRECT-dial", "TCP DIRECT connected", map[string]interface{}{"targetHost": targetHost, "selectedIP": resolvedTarget, "dialAddr": addr, "localAddr": debugAddrString(conn.LocalAddr()), "remoteAddr": debugAddrString(conn.RemoteAddr())})
				// #endregion
			}
			return conn, err
		})
	default: // proxy
		log.Printf("[TCP] PROXY  %s:%d", targetHost, dstPort)
		go e.relayConcurrent(appConn, func() (net.Conn, error) {
			return e.socks5.Connect(targetHost, dstPort)
		})
	}
}

// relayConcurrent 建立出站连接后双向转发数据
func (e *Engine) relayConcurrent(appConn *gonet.TCPConn, dial func() (net.Conn, error)) {
	defer appConn.Close()

	outConn, err := dial()
	if err != nil {
		log.Printf("[TCP] 出站连接失败: %v", err)
		return
	}
	defer outConn.Close()

	done := make(chan struct{}, 2)

	// app → out
	go func() {
		io.Copy(outConn, appConn)
		// 半关闭：通知对端写完了
		if cw, ok := outConn.(interface{ CloseWrite() error }); ok {
			cw.CloseWrite()
		}
		done <- struct{}{}
	}()

	// out → app
	go func() {
		io.Copy(appConn, outConn)
		appConn.CloseWrite()
		done <- struct{}{}
	}()

	// 等两个方向都结束
	<-done
	<-done
}

// ─── UDP ──────────────────────────────────────────────────────────────────────

// 广播/组播地址检测（这类流量不应通过代理转发）
var (
	_, udpMulticastRange, _ = net.ParseCIDR("224.0.0.0/4")
	udpLimitedBroadcast     = net.IPv4(255, 255, 255, 255)
)

func isUDPBroadcastOrMulticast(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.Equal(udpLimitedBroadcast) {
		return true
	}
	if udpMulticastRange.Contains(ip) {
		return true
	}
	// 子网广播：末字节为 255（简单检测）
	if ip4 := ip.To4(); ip4 != nil && ip4[3] == 255 {
		return true
	}
	return false
}

// handleUDPPacket 是 udp.Forwarder 的回调，每个新 UDP 四元组触发一次
func (e *Engine) handleUDPPacket(r *udp.ForwarderRequest) (handled bool) {
	id := r.ID()
	dstIP := net.IP(id.LocalAddress.AsSlice())
	dstPort := id.LocalPort
	srcKey := net.JoinHostPort(id.RemoteAddress.String(), fmt.Sprintf("%d", id.RemotePort))
	dstKey := net.JoinHostPort(id.LocalAddress.String(), fmt.Sprintf("%d", dstPort))
	key := udpSessionKey{srcAddr: srcKey, dstAddr: dstKey}

	// 广播/组播直接丢弃，不尝试转发（会导致 socket 耗尽）
	if isUDPBroadcastOrMulticast(dstIP) {
		return false
	}

	// FakeIP DNS 查询：在 gVisor 内联处理，不走代理/直连，避免环路
	if e.dnsIP != nil && dstIP.Equal(e.dnsIP) && dstPort == e.dnsPort {
		wq := new(waiter.Queue)
		ep, tcpipErr := r.CreateEndpoint(wq)
		if tcpipErr != nil {
			return false
		}
		appConn := gonet.NewUDPConn(wq, ep)
		ctx, cancel := context.WithCancel(e.ctx)
		e.udpMu.Lock()
		e.udpSessions[key] = &udpSession{appConn: appConn, cancel: cancel, lastSeen: time.Now()}
		e.udpMu.Unlock()
		go e.serveDNSConn(ctx, key, appConn)
		return true
	}

	// FakeIP → 还原真实域名
	targetHost := dstIP.String()
	if domain, ok := e.dnsServer.FakeIPMap().LookupDomain(dstIP); ok {
		targetHost = domain
	}

	// 查找源进程名（IPv4 only）
	var procName string
	if id.RemoteAddress.Len() == 4 {
		procName = lookupUDPProcess(id.RemoteAddress.As4(), id.RemotePort)
	}

	// 路由决策：代理进程自身强制 DIRECT，防止环路
	var action routing.Action
	if e.proxyProcName != "" && strings.EqualFold(procName, e.proxyProcName) {
		action = routing.ActionDirect
	} else {
		action = e.router.Match(dstIP, targetHost, procName)
	}
	if strings.EqualFold(procName, e.selfProcName) || targetHost == "8.8.8.8" {
		// #region agent log
		debugLogCurrent(debugRunInvestigation, "H9", "engine.go:handleUDPPacket", "UDP route decision", map[string]interface{}{"targetHost": targetHost, "dstPort": dstPort, "procName": procName, "selfProcName": e.selfProcName, "proxyProcName": e.proxyProcName, "action": fmt.Sprintf("%v", action), "srcIP": net.IP(id.RemoteAddress.AsSlice()).String(), "srcPort": id.RemotePort})
		// #endregion
	}
	if action == routing.ActionBlock {
		log.Printf("[UDP] BLOCK  %s:%d", targetHost, dstPort)
		return false
	}

	// 查找已有 session
	e.udpMu.Lock()
	sess, exists := e.udpSessions[key]
	if exists {
		sess.lastSeen = time.Now()
		e.udpMu.Unlock()
		// session 已存在，ForwarderRequest 会自动把包注入已有 endpoint
		return true
	}

	// 新建 session 前检查并发上限，满时驱逐最旧会话
	if len(e.udpSessions) >= e.maxUDPSessions {
		var oldestKey udpSessionKey
		var oldestTime time.Time
		first := true
		for k, s := range e.udpSessions {
			if first || s.lastSeen.Before(oldestTime) {
				oldestKey = k
				oldestTime = s.lastSeen
				first = false
			}
		}
		if !first {
			e.udpSessions[oldestKey].cancel()
			e.udpSessions[oldestKey].appConn.Close()
			delete(e.udpSessions, oldestKey)
			// #region agent log
			debugLog("engine.go:handleUDPPacket", "UDP session evicted (LRU)", map[string]interface{}{"hypothesisId": "H2", "evictedKey": oldestKey.dstAddr, "sessionCount": len(e.udpSessions), "newDst": fmt.Sprintf("%s:%d", targetHost, dstPort)})
			// #endregion
		}
	}
	wq := new(waiter.Queue)
	ep, tcpipErr := r.CreateEndpoint(wq)
	if tcpipErr != nil {
		e.udpMu.Unlock()
		log.Printf("[UDP] 创建端点失败 %s:%d: %v", targetHost, dstPort, tcpipErr)
		return false
	}

	ctx, cancel := context.WithCancel(e.ctx)
	appConn := gonet.NewUDPConn(wq, ep)
	sess = &udpSession{
		appConn:  appConn,
		cancel:   cancel,
		lastSeen: time.Now(),
	}
	e.udpSessions[key] = sess
	e.udpMu.Unlock()

	// #region agent log
	debugLog("engine.go:handleUDPPacket-newSession", "New UDP session created", map[string]interface{}{"hypothesisId": "H2", "sessionCount": len(e.udpSessions), "srcKey": srcKey, "dstKey": dstKey, "action": fmt.Sprintf("%v", action)})
	// #endregion

	switch action {
	case routing.ActionDirect:
		log.Printf("[UDP] DIRECT %s:%d", targetHost, dstPort)
		go e.relayUDPDirect(ctx, key, appConn, targetHost, dstPort)
	default:
		log.Printf("[UDP] PROXY  %s:%d", targetHost, dstPort)
		go e.relayUDPProxy(ctx, key, appConn, targetHost, dstPort)
	}
	return true
}

// relayUDPDirect 通过真实 UDP socket 直连转发
func (e *Engine) relayUDPDirect(ctx context.Context, key udpSessionKey, appConn *gonet.UDPConn, host string, port uint16) {
	defer e.removeUDPSession(key)
	defer appConn.Close()

	resolvedHost, _, _, err := e.resolveDirectHost(ctx, host)
	if err != nil {
		log.Printf("[UDP] DNS 解析失败 %s: %v", host, err)
		return
	}
	releaseRoute, err := e.acquireDirectRoute(resolvedHost)
	if err != nil {
		log.Printf("[UDP] 配置直连路由失败 %s: %v", resolvedHost, err)
		return
	}
	defer releaseRoute()

	lc := net.ListenConfig{}
	pc, err := lc.ListenPacket(ctx, "udp4", ":0")
	if err != nil {
		log.Printf("[UDP] 创建出站 socket 失败: %v", err)
		return
	}
	outConn := pc.(*net.UDPConn)
	defer outConn.Close()

	dstAddr := &net.UDPAddr{IP: net.ParseIP(resolvedHost), Port: int(port)}
	// #region agent log
	debugLog("engine.go:relayUDPDirect", "UDP DIRECT dstAddr", map[string]interface{}{"hypothesisId": "H3", "host": host, "resolvedHost": resolvedHost, "port": port, "dstAddrIPNil": dstAddr.IP == nil})
	// #endregion
	buf := make([]byte, udpBufSize)

	// app → out（每次 Read 是一个完整 datagram）
	go func() {
		pkt := make([]byte, udpBufSize)
		for {
			n, _, err := appConn.ReadFrom(pkt)
			if err != nil {
				return
			}
			outConn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			outConn.WriteTo(pkt[:n], dstAddr)
		}
	}()

	// out → app（30s 无响应则关闭，避免单向流量（如 DNS 请求）永久挂起）
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		outConn.SetReadDeadline(time.Now().Add(30 * time.Second))
		n, _, err := outConn.ReadFromUDP(buf)
		if err != nil {
			return // 超时或关闭，正常退出
		}
		appConn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		appConn.WriteTo(buf[:n], nil)
		e.touchUDPSession(key)
	}
}

// relayUDPProxy 通过 SOCKS5 UDP ASSOCIATE 转发
func (e *Engine) relayUDPProxy(ctx context.Context, key udpSessionKey, appConn *gonet.UDPConn, host string, port uint16) {
	defer e.removeUDPSession(key)
	defer appConn.Close()

	udpSess, err := e.socks5.UDPAssociate()
	if err != nil {
		log.Printf("[UDP] SOCKS5 UDP ASSOCIATE 失败 %s:%d: %v", host, port, err)
		return
	}
	defer udpSess.Close()

	buf := make([]byte, udpBufSize)

	// app → proxy relay
	go func() {
		pkt := make([]byte, udpBufSize)
		for {
			n, _, err := appConn.ReadFrom(pkt)
			if err != nil {
				return
			}
			udpSess.UDPConn.SetWriteDeadline(time.Now().Add(udpTimeout))
			if err := udpSess.SendUDP(pkt[:n], host, port); err != nil {
				log.Printf("[UDP] 发送到代理失败: %v", err)
				return
			}
		}
	}()

	// proxy relay → app
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		udpSess.UDPConn.SetReadDeadline(time.Now().Add(udpTimeout))
		payload, _, _, err := udpSess.RecvUDP(buf)
		if err != nil {
			return
		}
		appConn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		appConn.WriteTo(payload, nil)
		e.touchUDPSession(key)
	}
}

func (e *Engine) removeUDPSession(key udpSessionKey) {
	e.udpMu.Lock()
	delete(e.udpSessions, key)
	e.udpMu.Unlock()
}

func (e *Engine) touchUDPSession(key udpSessionKey) {
	e.udpMu.Lock()
	if sess, ok := e.udpSessions[key]; ok {
		sess.lastSeen = time.Now()
	}
	e.udpMu.Unlock()
}

// cleanupUDPSessions 定期清理超时的 UDP 会话
func (e *Engine) cleanupUDPSessions() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			e.udpMu.Lock()
			for key, sess := range e.udpSessions {
				if now.Sub(sess.lastSeen) > udpTimeout {
					sess.cancel()
					sess.appConn.Close()
					delete(e.udpSessions, key)
				}
			}
			e.udpMu.Unlock()
		}
	}
}

// serveDNSConn 在 gVisor UDP 端点内联处理 FakeIP DNS 查询，无需经过网络 socket。
// 避免 198.18.0.2:53 的流量被 TUN 当作普通 UDP 代理，形成回环。
func (e *Engine) serveDNSConn(ctx context.Context, key udpSessionKey, appConn *gonet.UDPConn) {
	defer e.removeUDPSession(key)
	defer appConn.Close()

	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		appConn.SetReadDeadline(time.Now().Add(5 * time.Second))
		n, _, err := appConn.ReadFrom(buf)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}
		resp := e.dnsServer.ProcessQuery(buf[:n])
		if resp != nil {
			appConn.SetWriteDeadline(time.Now().Add(2 * time.Second))
			appConn.WriteTo(resp, nil)
		}
	}
}
