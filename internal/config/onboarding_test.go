package config

import (
	"path/filepath"
	"testing"
)

func TestShouldShowOnboardingWhenConfigMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")

	state, err := DetectOnboardingState(path)
	if err != nil {
		t.Fatalf("DetectOnboardingState error: %v", err)
	}

	if state.ConfigExists {
		t.Fatal("expected config file to be treated as missing")
	}
	if !state.IsDefaultProxyConfig {
		t.Fatal("expected missing config to be treated as default proxy config")
	}
	if !state.ShouldShowOnboarding {
		t.Fatal("expected onboarding when config is missing")
	}
}

func TestShouldHideOnboardingWhenProxyConfigIsCustomized(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := DefaultConfig()
	cfg.Proxy.Host = "10.0.0.2"

	if err := Save(path, cfg); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	state, err := DetectOnboardingState(path)
	if err != nil {
		t.Fatalf("DetectOnboardingState error: %v", err)
	}

	if !state.ConfigExists {
		t.Fatal("expected config file to exist")
	}
	if state.IsDefaultProxyConfig {
		t.Fatal("expected onboarding state to detect customized proxy config")
	}
	if state.ShouldShowOnboarding {
		t.Fatal("expected onboarding to stay hidden once proxy config is customized")
	}
}
