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
	ctx                    context.Context
	App                    *McpNet.Mcp
	listenerMu             sync.Mutex
	listenerRunning        bool
	processEndpointRunning bool
	proxyEnabled           bool
	processDriverMu        sync.Mutex
	processDriverLoaded    bool
	processFilterMu        sync.RWMutex
	processPIDFilters      map[int]bool
	processNameFilters     map[string]bool
	processAllFiltered     bool

	mcpMu sync.Mutex
	mcp   *mcpServer
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		processPIDFilters:  make(map[int]bool),
		processNameFilters: make(map[string]bool),
	}
}

type Command struct {
	Command string
	Args    any
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	log.Printf("Wails OnStartup reached; initializing application services")
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
	log.Printf("Wails OnStartup initialization completed")
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
	if a.processEndpointRunning {
		a.App.Close()
		a.processEndpointRunning = false
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
	if a.listenerRunning || a.processEndpointRunning {
		a.App.Close()
		a.listenerRunning = false
		a.processEndpointRunning = false
		log.Printf("Listener stopped")
		// Mcp.Close detaches the process interception callback on Windows.
		// Keep the cached driver state in sync so the next load request can
		// reattach it instead of reporting a stale loaded state.
		a.processDriverMu.Lock()
		wasLoaded := a.processDriverLoaded
		a.processDriverLoaded = false
		a.processDriverMu.Unlock()
		if wasLoaded {
			log.Printf("Process driver callback detached with listener; cached driver state reset")
			a.notifyMCPUIStateChanged("process")
		}
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
			if a.processEndpointRunning {
				a.App.Close()
				a.processEndpointRunning = false
			}
			a.App.Error = nil
			if err := a.App.Start().Error; err != nil {
				log.Printf("Could not expand process endpoint for system proxy: %v", err)
				return false
			}
			a.listenerRunning = true
			log.Printf("Listener started for explicit system proxy request; port=%d", a.App.Port())
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
	return a.listenerRunning || a.processEndpointRunning, a.proxyEnabled
}

func (a *App) startProcessEndpoint() error {
	a.listenerMu.Lock()
	defer a.listenerMu.Unlock()
	if a.App == nil {
		return fmt.Errorf("McpNet is still initializing")
	}
	if a.listenerRunning || a.processEndpointRunning {
		return nil
	}
	if pid, name, ok := findOriginalMcpNet(); ok {
		return fmt.Errorf("检测到原版 %s 正在运行 (PID %d)，请先退出原版再启动进程攔截，避免驅動衝突", name, pid)
	}
	a.App.Error = nil
	if err := a.App.StartLocal().Error; err != nil {
		log.Printf("Start local process-capture endpoint failed: %v", err)
		return err
	}
	a.processEndpointRunning = true
	log.Printf("Local process-capture endpoint started; port=%d", a.App.Port())
	return nil
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

// notifyMCPUIStateChanged pushes authoritative backend state to the mounted UI
// immediately after an MCP mutation. Components that are currently closed can
// still query the same backend snapshot when they are opened.
func (a *App) notifyMCPUIStateChanged(section string) {
	if a == nil || a.ctx == nil {
		return
	}
	a.CallDo(&Command{Command: "MCP狀態已更新", Args: section})
}

func (a *App) notifyMCPUIOperation(message string) {
	if a == nil || a.ctx == nil || message == "" {
		return
	}
	a.CallDo(&Command{Command: "MCP操作已完成", Args: message})
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
