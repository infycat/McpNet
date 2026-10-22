package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const applicationVersion = "1.1.2"

var startupLogPath string
var startupLogFile *os.File
var startupLogError error

func init() {
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

func startupLogCandidates() []string {
	const logRelativePath = "McpNet_new/logs/McpNet_startup.log"
	candidates := make([]string, 0, 3)
	seen := make(map[string]struct{}, 3)
	add := func(base string) {
		if strings.TrimSpace(base) == "" {
			return
		}
		path := filepath.Clean(filepath.Join(base, filepath.FromSlash(logRelativePath)))
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
	add(os.TempDir())
	return candidates
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
		if len(failures) > 0 {
			return path, file, errors.New(strings.Join(failures, "; "))
		}
		return path, file, nil
	}
	return candidates[0], nil, errors.New(strings.Join(failures, "; "))
}
