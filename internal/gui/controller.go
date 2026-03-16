package gui

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"

	"github.com/shuairongzeng/aether/internal/control"
	"github.com/shuairongzeng/aether/internal/launcher"
	"github.com/shuairongzeng/aether/internal/logs"
	"github.com/shuairongzeng/aether/internal/paths"
	"github.com/shuairongzeng/aether/internal/runtime"
)

const (
	DefaultControlPort  = 43129
	DefaultControlToken = "aether-dev-token"
)

type LaunchOptions = launcher.LaunchOptions

type Launcher interface {
	LaunchElevatedCore(corePath string, options LaunchOptions) error
}

type Client interface {
	Status(ctx context.Context) (control.StatusResponse, error)
	RecentLogs(ctx context.Context, limit int) (control.RecentLogsResponse, error)
	Stop(ctx context.Context) error
}

type Controller struct {
	launcher      Launcher
	client        Client
	corePath      string
	launchOptions LaunchOptions
}

type realLauncher struct{}

func (realLauncher) LaunchElevatedCore(corePath string, options LaunchOptions) error {
	return launcher.LaunchElevatedCore(corePath, options)
}

func NewController(launcher Launcher, client Client) *Controller {
	return &Controller{
		launcher: launcher,
		client:   client,
		corePath: defaultCorePath(),
		launchOptions: LaunchOptions{
			ConfigPath:  paths.DefaultPaths().ConfigFile,
			ControlPort: DefaultControlPort,
			Token:       DefaultControlToken,
		},
	}
}

func NewDefaultController() *Controller {
	baseURL := "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(DefaultControlPort))
	return NewController(realLauncher{}, control.NewClient(baseURL, DefaultControlToken))
}

func (c *Controller) StartCore(ctx context.Context) error {
	status, err := c.client.Status(ctx)
	if err == nil {
		switch status.Phase {
		case runtime.PhaseRunning, runtime.PhaseStarting:
			return nil
		}
	}

	if _, err := os.Stat(c.corePath); err != nil {
		return fmt.Errorf("aether-core.exe not found at %s", c.corePath)
	}

	return c.launcher.LaunchElevatedCore(c.corePath, c.launchOptions)
}

func (c *Controller) StopCore(ctx context.Context) error {
	return c.client.Stop(ctx)
}

func (c *Controller) Status(ctx context.Context) (control.StatusResponse, error) {
	return c.client.Status(ctx)
}

func (c *Controller) RecentLogs(ctx context.Context, limit int) ([]logs.Entry, error) {
	response, err := c.client.RecentLogs(ctx, limit)
	if err != nil {
		return nil, err
	}

	return response.Entries, nil
}

func defaultCorePath() string {
	exePath, err := os.Executable()
	if err != nil {
		return "aether-core.exe"
	}

	return resolveCorePath(exePath, func(path string) bool {
		_, err := os.Stat(path)
		return err == nil
	})
}

func resolveCorePath(exePath string, fileExists func(string) bool) string {
	candidates := []string{
		filepath.Join(filepath.Dir(exePath), "aether-core.exe"),
		filepath.Join(filepath.Dir(exePath), "..", "aether-core.exe"),
		filepath.Join(filepath.Dir(exePath), "..", "..", "aether-core.exe"),
		"aether-core.exe",
	}

	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		cleaned := filepath.Clean(candidate)
		if _, ok := seen[cleaned]; ok {
			continue
		}
		seen[cleaned] = struct{}{}

		if fileExists(cleaned) {
			return cleaned
		}
	}

	return filepath.Clean(filepath.Join(filepath.Dir(exePath), "aether-core.exe"))
}
