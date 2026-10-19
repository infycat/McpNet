package main

import (
	"changeme/CommAnd"
	"context"
	"encoding/json"
	"fmt"
	"github.com/qtgolang/McpNet/McpNet"
	"github.com/qtgolang/McpNet/src/protobuf/JSON"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"log"
	"strconv"
	"sync"
)

// App struct
type App struct {
	ctx             context.Context
	App             *McpNet.Mcp
	listenerMu      sync.Mutex
	listenerRunning bool
	proxyEnabled    bool

	mcpMu sync.Mutex
	mcp   *mcpServer
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

type Command struct {
	Command string
	Args    any
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	log.Printf("Wails OnStartup reached")
	cleanupStaleSystemProxy()
	if pid, name, ok := findOriginalMcpNet(); ok {
		log.Printf("Detected original McpNet process: pid=%d name=%s", pid, name)
	}
	_ = ensurePluginDir()
	scanPlugins()
	go func() {
		a.startMCPServer()
	}()
	go a.mcpWatchdog()
}

func (a *App) startListener() error {
	a.listenerMu.Lock()
	defer a.listenerMu.Unlock()
	if a.listenerRunning {
		return nil
	}
	if a.App == nil {
		return fmt.Errorf("McpNet is still initializing")
	}
	if pid, name, ok := findOriginalMcpNet(); ok {
		return fmt.Errorf("检测到原版 %s 正在运行 (PID %d)，请先退出原版再启动监听，避免驱动冲突", name, pid)
	}
	a.App.Error = nil
	if err := a.App.Start().Error; err != nil {
		log.Printf("Start listener failed: %v", err)
		return err
	}
	a.listenerRunning = true
	log.Printf("Listener started; port=%d", a.App.Port())
	return nil
}

func (a *App) stopListener() {
	a.listenerMu.Lock()
	defer a.listenerMu.Unlock()
	if a.App == nil {
		return
	}
	if a.proxyEnabled {
		a.App.CancelIEProxy()
		a.proxyEnabled = false
		log.Printf("System proxy disabled")
	}
	if a.listenerRunning {
		a.App.Close()
		a.listenerRunning = false
		log.Printf("Listener stopped")
	}
}

func (a *App) setSystemProxy(enable bool) bool {
	a.listenerMu.Lock()
	defer a.listenerMu.Unlock()
	if a.App == nil {
		return false
	}
	if enable {
		if !a.listenerRunning {
			log.Printf("Refusing to enable system proxy: listener is not running")
			return false
		}
		if a.proxyEnabled {
			return true
		}
		if !a.App.SetIEProxy() {
			log.Printf("Failed to enable system proxy")
			return false
		}
		a.proxyEnabled = true
		log.Printf("System proxy enabled; port=%d", a.App.Port())
		return true
	}
	if !a.proxyEnabled {
		return true
	}
	if !a.App.CancelIEProxy() {
		log.Printf("Failed to disable system proxy")
		return false
	}
	a.proxyEnabled = false
	log.Printf("System proxy disabled")
	return true
}

func (a *App) listenerStatus() (bool, bool) {
	a.listenerMu.Lock()
	defer a.listenerMu.Unlock()
	return a.listenerRunning, a.proxyEnabled
}

func (a *App) shutdown(context.Context) {
	a.stopMCPServer()
	a.stopListener()
}

func (a *App) Do(arg any) any {
	defer func() {
		recover()
	}()
	as := arg.(map[string]any)
	if as == nil {
		return nil
	}
	_t := as["Command"]
	if _t == nil {
		return nil
	}
	command := _t.(string)
	Args := as["Args"]
	if Args == nil {
		return event(command, nil)
	}
	bs, _ := json.Marshal(Args)
	_json := string(bs)
	sj := JSON.NewSyJson()
	sj.Parse(_json)
	return event(command, sj)
}
func (a *App) CallDo(arg ...any) {
	runtime.EventsEmit(a.ctx, "Do", arg...)
}
func getFloat64(arg any) float64 {
	switch v := arg.(type) {
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case int32:
		return float64(v)
	case int16:
		return float64(v)
	case int8:
		return float64(v)
	case uint:
		return float64(v)
	case uint64:
		return float64(v)
	case uint32:
		return float64(v)
	case uint16:
		return float64(v)
	case uint8:
		return float64(v)
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0
		}
		return f
	case float32:
		return float64(v)
	case float64:
		return v
	default:
		return 0
	}
}
func getInt64(arg any) int64 {
	switch v := arg.(type) {
	case int:
		return int64(v)
	case int64:
		return int64(v)
	case int32:
		return int64(v)
	case int16:
		return int64(v)
	case int8:
		return int64(v)
	case uint:
		return int64(v)
	case uint64:
		return int64(v)
	case uint32:
		return int64(v)
	case uint16:
		return int64(v)
	case uint8:
		return int64(v)
	case string:
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0
		}
		return i
	case float32:
		return int64(v)
	case float64:
		return int64(v)
	default:
		return 0
	}
}
func getInt(arg any) int {
	return int(getInt64(arg))
}
func ClipboardText(text string) error {
	return CommAnd.ClipboardText(text)
}
