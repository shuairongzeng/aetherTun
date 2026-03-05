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
	defaultMTU   = 9000
	tcpQueueSize = 32
)

type Engine struct {
	cfg       *config.Config
	adapter   *wintun.Adapter
	session   wintun.Session
	stack     *stack.Stack
	linkEP    *channel.Endpoint
	dnsServer *dns.Server
	router    *routing.Engine
	socks5    *proxy.Socks5Client
	ctx       context.Context
	cancel    context.CancelFunc
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
	tunCfg := e.cfg.Tun
	mtu := tunCfg.MTU
	if mtu == 0 {
		mtu = defaultMTU
	}

	// 创建 wintun 适配器
	adapter, err := wintun.CreateAdapter(tunCfg.AdapterName, "Wintun", nil)
	if err != nil {
		return fmt.Errorf("创建 TUN 适配器失败: %w", err)
	}
	e.adapter = adapter

	session, err := adapter.StartSession(0x800000) // 8MB 环形缓冲
	if err != nil {
		adapter.Close()
		return fmt.Errorf("启动 wintun 会话失败: %w", err)
	}
	e.session = session

	// 初始化 gVisor 网络栈
	if err := e.initStack(mtu); err != nil {
		session.End()
		adapter.Close()
		return err
	}

	// 启动读写协程
	go e.readFromTUN()
	go e.writeToTUN()

	// 注册 TCP handler
	e.stack.SetTransportProtocolHandler(tcp.ProtocolNumber, e.handleTCP)

	fmt.Printf("[TUN] 适配器 %s 已启动 (MTU=%d)\n", tunCfg.AdapterName, mtu)
	return nil
}

func (e *Engine) Stop() {
	e.cancel()
	if e.session != (wintun.Session{}) {
		e.session.End()
	}
	if e.adapter != nil {
		e.adapter.Close()
	}
	if e.stack != nil {
		e.stack.Close()
	}
	fmt.Println("[TUN] 已停止")
}

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
	e.stack = s

	const nicID = 1
	if err := s.CreateNIC(nicID, e.linkEP); err != nil {
		return fmt.Errorf("创建 NIC 失败: %v", err)
	}

	// 添加 TUN 接口地址
	prefix, err := netip.ParsePrefix(e.cfg.Tun.Address)
	if err != nil {
		return fmt.Errorf("解析 TUN 地址失败: %w", err)
	}
	as4 := prefix.Addr().As4()
	addr := tcpip.AddrFromSlice(as4[:])
	s.AddProtocolAddress(nicID, tcpip.ProtocolAddress{
		Protocol: ipv4.ProtocolNumber,
		AddressWithPrefix: tcpip.AddressWithPrefix{
			Address:   addr,
			PrefixLen: prefix.Bits(),
		},
	}, stack.AddressProperties{})

	// 默认路由，所有流量进 TUN
	s.SetRouteTable([]tcpip.Route{
		{Destination: header.IPv4EmptySubnet, NIC: nicID},
		{Destination: header.IPv6EmptySubnet, NIC: nicID},
	})

	s.SetPromiscuousMode(nicID, true)
	s.SetSpoofing(nicID, true)

	return nil
}

// readFromTUN 从 wintun 读取 IP 包注入 gVisor
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

// writeToTUN 从 gVisor 取出 IP 包写入 wintun
func (e *Engine) writeToTUN() {
	for {
		pkt := e.linkEP.ReadContext(e.ctx)
		if pkt == nil {
			return
		}

		view := pkt.ToView()
		data := view.AsSlice()
		sendPkt, err := e.session.AllocateSendPacket(len(data))
		if err != nil {
			view.Release()
			pkt.DecRef()
			continue
		}
		copy(sendPkt, data)
		e.session.SendPacket(sendPkt)
		view.Release()
		pkt.DecRef()
	}
}

// handleTCP 处理 gVisor 捕获的 TCP 连接
func (e *Engine) handleTCP(id stack.TransportEndpointID, pkt *stack.PacketBuffer) bool {
	dstIP := net.IP(id.LocalAddress.AsSlice())
	dstPort := id.LocalPort

	// 查询 FakeIP 还原域名
	var targetHost string
	var fakeIPMap = e.dnsServer.FakeIPMap()
	if domain, ok := fakeIPMap.LookupDomain(dstIP); ok {
		targetHost = domain
	} else {
		targetHost = dstIP.String()
	}

	// 路由决策
	action := e.router.Match(dstIP, targetHost, "")
	switch action {
	case routing.ActionBlock:
		log.Printf("[TCP] BLOCK %s:%d", targetHost, dstPort)
		return false
	case routing.ActionDirect:
		log.Printf("[TCP] DIRECT %s:%d", targetHost, dstPort)
		go e.relayTCPDirect(id, pkt, targetHost, dstPort)
		return true
	default: // proxy
		log.Printf("[TCP] PROXY %s:%d", targetHost, dstPort)
		go e.relayTCPProxy(id, pkt, targetHost, dstPort)
		return true
	}
}

func (e *Engine) relayTCPProxy(id stack.TransportEndpointID, pkt *stack.PacketBuffer, host string, port uint16) {
	wq := new(waiter.Queue)
	ep, tcpipErr := e.stack.NewEndpoint(tcp.ProtocolNumber, ipv4.ProtocolNumber, wq)
	if tcpipErr != nil {
		log.Printf("[TCP] 创建端点失败: %v", tcpipErr)
		return
	}
	defer ep.Close()

	conn := gonet.NewTCPConn(wq, ep)
	defer conn.Close()

	proxyConn, err := e.socks5.Connect(host, port)
	if err != nil {
		log.Printf("[TCP] SOCKS5 连接失败 %s:%d: %v", host, port, err)
		return
	}
	defer proxyConn.Close()

	relay(conn, proxyConn)
}

func (e *Engine) relayTCPDirect(id stack.TransportEndpointID, pkt *stack.PacketBuffer, host string, port uint16) {
	wq := new(waiter.Queue)
	ep, tcpipErr := e.stack.NewEndpoint(tcp.ProtocolNumber, ipv4.ProtocolNumber, wq)
	if tcpipErr != nil {
		return
	}
	defer ep.Close()

	conn := gonet.NewTCPConn(wq, ep)
	defer conn.Close()

	target, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		log.Printf("[TCP] 直连失败 %s:%d: %v", host, port, err)
		return
	}
	defer target.Close()

	relay(conn, target)
}

func relay(a, b io.ReadWriter) {
	done := make(chan struct{}, 2)
	go func() { io.Copy(a, b); done <- struct{}{} }()
	go func() { io.Copy(b, a); done <- struct{}{} }()
	<-done
}
