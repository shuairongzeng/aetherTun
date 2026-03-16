package runtime

import "context"

type RuntimePhase string

const (
	PhaseStopped  RuntimePhase = "stopped"
	PhaseStarting RuntimePhase = "starting"
	PhaseRunning  RuntimePhase = "running"
	PhaseStopping RuntimePhase = "stopping"
	PhaseError    RuntimePhase = "error"
)

type RuntimeStatus struct {
	LastErrorCode string
	Phase         RuntimePhase
	LastErrorText string
}

type StartStopper interface {
	Start() error
	Stop()
}

type Factory interface {
	Prepare(ctx context.Context) error
	NewRouter(ctx context.Context) (any, error)
	NewDNSServer(ctx context.Context) (StartStopper, error)
	NewTunEngine(ctx context.Context, router any, dnsServer StartStopper) (StartStopper, error)
}
