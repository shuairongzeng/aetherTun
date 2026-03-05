package tun

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/netip"

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

const (
	defaultMTU      = 9000
	tcpQueueSize    = 512
	tcpMaxInFlight  = 1024
	tcpReceiveWnd   = 0 // 0 = gVisor 默认
)

type Engine struct {
	cfg        *config.Config
	adapter    *wintun.Adapter
	session    wintun.Session
	sessionOK  bool
	netStack   *stack.Stack
	linkEP     *channel.Endpoint
	dnsServer  *dns.Server
	router     *routing.Engine
	socks5     *proxy.Socks5Client
	ctx        context.Context
	cancel     context.CancelFunc
}

func New(cfg *config.Config, dnsServer *dns.Server, router *routing.Engine) *Engine {
	ctx, cancel := context.WithCancel(context.Background())
	return &Engine{
		cfg:       cfg,
		dnsServer: dnsServer,
		router:    router,
		socks5:    proxy.NewSocks5Client(cfg.Proxy.Host, cfg.Proxy.Port),
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (e *Engine) Start() error {
	mtu := e.cfg.Tun.MTU
	if mtu == 0 {
		mtu = defaultMTU
	}

	// 1. 创建 wintun 适配器
	adapter, err := wintun.CreateAdapter(e.cfg.Tun.AdapterName, "Wintun", nil)
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

	// 2. 初始化 gVisor 网络栈
	if err := e.initStack(mtu); err != nil {
		e.session.End()
		e.adapter.Close()
		return err
	}

	// 3. 注册 TCP forwarder（正确的连接级拦截方式）
	tcpFwd := tcp.NewForwarder(e.netStack, tcpReceiveWnd, tcpMaxInFlight, e.handleTCPConn)
	e.netStack.SetTransportProtocolHandler(tcp.ProtocolNumber, tcpFwd.HandlePacket)

	// 4. 启动 wintun ↔ gVisor 数据泵
	go e.readFromTUN()
	go e.writeToTUN()

	log.Printf("[TUN] 启动成功: 适配器=%s MTU=%d 代理=%s:%d",
		e.cfg.Tun.AdapterName, mtu, e.cfg.Proxy.Host, e.cfg.Proxy.Port)
	return nil
}

func (e *Engine) Stop() {
	e.cancel()
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

	// 路由决策
	action := e.router.Match(dstIP, targetHost, "")

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
			return net.Dial("tcp", fmt.Sprintf("%s:%d", targetHost, dstPort))
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
		if tc, ok := outConn.(*net.TCPConn); ok {
			tc.CloseWrite()
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
