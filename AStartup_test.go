package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIsGoTestExecutable(t *testing.T) {
	for _, path := range []string{
		`C:\build\main.test.exe`,
		`/tmp/main.test`,
	} {
		if !isGoTestExecutable(path) {
			t.Errorf("isGoTestExecutable(%q) = false, want true", path)
		}
	}
	for _, path := range []string{
		`C:\Program Files\McpNet.exe`,
		`/opt/McpNet`,
	} {
		if isGoTestExecutable(path) {
			t.Errorf("isGoTestExecutable(%q) = true, want false", path)
		}
	}
}

func TestOpenStartupLogUsesLocalAppData(t *testing.T) {
	localAppData := t.TempDir()
	t.Setenv("LOCALAPPDATA", localAppData)

	path, file, err := openStartupLog()
	if err != nil {
		t.Fatalf("openStartupLog() error = %v", err)
	}
	defer file.Close()

	wantDir := filepath.Join(localAppData, "McpNet_new", "logs")
	if filepath.Dir(path) != wantDir {
		t.Fatalf("openStartupLog() directory = %q, want %q", filepath.Dir(path), wantDir)
	}
	if !strings.HasPrefix(filepath.Base(path), "McpNet_startup_") || !strings.HasSuffix(path, ".log") {
		t.Fatalf("openStartupLog() path = %q, want a unique per-start log filename", path)
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
	if got := filepath.Dir(candidates[0]); got != filepath.Join(localAppData, "McpNet_new", "logs") {
		t.Fatalf("primary candidate directory = %q, want LocalAppData logs folder", got)
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

func TestStartupLogFileNameIsUniquePerRun(t *testing.T) {
	when := time.Date(2026, 10, 23, 12, 30, 0, 123456789, time.UTC)
	first := startupLogFileName(when, 10)
	second := startupLogFileName(when.Add(time.Nanosecond), 10)
	if first == second {
		t.Fatalf("startup log filename did not include enough timestamp precision")
	}
	if got := startupLogFileName(when, 11); got == first {
		t.Fatalf("startup log filename did not include the process ID")
	}
}

func TestPrepareWebviewDataPathUsesLocalAppDataWhenHomeIsMissing(t *testing.T) {
	localAppData := t.TempDir()
	t.Setenv("LOCALAPPDATA", localAppData)
	t.Setenv("USERPROFILE", filepath.Join(t.TempDir(), "missing-home"))

	path, err := prepareWebviewDataPath()
	if err != nil {
		t.Fatalf("prepareWebviewDataPath() error = %v", err)
	}
	want := filepath.Join(localAppData, "McpNet_new", "WebView2")
	if path != want {
		t.Fatalf("prepareWebviewDataPath() = %q, want %q", path, want)
	}
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		t.Fatalf("WebView2 data directory was not created: %v", err)
	}
}

func TestPruneStartupLogsKeepsNewestAndIgnoresOtherFiles(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 4; i++ {
		name := startupLogFileName(time.Unix(int64(i+1), 0), i+1)
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("test"), 0600); err != nil {
			t.Fatal(err)
		}
		stamp := time.Unix(int64(i+1), 0)
		if err := os.Chtimes(path, stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}
	other := filepath.Join(dir, "keep-me.log")
	if err := os.WriteFile(other, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}

	pruneStartupLogs(dir, 2)
	for i := 0; i < 4; i++ {
		path := filepath.Join(dir, startupLogFileName(time.Unix(int64(i+1), 0), i+1))
		_, err := os.Stat(path)
		wantPresent := i >= 2
		if wantPresent && err != nil {
			t.Errorf("newest log %q was removed: %v", path, err)
		}
		if !wantPresent && !os.IsNotExist(err) {
			t.Errorf("old log %q was not removed", path)
		}
	}
	if _, err := os.Stat(other); err != nil {
		t.Errorf("unrelated log was removed: %v", err)
	}
}
