package config

import (
	"encoding/json"
	"os"
)

type ProxyConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
	Type string `json:"type"` // socks5 | http
}

type TunConfig struct {
	Enabled     bool   `json:"enabled"`
	AdapterName string `json:"adapter_name"`
	Address     string `json:"address"`     // e.g. 198.18.0.1/15
	DNSListen   string `json:"dns_listen"`  // e.g. 198.18.0.2:53
	MTU         uint32 `json:"mtu"`
	AutoRoute   bool   `json:"auto_route"`
}

type DNSConfig struct {
	Mode       string `json:"mode"`        // fakeip | direct
	FakeIPCIDR string `json:"fakeip_cidr"` // e.g. 198.18.0.0/15
	Upstream   string `json:"upstream"`    // e.g. 8.8.8.8:53
}

type Rule struct {
	Type   string `json:"type"`   // process | cidr | domain
	Match  string `json:"match"`  // game.exe | 10.0.0.0/8 | *.local
	Action string `json:"action"` // proxy | direct | block
}

type RoutingConfig struct {
	DefaultAction     string `json:"default_action"`      // proxy | direct | block
	UseDefaultPrivate bool   `json:"use_default_private"` // auto direct for RFC1918
	Rules             []Rule `json:"rules"`
}

type Config struct {
	Proxy   ProxyConfig   `json:"proxy"`
	Tun     TunConfig     `json:"tun"`
	DNS     DNSConfig     `json:"dns"`
	Routing RoutingConfig `json:"routing"`
	LogLevel string       `json:"log_level"`
}

func DefaultConfig() *Config {
	return &Config{
		Proxy: ProxyConfig{
			Host: "127.0.0.1",
			Port: 10808,
			Type: "socks5",
		},
		Tun: TunConfig{
			Enabled:     true,
			AdapterName: "Aether-TUN",
			Address:     "198.18.0.1/15",
			DNSListen:   "198.18.0.2:53",
			MTU:         9000,
			AutoRoute:   true,
		},
		DNS: DNSConfig{
			Mode:       "fakeip",
			FakeIPCIDR: "198.18.0.0/15",
			Upstream:   "8.8.8.8:53",
		},
		Routing: RoutingConfig{
			DefaultAction:     "proxy",
			UseDefaultPrivate: true,
			Rules:             []Rule{},
		},
		LogLevel: "info",
	}
}

func Load(path string) (*Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func Save(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
