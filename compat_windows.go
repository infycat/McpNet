//go:build windows

package main

import (
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/Trisia/gosysproxy"
	gops "github.com/mitchellh/go-ps"
)

// findOriginalMcpNet reports whether another process named McpNet.exe is
// running. The original and rebuilt applications share the NetFilter driver,
// so only one of them may enable interception at a time.
func findOriginalMcpNet() (int, string, bool) {
	processes, err := gops.Processes()
	if err != nil {
		return 0, "", false
	}
	self := os.Getpid()
	for _, process := range processes {
		if process.Pid() == self {
			continue
		}
		name := process.Executable()
		if strings.EqualFold(name, "McpNet.exe") {
			return process.Pid(), name, true
		}
	}
	return 0, "", false
}

// cleanupStaleSystemProxy removes a dead loopback proxy left behind by an older
// McpNet build. It only touches 127.0.0.1:2024 (original) and 127.0.0.1:2025
// (rebuilt) and only when nothing is listening on that port.
func cleanupStaleSystemProxy() {
	if pid, name, ok := findOriginalMcpNet(); ok {
		log.Printf("Legacy proxy cleanup skipped: original %s is running (pid=%d)", name, pid)
		return
	}
	status, err := gosysproxy.Status()
	if err != nil {
		log.Printf("Legacy proxy cleanup: cannot read WinINet status: %v", err)
		return
	}
	if status == nil || status.Type != 3 {
		return
	}
	proxy := strings.ToLower(status.Proxy)
	for _, port := range []string{"2024", "2025"} {
		if !strings.Contains(proxy, "127.0.0.1:"+port) && !strings.Contains(proxy, "localhost:"+port) {
			continue
		}
		conn, dialErr := net.DialTimeout("tcp", "127.0.0.1:"+port, 300*time.Millisecond)
		if dialErr == nil {
			_ = conn.Close()
			log.Printf("Legacy proxy cleanup: %s is alive, keeping proxy %q", port, status.Proxy)
			return
		}
		log.Printf("Legacy proxy cleanup: disabling dead proxy %q (port %s is not listening)", status.Proxy, port)
		if err := gosysproxy.Off(); err != nil {
			log.Printf("Legacy proxy cleanup failed: %v", err)
		}
		return
	}
}
