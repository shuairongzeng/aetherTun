package logs

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Entry struct {
	Time    time.Time `json:"time"`
	Level   string    `json:"level"`
	Source  string    `json:"source"`
	Message string    `json:"message"`
}

const (
	LevelInfo  = "info"
	LevelError = "error"
)

type Store struct {
	mu       sync.RWMutex
	capacity int
	entries  []Entry
	file     *os.File
}

func NewStore(capacity int) *Store {
	if capacity <= 0 {
		capacity = 1
	}

	return &Store{
		capacity: capacity,
		entries:  make([]Entry, 0, capacity),
	}
}

func NewFileStore(capacity int, filePath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	store := NewStore(capacity)
	store.file = file
	return store, nil
}

func (s *Store) Append(entry Entry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry.Time.IsZero() {
		entry.Time = time.Now()
	}

	if len(s.entries) == s.capacity {
		copy(s.entries, s.entries[1:])
		s.entries[len(s.entries)-1] = entry
	} else {
		s.entries = append(s.entries, entry)
	}

	if s.file != nil {
		data, _ := json.Marshal(entry)
		s.file.Write(append(data, '\n'))
	}
}

func (s *Store) Writer(level, source string) io.Writer {
	return &lineWriter{
		store:  s,
		level:  level,
		source: source,
	}
}

func (s *Store) Recent(limit int) []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.entries) {
		limit = len(s.entries)
	}

	start := len(s.entries) - limit
	result := make([]Entry, limit)
	copy(result, s.entries[start:])
	return result
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.file == nil {
		return nil
	}

	err := s.file.Close()
	s.file = nil
	return err
}

type lineWriter struct {
	mu     sync.Mutex
	store  *Store
	level  string
	source string
	buffer string
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buffer += string(p)
	w.buffer = strings.ReplaceAll(w.buffer, "\r\n", "\n")

	lines := strings.Split(w.buffer, "\n")
	w.buffer = lines[len(lines)-1]

	for _, line := range lines[:len(lines)-1] {
		message := strings.TrimRight(line, "\r")
		if strings.TrimSpace(message) == "" {
			continue
		}

		w.store.Append(Entry{
			Level:   w.level,
			Source:  w.source,
			Message: message,
		})
	}

	return len(p), nil
}
