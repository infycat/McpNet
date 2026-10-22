//go:build windows

package main

import (
	"log"

	"github.com/wailsapp/go-webview2/webviewloader"
)

func logWebView2Runtime() {
	version, err := webviewloader.GetAvailableCoreWebView2BrowserVersionString("")
	if err != nil {
		log.Printf("Unable to detect the installed WebView2 Runtime: %v", err)
		return
	}
	if version == "" {
		log.Printf("WebView2 Runtime is not installed; Wails will offer the embedded Evergreen bootstrapper")
		return
	}
	log.Printf("Detected WebView2 Runtime version: %s", version)
}
