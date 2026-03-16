package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPortableConfigExampleMatchesDefaultConfig(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve current test file path")
	}

	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	examplePath := filepath.Join(repoRoot, "packaging", "portable", "config.example.json")

	data, err := os.ReadFile(examplePath)
	if err != nil {
		t.Fatalf("read config example: %v", err)
	}

	var got Config
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal config example: %v", err)
	}

	gotJSON, err := json.MarshalIndent(&got, "", "  ")
	if err != nil {
		t.Fatalf("marshal normalized example config: %v", err)
	}

	wantJSON, err := json.MarshalIndent(DefaultConfig(), "", "  ")
	if err != nil {
		t.Fatalf("marshal default config: %v", err)
	}

	if string(gotJSON) != string(wantJSON) {
		t.Fatalf("portable example config does not match defaults\nwant:\n%s\n\ngot:\n%s", string(wantJSON), string(gotJSON))
	}
}
