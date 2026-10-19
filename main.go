package main

import (
	"context"
	"embed"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS
var app = NewApp()


func main() {
	if startupLogFile != nil {
		defer startupLogFile.Close()
	}
	// Create an instance of the app structure
	Size := GlobalConfig.Size
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Unable to find user home directory: %v", err)
		showStartupError("Cannot find user home directory: " + err.Error())
		return
	}
	webviewDataPath := filepath.Join(homeDir, mcpDataDirectory, "WebView2")
	if err := os.MkdirAll(webviewDataPath, 0700); err != nil {
		webviewDataPath = filepath.Join(os.TempDir(), mcpDataDirectory, "WebView2")
		if fallbackErr := os.MkdirAll(webviewDataPath, 0700); fallbackErr != nil {
			log.Printf("Unable to create WebView2 data directory: %v; fallback: %v", err, fallbackErr)
			showStartupError("Cannot create WebView2 data directory: " + fallbackErr.Error())
			return
		}
	}
	logPath := startupLogPath
	var wailsLogger logger.Logger
	if logPath == "" {
		logPath = filepath.Join(filepath.Dir(webviewDataPath), "startup.log")
		if logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600); err == nil {
			log.SetOutput(logFile)
			log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
			defer logFile.Close()
		} else {
			log.Printf("Unable to open startup log: %v", err)
			logPath = ""
		}
	}
	if logPath != "" {
		wailsLogger = logger.NewFileLogger(logPath)
	}
	log.Printf("Startup begins; WebView2 data path: %s", webviewDataPath)
	ready := make(chan struct{})
	var readyOnce sync.Once
	markReady := func() { readyOnce.Do(func() { close(ready) }) }
	go func() {
		select {
		case <-ready:
		case <-time.After(30 * time.Second):
			log.Printf("WebView2 DOM not ready after 30 seconds")
			showStartupError("McpNet window did not open within 30 seconds. See " + logPath)
		}
	}()
	// Create application with options
	log.Printf("Starting Wails")
	err = wails.Run(&options.App{
		Frameless: true,
		Title:     "McpNet",
		Width:            Size.Width,
		Height:           Size.Height,
		MinWidth:         823,
		MinHeight:        388,
		Windows: &windows.Options{
			WebviewUserDataPath: webviewDataPath,
		},
		Logger:             wailsLogger,
		LogLevelProduction: logger.DEBUG,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnDomReady: func(ctx context.Context) {
			log.Printf("Wails DOM ready")
			runtime.WindowUnminimise(ctx)
			runtime.WindowCenter(ctx)
			runtime.WindowShow(ctx)
			// Windows foreground-lock rules can leave a freshly started GUI
			// behind the window that launched it. A short always-on-top pulse
			// raises the main window without leaving it permanently pinned.
			go func() {
				time.Sleep(200 * time.Millisecond)
				runtime.WindowUnminimise(ctx)
				runtime.WindowSetAlwaysOnTop(ctx, true)
				time.Sleep(150 * time.Millisecond)
				runtime.WindowShow(ctx)
				time.Sleep(2 * time.Second)
				runtime.WindowSetAlwaysOnTop(ctx, false)
				fitWindowToWorkArea()
				log.Printf("Main window raised")
			}()
			markReady()
		},
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
		},
	})
	markReady()

	log.Printf("wails.Run returned: %v", err)
	if err != nil {
		showStartupError("McpNet startup failed: " + err.Error() + "\nLog: " + logPath)
	}
}
