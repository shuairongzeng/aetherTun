package dns

import (
	"context"
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

func extractIPv4Answers(resp *dns.Msg) []string {
	if resp == nil {
		return nil
	}
	seen := make(map[string]struct{})
	ips := make([]string, 0, len(resp.Answer))
	for _, answer := range resp.Answer {
		a, ok := answer.(*dns.A)
		if !ok || a.A == nil {
			continue
		}
		ip := a.A.String()
		if _, exists := seen[ip]; exists {
			continue
		}
		seen[ip] = struct{}{}
		ips = append(ips, ip)
	}
	return ips
}

// LookupIPv4 queries the upstream resolver for A records and falls back to TCP
// when UDP fails or returns a truncated / empty response.
func (s *Server) LookupIPv4(ctx context.Context, host string) ([]string, string, error) {
	query := new(dns.Msg)
	query.SetQuestion(dns.Fqdn(host), dns.TypeA)
	query.RecursionDesired = true
	query.SetEdns0(1232, false)

	udpClient := &dns.Client{Net: "udp", UDPSize: 1232}
	udpResp, _, udpErr := udpClient.ExchangeContext(ctx, query.Copy(), s.upstream)
	udpIPs := extractIPv4Answers(udpResp)
	if udpErr == nil && udpResp != nil && udpResp.Rcode == dns.RcodeSuccess && !udpResp.Truncated && len(udpIPs) > 0 {
		return udpIPs, "udp", nil
	}

	tcpClient := &dns.Client{Net: "tcp"}
	tcpResp, _, tcpErr := tcpClient.ExchangeContext(ctx, query.Copy(), s.upstream)
	tcpIPs := extractIPv4Answers(tcpResp)
	if tcpErr == nil && tcpResp != nil && tcpResp.Rcode == dns.RcodeSuccess && len(tcpIPs) > 0 {
		return tcpIPs, "tcp-fallback", nil
	}

	if tcpErr != nil {
		return nil, "tcp-fallback", fmt.Errorf("udp lookup failed: %v; tcp lookup failed: %w", udpErr, tcpErr)
	}
	if tcpResp == nil {
		return nil, "tcp-fallback", fmt.Errorf("tcp lookup returned empty response for %s", host)
	}
	if tcpResp.Rcode != dns.RcodeSuccess {
		return nil, "tcp-fallback", fmt.Errorf("tcp lookup rcode=%s for %s", dns.RcodeToString[tcpResp.Rcode], host)
	}
	return nil, "tcp-fallback", fmt.Errorf("upstream DNS returned no IPv4 records for %s", host)
}

// ProcessQuery 接收一条原始 DNS 请求字节，处理后返回响应字节。
// 供 TUN 引擎在 gVisor UDP 层内联处理 DNS，无需经过网络。
func (s *Server) ProcessQuery(data []byte) []byte {
	msg := new(dns.Msg)
	if err := msg.Unpack(data); err != nil {
		return nil
	}
	resp := s.processMsg(msg)
	out, err := resp.Pack()
	if err != nil {
		return nil
	}
	return out
}

func (s *Server) processMsg(r *dns.Msg) *dns.Msg {
	if len(r.Question) == 0 {
		m := new(dns.Msg)
		m.SetRcode(r, dns.RcodeServerFailure)
		return m
	}
	q := r.Question[0]
	domain := dns.Fqdn(q.Name)
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
		fmt.Printf("[DNS] %s → FakeIP %s\n", domain, fakeIP)
		return m
	}

	// 非 A 记录转发到上游
	c := new(dns.Client)
	resp, _, err := c.Exchange(r, s.upstream)
	if err != nil {
		m.SetRcode(r, dns.RcodeServerFailure)
		return m
	}
	return resp
}

func (s *Server) handleDNS(w dns.ResponseWriter, r *dns.Msg) {
	resp := s.processMsg(r)
	w.WriteMsg(resp)
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
