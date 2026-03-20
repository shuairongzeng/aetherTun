//go:build windows

package autostart

import (
	"testing"

	"golang.org/x/sys/windows/registry"
)

func cleanup(t *testing.T) {
	t.Helper()
	key, err := registry.OpenKey(registry.CURRENT_USER, registryKeyPath, registry.SET_VALUE)
	if err != nil {
		return
	}
	defer key.Close()
	_ = key.DeleteValue(valueName)
}

func TestEnableAndDisable(t *testing.T) {
	cleanup(t)
	t.Cleanup(func() { cleanup(t) })

	if IsEnabled() {
		t.Fatal("expected auto-start to be disabled initially")
	}

	if err := Enable(); err != nil {
		t.Fatalf("Enable() failed: %v", err)
	}

	if !IsEnabled() {
		t.Fatal("expected auto-start to be enabled after Enable()")
	}

	if err := Disable(); err != nil {
		t.Fatalf("Disable() failed: %v", err)
	}

	if IsEnabled() {
		t.Fatal("expected auto-start to be disabled after Disable()")
	}
}

func TestToggle(t *testing.T) {
	cleanup(t)
	t.Cleanup(func() { cleanup(t) })

	enabled, err := Toggle()
	if err != nil {
		t.Fatalf("Toggle() failed: %v", err)
	}
	if !enabled {
		t.Fatal("expected Toggle() to enable auto-start")
	}
	if !IsEnabled() {
		t.Fatal("expected IsEnabled() to return true after toggle on")
	}

	enabled, err = Toggle()
	if err != nil {
		t.Fatalf("Toggle() failed: %v", err)
	}
	if enabled {
		t.Fatal("expected Toggle() to disable auto-start")
	}
	if IsEnabled() {
		t.Fatal("expected IsEnabled() to return false after toggle off")
	}
}
