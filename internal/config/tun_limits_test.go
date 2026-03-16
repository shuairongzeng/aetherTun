package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfigIncludesTunMaxUDPSessions(t *testing.T) {
	data, err := json.Marshal(DefaultConfig())
	if err != nil {
		t.Fatalf("marshal default config: %v", err)
	}

	if !strings.Contains(string(data), `"max_udp_sessions"`) {
		t.Fatalf("expected default config JSON to include %q, got %s", "max_udp_sessions", string(data))
	}
}

func TestLoadReadsTunMaxUDPSessions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	raw := `{
  "proxy": {"host":"127.0.0.1","port":10808,"type":"socks5"},
  "tun": {
    "enabled": true,
    "adapter_name": "Aether-TUN",
    "address": "198.18.0.1/15",
    "dns_listen": "198.18.0.2:53",
    "mtu": 9000,
    "auto_route": true,
    "max_udp_sessions": 2048
  },
  "dns": {"mode":"fakeip","fakeip_cidr":"198.18.0.0/15","upstream":"8.8.8.8:53","transport":"tcp"},
  "routing": {"default_action":"proxy","use_default_private":true,"rules":[]},
  "log_level": "info"
}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal loaded config: %v", err)
	}

	if !strings.Contains(string(data), `"max_udp_sessions":2048`) {
		t.Fatalf("expected loaded config JSON to preserve max_udp_sessions, got %s", string(data))
	}
}
