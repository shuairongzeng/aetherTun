package main

import (
	"context"
	"fmt"
	"os/exec"
	"sync/atomic"

	"github.com/shuairongzeng/aether/internal/autostart"
	"github.com/shuairongzeng/aether/internal/config"
	"github.com/shuairongzeng/aether/internal/control"
	"github.com/shuairongzeng/aether/internal/gui"
	"github.com/shuairongzeng/aether/internal/logs"
	"github.com/shuairongzeng/aether/internal/paths"
	agentruntime "github.com/shuairongzeng/aether/internal/runtime"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx           context.Context
	controller    *gui.Controller
	trayHintShown bool
	quitting      atomic.Bool
}

type AppStatus struct {
	Phase          string `json:"phase"`
	ProxyEndpoint  string `json:"proxyEndpoint,omitempty"`
	TunAdapterName string `json:"tunAdapterName,omitempty"`
	LastErrorCode  string `json:"lastErrorCode,omitempty"`
	LastErrorText  string `json:"lastErrorText,omitempty"`
}

type SaveBasicProxySettingsResult struct {
	Settings        config.BasicProxySettings `json:"settings"`
	RequiresRestart bool                      `json:"requiresRestart"`
}

func NewApp() *App {
	return &App{
		controller: gui.NewDefaultController(),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	if a.quitting.Load() {
		return false
	}

	if !a.trayHintShown {
		wailsruntime.LogInfo(ctx, "Aether 已最小化到系统托盘")
		a.trayHintShown = true
	}

	wailsruntime.WindowHide(ctx)
	return true
}

func (a *App) GetStatus() AppStatus {
	status, err := a.controller.Status(a.requestContext())
	if err != nil {
		if control.IsUnavailableError(err) {
			return AppStatus{
				Phase: string(agentruntime.PhaseStopped),
			}
		}

		return AppStatus{
			Phase:         string(agentruntime.PhaseStopped),
			LastErrorCode: "core_unreachable",
			LastErrorText: err.Error(),
		}
	}

	appConfig, _ := config.LoadOrCreate(paths.DefaultPaths().ConfigFile)
	proxyEndpoint := ""
	tunAdapterName := ""
	if appConfig != nil {
		proxyEndpoint = fmt.Sprintf("%s://%s:%d", appConfig.Proxy.Type, appConfig.Proxy.Host, appConfig.Proxy.Port)
		tunAdapterName = appConfig.Tun.AdapterName
	}

	return AppStatus{
		Phase:          string(status.Phase),
		ProxyEndpoint:  proxyEndpoint,
		TunAdapterName: tunAdapterName,
		LastErrorCode:  status.LastErrorCode,
		LastErrorText:  status.LastErrorText,
	}
}

func (a *App) StartCore() error {
	return a.controller.StartCore(a.requestContext())
}

func (a *App) StopCore() error {
	return a.controller.StopCore(a.requestContext())
}

func (a *App) GetBasicProxySettings() (config.BasicProxySettings, error) {
	return config.LoadBasicProxySettings(paths.DefaultPaths().ConfigFile)
}

func (a *App) SaveBasicProxySettings(input config.BasicProxySettings) (SaveBasicProxySettingsResult, error) {
	saved, err := config.SaveBasicProxySettings(paths.DefaultPaths().ConfigFile, input)
	if err != nil {
		return SaveBasicProxySettingsResult{}, err
	}

	return SaveBasicProxySettingsResult{
		Settings: config.BasicProxySettings{
			Host: saved.Proxy.Host,
			Port: saved.Proxy.Port,
			Type: saved.Proxy.Type,
		},
		RequiresRestart: a.GetStatus().Phase == string(agentruntime.PhaseRunning),
	}, nil
}

func (a *App) GetOnboardingState() (config.OnboardingState, error) {
	return config.DetectOnboardingState(paths.DefaultPaths().ConfigFile)
}

func (a *App) GetRecentLogs(limit int) []logs.Entry {
	entries, err := a.controller.RecentLogs(a.requestContext(), limit)
	if err != nil {
		if control.IsUnavailableError(err) {
			return nil
		}

		return []logs.Entry{
			{
				Level:   logs.LevelError,
				Source:  "gui",
				Message: err.Error(),
			},
		}
	}

	return entries
}

func (a *App) requestContext() context.Context {
	if a.ctx != nil {
		return a.ctx
	}

	return context.Background()
}

func (a *App) CurrentRuntimeStatus() agentruntime.RuntimeStatus {
	status := a.GetStatus()
	return agentruntime.RuntimeStatus{
		Phase:         agentruntime.RuntimePhase(status.Phase),
		LastErrorCode: status.LastErrorCode,
		LastErrorText: status.LastErrorText,
	}
}

func (a *App) ShowWindow() {
	if a.ctx == nil {
		return
	}

	wailsruntime.WindowShow(a.ctx)
	wailsruntime.Show(a.ctx)
}

func (a *App) OpenLogDirectory() {
	appPaths := paths.DefaultPaths()
	_ = paths.EnsureAppDirs(appPaths)
	_ = exec.Command("explorer", appPaths.LogDir).Start()
}

func (a *App) OpenConfigFile() {
	appPaths := paths.DefaultPaths()
	_ = paths.EnsureAppDirs(appPaths)
	_, _ = config.LoadOrCreate(appPaths.ConfigFile)
	_ = exec.Command("notepad", appPaths.ConfigFile).Start()
}

func (a *App) ToggleAutoStart() (bool, error) {
	return autostart.Toggle()
}

func (a *App) GetAutoStartEnabled() bool {
	return autostart.IsEnabled()
}

func (a *App) Quit() {
	a.quitting.Store(true)

	if a.ctx == nil {
		return
	}

	wailsruntime.Quit(a.ctx)
}
