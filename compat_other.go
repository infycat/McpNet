//go:build !windows

package main

func findOriginalMcpNet() (int, string, bool) {
	return 0, "", false
}

func cleanupStaleSystemProxy() {}
