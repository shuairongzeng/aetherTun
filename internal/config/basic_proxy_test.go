package config

import (
	"path/filepath"
	"testing"
)

func TestValidateBasicProxySettingsRejectsInvalidPort(t *testing.T) {
	err := ValidateBasicProxySettings(BasicProxySettings{
		Host: "127.0.0.1",
		Port: 70000,
		Type: "socks5",
	})

	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestSaveBasicProxySettingsPreservesAdvancedSections(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := DefaultConfig()
	cfg.Tun.AdapterName = "Custom-TUN"
	cfg.DNS.Upstream = "1.1.1.1:53"

	if err := Save(path, cfg); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	saved, err := SaveBasicProxySettings(path, BasicProxySettings{
		Host: " 10.0.0.2 ",
		Port: 7890,
		Type: "http",
	})
	if err != nil {
		t.Fatalf("save basic proxy settings: %v", err)
	}

	if saved.Proxy.Host != "10.0.0.2" {
		t.Fatalf("expected trimmed host %q, got %q", "10.0.0.2", saved.Proxy.Host)
	}
	if saved.Proxy.Port != 7890 {
		t.Fatalf("expected port %d, got %d", 7890, saved.Proxy.Port)
	}
	if saved.Proxy.Type != "http" {
		t.Fatalf("expected type %q, got %q", "http", saved.Proxy.Type)
	}
	if saved.Tun.AdapterName != "Custom-TUN" {
		t.Fatalf("expected TUN adapter %q, got %q", "Custom-TUN", saved.Tun.AdapterName)
	}
	if saved.DNS.Upstream != "1.1.1.1:53" {
		t.Fatalf("expected DNS upstream %q, got %q", "1.1.1.1:53", saved.DNS.Upstream)
	}
	if saved.Routing.DefaultAction != cfg.Routing.DefaultAction {
		t.Fatalf("expected routing action %q, got %q", cfg.Routing.DefaultAction, saved.Routing.DefaultAction)
	}
	if saved.LogLevel != cfg.LogLevel {
		t.Fatalf("expected log level %q, got %q", cfg.LogLevel, saved.LogLevel)
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}

	if reloaded.Proxy.Host != "10.0.0.2" {
		t.Fatalf("expected persisted host %q, got %q", "10.0.0.2", reloaded.Proxy.Host)
	}
	if reloaded.Tun.AdapterName != "Custom-TUN" {
		t.Fatalf("expected persisted TUN adapter %q, got %q", "Custom-TUN", reloaded.Tun.AdapterName)
	}
	if reloaded.DNS.Upstream != "1.1.1.1:53" {
		t.Fatalf("expected persisted DNS upstream %q, got %q", "1.1.1.1:53", reloaded.DNS.Upstream)
	}
}
