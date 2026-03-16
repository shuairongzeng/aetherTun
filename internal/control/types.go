package control

import (
	"context"
	"time"

	"github.com/shuairongzeng/aether/internal/logs"
	"github.com/shuairongzeng/aether/internal/runtime"
)

type Manager interface {
	Status() runtime.RuntimeStatus
	Stop(ctx context.Context) error
}

type LogReader interface {
	Recent(limit int) []logs.Entry
}

type StatusResponse struct {
	Phase         runtime.RuntimePhase `json:"phase"`
	LastErrorCode string               `json:"lastErrorCode,omitempty"`
	LastErrorText string               `json:"lastErrorText,omitempty"`
}

type MetaResponse struct {
	Name      string    `json:"name"`
	PID       int       `json:"pid"`
	Timestamp time.Time `json:"timestamp"`
}

type RecentLogsResponse struct {
	Entries []logs.Entry `json:"entries"`
}

type StopResponse struct {
	OK bool `json:"ok"`
}
