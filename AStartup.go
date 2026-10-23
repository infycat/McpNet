package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const applicationVersion = "1.2.1"

var startupLogPath string
var startupLogFile *os.File
var startupLogError error

func init() {
	// `go test` runs package init functions in its generated test executable.
	// Do not let test runs append misleading records to the user's app log.
	if isGoTestExecutable(os.Args[0]) {
		return
	}
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	startupLogPath, startupLogFile, startupLogError = openStartupLog()
	if startupLogFile == nil {
		log.SetOutput(os.Stdout)
		return
	}
	log.SetOutput(startupLogFile)
	log.Printf("========== McpNet startup ==========")
	log.Printf("Version=%s PID=%d Go=%s Platform=%s/%s", applicationVersion, os.Getpid(), runtime.Version(), runtime.GOOS, runtime.GOARCH)
	log.Printf("Startup diagnostic log: %s", startupLogPath)
	exe, err := os.Executable()
	if err != nil {
		log.Printf("Unable to resolve executable path: %v", err)
	} else {
		log.Printf("Executable: %s", exe)
	}
	if startupLogError != nil {
		log.Printf("Startup log used fallback path after earlier failure: %v", startupLogError)
	}
}

func isGoTestExecutable(path string) bool {
	name := strings.ToLower(filepath.Base(path))
	return strings.HasSuffix(name, ".test") || strings.HasSuffix(name, ".test.exe")
}

func startupLogCandidates() []string {
	const logRelativeDir = "McpNet_new/logs"
	candidates := make([]string, 0, 4)
	seen := make(map[string]struct{}, 4)
	fileName := startupLogFileName(time.Now(), os.Getpid())
	add := func(base string) {
		if strings.TrimSpace(base) == "" {
			return
		}
		path := filepath.Clean(filepath.Join(base, filepath.FromSlash(logRelativeDir), fileName))
		key := strings.ToLower(path)
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		candidates = append(candidates, path)
	}

	add(os.Getenv("LOCALAPPDATA"))
	if configDir, err := os.UserConfigDir(); err == nil {
		add(configDir)
	}
	if homeDir, err := os.UserHomeDir(); err == nil {
		add(homeDir)
	}
	add(os.TempDir())
	return candidates
}

func webviewDataPathCandidates() []string {
	const relative = "McpNet_new/WebView2"
	candidates := make([]string, 0, 4)
	seen := make(map[string]struct{}, 4)
	add := func(base string) {
		if strings.TrimSpace(base) == "" {
			return
		}
		path := filepath.Clean(filepath.Join(base, filepath.FromSlash(relative)))
		key := strings.ToLower(path)
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		candidates = append(candidates, path)
	}

	// Preserve an existing profile when possible, then prefer local writable
	// per-user storage over redirected home folders and finally the temp folder.
	if homeDir, err := os.UserHomeDir(); err == nil {
		existing := filepath.Join(homeDir, filepath.FromSlash(relative))
		if info, err := os.Stat(existing); err == nil && info.IsDir() {
			add(homeDir)
		}
	}
	add(os.Getenv("LOCALAPPDATA"))
	if homeDir, err := os.UserHomeDir(); err == nil {
		add(homeDir)
	}
	if configDir, err := os.UserConfigDir(); err == nil {
		add(configDir)
	}
	add(os.TempDir())
	return candidates
}

func prepareWebviewDataPath() (string, error) {
	candidates := webviewDataPathCandidates()
	if len(candidates) == 0 {
		return "", errors.New("no writable WebView2 data directory is available")
	}
	var failures []string
	for _, path := range candidates {
		if err := os.MkdirAll(path, 0700); err != nil {
			log.Printf("WebView2 data path unavailable: %s: %v", path, err)
			failures = append(failures, path+": "+err.Error())
			continue
		}
		log.Printf("Using WebView2 data path: %s", path)
		return path, nil
	}
	return "", errors.New(strings.Join(failures, "; "))
}

func startupLogFileName(start time.Time, pid int) string {
	return "McpNet_startup_" + start.Format("20060102_150405.000000000") + "_" + strconv.Itoa(pid) + ".log"
}

func openStartupLog() (string, *os.File, error) {
	candidates := startupLogCandidates()
	if len(candidates) == 0 {
		return "", nil, errors.New("no writable startup log location is available")
	}

	var failures []string
	for _, path := range candidates {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			failures = append(failures, path+": "+err.Error())
			continue
		}
		file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			failures = append(failures, path+": "+err.Error())
			continue
		}
		pruneStartupLogs(filepath.Dir(path), 50)
		if len(failures) > 0 {
			return path, file, errors.New(strings.Join(failures, "; "))
		}
		return path, file, nil
	}
	return candidates[0], nil, errors.New(strings.Join(failures, "; "))
}

// Keep a bounded set of per-run logs so repeated failed starts do not grow the
// profile indefinitely. Only files created by this diagnostic logger qualify.
func pruneStartupLogs(dir string, keep int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	type logEntry struct {
		path    string
		modTime time.Time
	}
	logs := make([]logEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "McpNet_startup_") || !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}
		info, err := entry.Info()
		if err == nil {
			logs = append(logs, logEntry{path: filepath.Join(dir, entry.Name()), modTime: info.ModTime()})
		}
	}
	if keep < 1 {
		keep = 1
	}
	sort.Slice(logs, func(i, j int) bool { return logs[i].modTime.After(logs[j].modTime) })
	for _, old := range logs[min(keep, len(logs)):] {
		_ = os.Remove(old.path)
	}
}
