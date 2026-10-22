package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenStartupLogUsesLocalAppData(t *testing.T) {
	localAppData := t.TempDir()
	t.Setenv("LOCALAPPDATA", localAppData)

	path, file, err := openStartupLog()
	if err != nil {
		t.Fatalf("openStartupLog() error = %v", err)
	}
	defer file.Close()

	want := filepath.Join(localAppData, "McpNet_new", "logs", "McpNet_startup.log")
	if path != want {
		t.Fatalf("openStartupLog() path = %q, want %q", path, want)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("startup log was not created: %v", err)
	}
}

func TestStartupLogCandidatesDeduplicatesLocations(t *testing.T) {
	localAppData := t.TempDir()
	t.Setenv("LOCALAPPDATA", localAppData)
	t.Setenv("APPDATA", localAppData)

	candidates := startupLogCandidates()
	if len(candidates) < 2 {
		t.Fatalf("startupLogCandidates() returned %d candidates, want primary and fallback", len(candidates))
	}
	if got, want := candidates[0], filepath.Join(localAppData, "McpNet_new", "logs", "McpNet_startup.log"); got != want {
		t.Fatalf("primary candidate = %q, want %q", got, want)
	}
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		key := filepath.Clean(candidate)
		if _, exists := seen[key]; exists {
			t.Fatalf("duplicate startup log candidate: %q", candidate)
		}
		seen[key] = struct{}{}
	}
}
