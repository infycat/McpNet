//go:build !windows

package main

import "log"

func showStartupError(message string) {
	log.Print(message)
}
