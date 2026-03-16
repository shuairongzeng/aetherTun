package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type ProxyConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
	Type string `json:"type"`
}

type BasicProxySettings struct {
	Host string
	Port int
	Type string
}

type OnboardingState struct {
	ConfigExists         bool `json:"configExists"`
	IsDefaultProxyConfig bool `json:"isDefaultProxyConfig"`
	ShouldShowOnboarding bool `json:"shouldShowOnboarding"`
}

type TunConfig struct {
	Enabled        bool   `json:"enabled"`
	AdapterName    string `json:"adapter_name"`
	Address        string `json:"address"`
	DNSListen      string `json:"dns_listen"`
	MTU            uint32 `json:"mtu"`
	AutoRoute      bool   `json:"auto_route"`
	MaxUDPSessions int    `json:"max_udp_sessions"`
}

type DNSConfig struct {
	Mode       string `json:"mode"`
	FakeIPCIDR string `json:"fakeip_cidr"`
	Upstream   string `json:"upstream"`
	Transport  string `json:"transport,omitempty"`
}

type Rule struct {
	Type   string `json:"type"`
	Match  string `json:"match"`
	Action string `json:"action"`
}

type RoutingConfig struct {
	DefaultAction     string `json:"default_action"`
	UseDefaultPrivate bool   `json:"use_default_private"`
	Rules             []Rule `json:"rules"`
}

type Config struct {
	Proxy    ProxyConfig   `json:"proxy"`
	Tun      TunConfig     `json:"tun"`
	DNS      DNSConfig     `json:"dns"`
	Routing  RoutingConfig `json:"routing"`
	LogLevel string        `json:"log_level"`
}

func DefaultConfig() *Config {
	return &Config{
		Proxy: ProxyConfig{
			Host: "127.0.0.1",
			Port: 10808,
			Type: "socks5",
		},
		Tun: TunConfig{
			Enabled:        true,
			AdapterName:    "Aether-TUN",
			Address:        "198.18.0.1/15",
			DNSListen:      "198.18.0.2:53",
			MTU:            9000,
			AutoRoute:      true,
			MaxUDPSessions: 2048,
		},
		DNS: DNSConfig{
			Mode:       "fakeip",
			FakeIPCIDR: "198.18.0.0/15",
			Upstream:   "8.8.8.8:53",
			Transport:  "tcp",
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
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func LoadOrCreate(path string) (*Config, error) {
	_, err := os.Stat(path)
	switch {
	case err == nil:
		return Load(path)
	case !os.IsNotExist(err):
		return nil, err
	}

	cfg := DefaultConfig()
	if err := Save(path, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func LoadBasicProxySettings(path string) (BasicProxySettings, error) {
	cfg, err := LoadOrCreate(path)
	if err != nil {
		return BasicProxySettings{}, err
	}

	return BasicProxySettings{
		Host: cfg.Proxy.Host,
		Port: cfg.Proxy.Port,
		Type: cfg.Proxy.Type,
	}, nil
}

func ValidateBasicProxySettings(input BasicProxySettings) error {
	host := strings.TrimSpace(input.Host)
	if host == "" {
		return errors.New("proxy host is required")
	}

	if input.Port < 1 || input.Port > 65535 {
		return errors.New("proxy port must be between 1 and 65535")
	}

	switch strings.ToLower(strings.TrimSpace(input.Type)) {
	case "socks5", "http":
		return nil
	default:
		return errors.New("proxy type must be socks5 or http")
	}
}

func SaveBasicProxySettings(path string, input BasicProxySettings) (*Config, error) {
	normalized := BasicProxySettings{
		Host: strings.TrimSpace(input.Host),
		Port: input.Port,
		Type: strings.ToLower(strings.TrimSpace(input.Type)),
	}

	if err := ValidateBasicProxySettings(normalized); err != nil {
		return nil, err
	}

	cfg, err := LoadOrCreate(path)
	if err != nil {
		return nil, err
	}

	cfg.Proxy.Host = normalized.Host
	cfg.Proxy.Port = normalized.Port
	cfg.Proxy.Type = normalized.Type

	if err := Save(path, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func DetectOnboardingState(path string) (OnboardingState, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return OnboardingState{
				ConfigExists:         false,
				IsDefaultProxyConfig: true,
				ShouldShowOnboarding: true,
			}, nil
		}

		return OnboardingState{}, err
	}

	cfg, err := Load(path)
	if err != nil {
		return OnboardingState{}, err
	}

	defaultProxy := DefaultConfig().Proxy
	isDefaultProxyConfig := cfg.Proxy.Host == defaultProxy.Host &&
		cfg.Proxy.Port == defaultProxy.Port &&
		cfg.Proxy.Type == defaultProxy.Type

	return OnboardingState{
		ConfigExists:         true,
		IsDefaultProxyConfig: isDefaultProxyConfig,
		ShouldShowOnboarding: isDefaultProxyConfig,
	}, nil
}
