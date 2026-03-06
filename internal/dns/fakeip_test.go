package dns_test

import (
	"net"
	"testing"

	"github.com/shuairongzeng/aether/internal/dns"
)

func TestFakeIPAssignAndLookup(t *testing.T) {
	m, err := dns.NewFakeIPMap("198.18.0.0/15")
	if err != nil {
		t.Fatal(err)
	}

	ip1 := m.Assign("example.com")
	ip2 := m.Assign("github.com")

	if ip1.Equal(ip2) {
		t.Error("不同域名应分配不同 FakeIP")
	}

	// 同一域名复用同一 IP
	ip1b := m.Assign("example.com")
	if !ip1.Equal(ip1b) {
		t.Error("同域名应返回相同 FakeIP")
	}

	// 反向查找
	domain, ok := m.LookupDomain(ip1)
	if !ok || domain != "example.com" {
		t.Errorf("反向查找失败: ok=%v domain=%s", ok, domain)
	}

	domain2, ok := m.LookupDomain(ip2)
	if !ok || domain2 != "github.com" {
		t.Errorf("反向查找失败: ok=%v domain=%s", ok, domain2)
	}
}

func TestFakeIPInCIDR(t *testing.T) {
	m, err := dns.NewFakeIPMap("198.18.0.0/15")
	if err != nil {
		t.Fatal(err)
	}

	_, cidr, _ := net.ParseCIDR("198.18.0.0/15")

	for i := 0; i < 100; i++ {
		domain := "test" + string(rune('a'+i%26)) + ".com"
		ip := m.Assign(domain)
		if !cidr.Contains(ip) {
			t.Errorf("FakeIP %s 不在 198.18.0.0/15 范围内", ip)
		}
	}
}

func TestIsFakeIP(t *testing.T) {
	m, err := dns.NewFakeIPMap("198.18.0.0/15")
	if err != nil {
		t.Fatal(err)
	}

	fakeIP := m.Assign("example.com")
	if !m.IsFakeIP(fakeIP) {
		t.Error("已分配的 FakeIP 应返回 true")
	}
	if m.IsFakeIP(net.ParseIP("8.8.8.8")) {
		t.Error("外部 IP 不应被认为是 FakeIP")
	}
}

func TestLookupUnknownIP(t *testing.T) {
	m, _ := dns.NewFakeIPMap("198.18.0.0/15")
	_, ok := m.LookupDomain(net.ParseIP("198.18.100.100"))
	if ok {
		t.Error("未分配的 IP 查找应返回 false")
	}
}
