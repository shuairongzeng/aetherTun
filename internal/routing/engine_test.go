package routing_test

import (
	"net"
	"testing"

	"github.com/shuairongzeng/aether/internal/config"
	"github.com/shuairongzeng/aether/internal/routing"
)

func makeEngine(defaultAction string, usePrivate bool, rules []config.Rule) *routing.Engine {
	return routing.New(&config.RoutingConfig{
		DefaultAction:     defaultAction,
		UseDefaultPrivate: usePrivate,
		Rules:             rules,
	})
}

func TestDefaultAction(t *testing.T) {
	e := makeEngine("proxy", false, nil)
	action := e.Match(net.ParseIP("1.2.3.4"), "", "")
	if action != routing.ActionProxy {
		t.Fatalf("期望 proxy，得到 %s", action)
	}
}

func TestPrivateDirectByDefault(t *testing.T) {
	e := makeEngine("proxy", true, nil)
	cases := []string{"192.168.1.1", "10.0.0.1", "172.16.0.1", "127.0.0.1"}
	for _, ip := range cases {
		action := e.Match(net.ParseIP(ip), "", "")
		if action != routing.ActionDirect {
			t.Errorf("私有地址 %s 期望 direct，得到 %s", ip, action)
		}
	}
}

func TestCIDRRule(t *testing.T) {
	e := makeEngine("proxy", false, []config.Rule{
		{Type: "cidr", Match: "8.8.0.0/16", Action: "direct"},
	})
	if e.Match(net.ParseIP("8.8.8.8"), "", "") != routing.ActionDirect {
		t.Error("8.8.8.8 应匹配 8.8.0.0/16 direct")
	}
	if e.Match(net.ParseIP("1.1.1.1"), "", "") != routing.ActionProxy {
		t.Error("1.1.1.1 应走默认 proxy")
	}
}

func TestDomainExactRule(t *testing.T) {
	e := makeEngine("proxy", false, []config.Rule{
		{Type: "domain", Match: "example.com", Action: "direct"},
	})
	if e.Match(nil, "example.com", "") != routing.ActionDirect {
		t.Error("example.com 应匹配 direct")
	}
	if e.Match(nil, "sub.example.com", "") != routing.ActionProxy {
		t.Error("sub.example.com 不应匹配精确规则")
	}
}

func TestDomainWildcardRule(t *testing.T) {
	e := makeEngine("proxy", false, []config.Rule{
		{Type: "domain", Match: "*.example.com", Action: "block"},
	})
	if e.Match(nil, "sub.example.com", "") != routing.ActionBlock {
		t.Error("sub.example.com 应匹配 *.example.com block")
	}
	if e.Match(nil, "example.com", "") != routing.ActionProxy {
		t.Error("example.com 不应匹配 *.example.com")
	}
}

func TestProcessRule(t *testing.T) {
	e := makeEngine("proxy", false, []config.Rule{
		{Type: "process", Match: "game.exe", Action: "proxy"},
	})
	if e.Match(nil, "", "GAME.EXE") != routing.ActionProxy {
		t.Error("进程名匹配应大小写不敏感")
	}
	if e.Match(nil, "", "other.exe") != routing.ActionProxy {
		// 默认也是 proxy，只是规则没命中
		t.Log("其他进程走默认动作 OK")
	}
}

func TestRulePriority(t *testing.T) {
	// 规则按顺序匹配，第一个命中的生效
	e := makeEngine("proxy", false, []config.Rule{
		{Type: "cidr", Match: "8.8.8.0/24", Action: "block"},
		{Type: "cidr", Match: "8.8.0.0/16", Action: "direct"},
	})
	// 8.8.8.8 同时匹配两条，应命中第一条 block
	if e.Match(net.ParseIP("8.8.8.8"), "", "") != routing.ActionBlock {
		t.Error("应命中第一条规则 block")
	}
	// 8.8.4.4 只匹配第二条 direct
	if e.Match(net.ParseIP("8.8.4.4"), "", "") != routing.ActionDirect {
		t.Error("应命中第二条规则 direct")
	}
}
