package main

import (
	"context"
	"errors"
	"testing"

	"github.com/shuairongzeng/aether/internal/config"
	"github.com/shuairongzeng/aether/internal/control"
	"github.com/shuairongzeng/aether/internal/gui"
	"github.com/shuairongzeng/aether/internal/logs"
	"github.com/shuairongzeng/aether/internal/paths"
	"github.com/shuairongzeng/aether/internal/runtime"
)

type unavailableClient struct{}

func (unavailableClient) Status(context.Context) (control.StatusResponse, error) {
	return control.StatusResponse{}, control.ErrUnavailable
}

func (unavailableClient) RecentLogs(context.Context, int) (control.RecentLogsResponse, error) {
	return control.RecentLogsResponse{}, control.ErrUnavailable
}

func (unavailableClient) Stop(context.Context) error {
	return nil
}

type failingLogsClient struct{}

func (failingLogsClient) Status(context.Context) (control.StatusResponse, error) {
	return control.StatusResponse{Phase: runtime.PhaseStopped}, nil
}

func (failingLogsClient) RecentLogs(context.Context, int) (control.RecentLogsResponse, error) {
	return control.RecentLogsResponse{}, errors.New("recent logs failed")
}

func (failingLogsClient) Stop(context.Context) error {
	return nil
}

type runningClient struct{}

func (runningClient) Status(context.Context) (control.StatusResponse, error) {
	return control.StatusResponse{Phase: runtime.PhaseRunning}, nil
}

func (runningClient) RecentLogs(context.Context, int) (control.RecentLogsResponse, error) {
	return control.RecentLogsResponse{}, nil
}

func (runningClient) Stop(context.Context) error {
	return nil
}

func TestGetStatusSuppressesUnavailableCoreError(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())

	app := &App{
		controller: gui.NewController(guiTestLauncher{}, unavailableClient{}),
	}

	status := app.GetStatus()
	if status.Phase != string(runtime.PhaseStopped) {
		t.Fatalf("expected stopped phase, got %q", status.Phase)
	}
	if status.LastErrorCode != "" {
		t.Fatalf("expected empty error code, got %q", status.LastErrorCode)
	}
	if status.LastErrorText != "" {
		t.Fatalf("expected empty error text, got %q", status.LastErrorText)
	}
}

func TestGetRecentLogsReturnsEmptyWhenCoreUnavailable(t *testing.T) {
	app := &App{
		controller: gui.NewController(guiTestLauncher{}, unavailableClient{}),
	}

	entries := app.GetRecentLogs(20)
	if len(entries) != 0 {
		t.Fatalf("expected no log entries, got %d", len(entries))
	}
}

func TestGetRecentLogsStillReturnsUnexpectedErrors(t *testing.T) {
	app := &App{
		controller: gui.NewController(guiTestLauncher{}, failingLogsClient{}),
	}

	entries := app.GetRecentLogs(20)
	if len(entries) != 1 {
		t.Fatalf("expected 1 error log entry, got %d", len(entries))
	}
	if entries[0].Level != logs.LevelError {
		t.Fatalf("expected error level %q, got %q", logs.LevelError, entries[0].Level)
	}
	if entries[0].Message != "recent logs failed" {
		t.Fatalf("expected error message %q, got %q", "recent logs failed", entries[0].Message)
	}
}

func TestBeforeCloseAllowsQuitToProceed(t *testing.T) {
	app := &App{}
	app.quitting.Store(true)

	if prevent := app.beforeClose(context.Background()); prevent {
		t.Fatal("expected quit-triggered close to proceed")
	}
}

func TestGetBasicProxySettingsLoadsCurrentConfig(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())

	cfg := config.DefaultConfig()
	cfg.Proxy.Host = "192.168.1.2"
	cfg.Proxy.Port = 8899
	cfg.Proxy.Type = "http"
	if err := config.Save(paths.DefaultPaths().ConfigFile, cfg); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	app := &App{
		controller: gui.NewController(guiTestLauncher{}, unavailableClient{}),
	}

	got, err := app.GetBasicProxySettings()
	if err != nil {
		t.Fatalf("GetBasicProxySettings error: %v", err)
	}

	if got.Host != "192.168.1.2" {
		t.Fatalf("expected host %q, got %q", "192.168.1.2", got.Host)
	}
	if got.Port != 8899 {
		t.Fatalf("expected port %d, got %d", 8899, got.Port)
	}
	if got.Type != "http" {
		t.Fatalf("expected type %q, got %q", "http", got.Type)
	}
}

func TestSaveBasicProxySettingsMarksRunningConfigForRestart(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())

	app := &App{
		controller: gui.NewController(guiTestLauncher{}, runningClient{}),
	}

	result, err := app.SaveBasicProxySettings(config.BasicProxySettings{
		Host: "127.0.0.1",
		Port: 7890,
		Type: "socks5",
	})
	if err != nil {
		t.Fatalf("SaveBasicProxySettings error: %v", err)
	}

	if !result.RequiresRestart {
		t.Fatal("expected requires restart when runtime is running")
	}
	if result.Settings.Host != "127.0.0.1" {
		t.Fatalf("expected saved host %q, got %q", "127.0.0.1", result.Settings.Host)
	}
	if result.Settings.Port != 7890 {
		t.Fatalf("expected saved port %d, got %d", 7890, result.Settings.Port)
	}
	if result.Settings.Type != "socks5" {
		t.Fatalf("expected saved type %q, got %q", "socks5", result.Settings.Type)
	}
}

func TestGetOnboardingStateUsesDefaultPathsConfig(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())

	app := &App{
		controller: gui.NewController(guiTestLauncher{}, unavailableClient{}),
	}

	state, err := app.GetOnboardingState()
	if err != nil {
		t.Fatalf("GetOnboardingState error: %v", err)
	}

	if !state.ShouldShowOnboarding {
		t.Fatal("expected onboarding on first run")
	}
	if state.ConfigExists {
		t.Fatal("expected config to be missing on first run")
	}
}

type guiTestLauncher struct{}

func (guiTestLauncher) LaunchElevatedCore(string, gui.LaunchOptions) error {
	return nil
}
