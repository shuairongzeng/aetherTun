package runtime

import (
	"context"
	"fmt"
	"sync"

	"github.com/shuairongzeng/aether/internal/config"
	"github.com/shuairongzeng/aether/internal/dns"
	"github.com/shuairongzeng/aether/internal/paths"
	"github.com/shuairongzeng/aether/internal/routing"
	"github.com/shuairongzeng/aether/internal/tun"
)

type LiveFactory struct {
	configPath  string
	prepareHook func() error

	mu  sync.Mutex
	cfg *config.Config
}

func NewLiveFactory(configPath string) *LiveFactory {
	if configPath == "" {
		configPath = paths.DefaultPaths().ConfigFile
	}

	return &LiveFactory{configPath: configPath}
}

func (f *LiveFactory) SetPrepareHook(hook func() error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.prepareHook = hook
}

func (f *LiveFactory) LoadConfig() (*config.Config, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.cfg != nil {
		return f.cfg, nil
	}

	cfg, err := config.LoadOrCreate(f.configPath)
	if err != nil {
		return nil, err
	}

	f.cfg = cfg
	return f.cfg, nil
}

func (f *LiveFactory) Prepare(ctx context.Context) error {
	f.mu.Lock()
	prepareHook := f.prepareHook
	f.mu.Unlock()

	if prepareHook != nil {
		if err := prepareHook(); err != nil {
			return fmt.Errorf("释放 wintun.dll 失败: %w", err)
		}
	}

	if _, err := f.LoadConfig(); err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	return nil
}

func (f *LiveFactory) NewRouter(ctx context.Context) (any, error) {
	cfg, err := f.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	return routing.New(&cfg.Routing), nil
}

func (f *LiveFactory) NewDNSServer(ctx context.Context) (StartStopper, error) {
	cfg, err := f.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	dnsServer, err := dns.NewServer(
		cfg.Tun.DNSListen,
		cfg.DNS.Upstream,
		cfg.DNS.FakeIPCIDR,
	)
	if err != nil {
		return nil, fmt.Errorf("初始化 DNS 失败: %w", err)
	}
	dnsServer.SetUpstreamTransport(cfg.DNS.Transport)

	return dnsServer, nil
}

func (f *LiveFactory) NewTunEngine(ctx context.Context, router any, dnsServer StartStopper) (StartStopper, error) {
	cfg, err := f.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	routerEngine, ok := router.(*routing.Engine)
	if !ok {
		return nil, fmt.Errorf("路由引擎类型错误: %T", router)
	}

	dnsRuntime, ok := dnsServer.(*dns.Server)
	if !ok {
		return nil, fmt.Errorf("DNS 服务类型错误: %T", dnsServer)
	}

	return tun.New(cfg, dnsRuntime, routerEngine), nil
}
