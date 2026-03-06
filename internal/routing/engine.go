package routing

import (
	"net"
	"strings"

	"github.com/shuairongzeng/aether/internal/config"
)

type Action string

const (
	ActionProxy  Action = "proxy"
	ActionDirect Action = "direct"
	ActionBlock  Action = "block"
)

// 私有地址段（RFC1918 + loopback + link-local）
var privateRanges = []string{
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"::1/128",
	"fc00::/7",
}

type Engine struct {
	rules             []config.Rule
	privateNets       []*net.IPNet
	defaultAction     Action
	useDefaultPrivate bool
}

func New(cfg *config.RoutingConfig) *Engine {
	e := &Engine{
		rules:             cfg.Rules,
		defaultAction:     Action(cfg.DefaultAction),
		useDefaultPrivate: cfg.UseDefaultPrivate,
	}
	for _, cidr := range privateRanges {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err == nil {
			e.privateNets = append(e.privateNets, ipNet)
		}
	}
	return e
}

// Match 根据目标 IP、端口、域名、进程名判断路由动作
func (e *Engine) Match(ip net.IP, domain, process string) Action {
	// 私有地址直连
	if e.useDefaultPrivate && e.isPrivate(ip) {
		return ActionDirect
	}

	// 按顺序匹配规则
	for _, rule := range e.rules {
		if e.matchRule(rule, ip, domain, process) {
			return Action(rule.Action)
		}
	}

	return e.defaultAction
}

func (e *Engine) matchRule(rule config.Rule, ip net.IP, domain, process string) bool {
	switch rule.Type {
	case "cidr":
		_, ipNet, err := net.ParseCIDR(rule.Match)
		if err != nil || ip == nil {
			return false
		}
		return ipNet.Contains(ip)

	case "domain":
		if domain == "" {
			return false
		}
		pattern := rule.Match
		if strings.HasPrefix(pattern, "*.") {
			// *.example.com 只匹配子域名（sub.example.com），不匹配 example.com 本身
			suffix := pattern[1:] // ".example.com"
			return strings.HasSuffix(domain, suffix)
		}
		return domain == pattern

	case "process":
		if process == "" {
			return false
		}
		return strings.EqualFold(process, rule.Match)
	}
	return false
}

func (e *Engine) isPrivate(ip net.IP) bool {
	if ip == nil {
		return false
	}
	for _, network := range e.privateNets {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
