package control

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shuairongzeng/aether/internal/logs"
	"github.com/shuairongzeng/aether/internal/runtime"
)

type fakeManager struct {
	status     runtime.RuntimeStatus
	stopCalled bool
}

func newFakeManager(phase runtime.RuntimePhase) *fakeManager {
	return &fakeManager{
		status: runtime.RuntimeStatus{Phase: phase},
	}
}

func (m *fakeManager) Status() runtime.RuntimeStatus {
	return m.status
}

func (m *fakeManager) Stop(context.Context) error {
	m.stopCalled = true
	m.status.Phase = runtime.PhaseStopped
	return nil
}

func TestServerReturnsStatusAndStopsManager(t *testing.T) {
	manager := newFakeManager(runtime.PhaseRunning)
	store := logs.NewStore(10)
	store.Append(logs.Entry{Level: logs.LevelInfo, Source: "runtime", Message: "runtime running"})

	srv := NewServer(manager, store, "token-123")

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/v1/status", nil)
	if err != nil {
		t.Fatalf("NewRequest returned error: %v", err)
	}
	req.Header.Set("Authorization", "Bearer token-123")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var status StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if status.Phase != runtime.PhaseRunning {
		t.Fatalf("expected phase %q, got %q", runtime.PhaseRunning, status.Phase)
	}

	stopReq, err := http.NewRequest(http.MethodPost, ts.URL+"/v1/stop", strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("NewRequest returned error: %v", err)
	}
	stopReq.Header.Set("Authorization", "Bearer token-123")
	stopReq.Header.Set("Content-Type", "application/json")

	stopResp, err := http.DefaultClient.Do(stopReq)
	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	defer stopResp.Body.Close()

	if stopResp.StatusCode != http.StatusOK {
		t.Fatalf("expected stop status code %d, got %d", http.StatusOK, stopResp.StatusCode)
	}
	if !manager.stopCalled {
		t.Fatal("expected manager.Stop to be called")
	}
}

func TestServerRejectsUnauthorizedRequests(t *testing.T) {
	srv := NewServer(newFakeManager(runtime.PhaseRunning), logs.NewStore(10), "token-123")

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/v1/status")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, resp.StatusCode)
	}
}

func TestServerReturnsRecentLogs(t *testing.T) {
	store := logs.NewStore(10)
	store.Append(logs.Entry{Level: logs.LevelInfo, Source: "runtime", Message: "one"})
	store.Append(logs.Entry{Level: logs.LevelInfo, Source: "runtime", Message: "two"})

	srv := NewServer(newFakeManager(runtime.PhaseRunning), store, "token-123")
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/v1/logs/recent?limit=1", nil)
	if err != nil {
		t.Fatalf("NewRequest returned error: %v", err)
	}
	req.Header.Set("Authorization", "Bearer token-123")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var logsResponse RecentLogsResponse
	if err := json.NewDecoder(resp.Body).Decode(&logsResponse); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}

	if len(logsResponse.Entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(logsResponse.Entries))
	}
	if logsResponse.Entries[0].Message != "two" {
		t.Fatalf("expected recent log %q, got %q", "two", logsResponse.Entries[0].Message)
	}
}
