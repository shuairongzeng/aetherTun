package gui

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/shuairongzeng/aether/internal/control"
	"github.com/shuairongzeng/aether/internal/logs"
	"github.com/shuairongzeng/aether/internal/runtime"
)

type fakeLauncher struct {
	called bool
}

func newFakeLauncher() *fakeLauncher {
	return &fakeLauncher{}
}

func (l *fakeLauncher) LaunchElevatedCore(string, LaunchOptions) error {
	l.called = true
	return nil
}

type fakeClient struct {
	status control.StatusResponse
}

func newFakeClient(phase runtime.RuntimePhase) *fakeClient {
	return &fakeClient{
		status: control.StatusResponse{Phase: phase},
	}
}

func (c *fakeClient) Status(context.Context) (control.StatusResponse, error) {
	return c.status, nil
}

func (c *fakeClient) RecentLogs(context.Context, int) (control.RecentLogsResponse, error) {
	return control.RecentLogsResponse{
		Entries: []logs.Entry{},
	}, nil
}

func (c *fakeClient) Stop(context.Context) error {
	c.status.Phase = runtime.PhaseStopped
	return nil
}

func TestControllerStartsCoreWhenNotRunning(t *testing.T) {
	launcher := newFakeLauncher()
	client := newFakeClient(runtime.PhaseStopped)
	controller := NewController(launcher, client)
	controller.corePath = filepath.Join(t.TempDir(), "aether-core.exe")
	if err := os.WriteFile(controller.corePath, []byte("stub"), 0o644); err != nil {
		t.Fatalf("failed to create fake core binary: %v", err)
	}

	err := controller.StartCore(context.Background())
	if err != nil {
		t.Fatalf("StartCore returned error: %v", err)
	}
	if !launcher.called {
		t.Fatal("expected launcher to be called")
	}
}

func TestResolveCorePathFallsBackToRepoRootBinaryForBuildBinGUI(t *testing.T) {
	exePath := filepath.Join(`C:\repo`, "build", "bin", "Aether.exe")
	want := filepath.Clean(filepath.Join(`C:\repo`, "aether-core.exe"))

	got := resolveCorePath(exePath, func(path string) bool {
		return filepath.Clean(path) == want
	})

	if got != want {
		t.Fatalf("expected fallback core path %q, got %q", want, got)
	}
}

func TestResolveCorePathPrefersSiblingBinary(t *testing.T) {
	exePath := filepath.Join(`C:\repo`, "build", "bin", "Aether.exe")
	want := filepath.Clean(filepath.Join(`C:\repo`, "build", "bin", "aether-core.exe"))

	got := resolveCorePath(exePath, func(path string) bool {
		return filepath.Clean(path) == want
	})

	if got != want {
		t.Fatalf("expected sibling core path %q, got %q", want, got)
	}
}
