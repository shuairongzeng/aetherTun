//go:build windows

package tun

import (
	"strings"
	"testing"

	"golang.org/x/sys/windows"
	"golang.zx2c4.com/wintun"
)

func TestCreateAdapterSafeRecoversFromPanic(t *testing.T) {
	original := createAdapterFn
	t.Cleanup(func() {
		createAdapterFn = original
	})

	createAdapterFn = func(name string, tunnelType string, requestedGUID *windows.GUID) (*wintun.Adapter, error) {
		panic("boom")
	}

	adapter, err := createAdapterSafe("Aether-TUN")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if adapter != nil {
		t.Fatal("expected nil adapter on panic")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected panic text in error, got %v", err)
	}
}

func TestCommandRunnerHidesConsoleWindow(t *testing.T) {
	cmd := newHiddenCommand("powershell", "-NoProfile", "-Command", "Write-Output ok")
	if cmd.SysProcAttr == nil {
		t.Fatal("expected SysProcAttr to be configured")
	}
	if !cmd.SysProcAttr.HideWindow {
		t.Fatal("expected child process window to be hidden")
	}
	if cmd.SysProcAttr.CreationFlags&0x08000000 == 0 {
		t.Fatal("expected CREATE_NO_WINDOW flag to be set")
	}
}
