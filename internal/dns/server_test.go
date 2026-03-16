package dns

import (
	"context"
	"net"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	mdns "github.com/miekg/dns"
)

func waitForListener(t *testing.T, network, addr string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout(network, addr, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("listener %s on %s did not become ready", network, addr)
}

func startTCPDNSServer(t *testing.T, handler mdns.HandlerFunc) (string, *atomic.Int32, func()) {
	t.Helper()
	counter := &atomic.Int32{}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen tcp: %v", err)
	}
	addr := ln.Addr().String()
	srv := &mdns.Server{
		Net: "tcp",
		Listener: ln,
		Handler: mdns.HandlerFunc(func(w mdns.ResponseWriter, r *mdns.Msg) {
			counter.Add(1)
			handler(w, r)
		}),
	}
	go func() { _ = srv.ActivateAndServe() }()
	waitForListener(t, "tcp", addr)
	cleanup := func() { _ = srv.Shutdown() }
	t.Cleanup(cleanup)
	return addr, counter, cleanup
}

func startUDPDNSServer(t *testing.T, addr string, handler mdns.HandlerFunc) (string, *atomic.Int32, func()) {
	t.Helper()
	counter := &atomic.Int32{}
	if addr == "" {
		addr = "127.0.0.1:0"
	}
	pc, err := net.ListenPacket("udp", addr)
	if err != nil {
		t.Fatalf("listen udp: %v", err)
	}
	actualAddr := pc.LocalAddr().String()
	srv := &mdns.Server{
		Net: "udp",
		PacketConn: pc,
		Handler: mdns.HandlerFunc(func(w mdns.ResponseWriter, r *mdns.Msg) {
			counter.Add(1)
			handler(w, r)
		}),
	}
	go func() { _ = srv.ActivateAndServe() }()
	cleanup := func() { _ = srv.Shutdown() }
	t.Cleanup(cleanup)
	return actualAddr, counter, cleanup
}

func startSharedPortDNSServers(t *testing.T, tcpHandler, udpHandler mdns.HandlerFunc) (string, *atomic.Int32, *atomic.Int32, func()) {
	t.Helper()

	tcpLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen tcp: %v", err)
	}
	host, port, err := net.SplitHostPort(tcpLn.Addr().String())
	if err != nil {
		_ = tcpLn.Close()
		t.Fatalf("split host port: %v", err)
	}
	udpAddr := net.JoinHostPort(host, port)
	pc, err := net.ListenPacket("udp", udpAddr)
	if err != nil {
		_ = tcpLn.Close()
		t.Fatalf("listen udp shared port: %v", err)
	}

	tcpHits := &atomic.Int32{}
	udpHits := &atomic.Int32{}
	tcpSrv := &mdns.Server{
		Net: "tcp",
		Listener: tcpLn,
		Handler: mdns.HandlerFunc(func(w mdns.ResponseWriter, r *mdns.Msg) {
			tcpHits.Add(1)
			tcpHandler(w, r)
		}),
	}
	udpSrv := &mdns.Server{
		Net: "udp",
		PacketConn: pc,
		Handler: mdns.HandlerFunc(func(w mdns.ResponseWriter, r *mdns.Msg) {
			udpHits.Add(1)
			udpHandler(w, r)
		}),
	}
	go func() { _ = tcpSrv.ActivateAndServe() }()
	go func() { _ = udpSrv.ActivateAndServe() }()
	waitForListener(t, "tcp", net.JoinHostPort(host, port))
	cleanup := func() {
		_ = tcpSrv.Shutdown()
		_ = udpSrv.Shutdown()
	}
	t.Cleanup(cleanup)
	return net.JoinHostPort(host, port), tcpHits, udpHits, cleanup
}

func ipv4AnswerHandler(ip string) mdns.HandlerFunc {
	return func(w mdns.ResponseWriter, r *mdns.Msg) {
		resp := new(mdns.Msg)
		resp.SetReply(r)
		for _, q := range r.Question {
			if q.Qtype == mdns.TypeA {
				resp.Answer = append(resp.Answer, &mdns.A{
					Hdr: mdns.RR_Header{Name: q.Name, Rrtype: mdns.TypeA, Class: mdns.ClassINET, Ttl: 60},
					A:   net.ParseIP(ip).To4(),
				})
			}
		}
		_ = w.WriteMsg(resp)
	}
}

func aaaaAnswerHandler(ip string) mdns.HandlerFunc {
	return func(w mdns.ResponseWriter, r *mdns.Msg) {
		resp := new(mdns.Msg)
		resp.SetReply(r)
		for _, q := range r.Question {
			if q.Qtype == mdns.TypeAAAA {
				resp.Answer = append(resp.Answer, &mdns.AAAA{
					Hdr:  mdns.RR_Header{Name: q.Name, Rrtype: mdns.TypeAAAA, Class: mdns.ClassINET, Ttl: 60},
					AAAA: net.ParseIP(ip),
				})
			}
		}
		_ = w.WriteMsg(resp)
	}
}

func TestLookupIPv4PrefersTCPUpstreamByDefault(t *testing.T) {
	upstream, tcpHits, udpHits, _ := startSharedPortDNSServers(t, ipv4AnswerHandler("1.1.1.1"), ipv4AnswerHandler("8.8.8.8"))

	srv, err := NewServer("127.0.0.1:0", upstream, "198.18.0.0/15")
	if err != nil {
		t.Fatal(err)
	}

	ips, mode, err := srv.LookupIPv4(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("LookupIPv4 returned error: %v", err)
	}
	if mode != "tcp" {
		t.Fatalf("expected tcp mode, got %q", mode)
	}
	if len(ips) != 1 || ips[0] != "1.1.1.1" {
		t.Fatalf("expected tcp IPv4 answer 1.1.1.1, got %#v", ips)
	}
	if tcpHits.Load() == 0 {
		t.Fatal("expected tcp upstream to be used")
	}
	if udpHits.Load() != 0 {
		t.Fatalf("expected udp upstream to be skipped, got %d hits", udpHits.Load())
	}
}

func TestLookupIPv4FallsBackToUDPWhenTCPUnavailable(t *testing.T) {
	udpAddr, udpHits, _ := startUDPDNSServer(t, "", ipv4AnswerHandler("8.8.4.4"))

	srv, err := NewServer("127.0.0.1:0", udpAddr, "198.18.0.0/15")
	if err != nil {
		t.Fatal(err)
	}

	srv.SetUpstreamTransport("tcp")
	ips, mode, err := srv.LookupIPv4(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("LookupIPv4 returned error: %v", err)
	}
	if mode != "udp-fallback" {
		t.Fatalf("expected udp-fallback mode, got %q", mode)
	}
	if len(ips) != 1 || ips[0] != "8.8.4.4" {
		t.Fatalf("expected udp fallback IPv4 answer 8.8.4.4, got %#v", ips)
	}
	if udpHits.Load() == 0 {
		t.Fatal("expected udp fallback to be used")
	}
}

func TestProcessMsgPrefersTCPForForwardedRecords(t *testing.T) {
	upstream, tcpHits, udpHits, _ := startSharedPortDNSServers(t, aaaaAnswerHandler("2001:4860:4860::8888"), aaaaAnswerHandler("::1"))

	srv, err := NewServer("127.0.0.1:0", upstream, "198.18.0.0/15")
	if err != nil {
		t.Fatal(err)
	}

	query := new(mdns.Msg)
	query.SetQuestion(mdns.Fqdn("example.com"), mdns.TypeAAAA)
	resp := srv.processMsg(query)
	if resp == nil {
		t.Fatal("expected forwarded response, got nil")
	}
	if resp.Rcode != mdns.RcodeSuccess {
		t.Fatalf("expected success rcode, got %d", resp.Rcode)
	}
	if len(resp.Answer) != 1 {
		t.Fatalf("expected one AAAA answer, got %d", len(resp.Answer))
	}
	aaaa, ok := resp.Answer[0].(*mdns.AAAA)
	if !ok {
		t.Fatalf("expected AAAA answer, got %T", resp.Answer[0])
	}
	if got := aaaa.AAAA.String(); got != "2001:4860:4860::8888" {
		t.Fatalf("expected tcp AAAA answer, got %s", got)
	}
	if tcpHits.Load() == 0 {
		t.Fatal("expected tcp upstream to be used for forwarded records")
	}
	if udpHits.Load() != 0 {
		t.Fatalf("expected udp upstream to be skipped, got %d hits", udpHits.Load())
	}
}

func TestSharedPortFixtureUsesSamePort(t *testing.T) {
	upstream, _, _, _ := startSharedPortDNSServers(t, ipv4AnswerHandler("1.1.1.1"), ipv4AnswerHandler("8.8.8.8"))
	_, port, err := net.SplitHostPort(upstream)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := strconv.Atoi(port); err != nil {
		t.Fatalf("expected numeric port, got %q", port)
	}
}
