package main

import (
	"log"
	"os"
	"path/filepath"
)

var startupLogPath string
var startupLogFile *os.File

func init() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	startupLogPath = filepath.Join(filepath.Dir(exe), "McpNet_new_startup.log")
	startupLogFile, err = os.OpenFile(startupLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		startupLogPath = ""
		return
	}
	log.SetOutput(startupLogFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	log.Printf("Process initialized: %s", exe)
}
