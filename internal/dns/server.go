package dns

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	mdns "github.com/miekg/dns"
)

const defaultUpstreamTransport = "tcp"

type Server struct {
	listenAddr        string
	fakeIPMap         *FakeIPMap
	upstream          string
	upstreamTransport string
	server            *mdns.Server
	eventHook         func(event string, err error)
	localIP           net.IP
}

func NewServer(listenAddr, upstream, fakeIPCIDR string) (*Server, error) {
	fakeMap, err := NewFakeIPMap(fakeIPCIDR)
	if err != nil {
		return nil, fmt.Errorf("初始化 FakeIP 池失败: %w", err)
	}
	return &Server{
		listenAddr:        listenAddr,
		fakeIPMap:         fakeMap,
		upstream:          upstream,
		upstreamTransport: defaultUpstreamTransport,
	}, nil
}

func (s *Server) Start() error {
	pc, err := waitBind(s.listenAddr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("DNS 监听地址不可用 %s: %w", s.listenAddr, err)
	}

	mux := mdns.NewServeMux()
	mux.HandleFunc(".", s.handleDNS)
	s.server = &mdns.Server{
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

func (s *Server) SetLocalIP(ip net.IP) {
	s.localIP = ip
}

func (s *Server) SetUpstreamTransport(transport string) {
	s.upstreamTransport = normalizeUpstreamTransport(transport)
}

func (s *Server) FakeIPMap() *FakeIPMap {
	return s.fakeIPMap
}

func (s *Server) Upstream() string {
	return s.upstream
}

func extractIPv4Answers(resp *mdns.Msg) []string {
	if resp == nil {
		return nil
	}
	seen := make(map[string]struct{})
	ips := make([]string, 0, len(resp.Answer))
	for _, answer := range resp.Answer {
		a, ok := answer.(*mdns.A)
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

func (s *Server) LookupIPv4(ctx context.Context, host string) ([]string, string, error) {
	query := new(mdns.Msg)
	query.SetQuestion(mdns.Fqdn(host), mdns.TypeA)
	query.RecursionDesired = true
	query.SetEdns0(1232, false)

	networks := s.preferredNetworks()
	lastMode := networks[len(networks)-1] + "-fallback"
	failures := make([]string, 0, len(networks))

	for index, network := range networks {
		resp, err := s.exchangeUpstream(ctx, query.Copy(), network)
		mode := network
		if index > 0 {
			mode += "-fallback"
		}
		lastMode = mode

		if err != nil {
			failures = append(failures, fmt.Sprintf("%s exchange failed: %v", network, err))
			continue
		}
		if resp == nil {
			failures = append(failures, fmt.Sprintf("%s exchange returned empty response", network))
			continue
		}
		if resp.Rcode != mdns.RcodeSuccess {
			failures = append(failures, fmt.Sprintf("%s lookup rcode=%s for %s", network, mdns.RcodeToString[resp.Rcode], host))
			continue
		}
		if network == "udp" && resp.Truncated {
			failures = append(failures, fmt.Sprintf("%s lookup returned truncated response for %s", network, host))
			continue
		}

		ips := extractIPv4Answers(resp)
		if len(ips) == 0 {
			failures = append(failures, fmt.Sprintf("%s lookup returned no IPv4 records for %s", network, host))
			continue
		}
		return ips, mode, nil
	}

	return nil, lastMode, fmt.Errorf("%s", strings.Join(failures, "; "))
}

func (s *Server) ProcessQuery(data []byte) []byte {
	msg := new(mdns.Msg)
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

func (s *Server) processMsg(r *mdns.Msg) *mdns.Msg {
	if len(r.Question) == 0 {
		m := new(mdns.Msg)
		m.SetRcode(r, mdns.RcodeServerFailure)
		return m
	}
	q := r.Question[0]
	domain := mdns.Fqdn(q.Name)
	domain = domain[:len(domain)-1]

	m := new(mdns.Msg)
	m.SetReply(r)
	m.Authoritative = true

	if q.Qtype == mdns.TypeA {
		fakeIP := s.fakeIPMap.Assign(domain)
		m.Answer = append(m.Answer, &mdns.A{
			Hdr: mdns.RR_Header{
				Name:   q.Name,
				Rrtype: mdns.TypeA,
				Class:  mdns.ClassINET,
				Ttl:    1,
			},
			A: fakeIP,
		})
		fmt.Printf("[DNS] %s -> FakeIP %s\n", domain, fakeIP)
		return m
	}

	resp, _, err := s.exchangeUpstreamWithFallback(context.Background(), r)
	if err != nil {
		m.SetRcode(r, mdns.RcodeServerFailure)
		return m
	}
	return resp
}

func (s *Server) handleDNS(w mdns.ResponseWriter, r *mdns.Msg) {
	resp := s.processMsg(r)
	w.WriteMsg(resp)
}

func (s *Server) forward(w mdns.ResponseWriter, r *mdns.Msg) {
	resp, _, err := s.exchangeUpstreamWithFallback(context.Background(), r)
	if err != nil {
		mdns.HandleFailed(w, r)
		return
	}
	w.WriteMsg(resp)
}

func (s *Server) IsLocalAddr(ip net.IP) bool {
	listenIP, _, _ := net.SplitHostPort(s.listenAddr)
	return ip.String() == listenIP
}

func normalizeUpstreamTransport(transport string) string {
	switch strings.ToLower(strings.TrimSpace(transport)) {
	case "udp":
		return "udp"
	default:
		return defaultUpstreamTransport
	}
}

func (s *Server) preferredNetworks() []string {
	if normalizeUpstreamTransport(s.upstreamTransport) == "udp" {
		return []string{"udp", "tcp"}
	}
	return []string{"tcp", "udp"}
}

func (s *Server) exchangeUpstreamWithFallback(ctx context.Context, msg *mdns.Msg) (*mdns.Msg, string, error) {
	networks := s.preferredNetworks()
	lastMode := networks[len(networks)-1] + "-fallback"
	failures := make([]string, 0, len(networks))

	for index, network := range networks {
		resp, err := s.exchangeUpstream(ctx, msg.Copy(), network)
		mode := network
		if index > 0 {
			mode += "-fallback"
		}
		lastMode = mode

		if err != nil {
			failures = append(failures, fmt.Sprintf("%s exchange failed: %v", network, err))
			continue
		}
		if resp == nil {
			failures = append(failures, fmt.Sprintf("%s exchange returned empty response", network))
			continue
		}
		if network == "udp" && resp.Truncated {
			failures = append(failures, fmt.Sprintf("%s exchange returned truncated response", network))
			continue
		}
		return resp, mode, nil
	}

	return nil, lastMode, fmt.Errorf("%s", strings.Join(failures, "; "))
}

func (s *Server) exchangeUpstream(ctx context.Context, msg *mdns.Msg, network string) (*mdns.Msg, error) {
	client := &mdns.Client{Net: network}
	dialer := &net.Dialer{}
	if s.localIP != nil {
		if network == "tcp" {
			dialer.LocalAddr = &net.TCPAddr{IP: s.localIP}
		} else {
			dialer.LocalAddr = &net.UDPAddr{IP: s.localIP}
		}
	}
	if network == "udp" {
		client.UDPSize = 1232
	}
	client.Dialer = dialer

	resp, _, err := client.ExchangeContext(ctx, msg, s.upstream)
	return resp, err
}

