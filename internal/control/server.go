package control

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/shuairongzeng/aether/internal/logs"
)

type Server struct {
	manager Manager
	logs    LogReader
	token   string
}

func NewServer(manager Manager, store LogReader, token string) *Server {
	return &Server{
		manager: manager,
		logs:    store,
		token:   token,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/status", s.withAuth(s.handleStatus))
	mux.HandleFunc("/v1/meta", s.withAuth(s.handleMeta))
	mux.HandleFunc("/v1/logs/recent", s.withAuth(s.handleRecentLogs))
	mux.HandleFunc("/v1/stop", s.withAuth(s.handleStop))
	return mux
}

func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.authorized(r) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		next(w, r)
	}
}

func (s *Server) authorized(r *http.Request) bool {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth == "" || s.token == "" {
		return false
	}

	return auth == "Bearer "+s.token
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := s.manager.Status()
	writeJSON(w, http.StatusOK, StatusResponse{
		Phase:         status.Phase,
		LastErrorCode: status.LastErrorCode,
		LastErrorText: status.LastErrorText,
	})
}

func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, MetaResponse{
		Name:      "aether-core",
		PID:       os.Getpid(),
		Timestamp: time.Now(),
	})
}

func (s *Server) handleRecentLogs(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil {
			limit = parsed
		}
	}

	entries := []logs.Entry{}
	if s.logs != nil {
		entries = s.logs.Recent(limit)
	}

	writeJSON(w, http.StatusOK, RecentLogsResponse{Entries: entries})
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	if err := s.manager.Stop(context.Background()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, StopResponse{OK: true})
}

func writeJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(value)
}
