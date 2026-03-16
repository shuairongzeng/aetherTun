package logs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStoreKeepsRecentEntriesInOrder(t *testing.T) {
	store := NewStore(3)

	store.Append(Entry{Message: "1"})
	store.Append(Entry{Message: "2"})
	store.Append(Entry{Message: "3"})
	store.Append(Entry{Message: "4"})

	entries := store.Recent(3)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	got := []string{entries[0].Message, entries[1].Message, entries[2].Message}
	want := []string{"2", "3", "4"}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("expected entries %v, got %v", want, got)
		}
	}
}

func TestFileStoreWritesEntriesToDisk(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "aether.log")

	store, err := NewFileStore(5, logPath)
	if err != nil {
		t.Fatalf("NewFileStore returned error: %v", err)
	}
	defer store.Close()

	store.Append(Entry{
		Time:    time.Now(),
		Level:   "info",
		Source:  "runtime",
		Message: "runtime started",
	})

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "runtime started") {
		t.Fatalf("expected log file to contain message %q, got %q", "runtime started", content)
	}
}
