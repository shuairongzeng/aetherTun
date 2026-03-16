package main

import "testing"

func TestParseCoreFlags(t *testing.T) {
	cfg := parseFlags([]string{
		"--config", "x.json",
		"--control-port", "43129",
		"--token", "abc",
	})

	if cfg.ConfigPath != "x.json" {
		t.Fatalf("expected config path %q, got %q", "x.json", cfg.ConfigPath)
	}
	if cfg.ControlPort != 43129 {
		t.Fatalf("expected control port %d, got %d", 43129, cfg.ControlPort)
	}
	if cfg.Token != "abc" {
		t.Fatalf("expected token %q, got %q", "abc", cfg.Token)
	}
}
