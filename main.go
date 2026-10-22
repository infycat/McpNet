package main

import (
	"context"
	"embed"
	"fmt"
	"log"
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
	if startupLogError != nil && startupLogFile == nil {
		log.Printf("Unable to create startup diagnostic log at %s: %v", startupLogPath, startupLogError)
	}
	// Create an instance of the app structure
	Size := GlobalConfig.Size
	webviewDataPath, err := prepareWebviewDataPath()
	if err != nil {
		log.Printf("Unable to prepare WebView2 user data directory: %v", err)
		showStartupError(startupFailureMessage("Cannot prepare WebView2 data directory", err))
		return
	}
	logPath := startupLogPath
	var wailsLogger logger.Logger
	if startupLogFile != nil {
		wailsLogger = logger.NewFileLogger(logPath)
	}
	log.Printf("Startup continues; WebView2 data path: %s", webviewDataPath)
	logWebView2Runtime()
	ready := make(chan struct{})
	var readyOnce sync.Once
	markReady := func() { readyOnce.Do(func() { close(ready) }) }
	go func() {
		started := time.Now()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ready:
				return
			case <-ticker.C:
				log.Printf("Startup is still in progress; WebView2 DOM is not ready; elapsed=%s", time.Since(started).Round(time.Second))
			}
		}
	}()
	// Create application with options
	log.Printf("Starting Wails")
	err = wails.Run(&options.App{
		Frameless: true,
		Title:     "McpNet",
		Width:     Size.Width,
		Height:    Size.Height,
		MinWidth:  823,
		MinHeight: 388,
		Windows: &windows.Options{
			WebviewUserDataPath: webviewDataPath,
		},
		Logger:             wailsLogger,
		LogLevelProduction: logger.INFO,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnDomReady: func(ctx context.Context) {
			log.Printf("Wails DOM ready")
			log.Printf("Showing and centering main window")
			runtime.WindowUnminimise(ctx)
			runtime.WindowCenter(ctx)
			runtime.WindowShow(ctx)
			log.Printf("Main window show requested")
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
		showStartupError(startupFailureMessage("McpNet startup failed", err))
	}
}

func startupFailureMessage(summary string, cause error) string {
	message := fmt.Sprintf("%s: %v", summary, cause)
	if startupLogFile != nil && startupLogPath != "" {
		return message + "\n\nStartup log: " + startupLogPath
	}
	if startupLogPath != "" {
		return message + "\n\nCould not create the startup log at: " + startupLogPath + "\n" + fmt.Sprint(startupLogError)
	}
	return message + "\n\nCould not determine a startup log path: " + fmt.Sprint(startupLogError)
}
