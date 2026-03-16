package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLiveFactoryCreatesDefaultConfigWhenMissing(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	factory := NewLiveFactory(configPath)
	cfg, err := factory.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if cfg == nil {
		t.Fatal("expected config, got nil")
	}

	if cfg.Proxy.Host != "127.0.0.1" {
		t.Fatalf("expected default proxy host %q, got %q", "127.0.0.1", cfg.Proxy.Host)
	}

	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("expected config file to be created at %q: %v", configPath, err)
	}
}

func TestNewLiveFactoryUsesDefaultAppPathWhenConfigPathEmpty(t *testing.T) {
	t.Setenv("LOCALAPPDATA", `C:\Users\Test\AppData\Local`)

	factory := NewLiveFactory("")

	if factory.configPath != `C:\Users\Test\AppData\Local\Aether\config.json` {
		t.Fatalf("expected default config path %q, got %q", `C:\Users\Test\AppData\Local\Aether\config.json`, factory.configPath)
	}
}
