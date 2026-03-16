package runtime

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/shuairongzeng/aether/internal/logs"
)

type Manager struct {
	factory Factory

	mu     sync.RWMutex
	status RuntimeStatus
	logs   *logs.Store

	router any
	dns    StartStopper
	tun    StartStopper
}

const defaultLogCapacity = 1000

func NewManager(factory Factory) *Manager {
	return &Manager{
		factory: factory,
		status:  RuntimeStatus{Phase: PhaseStopped},
		logs:    logs.NewStore(defaultLogCapacity),
	}
}

func (m *Manager) Status() RuntimeStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

func (m *Manager) RecentLogs(limit int) []logs.Entry {
	return m.logs.Recent(limit)
}

func (m *Manager) LogWriter(level, source string) io.Writer {
	return m.logs.Writer(level, source)
}

func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.status.Phase != PhaseStopped && m.status.Phase != PhaseError {
		phase := m.status.Phase
		m.mu.Unlock()
		return fmt.Errorf("runtime is %s", phase)
	}
	m.status = RuntimeStatus{Phase: PhaseStarting}
	m.mu.Unlock()
	m.appendInfo("runtime starting")

	if err := m.factory.Prepare(ctx); err != nil {
		m.fail("prepare_failed", err)
		return err
	}

	router, err := m.factory.NewRouter(ctx)
	if err != nil {
		m.fail("router_init_failed", err)
		return err
	}

	dnsServer, err := m.factory.NewDNSServer(ctx)
	if err != nil {
		stopIfPossible(router)
		m.fail("dns_init_failed", err)
		return err
	}

	tunEngine, err := m.factory.NewTunEngine(ctx, router, dnsServer)
	if err != nil {
		dnsServer.Stop()
		stopIfPossible(router)
		m.fail("tun_init_failed", err)
		return err
	}

	if err := tunEngine.Start(); err != nil {
		tunEngine.Stop()
		dnsServer.Stop()
		stopIfPossible(router)
		m.fail("tun_start_failed", err)
		return err
	}

	if err := dnsServer.Start(); err != nil {
		dnsServer.Stop()
		tunEngine.Stop()
		stopIfPossible(router)
		m.fail("dns_start_failed", err)
		return err
	}

	m.mu.Lock()
	m.router = router
	m.dns = dnsServer
	m.tun = tunEngine
	m.status = RuntimeStatus{Phase: PhaseRunning}
	m.mu.Unlock()
	m.appendInfo("runtime running")

	return nil
}

func (m *Manager) Stop(context.Context) error {
	m.mu.Lock()
	if m.status.Phase == PhaseStopped {
		m.mu.Unlock()
		return nil
	}

	dnsServer := m.dns
	tunEngine := m.tun
	router := m.router

	m.status.Phase = PhaseStopping
	m.mu.Unlock()
	m.appendInfo("runtime stopping")

	if dnsServer != nil {
		dnsServer.Stop()
	}
	if tunEngine != nil {
		tunEngine.Stop()
	}
	stopIfPossible(router)

	m.mu.Lock()
	m.router = nil
	m.dns = nil
	m.tun = nil
	m.status = RuntimeStatus{Phase: PhaseStopped}
	m.mu.Unlock()
	m.appendInfo("runtime stopped")

	return nil
}

func (m *Manager) fail(code string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.router = nil
	m.dns = nil
	m.tun = nil
	m.status = RuntimeStatus{
		LastErrorCode: code,
		Phase:         PhaseError,
		LastErrorText: err.Error(),
	}
	m.logs.Append(logs.Entry{
		Level:   logs.LevelError,
		Source:  "runtime",
		Message: "runtime failed",
	})
}

func stopIfPossible(component any) {
	if stopper, ok := component.(interface{ Stop() }); ok {
		stopper.Stop()
	}
}

func (m *Manager) appendInfo(message string) {
	m.logs.Append(logs.Entry{
		Level:   logs.LevelInfo,
		Source:  "runtime",
		Message: message,
	})
}
