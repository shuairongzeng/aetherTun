package runtime

import (
	"context"
	"errors"
	"testing"

	"github.com/shuairongzeng/aether/internal/logs"
)

type fakeFactory struct {
	started []string
	stopped []string
}

func newFakeFactory() *fakeFactory {
	return &fakeFactory{}
}

func (f *fakeFactory) Prepare(context.Context) error {
	return nil
}

func (f *fakeFactory) NewRouter(context.Context) (any, error) {
	f.started = append(f.started, "router")
	return &fakeRouter{factory: f}, nil
}

func (f *fakeFactory) NewDNSServer(context.Context) (StartStopper, error) {
	return &fakeStartStopper{name: "dns", factory: f}, nil
}

func (f *fakeFactory) NewTunEngine(context.Context, any, StartStopper) (StartStopper, error) {
	return &fakeStartStopper{name: "tun", factory: f}, nil
}

type fakeRouter struct {
	factory *fakeFactory
}

func (r *fakeRouter) Stop() {
	r.factory.stopped = append(r.factory.stopped, "router")
}

type fakeStartStopper struct {
	name    string
	factory *fakeFactory
}

func (s *fakeStartStopper) Start() error {
	s.factory.started = append(s.factory.started, s.name)
	return nil
}

func (s *fakeStartStopper) Stop() {
	s.factory.stopped = append(s.factory.stopped, s.name)
}

type failingFactory struct {
	err error
}

func (f *failingFactory) Prepare(context.Context) error {
	return nil
}

func (f *failingFactory) NewRouter(context.Context) (any, error) {
	return nil, f.err
}

func (f *failingFactory) NewDNSServer(context.Context) (StartStopper, error) {
	return nil, nil
}

func (f *failingFactory) NewTunEngine(context.Context, any, StartStopper) (StartStopper, error) {
	return nil, nil
}

func TestManagerTransitionsThroughStartAndStop(t *testing.T) {
	fake := newFakeFactory()
	manager := NewManager(fake)

	if manager.Status().Phase != PhaseStopped {
		t.Fatalf("expected initial phase %q, got %q", PhaseStopped, manager.Status().Phase)
	}

	if err := manager.Start(context.Background()); err != nil {
		t.Fatalf("start returned error: %v", err)
	}

	if manager.Status().Phase != PhaseRunning {
		t.Fatalf("expected running phase %q, got %q", PhaseRunning, manager.Status().Phase)
	}

	if err := manager.Stop(context.Background()); err != nil {
		t.Fatalf("stop returned error: %v", err)
	}

	if manager.Status().Phase != PhaseStopped {
		t.Fatalf("expected final phase %q, got %q", PhaseStopped, manager.Status().Phase)
	}

	expectedStarted := []string{"router", "tun", "dns"}
	if len(fake.started) != len(expectedStarted) {
		t.Fatalf("expected started sequence %v, got %v", expectedStarted, fake.started)
	}
	for index, want := range expectedStarted {
		if fake.started[index] != want {
			t.Fatalf("expected started[%d] = %q, got %q", index, want, fake.started[index])
		}
	}

	expectedStopped := []string{"dns", "tun", "router"}
	if len(fake.stopped) != len(expectedStopped) {
		t.Fatalf("expected stopped sequence %v, got %v", expectedStopped, fake.stopped)
	}
	for index, want := range expectedStopped {
		if fake.stopped[index] != want {
			t.Fatalf("expected stopped[%d] = %q, got %q", index, want, fake.stopped[index])
		}
	}
}

func TestManagerRecordsPhaseTransitionsAndErrors(t *testing.T) {
	manager := NewManager(&failingFactory{err: errors.New("router exploded")})

	err := manager.Start(context.Background())
	if err == nil {
		t.Fatal("expected start error, got nil")
	}

	status := manager.Status()
	if status.Phase != PhaseError {
		t.Fatalf("expected phase %q, got %q", PhaseError, status.Phase)
	}
	if status.LastErrorCode != "router_init_failed" {
		t.Fatalf("expected error code %q, got %q", "router_init_failed", status.LastErrorCode)
	}
	if status.LastErrorText != "router exploded" {
		t.Fatalf("expected error text %q, got %q", "router exploded", status.LastErrorText)
	}

	entries := manager.RecentLogs(10)
	if len(entries) != 2 {
		t.Fatalf("expected 2 recent log entries, got %d", len(entries))
	}
	if entries[0].Message != "runtime starting" {
		t.Fatalf("expected first log %q, got %q", "runtime starting", entries[0].Message)
	}
	if entries[1].Message != "runtime failed" {
		t.Fatalf("expected second log %q, got %q", "runtime failed", entries[1].Message)
	}
	if entries[1].Level != logs.LevelError {
		t.Fatalf("expected error log level %q, got %q", logs.LevelError, entries[1].Level)
	}
}

func TestManagerCapturesExternalLogWriterOutput(t *testing.T) {
	manager := NewManager(newFakeFactory())

	written, err := manager.LogWriter(logs.LevelInfo, "core").Write([]byte("first line\nsecond line\n\n"))
	if err != nil {
		t.Fatalf("write returned error: %v", err)
	}
	if written != len("first line\nsecond line\n\n") {
		t.Fatalf("expected written bytes %d, got %d", len("first line\nsecond line\n\n"), written)
	}

	entries := manager.RecentLogs(10)
	if len(entries) != 2 {
		t.Fatalf("expected 2 external log entries, got %d", len(entries))
	}
	if entries[0].Source != "core" {
		t.Fatalf("expected first log source %q, got %q", "core", entries[0].Source)
	}
	if entries[0].Message != "first line" {
		t.Fatalf("expected first log message %q, got %q", "first line", entries[0].Message)
	}
	if entries[1].Message != "second line" {
		t.Fatalf("expected second log message %q, got %q", "second line", entries[1].Message)
	}
}
