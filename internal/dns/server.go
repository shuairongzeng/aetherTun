package dns

import (
	"fmt"
	"net"
	"time"

	"github.com/miekg/dns"
)

type Server struct {
	listenAddr string
	fakeIPMap  *FakeIPMap
	upstream   string
	server     *dns.Server
}

func NewServer(listenAddr, upstream, fakeIPCIDR string) (*Server, error) {
	fakeMap, err := NewFakeIPMap(fakeIPCIDR)
	if err != nil {
		return nil, fmt.Errorf("初始化 FakeIP 池失败: %w", err)
	}
	return &Server{
		listenAddr: listenAddr,
		fakeIPMap:  fakeMap,
		upstream:   upstream,
	}, nil
}

func (s *Server) Start() error {
	// 等待绑定地址可用（最多 5 秒，适配器 IP 分配可能有延迟）
	pc, err := waitBind(s.listenAddr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("DNS 监听地址不可用 %s: %w", s.listenAddr, err)
	}

	mux := dns.NewServeMux()
	mux.HandleFunc(".", s.handleDNS)
	s.server = &dns.Server{
		PacketConn: pc,
		Net:        "udp",
		Handler:    mux,
	}
	go func() {
		if err := s.server.ActivateAndServe(); err != nil {
			fmt.Printf("[DNS] 服务器停止: %v\n", err)
		}
	}()
	fmt.Printf("[DNS] 监听 %s (FakeIP 模式)\n", s.listenAddr)
	return nil
}

// waitBind 重试绑定 UDP 地址，等待系统网络接口 IP 分配完成
func waitBind(addr string, timeout time.Duration) (net.PacketConn, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		pc, err := net.ListenPacket("udp", addr)
		if err == nil {
			return pc, nil
		}
		lastErr = err
		time.Sleep(300 * time.Millisecond)
	}
	return nil, lastErr
}

func (s *Server) Stop() {
	if s.server != nil {
		s.server.Shutdown()
	}
}

// FakeIPMap 暴露给 TUN handler 查询域名
func (s *Server) FakeIPMap() *FakeIPMap {
	return s.fakeIPMap
}

// Upstream 返回上游 DNS 地址（供 TUN 引擎直接解析域名用）
func (s *Server) Upstream() string {
	return s.upstream
}

func (s *Server) handleDNS(w dns.ResponseWriter, r *dns.Msg) {
	if len(r.Question) == 0 {
		dns.HandleFailed(w, r)
		return
	}

	q := r.Question[0]
	domain := dns.Fqdn(q.Name)
	// 去掉末尾的 .
	domain = domain[:len(domain)-1]

	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true

	if q.Qtype == dns.TypeA {
		fakeIP := s.fakeIPMap.Assign(domain)
		m.Answer = append(m.Answer, &dns.A{
			Hdr: dns.RR_Header{
				Name:   q.Name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    1,
			},
			A: fakeIP,
		})
		w.WriteMsg(m)
		fmt.Printf("[DNS] %s → FakeIP %s\n", domain, fakeIP)
		return
	}

	// AAAA / 其他类型：转发到上游
	s.forward(w, r)
}

func (s *Server) forward(w dns.ResponseWriter, r *dns.Msg) {
	c := new(dns.Client)
	resp, _, err := c.Exchange(r, s.upstream)
	if err != nil {
		dns.HandleFailed(w, r)
		return
	}
	w.WriteMsg(resp)
}

// IsLocalAddr 判断是否是 DNS 监听地址本身（避免循环）
func (s *Server) IsLocalAddr(ip net.IP) bool {
	listenIP, _, _ := net.SplitHostPort(s.listenAddr)
	return ip.String() == listenIP
}
