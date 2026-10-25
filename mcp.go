package main

import (
	"changeme/CommAnd"
	"changeme/MapHash"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/qtgolang/McpNet/public"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	defaultMCPPort     = 2026
	mcpServerName      = "mcpnet"
	mcpCurrentProtocol = "2025-11-25"
)

var supportedMCPProtocols = map[string]bool{
	"2024-11-05":       true,
	"2025-03-26":       true,
	"2025-06-18":       true,
	mcpCurrentProtocol: true,
}

type mcpServer struct {
	app     *App
	confirm func(title, message string) bool

	mu      sync.RWMutex
	ln      net.Listener
	srv     *http.Server
	port    int
	running bool
	lastErr string
	path    string
}

type mcpSettings struct {
	Path string `json:"path"`
}

type mcpServerEntry struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type mcpConfigFile struct {
	McpServers map[string]mcpServerEntry `json:"mcpServers"`
}

type mcpTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func mcpHealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !validMCPRequestHostAndOrigin(r) {
			http.Error(w, "MCP health endpoint only accepts loopback Host and Origin values", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"ok":true,"name":"mcpnet","version":"%s","port":%d}`, applicationVersion, defaultMCPPort)
	}
}

func (a *App) startMCPServer() map[string]any {
	a.mcpMu.Lock()
	defer a.mcpMu.Unlock()

	if a.mcp != nil {
		a.mcp.stop()
	}

	path := loadMCPPath()
	s := &mcpServer{app: a, path: path}
	a.mcp = s
	if err := s.start(); err != nil {
		s.mu.Lock()
		s.lastErr = err.Error()
		s.mu.Unlock()
		log.Printf("MCP server start failed: %v", err)
	}
	return a.mcpInfoLocked()
}

func (a *App) mcpWatchdog() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		a.mcpMu.Lock()
		s := a.mcp
		a.mcpMu.Unlock()
		if s == nil {
			a.startMCPServer()
			continue
		}
		s.mu.RLock()
		running := s.running
		s.mu.RUnlock()
		if !running {
			if err := s.start(); err != nil {
				s.mu.Lock()
				s.lastErr = err.Error()
				s.mu.Unlock()
			}
		}
	}
}

func (a *App) restartMCPServer() map[string]any {
	a.mcpMu.Lock()
	old := a.mcp
	a.mcp = nil
	a.mcpMu.Unlock()
	if old != nil {
		old.stop()
	}
	return a.startMCPServer()
}

func (a *App) stopMCPServer() {
	a.mcpMu.Lock()
	s := a.mcp
	a.mcp = nil
	a.mcpMu.Unlock()
	if s != nil {
		s.stop()
	}
}

func (a *App) mcpInfo() map[string]any {
	a.mcpMu.Lock()
	defer a.mcpMu.Unlock()
	return a.mcpInfoLocked()
}

func (a *App) mcpInfoLocked() map[string]any {
	s := a.mcp
	if s == nil {
		return map[string]any{
			"running": false,
			"url":     "",
			"port":    0,
			"path":    loadMCPPath(),
			"json":    "",
			"error":   "MCP server is not initialized",
		}
	}
	s.mu.RLock()
	running := s.running
	port := s.port
	path := s.path
	lastErr := s.lastErr
	s.mu.RUnlock()

	url := ""
	cfg := ""
	if running && port > 0 {
		url = fmt.Sprintf("http://127.0.0.1:%d/mcp", port)
		cfg = mcpConfigJSON(url)
	}
	return map[string]any{
		"running": running,
		"url":     url,
		"port":    port,
		"path":    path,
		"json":    cfg,
		"error":   lastErr,
	}
}

func (a *App) setMCPPath(path string) map[string]any {
	var err error
	path, err = validateMCPPath(path)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return map[string]any{"error": err.Error()}
	}
	saveMCPPath(path)

	a.mcpMu.Lock()
	s := a.mcp
	if s != nil {
		s.mu.Lock()
		s.path = path
		s.mu.Unlock()
	}
	a.mcpMu.Unlock()

	if s != nil {
		if err := s.writeConfig(); err != nil {
			return map[string]any{"error": err.Error(), "path": path}
		}
	}
	return a.mcpInfo()
}

func mcpDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	return filepath.Join(home, mcpDataDirectory)
}

func mcpSettingsPath() string {
	return filepath.Join(mcpDataDir(), "mcp-settings.json")
}

func loadMCPPath() string {
	def := filepath.Join(mcpDataDir(), "mcp.json")
	bs, err := os.ReadFile(mcpSettingsPath())
	if err != nil {
		return def
	}
	var s mcpSettings
	if json.Unmarshal(bs, &s) != nil || strings.TrimSpace(s.Path) == "" {
		return def
	}
	p, err := validateMCPPath(s.Path)
	if err != nil {
		return def
	}
	return p
}

func saveMCPPath(path string) {
	_ = os.MkdirAll(mcpDataDir(), 0700)
	bs, _ := json.MarshalIndent(mcpSettings{Path: path}, "", "  ")
	_ = os.WriteFile(mcpSettingsPath(), bs, 0600)
}

func validateMCPPath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return filepath.Join(mcpDataDir(), "mcp.json"), nil
	}
	if strings.ContainsRune(raw, 0) {
		return "", fmt.Errorf("MCP path contains an invalid NUL character")
	}
	for _, r := range raw {
		if r < 32 || r == 127 {
			return "", fmt.Errorf("MCP path contains control characters")
		}
	}
	if len(raw) > 240 {
		return "", fmt.Errorf("MCP path is too long (max 240 characters)")
	}
	// Accept both separators, then collapse duplicate separators. This is what
	// prevents "C:\\Users\\..." from growing on every save.
	raw = strings.ReplaceAll(raw, "/", string(os.PathSeparator))
	clean := filepath.Clean(raw)
	if clean == "." || clean == string(os.PathSeparator) {
		return "", fmt.Errorf("MCP path is invalid")
	}
	if os.PathSeparator == '\\' {
		vol := filepath.VolumeName(clean)
		rest := strings.TrimPrefix(clean, vol)
		if strings.ContainsAny(rest, "<>:\"|?*") {
			return "", fmt.Errorf("MCP path contains invalid Windows characters")
		}
		for _, seg := range strings.Split(strings.Trim(rest, "\\"), "\\") {
			base := strings.ToUpper(strings.TrimSuffix(seg, filepath.Ext(seg)))
			switch base {
			case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
				return "", fmt.Errorf("MCP path uses a reserved Windows name: %s", seg)
			}
		}
	}
	abs, err := filepath.Abs(clean)
	if err != nil {
		return "", fmt.Errorf("MCP path is invalid: %w", err)
	}
	if st, err := os.Stat(abs); err == nil && st.IsDir() {
		abs = filepath.Join(abs, "mcp.json")
	} else if !strings.EqualFold(filepath.Ext(abs), ".json") {
		abs = filepath.Join(abs, "mcp.json")
	}
	if !strings.EqualFold(filepath.Ext(abs), ".json") {
		return "", fmt.Errorf("MCP path must be a directory or a .json file")
	}
	return filepath.Clean(abs), nil
}

func mcpConfigJSON(url string) string {
	cfg := mcpConfigFile{McpServers: map[string]mcpServerEntry{
		mcpServerName: {Type: "http", URL: url},
	}}
	bs, _ := json.MarshalIndent(cfg, "", "  ")
	return string(bs)
}

func (s *mcpServer) start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	// Keep the generated JSON stable: a fixed port is more important than
	// silently moving to another port, which would invalidate copied configs.
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", defaultMCPPort))
	if err != nil {
		s.lastErr = fmt.Sprintf("MCP port %d is occupied; close the other program or restart this app: %v", defaultMCPPort, err)
		s.mu.Unlock()
		return fmt.Errorf("%s", s.lastErr)
	}
	s.port = defaultMCPPort
	s.ln = ln
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", s.handleMCP)
	mux.HandleFunc("/health", mcpHealthHandler())
	s.srv = &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	s.running = true
	s.lastErr = ""
	port := s.port
	path := s.path
	srv := s.srv
	s.mu.Unlock()

	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			s.mu.Lock()
			s.lastErr = err.Error()
			s.running = false
			s.mu.Unlock()
		}
	}()
	if err := s.writeConfig(); err != nil {
		log.Printf("MCP config write failed: %v", err)
	}
	log.Printf("MCP server listening on http://127.0.0.1:%d/mcp (config %s)", port, path)
	return nil
}

func (s *mcpServer) stop() {
	s.mu.Lock()
	srv := s.srv
	s.srv = nil
	s.running = false
	s.mu.Unlock()
	if srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = srv.Shutdown(ctx)
		cancel()
	}
}

func (s *mcpServer) writeConfig() error {
	s.mu.RLock()
	port := s.port
	path := s.path
	s.mu.RUnlock()
	if port == 0 || path == "" {
		return fmt.Errorf("MCP server is not ready")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	cfg := mcpConfigJSON(fmt.Sprintf("http://127.0.0.1:%d/mcp", port))
	if err := os.WriteFile(path, []byte(cfg), 0600); err != nil {
		return err
	}
	saveMCPPath(path)
	return nil
}

func mcpLoopbackHost(raw string) bool {
	host := raw
	if ip := net.ParseIP(strings.Trim(host, "[]")); ip != nil {
		return ip.IsLoopback()
	}
	if parsedHost, port, err := net.SplitHostPort(raw); err == nil {
		if port != "" {
			portNumber, err := strconv.Atoi(port)
			if err != nil || portNumber < 1 || portNumber > 65535 {
				return false
			}
		}
		host = parsedHost
	} else if strings.Contains(raw, ":") {
		return false
	}
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func validMCPOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Opaque != "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "" {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	return parsed.Host != "" && mcpLoopbackHost(parsed.Host)
}

func validMCPRequestHostAndOrigin(r *http.Request) bool {
	if r == nil || !mcpLoopbackHost(r.Host) {
		return false
	}
	origins := r.Header.Values("Origin")
	if len(origins) > 1 {
		return false
	}
	return len(origins) == 0 || validMCPOrigin(origins[0])
}

func supportedMCPProtocol(version string) bool {
	return supportedMCPProtocols[strings.TrimSpace(version)]
}

func (s *mcpServer) handleMCP(w http.ResponseWriter, r *http.Request) {
	origins := r.Header.Values("Origin")
	if !validMCPRequestHostAndOrigin(r) {
		http.Error(w, "MCP endpoint only accepts loopback Host and Origin values", http.StatusForbidden)
		return
	}
	if protocolVersion := strings.TrimSpace(r.Header.Get("MCP-Protocol-Version")); protocolVersion != "" && !supportedMCPProtocol(protocolVersion) {
		http.Error(w, "unsupported MCP protocol version", http.StatusBadRequest)
		return
	}
	if len(origins) == 1 {
		w.Header().Set("Access-Control-Allow-Origin", origins[0])
		w.Header().Add("Vary", "Origin")
	}
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, MCP-Protocol-Version, Mcp-Session-Id, Last-Event-ID")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method == http.MethodGet {
		s.handleSSE(w, r)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxMCPRequestBytes+1))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(body) > maxMCPRequestBytes {
		http.Error(w, "MCP request body exceeds 24 MiB", http.StatusRequestEntityTooLarge)
		return
	}
	var raw json.RawMessage = body
	var one map[string]json.RawMessage
	if json.Unmarshal(body, &one) == nil {
		// single request
	} else {
		var batch []json.RawMessage
		if json.Unmarshal(body, &batch) != nil {
			http.Error(w, "invalid JSON-RPC body", http.StatusBadRequest)
			return
		}
		responses := make([]any, 0, len(batch))
		allNotifications := true
		for _, b := range batch {
			resp, notification := s.handleOne(b)
			if !notification {
				allNotifications = false
				responses = append(responses, resp)
			}
		}
		if allNotifications {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		s.writeJSON(w, r, responses)
		return
	}

	resp, notification := s.handleOne(raw)
	if notification {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	s.writeJSON(w, r, resp)
}

func (s *mcpServer) writeJSON(w http.ResponseWriter, r *http.Request, v any) {
	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "text/event-stream") && !strings.Contains(accept, "application/json") {
		bs, _ := json.Marshal(v)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = fmt.Fprintf(w, "event: message\ndata: %s\n\n", bs)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (s *mcpServer) handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	f, _ := w.(http.Flusher)
	_, _ = io.WriteString(w, "event: endpoint\ndata: /mcp\n\n")
	if f != nil {
		f.Flush()
	}
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			_, _ = io.WriteString(w, ": ping\n\n")
			if f != nil {
				f.Flush()
			}
		}
	}
}

func (s *mcpServer) handleOne(raw json.RawMessage) (any, bool) {
	var req struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(raw, &req); err != nil {
		return mcpError(nil, -32700, "parse error"), false
	}
	if req.JSONRPC != "2.0" || req.Method == "" {
		return mcpError(req.ID, -32600, "invalid request"), false
	}
	notification := len(req.ID) == 0 || string(req.ID) == "null"

	switch req.Method {
	case "initialize":
		var params struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil || strings.TrimSpace(params.ProtocolVersion) == "" {
			return mcpError(req.ID, -32602, "initialize requires a protocolVersion"), false
		}
		protocolVersion := strings.TrimSpace(params.ProtocolVersion)
		if !supportedMCPProtocol(protocolVersion) {
			protocolVersion = mcpCurrentProtocol
		}
		return mcpResult(req.ID, map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "mcpnet",
				"version": applicationVersion,
			},
		}), false
	case "notifications/initialized", "notifications/cancelled":
		return nil, true
	case "ping":
		return mcpResult(req.ID, map[string]any{}), false
	case "tools/list":
		return mcpResult(req.ID, map[string]any{"tools": mcpTools()}), false
	case "tools/call":
		var p struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil || p.Name == "" {
			return mcpError(req.ID, -32602, "invalid tools/call params"), false
		}
		out, err := s.callTool(p.Name, p.Arguments)
		if err != nil {
			return mcpResult(req.ID, toolText(err.Error(), true)), false
		}
		return mcpResult(req.ID, toolText(out, false)), false
	case "resources/list":
		return mcpResult(req.ID, map[string]any{"resources": []any{}}), false
	case "prompts/list":
		return mcpResult(req.ID, map[string]any{"prompts": []any{}}), false
	default:
		if notification {
			return nil, true
		}
		return mcpError(req.ID, -32601, "method not found: "+req.Method), false
	}
}

func mcpResult(id json.RawMessage, result any) map[string]any {
	return map[string]any{"jsonrpc": "2.0", "id": id, "result": result}
}

func mcpError(id json.RawMessage, code int, msg string) map[string]any {
	return map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": code, "message": msg}}
}

func toolText(v any, isError bool) map[string]any {
	var text string
	switch t := v.(type) {
	case string:
		text = t
	default:
		bs, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			text = fmt.Sprint(v)
		} else {
			text = string(bs)
		}
	}
	if len(text) > 200000 {
		text = text[:200000] + "\n... (truncated)"
	}
	return map[string]any{
		"content": []any{map[string]any{"type": "text", "text": text}},
		"isError": isError,
	}
}

const maxMCPBodyBytes = 64 << 10
const maxMCPRequestBytes = 24 << 20

func argInt(args map[string]any, key string, fallback int) int {
	v, ok := args[key]
	if !ok {
		return fallback
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case float32:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	case json.Number:
		i, err := n.Int64()
		if err == nil {
			return int(i)
		}
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(n))
		if err == nil {
			return i
		}
	}
	return fallback
}

func argBool(args map[string]any, key string) bool {
	v, _ := args[key].(bool)
	return v
}

func boundedBase64(body []byte) (string, bool) {
	truncated := len(body) > maxMCPBodyBytes
	if truncated {
		body = body[:maxMCPBodyBytes]
	}
	return base64.StdEncoding.EncodeToString(body), truncated
}

func captureMetadata(c MapHash.CaptureSnapshot) map[string]any {
	return map[string]any{
		"capture_id": c.ID, "protocol": c.Protocol, "method": c.Method, "url": redactCaptureURL(c.URL),
		"status_code":  c.StatusCode,
		"request_time": c.RequestAt, "response_time": c.ResponseAt,
		"request_body_length": len(c.RequestBody), "response_body_length": len(c.ResponseBody),
		"sent_bytes": c.SentBytes, "received_bytes": c.RecvBytes, "message_count": len(c.Messages),
	}
}

// redactCaptureURL exposes only the scheme and host to metadata-only MCP tools.
func redactCaptureURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "[invalid URL omitted]"
	}
	parsed.User = nil
	if parsed.Path != "" && parsed.Path != "/" {
		parsed.Path = "/[redacted]"
		parsed.RawPath = ""
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func hasCaptureAfter(captures []MapHash.CaptureSnapshot, id int, protocol string) bool {
	for _, c := range captures {
		if c.ID > id && (protocol == "" || protocol == "<nil>" || strings.EqualFold(c.Protocol, protocol)) {
			return true
		}
	}
	return false
}

func (s *mcpServer) callTool(name string, args map[string]any) (any, error) {
	switch name {
	case "mcpnet_status":
		running, proxy := s.app.listenerStatus()
		processDriverLoaded := s.app.processDriverStatus()
		info := s.app.mcpInfo()
		mcpStatus := map[string]any{
			"running": info["running"],
			"url":     info["url"],
			"port":    info["port"],
		}
		return map[string]any{
			"listener_running":      running,
			"system_proxy":          proxy,
			"process_driver_loaded": processDriverLoaded,
			"mcp":                   mcpStatus,
		}, nil
	case "mcpnet_list_processes":
		name := strings.TrimSpace(stringArg(args, "name"))
		if name == "" || filepath.Base(name) != name {
			return nil, fmt.Errorf("name must be an executable file name such as MiniGameApp.exe")
		}
		if !s.authorize("private_read", "McpNet process lookup", fmt.Sprintf("AI 客戶端要求查找本機名為 %q 的進程及 PID。是否分享？\n\nAn AI client requests local process IDs matching %q. Share?", name, name)) {
			return nil, fmt.Errorf("user denied process lookup")
		}
		return map[string]any{"processes": matchingProcesses(name, CommAnd.EnumerateProcesses())}, nil
	case "mcpnet_process_control":
		action := strings.ToLower(strings.TrimSpace(stringArg(args, "action")))
		name := strings.TrimSpace(stringArg(args, "name"))
		pid := argInt(args, "pid", -1)
		if action != "load_driver" && action != "add_name" && action != "remove_name" && action != "add_pid" && action != "remove_pid" && action != "clear" {
			return nil, fmt.Errorf("action must be load_driver, add_name, remove_name, add_pid, remove_pid, or clear")
		}
		if (action == "add_name" || action == "remove_name") && (name == "" || filepath.Base(name) != name) {
			return nil, fmt.Errorf("name must be an executable file name such as MiniGameApp.exe")
		}
		if (action == "add_pid" || action == "remove_pid") && pid <= 0 {
			return nil, fmt.Errorf("pid must be a positive process ID")
		}
		if !s.authorize("settings", "McpNet process interception", fmt.Sprintf("AI 客戶端要求執行進程攔截操作 %q（進程 %q，PID %d）。加載驅動會安裝／啟用網絡攔截驅動，進程規則會改變被捕獲的流量範圍。允許？\n\nAn AI client requests process-interception action %q (process %q, PID %d). Loading the driver installs/enables a network interception driver; process rules change which traffic is captured. Allow?", action, name, pid, action, name, pid)) {
			return nil, fmt.Errorf("user denied process interception action %q", action)
		}
		if action == "add_pid" {
			if _, exists := CommAnd.EnumerateProcesses()[pid]; !exists {
				return nil, fmt.Errorf("process ID %d is not currently running", pid)
			}
		}
		if s.app == nil || s.app.App == nil {
			return nil, fmt.Errorf("McpNet core is not initialized")
		}
		switch action {
		case "load_driver":
			loaded, err := s.app.startProcessDriver()
			if err != nil {
				return nil, err
			}
			s.app.notifyMCPUIStateChanged("process")
			return map[string]any{"action": action, "loaded": loaded}, nil
		case "add_name":
			if !s.app.processDriverStatus() {
				return nil, fmt.Errorf("load the process interception driver before changing process filters")
			}
			s.app.setProcessNameFilter(name, true)
		case "remove_name":
			if !s.app.processDriverStatus() {
				return nil, fmt.Errorf("load the process interception driver before changing process filters")
			}
			s.app.setProcessNameFilter(name, false)
		case "add_pid":
			if !s.app.processDriverStatus() {
				return nil, fmt.Errorf("load the process interception driver before changing process filters")
			}
			s.app.setProcessPIDFilterMCP(pid, true)
		case "remove_pid":
			if !s.app.processDriverStatus() {
				return nil, fmt.Errorf("load the process interception driver before changing process filters")
			}
			s.app.setProcessPIDFilterMCP(pid, false)
		case "clear":
			if !s.app.processDriverStatus() {
				return nil, fmt.Errorf("load the process interception driver before changing process filters")
			}
			s.app.clearProcessFilters()
		}
		return map[string]any{"action": action, "name": name, "pid": pid, "applied": true}, nil
	case "mcpnet_start_listener":
		if !s.authorize("settings", "McpNet MCP request", "AI 客戶端要求啟動流量監聽，可能檢視經過 McpNet 的流量。允許？\n\nAn AI client wants to start traffic monitoring. This may inspect traffic that passes through McpNet. Allow?") {
			return nil, fmt.Errorf("user denied starting traffic monitoring")
		}
		if err := s.app.startListener(); err != nil {
			return nil, err
		}
		s.app.notifyMCPUIStateChanged("listener")
		running, proxy := s.app.listenerStatus()
		return map[string]any{"listener_running": running, "system_proxy": proxy}, nil
	case "mcpnet_stop_listener":
		if !s.authorize("settings", "McpNet MCP request", "AI 客戶端要求停止流量監聽並清除系統代理。允許？\n\nAn AI client wants to stop traffic monitoring and clear the system proxy. Allow?") {
			return nil, fmt.Errorf("user denied stopping traffic monitoring")
		}
		s.app.stopListener()
		s.app.notifyMCPUIStateChanged("listener")
		running, proxy := s.app.listenerStatus()
		return map[string]any{"listener_running": running, "system_proxy": proxy}, nil
	case "mcpnet_set_system_proxy":
		enable, _ := args["enable"].(bool)
		action := "停用 / disable"
		if enable {
			action = "啟用 / enable"
		}
		if !s.authorize("settings", "McpNet MCP request", fmt.Sprintf("AI 客戶端要求%s Windows 系統代理，會改變系統流量路由。允許？\n\nAn AI client wants to %s the Windows system proxy. This changes system traffic routing. Allow?", action, map[bool]string{true: "enable", false: "disable"}[enable])) {
			return nil, fmt.Errorf("user denied changing the system proxy")
		}
		ok := s.app.setSystemProxy(enable)
		s.app.notifyMCPUIStateChanged("listener")
		running, proxy := s.app.listenerStatus()
		return map[string]any{"ok": ok, "listener_running": running, "system_proxy": proxy}, nil
	case "mcpnet_command":
		command, _ := args["command"].(string)
		if strings.TrimSpace(command) == "" {
			return nil, fmt.Errorf("command is required")
		}
		category, operation := mcpUICommandPermission(command, args)
		prompt := fmt.Sprintf("AI 客戶端要求執行命令「%s」：%s。允許？\n\nAn AI client requests UI command “%s”: %s. Allow?", command, operation, command, operation)
		if !s.authorizeMCPUICommand(command, category, prompt) {
			return nil, fmt.Errorf("user denied UI command %q (%s permission)", command, category)
		}
		var cmdArgs any
		if v, ok := args["args"]; ok {
			cmdArgs = v
		}
		if command == "选择文件" || command == "保存文件对话框" {
			values, _ := cmdArgs.(map[string]any)
			path, _ := values["Path"].(string)
			selected, err := mcpDialogPath(command, path)
			if err != nil {
				return nil, err
			}
			return map[string]any{"command": command, "result": selected, "native_dialog_skipped": true}, nil
		}
		if command == "导出默认证书" {
			values, _ := cmdArgs.(map[string]any)
			path, _ := values["Path"].(string)
			exported, err := mcpExportDefaultCertificate(path)
			if err != nil {
				return nil, err
			}
			return map[string]any{"command": command, "result": exported, "native_dialog_skipped": true}, nil
		}
		res, err := s.doUICommand(command, cmdArgs)
		if err != nil {
			return nil, err
		}
		s.app.notifyMCPUIStateChanged(mcpUIStateSection(command))
		return map[string]any{"command": command, "result": res}, nil
	case "mcpnet_list_commands":
		return map[string]any{
			"commands": mcpUICommandCatalog(),
			"note":     "Use mcpnet_command with one of these exact command names and the same Args object used by the UI. Each command lists argument_fields, permission_category, and authorization. The authorization value requires the named permission category or a direct user confirmation; MCP permission changes always require direct confirmation. socket点击右键菜单 is context-dependent: Type=empty requires settings, while other Type values require private-read. File-picker and default-certificate-export commands advertise mcp_path_override and accept an absolute Args.Path so MCP clients do not open native dialogs. Unknown commands require direct confirmation.",
		}, nil
	case "mcpnet_list_plugins":
		return mcpPluginListResult(), nil
	case "mcpnet_get_plugin":
		id := strings.TrimSpace(stringArg(args, "plugin_id"))
		if id == "" {
			return nil, fmt.Errorf("plugin_id is required")
		}
		var manifest pluginManifest
		found := false
		for _, p := range plugins() {
			if p.Manifest.ID == id {
				manifest = p.Manifest
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("plugin %q was not found", id)
		}
		result := map[string]any{"manifest": manifest}
		if argBool(args, "include_source") {
			if !s.authorize("private_read", "McpNet plugin source", fmt.Sprintf("AI 客戶端要求讀取插件 %q 的本機 Go 原始碼。是否分享？\n\nAn AI client requests the local Go source for plugin %q. Share?", id, id)) {
				return nil, fmt.Errorf("user denied plugin source access")
			}
			result["source"] = pluginSourceOf(id)
		}
		return result, nil
	case "mcpnet_save_plugin":
		manifestJSON := stringArg(args, "manifest_json")
		manifest, err := parsePluginManifest(manifestJSON)
		if err != nil {
			return nil, err
		}
		if manifest.Enabled {
			return nil, fmt.Errorf("plugin manifest must keep enabled=false; use mcpnet_set_plugin_enabled after reviewing the source")
		}
		source := stringArg(args, "source")
		var sourcePtr *string
		if source != "" {
			sourcePtr = &source
		}
		if !s.authorize("plugin_code", "McpNet save plugin", fmt.Sprintf("AI 客戶端要求建立或更新插件 %q。保存的 Go 程式碼會在啟用後於本機執行；此操作不會啟用插件。允許保存？\n\nAn AI client requests creating or updating plugin %q. Saved Go code runs locally only after it is enabled; this operation will leave the plugin disabled. Allow?", manifest.ID, manifest.ID)) {
			return nil, fmt.Errorf("user denied saving plugin")
		}
		if err := savePluginManifest(manifest, sourcePtr); err != nil {
			return nil, err
		}
		s.app.notifyMCPUIStateChanged("plugins")
		return mcpPluginListResult(), nil
	case "mcpnet_delete_plugin":
		id := strings.TrimSpace(stringArg(args, "plugin_id"))
		if id == "" {
			return nil, fmt.Errorf("plugin_id is required")
		}
		if !s.authorize("plugin_code", "McpNet delete plugin", fmt.Sprintf("AI 客戶端要求永久刪除本機插件 %q 的清單與原始碼。允許？\n\nAn AI client requests permanently deleting local plugin %q, including its manifest and source. Allow?", id, id)) {
			return nil, fmt.Errorf("user denied deleting plugin")
		}
		if err := deletePluginDir(id); err != nil {
			return nil, err
		}
		s.app.notifyMCPUIStateChanged("plugins")
		return mcpPluginListResult(), nil
	case "mcpnet_plugin_ui_action":
		id := strings.TrimSpace(stringArg(args, "plugin_id"))
		action := stringArg(args, "action")
		payload := stringArg(args, "payload")
		if id == "" || strings.TrimSpace(action) == "" {
			return nil, fmt.Errorf("plugin_id and action are required")
		}
		if !s.authorize("plugin_code", "McpNet plugin UI action", fmt.Sprintf("AI 客戶端要求執行插件 %q 的自訂介面動作 %q。這會呼叫插件本機 Go 程式碼。允許？\n\nAn AI client requests plugin %q UI action %q. This invokes local plugin Go code. Allow?", id, action, id, action)) {
			return nil, fmt.Errorf("user denied plugin UI action")
		}
		return runPluginUIAction(id, action, payload)
	case "mcpnet_import_plugin":
		if !s.authorize("plugin_code", "McpNet plugin import", "AI 客戶端要求安裝 Go 插件程式碼。插件匯入後預設停用，啟用時可讀取／修改本機流量；請先審查來源。允許匯入？\n\nAn AI client wants to install Go plugin code. Imports are disabled by default but can access local traffic when enabled. Review the source before enabling. Allow import?") {
			return nil, fmt.Errorf("user denied plugin import")
		}
		if zipData, _ := args["zip_base64"].(string); zipData != "" {
			decoded, err := base64.StdEncoding.DecodeString(zipData)
			if err != nil {
				return nil, fmt.Errorf("invalid ZIP base64: %w", err)
			}
			if len(decoded) > maxPluginPackageSize {
				return nil, fmt.Errorf("plugin ZIP exceeds the 16 MiB limit")
			}
			if err := importPluginZipBytes(decoded); err != nil {
				return nil, err
			}
			s.app.notifyMCPUIStateChanged("plugins")
			return mcpPluginListResult(), nil
		}
		manifest, _ := args["manifest_json"].(string)
		source, _ := args["source"].(string)
		if manifest == "" || source == "" {
			return nil, fmt.Errorf("provide zip_base64 or both manifest_json and source")
		}
		if err := importPluginFiles(manifest, source); err != nil {
			return nil, err
		}
		s.app.notifyMCPUIStateChanged("plugins")
		return mcpPluginListResult(), nil
	case "mcpnet_set_plugin_enabled":
		id, _ := args["plugin_id"].(string)
		enabled, _ := args["enabled"].(bool)
		if strings.TrimSpace(id) == "" {
			return nil, fmt.Errorf("plugin_id is required")
		}
		if enabled && !s.authorize("plugin_code", "McpNet enable plugin", fmt.Sprintf("AI 客戶端要求啟用插件 %q。Go 程式碼會在本機執行，可能檢視或修改流量；請確認你信任此插件。允許啟用？\n\nAn AI client wants to enable plugin %q. Go code runs locally and may inspect or modify traffic. Review and trust this plugin before allowing. Enable?", id, id)) {
			return nil, fmt.Errorf("user denied enabling plugin %q", id)
		}
		if err := setPluginEnabled(id, enabled); err != nil {
			return nil, err
		}
		s.app.notifyMCPUIStateChanged("plugins")
		return mcpPluginListResult(), nil
	case "mcpnet_send_session_data":
		id := argInt(args, "capture_id", -1)
		protocol, _ := args["protocol"].(string)
		data, _ := args["data_base64"].(string)
		direction, _ := args["direction"].(string)
		decoded, err := base64.StdEncoding.DecodeString(data)
		if id < 0 || err != nil || len(decoded) == 0 {
			return nil, fmt.Errorf("capture_id and a non-empty valid data_base64 value are required")
		}
		isWS := strings.EqualFold(protocol, "websocket") || strings.EqualFold(protocol, "ws")
		isTCP := strings.EqualFold(protocol, "tcp")
		isUDP := strings.EqualFold(protocol, "udp")
		if !isWS && !isTCP && !isUDP {
			return nil, fmt.Errorf("protocol must be websocket, tcp, or udp; use the UI command tool for HTTP request replay")
		}
		if direction != "Server" && direction != "Client" {
			return nil, fmt.Errorf("direction must be Server or Client")
		}
		if !s.authorize("network_write", "McpNet send packet", fmt.Sprintf("AI 客戶端要求透過捕獲 %d 的既有 %s 連線，向%s發送 %d bytes。資料會傳給已連線的對端。允許？\n\nAn AI client wants to send %d bytes via %s through capture %d toward the %s. This transmits data to the connected peer. Allow?", id, protocol, direction, len(decoded), len(decoded), protocol, id, direction)) {
			return nil, fmt.Errorf("user denied packet send")
		}
		encodedInput := base64.StdEncoding.EncodeToString([]byte(data))
		res, err := s.doUICommand("主动发送", map[string]any{
			"Theology": strconv.Itoa(id), "IsWs": strconv.FormatBool(isWS), "IsTCP": strconv.FormatBool(isTCP), "IsUDP": strconv.FormatBool(isUDP),
			"wsType": "Binary", "SendType": "Base64", "direction": direction, "Data": encodedInput,
		})
		if err != nil {
			return nil, err
		}
		s.app.notifyMCPUIStateChanged("refresh")
		return map[string]any{"capture_id": id, "protocol": protocol, "result": res}, nil
	case "mcpnet_send_http_request":
		result, err := s.sendHTTPMCPRequest(args)
		if err == nil {
			s.app.notifyMCPUIOperation(fmt.Sprintf("MCP HTTP 請求已完成（狀態碼 %v；不會加入封包捕獲列表）", result["status_code"]))
		}
		return result, err
	case "mcpnet_send_socket_request":
		result, err := s.sendSocketMCPRequest(args)
		if err == nil {
			s.app.notifyMCPUIOperation(fmt.Sprintf("MCP %s 請求已完成（不會加入封包捕獲列表）", result["protocol"]))
		}
		return result, err
	case "mcpnet_send_websocket_request":
		result, err := s.sendWebSocketMCPRequest(args)
		if err == nil {
			s.app.notifyMCPUIOperation(fmt.Sprintf("MCP WebSocket 請求已完成（狀態碼 %v；不會加入封包捕獲列表）", result["status_code"]))
		}
		return result, err
	case "mcpnet_replay_capture":
		id := argInt(args, "capture_id", -1)
		mode := argInt(args, "mode", 3)
		if id < 0 {
			return nil, fmt.Errorf("capture_id must be a non-negative integer")
		}
		if mode != 3 && mode != 1 && mode != 2 {
			return nil, fmt.Errorf("mode must be 3 (normal), 1 (intercept upload), or 2 (intercept download)")
		}
		capture, ok := HashMap.SnapshotCapture(id)
		if !ok {
			return nil, fmt.Errorf("capture %d was not found", id)
		}
		if !strings.EqualFold(capture.Protocol, "HTTP") {
			return nil, fmt.Errorf("HTTP replay is only available for HTTP/HTTPS captures; use mcpnet_send_session_data for existing socket sessions")
		}
		if !s.authorize("network_write", "McpNet replay request", fmt.Sprintf("AI 客戶端要求重播捕獲請求 %d（%s %s，模式 %d）。這會向原始目標送出新的網路請求。允許？\n\nAn AI client wants to replay capture %d (%s %s, mode %d). This sends a new network request to the original destination. Allow?", id, capture.Method, redactCaptureURL(capture.URL), mode, id, capture.Method, redactCaptureURL(capture.URL), mode)) {
			return nil, fmt.Errorf("user denied request replay")
		}
		if mode == 3 {
			statusCode, err := HashMap.ResendHTTPNow(id, mode, s.app.App.Port())
			if err != nil {
				return nil, err
			}
			s.app.notifyMCPUIOperation(fmt.Sprintf("MCP 已重播捕獲 %d（狀態碼 %d；不會建立重複捕獲）", id, statusCode))
			return map[string]any{"capture_id": id, "mode": mode, "status_code": statusCode, "replayed": true}, nil
		}
		return s.doUICommand("重发请求", map[string]any{"Data": []int{id}, "Mode": mode})
	case "mcpnet_help":
		return map[string]any{
			"usage": "Use named MCP tools for captures, plugins, listener/proxy control, session sends, new HTTP/WebSocket/TCP/UDP requests, and HTTP request replay. Use mcpnet_list_commands to discover UI-backed commands, then mcpnet_command with the exact command name and UI-compatible Args. The local MCP access policy can require confirmation, grant selected categories, or grant unrestricted access.",
			"examples": []string{
				"启动监听", "停止监听", "监听状态", "设置IE代理",
				"插件列表", "導入插件ZIP", "设置插件状态", "重发请求",
				"mcpnet_send_http_request(url=http://127.0.0.1:8080/health, method=GET)",
				"mcpnet_send_socket_request(protocol=tcp, address=127.0.0.1:9000, data_base64=SGVsbG8=)",
				"mcpnet_send_websocket_request(url=ws://127.0.0.1:8080/ws, data_base64=SGVsbG8=, message_type=text)",
				"mcpnet_list_processes(name=MiniGameApp.exe)", "mcpnet_process_control(load_driver)", "mcpnet_process_control(add_name, name=MiniGameApp.exe)",
				"加载配置", "保存配置", "获取运行端口", "获取内网IP",
			},
			"capture_tools": []string{"mcpnet_list_captures", "mcpnet_get_capture", "mcpnet_get_messages"},
			"plugin_tools":  []string{"mcpnet_list_plugins", "mcpnet_get_plugin", "mcpnet_save_plugin", "mcpnet_delete_plugin", "mcpnet_plugin_ui_action", "mcpnet_import_plugin", "mcpnet_set_plugin_enabled"},
			"send_tools":    []string{"mcpnet_send_session_data", "mcpnet_send_http_request", "mcpnet_send_socket_request", "mcpnet_send_websocket_request", "mcpnet_replay_capture"},
			"process_tools": []string{"mcpnet_list_processes", "mcpnet_process_control"},
			"command_tools": []string{"mcpnet_list_commands", "mcpnet_command"},
			"note":          "Capture metadata omits private URL components, client IPs, and process IDs. Headers/payloads require include_body=true and permission under the local MCP access policy. Sends, replay, plugin installation/enabling, listener/proxy changes, and process-driver/filter changes use their respective permission categories. UI commands use private-read, settings, plugin-code, or network-write permission; unknown commands require direct confirmation. MCP permission changes always require direct confirmation.",
		}, nil
	case "mcpnet_list_captures":
		limit := argInt(args, "limit", 50)
		if limit < 1 {
			limit = 1
		}
		if limit > 200 {
			limit = 200
		}
		afterID := argInt(args, "after_id", 0)
		protocol := strings.ToLower(strings.TrimSpace(fmt.Sprint(args["protocol"])))
		all := HashMap.SnapshotCaptures()
		items := make([]map[string]any, 0, limit)
		next := afterID
		for _, c := range all {
			if c.ID <= afterID || (protocol != "" && protocol != "<nil>" && !strings.EqualFold(c.Protocol, protocol)) {
				continue
			}
			items = append(items, captureMetadata(c))
			next = c.ID
			if len(items) == limit {
				break
			}
		}
		return map[string]any{"captures": items, "count": len(items), "next_after_id": next, "has_more": hasCaptureAfter(all, next, protocol)}, nil
	case "mcpnet_get_capture":
		id := argInt(args, "capture_id", -1)
		if id < 0 {
			return nil, fmt.Errorf("capture_id must be a non-negative integer")
		}
		c, ok := HashMap.SnapshotCapture(id)
		if !ok {
			return nil, fmt.Errorf("capture %d was not found (it may have been cleared or not displayed)", id)
		}
		out := captureMetadata(c)
		if argBool(args, "include_body") {
			if !s.authorize("private_read", "McpNet private capture data", fmt.Sprintf("AI 客戶端要求讀取捕獲 %d 的完整 URL、IP/PID、標頭與請求／回應本文；可能含憑證或個人資料。是否分享？\n\nAn AI client requests the full URL, IP/PID, headers, and request/response bodies for capture %d. This may contain credentials or personal information. Share?", id, id)) {
				return nil, fmt.Errorf("user denied access to private capture data")
			}
			out["request_headers"] = c.RequestHead
			out["response_headers"] = c.ResponseHead
			out["url_private"] = c.URL
			out["pid"] = c.PID
			out["client_ip"] = c.ClientIP
			out["request_body_base64"], out["request_body_truncated"] = boundedBase64(c.RequestBody)
			out["response_body_base64"], out["response_body_truncated"] = boundedBase64(c.ResponseBody)
		}
		return out, nil
	case "mcpnet_get_messages":
		id := argInt(args, "capture_id", -1)
		if id < 0 {
			return nil, fmt.Errorf("capture_id must be a non-negative integer")
		}
		c, ok := HashMap.SnapshotCapture(id)
		if !ok {
			return nil, fmt.Errorf("capture %d was not found (it may have been cleared or not displayed)", id)
		}
		offset := argInt(args, "offset", 0)
		if offset < 0 {
			offset = 0
		}
		if offset > len(c.Messages) {
			offset = len(c.Messages)
		}
		limit := argInt(args, "limit", 100)
		if limit < 1 {
			limit = 1
		}
		if limit > 200 {
			limit = 200
		}
		includeBody := argBool(args, "include_body")
		if includeBody && !s.authorize("private_read", "McpNet private capture data", fmt.Sprintf("AI 客戶端要求讀取捕獲 %d 的第 %d 則訊息本文；可能含憑證或個人資料。是否分享？\n\nAn AI client requests message %d in capture %d. Its payload may contain credentials or personal information. Share?", id, offset, offset, id)) {
			return nil, fmt.Errorf("user denied access to private capture data")
		}
		if includeBody && limit > 1 {
			limit = 1
		}
		end := offset + limit
		if end > len(c.Messages) {
			end = len(c.Messages)
		}
		messages := make([]map[string]any, 0, end-offset)
		for _, m := range c.Messages[offset:end] {
			row := map[string]any{"index": m.Index, "direction": m.Direction, "type": m.Type, "time": m.Time, "length": m.Length}
			if includeBody {
				row["payload_base64"], row["payload_truncated"] = boundedBase64(m.Body)
			}
			messages = append(messages, row)
		}
		return map[string]any{"capture_id": id, "messages": messages, "count": len(messages), "total": len(c.Messages), "offset": offset, "has_more": end < len(c.Messages)}, nil
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func (s *mcpServer) authorizeMCPUICommand(command, category, prompt string) bool {
	// A client must never be able to grant itself broader MCP access by
	// invoking the settings UI command, even when the current profile is
	// unrestricted. Require an explicit native confirmation for this one.
	if command == "mcp設定權限" {
		return s.confirmUser("McpNet MCP permission change", prompt)
	}
	return s.authorize(category, "McpNet MCP command", prompt)
}

func (s *mcpServer) doUICommand(command string, args any) (any, error) {
	if s == nil || s.app == nil {
		return nil, fmt.Errorf("McpNet application is not available")
	}
	var wrapped map[string]any
	callArgs := any(nil)
	if args != nil {
		callArgs = map[string]any{"Command": command, "Args": args}
	} else {
		callArgs = map[string]any{"Command": command}
	}
	result := s.app.Do(callArgs)
	if result == nil {
		return nil, fmt.Errorf("UI command %q returned no result", command)
	}
	if value, ok := result.(map[string]any); ok {
		wrapped = value
		if message, ok := wrapped["error"].(string); ok && message != "" {
			return nil, fmt.Errorf("%s", message)
		}
	}
	return result, nil
}

func mcpUIStateSection(command string) string {
	switch command {
	case "启动监听", "停止监听", "设置IE代理":
		return "listener"
	case "加载驱动", "进程驱动添加进程名", "进程驱动添加PID":
		return "process"
	case "插件列表", "插件清單", "導入插件ZIP", "導入插件資料夾", "保存插件", "设置插件状态", "删除插件":
		return "plugins"
	default:
		return "refresh"
	}
}

// mcpUICommandPermission assigns UI-backed MCP calls to the same permission
// groups as their dedicated MCP tools. This lets custom permission profiles
// use ordinary UI features without granting the broad unrestricted mode, while
// unknown commands still require explicit per-call confirmation.
func mcpUICommandPermission(command string, args map[string]any) (category, operation string) {
	if command == "选择文件" || command == "保存文件对话框" {
		return "private_read", "檢查使用者明確提供的本機路徑 / inspect a user-supplied local path"
	}
	switch command {
	case "HTTP请求获取", "socket请求获取", "获取请求图片", "获取脚本日志", "获取脚本模板列表", "查找", "打开记录文件", "创建请求代码", "保存文件", "保存响应图片", "protobufToJson", "mcp權限", "mcp信息", "主题信息", "监听状态", "获取内网IP", "获取运行端口", "查询证书CommonName", "枚举进程":
		return "private_read", "讀取捕獲、腳本、設定或本機資訊 / read captured data, scripts, settings, or local information"
	case "插件界面動作", "插件源码", "保存插件", "删除插件", "導入插件ZIP", "導入插件資料夾", "设置插件状态", "获取脚本代码", "获取默认Go脚本代码", "格式化Go脚本代码", "保存Go脚本代码", "插件清單", "插件列表":
		return "plugin_code", "讀取或執行本機插件／腳本程式碼 / read or execute local plugin or script code"
	case "主动发送", "重发请求", "保存修改数据", "全部放行", "断点点击", "关闭请求会话":
		return "network_write", "發送或修改網路請求資料 / send or modify network request data"
	case "socket点击右键菜单":
		if strings.EqualFold(stringArg(args, "Type"), "empty") {
			return "settings", "清除所選連線的已捕獲資料 / clear captured data for the selected connection"
		}
		return "private_read", "複製所選連線的封包資料 / copy captured packet data from the selected connection"
	case "设置断点模式", "清空", "删除请求会话", "更新注释", "更新主题", "标记颜色", "工作状态", "启动监听", "停止监听", "mcp設定權限", "mcp设置路径", "mcp重启", "保存主题", "设置IE代理", "选择文件", "保存文件对话框", "取消搜索颜色标记", "CloseWindow", "保存配置", "加载配置", "重置所有配置", "设置右键菜单配置", "重置颜色列表", "保存上游代理使用规则", "设置上游代理", "保存强制TCP使用规则", "创建证书", "创建证书管理器", "安装CA证书", "导入证书", "安装默认证书", "导出默认证书", "应用默认证书", "修改端口号", "禁止UDP", "禁止TCP", "禁止缓存", "身份验证模式", "更新身份验证账号信息", "加载驱动", "枚举进程图标", "枚举全部进程图标", "进程驱动添加进程名", "进程驱动添加PID", "载入请求证书", "保存替换规则", "保存HOSTS规则", "删除证书管理器", "init", "置剪辑版文本":
		return "settings", "更改應用程式、代理、監聽、進程或憑證設定 / change application, proxy, capture, process, or certificate settings"
	default:
		return "generic", "命令可能讀取私人資料或更改應用程式／系統狀態 / the command may read private data or change application/system state"
	}
}

func mcpPluginListResult() map[string]any {
	info := pluginInfo()
	return map[string]any{"plugins": info["plugins"]}
}

func (s *mcpServer) confirmUser(title, message string) bool {
	if s != nil && s.confirm != nil {
		return s.confirm(title, message)
	}
	if s == nil || s.app == nil || s.app.ctx == nil {
		return false
	}
	answer, err := runtime.MessageDialog(s.app.ctx, runtime.MessageDialogOptions{
		Type:    runtime.QuestionDialog,
		Title:   title,
		Message: message,
		// Wails uses the native Yes/No buttons on Windows and returns the
		// canonical answer ("Yes" or "No"); custom labels are ignored there.
		// Use the standard values so No remains the safe default on every OS.
		Buttons:       []string{"Yes", "No"},
		DefaultButton: "No",
		CancelButton:  "No",
	})
	return err == nil && affirmativeDialogAnswer(answer)
}

func (s *mcpServer) authorize(category, title, message string) bool {
	// An injected confirmation callback is used by tests and embedding hosts to
	// control authorization deterministically. Do not let the user's persisted
	// unrestricted policy leak into those callers.
	if s != nil && s.confirm != nil {
		return s.confirm(title, message)
	}
	if loadMCPAccessSettings().permits(category) {
		return true
	}
	return s.confirmUser(title, message)
}

type mcpUICommand struct {
	Name               string   `json:"command"`
	Description        string   `json:"description"`
	ArgumentFields     []string `json:"argument_fields"`
	PermissionCategory string   `json:"permission_category"`
	Authorization      string   `json:"authorization"`
	PathOverride       bool     `json:"mcp_path_override,omitempty"`
}

// mcpUICommandCatalog lists the Go commands actually invoked by the Vue UI.
// AI clients can discover these names before using the generic command tool.
func mcpUICommandCatalog() []mcpUICommand {
	names := []string{
		"工作状态", "主动发送", "加载驱动", "加载配置", "打开记录文件", "全部放行", "关闭请求会话", "创建请求代码",
		"启动监听", "停止监听", "枚举进程", "枚举进程图标", "枚举全部进程图标",
		"创建证书", "创建证书管理器", "安装默认证书", "安装CA证书", "导入证书", "导出默认证书",
		"设置上游代理", "设置右键菜单配置", "设置断点模式", "设置IE代理", "删除证书管理器", "删除请求会话",
		"应用默认证书", "更新主题", "更新身份验证账号信息", "更新注释", "身份验证模式", "取消搜索颜色标记",
		"进程驱动添加进程名", "进程驱动添加PID", "保存上游代理使用规则", "保存文件", "保存文件对话框",
		"保存响应图片", "保存修改数据", "保存配置", "保存强制TCP使用规则", "保存替换规则", "保存Go脚本代码",
		"保存HOSTS规则", "查找", "查询证书CommonName", "标记颜色", "重发请求", "重置所有配置", "重置颜色列表",
		"修改端口号", "格式化Go脚本代码", "载入请求证书", "选择文件", "断点点击", "清空", "获取内网IP",
		"获取运行端口", "获取请求图片", "获取脚本日志", "获取脚本代码", "获取脚本模板列表", "获取默认Go脚本代码",
		"插件界面動作", "禁止缓存", "禁止TCP", "禁止UDP", "置剪辑版文本", "CloseWindow", "HTTP请求获取",
		"protobufToJson", "socket点击右键菜单", "socket请求获取", "init",
		"主题信息", "设置插件状态", "保存主题", "监听状态", "插件列表", "mcp设置路径", "mcp信息", "mcp重启", "mcp設定權限", "mcp權限",
		// Plugin editor backend operations are not all called directly by the
		// current Vue UI, but are part of the supported plugin workflow.
		"插件源码", "插件清單", "保存插件", "删除插件", "導入插件ZIP", "導入插件資料夾",
	}
	commands := make([]mcpUICommand, 0, len(names))
	for _, name := range names {
		description := "Invoke this existing McpNet UI/backend command using its UI-compatible Args object. Operations may read private data or change local/network state and are subject to MCP authorization."
		category, _ := mcpUICommandPermission(name, nil)
		authorization := "requires_permission_category_or_direct_confirmation"
		pathOverride := name == "选择文件" || name == "保存文件对话框" || name == "导出默认证书"
		if name == "mcp設定權限" {
			authorization = "always_requires_direct_confirmation"
		}
		switch name {
		case "插件源码":
			description = `Args: {"Id":"plugin-id"}. Returns local plugin Go source; treat as private code.`
		case "插件清單":
			description = `Args: {"Id":"plugin-id"}. Returns the plugin manifest.`
		case "保存插件":
			description = `Args: {"Manifest":{"id":"plugin-id","name":"Name","version":"1.0.0","author":"Author","description":"Description","enabled":false,"order":100},"WithSource":true,"Source":"package main ..."}. Keep enabled=false; separately approve enabling.`
		case "删除插件":
			description = `Args: {"Id":"plugin-id"}. Permanently deletes that local plugin and its source.`
		case "導入插件ZIP":
			description = `Args: {"Data":"<base64 ZIP containing main.go and plugin.json>"}. Imported plugin stays disabled.`
		case "導入插件資料夾":
			description = `Args: {"Manifest":"<plugin.json JSON>","Source":"<main.go source>"}. Imported plugin stays disabled.`
		case "插件界面動作":
			description = `Args: {"Id":"plugin-id","Action":"action-name","Payload":"JSON or text"}. Calls the enabled plugin's OnPluginAction Go hook.`
		case "socket点击右键菜单":
			description = `Args: {"Type":"operation","Theology":connection-id,...}. Type="empty" clears captured data and requires settings permission; other types copy packet data and require private-read permission.`
		case "选择文件":
			description = `For MCP, pass Args.Path as an absolute path to an existing file. The file is not read by this command; it returns the path without opening a native file dialog.`
		case "保存文件对话框":
			description = `For MCP, pass Args.Path as an absolute output path. The command returns the path without opening a native save dialog or writing a file.`
		case "导出默认证书":
			description = `For MCP, pass Args.Path as an absolute .cer output path. The default CA certificate is written without opening a native dialog; existing files are never overwritten.`
		}
		if name == "socket点击右键菜单" {
			category = "context_dependent"
		}
		commands = append(commands, mcpUICommand{
			Name:               name,
			Description:        description,
			ArgumentFields:     mcpUICommandArgumentFields(name),
			PermissionCategory: category,
			Authorization:      authorization,
			PathOverride:       pathOverride,
		})
	}
	return commands
}

// mcpDialogPath replaces file-picker UI for MCP callers that provide an
// explicit absolute path. It never reads or writes the selected file.
func mcpDialogPath(command, path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" || !filepath.IsAbs(path) {
		return "", fmt.Errorf("MCP command %q does not open native file dialogs; provide Args.Path as an absolute path", command)
	}
	path = filepath.Clean(path)
	if command == "选择文件" {
		info, err := os.Stat(path)
		if err != nil {
			return "", fmt.Errorf("selected file %q: %w", path, err)
		}
		if info.IsDir() {
			return "", fmt.Errorf("selected path %q is a directory; provide a file path", path)
		}
		return path, nil
	}
	if command == "保存文件对话框" {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return "", fmt.Errorf("output path %q is a directory; provide a file path", path)
		}
		parent := filepath.Dir(path)
		info, err := os.Stat(parent)
		if err != nil {
			return "", fmt.Errorf("output directory %q: %w", parent, err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("output parent %q is not a directory", parent)
		}
		return path, nil
	}
	return "", fmt.Errorf("command %q is not an MCP file-dialog command", command)
}

func mcpExportDefaultCertificate(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" || !filepath.IsAbs(path) {
		return "", fmt.Errorf("MCP command %q does not open native file dialogs; provide Args.Path as an absolute path", "导出默认证书")
	}
	path = filepath.Clean(path)
	if !strings.HasSuffix(strings.ToLower(path), ".cer") {
		path += ".cer"
	}
	parent := filepath.Dir(path)
	info, err := os.Stat(parent)
	if err != nil {
		return "", fmt.Errorf("output directory %q: %w", parent, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("output parent %q is not a directory", parent)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", fmt.Errorf("create certificate file %q: %w", path, err)
	}
	if _, err := file.Write([]byte(public.RootCa)); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return "", fmt.Errorf("write certificate file %q: %w", path, err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("close certificate file %q: %w", path, err)
	}
	return path, nil
}

// mcpUICommandArgumentFields exposes the Args keys used by the Vue UI and
// backend command handlers. Values keep their UI-compatible JSON types; this
// list is discovery metadata, not a coercion layer.
func mcpUICommandArgumentFields(command string) []string {
	fields := map[string][]string{
		"置剪辑版文本":         {"Data"},
		"protobufToJson": {"Data", "skip"},
		"保存配置":           {"Type", "Data"},
		"查找":             {"Options", "Value", "Type", "Range", "Color", "ProtoSkip"},
		"CloseWindow":    {"Filter", "KeysStrings", "StorageColumns"},
		"保存文件对话框":        {"Title", "Filters", "Path"},
		"保存文件":           {"Path", "ALL", "Data"},
		"选择文件":           {"Title", "Filters", "Path"},
		"导出默认证书":         {"Path"},
		"打开记录文件":         {"Path"},
		"更新注释":           {"Theology", "Data"},
		"更新主题":           {"Dark"},
		"创建请求代码":         {"Data", "Lang", "Module"},
		"标记颜色":           {"Data", "empty"},
		"删除请求会话":         {"Data"},
		"关闭请求会话":         {"Data"},
		"重发请求":           {"Data", "Mode"},
		"工作状态":           {"State"},
		"设置断点模式":         {"break"},
		"断点点击":           {"NextBreak", "Theology"},
		"获取请求图片":         {"Theology"},
		"保存修改数据":         {"Theology", "Type", "Tabs", "Data", "UTF8"},
		"HTTP请求获取":       {"Theology"},
		"设置右键菜单配置":       {"StopSend", "StopRec", "StopALL", "Theology", "IsWs", "IsTCP"},
		"socket点击右键菜单":   {"Type", "Theology", "IsWs", "IsTCP", "SelectedID"},
		"主动发送":           {"direction", "SendType", "wsType", "Data", "Theology", "IsWs", "IsTCP", "IsUDP"},
		"保存响应图片":         {"Theology", "type", "IsRequest"},
		"socket请求获取":     {"Theology", "Index"},
		"设置IE代理":         {"Set"},
		"设置上游代理":         {"Data", "Set"},
		"保存上游代理使用规则":     {"Data"},
		"禁止TCP":          {"DisableTCP"},
		"禁止UDP":          {"DisableUDP"},
		"身份验证模式":         {"authentication"},
		"禁止缓存":           {"DisableBrowserCache"},
		"重置所有配置":         {"Port"},
		"修改端口号":          {"Port"},
		"更新身份验证账号信息":     {"Data"},
		"保存HOSTS规则":      {"Data"},
		"重置颜色列表":         {"dark"},
		"保存强制TCP使用规则":    {"Data"},
		"进程驱动添加PID":      {"PID", "isSelected"},
		"进程驱动添加进程名":      {"Name", "isSet"},
		"保存替换规则":         {"Data"},
		"删除证书管理器":        {"context"},
		"查询证书CommonName": {"context", "rule"},
		"载入请求证书":         {"Data"},
		"创建证书":           {"domain", "country", "company", "department", "province", "city", "outTime"},
		"安装CA证书":         {"CaFilePath"},
		"导入证书":           {"CaFilePath", "KeyFilePath"},
		"格式化Go脚本代码":      {"code"},
		"保存Go脚本代码":       {"code"},
		"获取默认Go脚本代码":     {"code"},
		"插件界面動作":         {"Id", "Action", "Payload"},
		"插件源码":           {"Id"},
		"插件清單":           {"Id"},
		"保存插件":           {"Manifest", "WithSource", "Source"},
		"删除插件":           {"Id"},
		"導入插件ZIP":        {"Data"},
		"導入插件資料夾":        {"Manifest", "Source"},
		"设置插件状态":         {"Id", "Enabled"},
		"mcp设置路径":        {"Path"},
		"mcp設定權限":        {"Mode", "AllowPrivateRead", "AllowSettings", "AllowPluginCode", "AllowNetworkWrites"},
	}
	if found, ok := fields[command]; ok {
		return found
	}
	return []string{}
}

func mcpTools() []mcpTool {
	obj := func(props map[string]any, required []string) map[string]any {
		m := map[string]any{"type": "object", "properties": props}
		if len(required) > 0 {
			m["required"] = required
		}
		return m
	}
	return []mcpTool{
		{
			Name:        "mcpnet_list_captures",
			Description: "List captured HTTP/HTTPS, WebSocket, TCP, and UDP sessions. Returns metadata only; no headers, payloads, process IDs, client IPs, or private URL path/query/userinfo/fragment components.",
			InputSchema: obj(map[string]any{"after_id": map[string]any{"type": "integer", "minimum": 0}, "limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 200}, "protocol": map[string]any{"type": "string", "description": "Optional HTTP, WebSocket, TCP, or UDP filter."}}, nil),
		},
		{
			Name:        "mcpnet_get_capture",
			Description: "Get one captured session's metadata. Private URL components are redacted. Set include_body=true to request the original URL, client IP, process ID, headers, and request/response payloads (Base64, capped at 64 KiB each); a direct user confirmation dialog appears first because these may contain personal data.",
			InputSchema: obj(map[string]any{"capture_id": map[string]any{"type": "integer", "minimum": 0}, "include_body": map[string]any{"type": "boolean", "default": false}}, []string{"capture_id"}),
		},
		{
			Name:        "mcpnet_get_messages",
			Description: "List WebSocket/TCP/UDP messages for a captured session. Payloads are omitted unless include_body=true; then one message per call is returned with a Base64 payload capped at 64 KiB after direct user confirmation. Payloads may contain private data.",
			InputSchema: obj(map[string]any{"capture_id": map[string]any{"type": "integer", "minimum": 0}, "offset": map[string]any{"type": "integer", "minimum": 0}, "limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 200}, "include_body": map[string]any{"type": "boolean", "default": false}}, []string{"capture_id"}),
		},
		{
			Name:        "mcpnet_status",
			Description: "Return listener, system-proxy, MCP, and process-driver status for McpNet.",
			InputSchema: obj(map[string]any{}, nil),
		},
		{
			Name:        "mcpnet_list_processes",
			Description: "Find currently running local processes by exact executable file name and return matching PID/name pairs only. Process metadata may be private and requires private-read permission or confirmation.",
			InputSchema: obj(map[string]any{"name": map[string]any{"type": "string", "description": "Exact executable file name, e.g. MiniGameApp.exe; paths are not accepted."}}, []string{"name"}),
		},
		{
			Name:        "mcpnet_process_control",
			Description: "Control process interception without UI automation: load the network interception driver, add/remove a process-name or PID rule, or clear rules. Driver loading and filter changes require settings permission or direct user confirmation.",
			InputSchema: obj(map[string]any{
				"action": map[string]any{"type": "string", "enum": []string{"load_driver", "add_name", "remove_name", "add_pid", "remove_pid", "clear"}},
				"name":   map[string]any{"type": "string", "description": "Executable file name for add_name/remove_name."},
				"pid":    map[string]any{"type": "integer", "minimum": 1, "description": "PID for add_pid/remove_pid."},
			}, []string{"action"}),
		},
		{
			Name:        "mcpnet_list_plugins",
			Description: "List local plugins and their enabled/error status. Does not include source code.",
			InputSchema: obj(map[string]any{}, nil),
		},
		{
			Name:        "mcpnet_get_plugin",
			Description: "Get a local plugin manifest. Set include_source=true to request its Go source; a direct user confirmation is required because local source may be private.",
			InputSchema: obj(map[string]any{
				"plugin_id":      map[string]any{"type": "string", "description": "Exact plugin id from mcpnet_list_plugins."},
				"include_source": map[string]any{"type": "boolean", "default": false},
			}, []string{"plugin_id"}),
		},
		{
			Name:        "mcpnet_save_plugin",
			Description: "Create or update a local plugin manifest and optionally its main.go source. The manifest must keep enabled=false; use mcpnet_set_plugin_enabled separately after reviewing the code. Requires plugin-code permission or direct confirmation.",
			InputSchema: obj(map[string]any{
				"manifest_json": map[string]any{"type": "string", "description": "JSON plugin manifest with id, name, version, author, description, enabled:false, and order."},
				"source":        map[string]any{"type": "string", "description": "Optional main.go source. Omit to update only the manifest."},
			}, []string{"manifest_json"}),
		},
		{
			Name:        "mcpnet_delete_plugin",
			Description: "Permanently delete a local plugin manifest and source. Requires plugin-code permission or direct confirmation.",
			InputSchema: obj(map[string]any{"plugin_id": map[string]any{"type": "string", "description": "Exact plugin id from mcpnet_list_plugins."}}, []string{"plugin_id"}),
		},
		{
			Name:        "mcpnet_plugin_ui_action",
			Description: "Call an enabled plugin's OnPluginAction hook with an action name and text payload. Executes trusted local Go code and requires plugin-code permission or direct confirmation.",
			InputSchema: obj(map[string]any{
				"plugin_id": map[string]any{"type": "string"},
				"action":    map[string]any{"type": "string", "maxLength": 128},
				"payload":   map[string]any{"type": "string", "maxLength": 16384},
			}, []string{"plugin_id", "action"}),
		},
		{
			Name:        "mcpnet_import_plugin",
			Description: "Install a community Go plugin from Base64 ZIP or from a JSON manifest plus main.go source. Imported code remains disabled by default; the user must separately approve enabling it.",
			InputSchema: obj(map[string]any{
				"zip_base64":    map[string]any{"type": "string", "description": "Base64-encoded ZIP containing main.go and one JSON manifest; max 16 MiB."},
				"manifest_json": map[string]any{"type": "string", "description": "Plugin JSON manifest; use with source."},
				"source":        map[string]any{"type": "string", "description": "main.go source; use with manifest_json."},
			}, nil),
		},
		{
			Name:        "mcpnet_set_plugin_enabled",
			Description: "Enable or disable a local plugin. Enabling executes trusted local Go code and requires direct user approval.",
			InputSchema: obj(map[string]any{"plugin_id": map[string]any{"type": "string"}, "enabled": map[string]any{"type": "boolean"}}, []string{"plugin_id", "enabled"}),
		},
		{
			Name:        "mcpnet_send_session_data",
			Description: "Send Base64 bytes through an existing WebSocket, TCP, or UDP capture session. Each send requires direct user approval. This does not create arbitrary new connections.",
			InputSchema: obj(map[string]any{
				"capture_id":  map[string]any{"type": "integer", "minimum": 0},
				"protocol":    map[string]any{"type": "string", "enum": []string{"websocket", "tcp", "udp"}},
				"direction":   map[string]any{"type": "string", "enum": []string{"Server", "Client"}},
				"data_base64": map[string]any{"type": "string", "description": "Bytes to send, Base64 encoded."},
			}, []string{"capture_id", "protocol", "direction", "data_base64"}),
		},
		{
			Name:        "mcpnet_send_http_request",
			Description: "Create a new direct HTTP/HTTPS request to the specified URL. Each request requires network-write approval; response headers and body are returned only when include_response=true and private-read is approved. Redirects are not followed; request and response bodies are capped at 64 KiB.",
			InputSchema: obj(map[string]any{
				"url":              map[string]any{"type": "string", "description": "Absolute http:// or https:// URL; embedded credentials are rejected."},
				"method":           map[string]any{"type": "string", "default": "GET"},
				"headers":          map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}, "description": "Optional single-line request headers; Host and Content-Length are controlled by the client."},
				"body_base64":      map[string]any{"type": "string", "description": "Optional request body encoded as Base64; maximum 64 KiB."},
				"include_response": map[string]any{"type": "boolean", "default": true, "description": "Whether to return response headers and body; requires private-read approval."},
				"timeout_ms":       map[string]any{"type": "integer", "minimum": 100, "maximum": 60000, "default": 10000},
			}, []string{"url"}),
		},
		{
			Name:        "mcpnet_send_socket_request",
			Description: "Create a new TCP or UDP connection and send arbitrary Base64 bytes. Each send requires network-write approval; returned data requires private-read approval. TCP reads until EOF or timeout; UDP reads one datagram. Optional TLS is supported for TCP.",
			InputSchema: obj(map[string]any{
				"protocol":      map[string]any{"type": "string", "enum": []string{"tcp", "udp"}},
				"address":       map[string]any{"type": "string", "description": "Destination host:port, including bracketed IPv6 such as [::1]:9000."},
				"data_base64":   map[string]any{"type": "string", "description": "Non-empty payload encoded as Base64; maximum 64 KiB."},
				"use_tls":       map[string]any{"type": "boolean", "default": false, "description": "Use TLS over TCP with certificate verification."},
				"server_name":   map[string]any{"type": "string", "description": "Optional TLS certificate/SNI name; defaults to the address host."},
				"read_response": map[string]any{"type": "boolean", "default": true, "description": "Whether to read and return response bytes; requires private-read approval."},
				"timeout_ms":    map[string]any{"type": "integer", "minimum": 100, "maximum": 60000, "default": 5000},
			}, []string{"protocol", "address", "data_base64"}),
		},
		{
			Name:        "mcpnet_send_websocket_request",
			Description: "Create a new ws:// or wss:// connection using the WebSocket handshake and send one text or binary message. Each send requires network-write authorization; response messages require private-read authorization. Message size is capped at 64 KiB.",
			InputSchema: obj(map[string]any{
				"url":           map[string]any{"type": "string", "description": "Absolute ws:// or wss:// URL; credentials and fragments are rejected."},
				"data_base64":   map[string]any{"type": "string", "description": "Non-empty message payload encoded as Base64; maximum 64 KiB."},
				"message_type":  map[string]any{"type": "string", "enum": []string{"text", "binary"}, "default": "text"},
				"headers":       map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}, "description": "Optional request headers; WebSocket handshake headers are managed by the client."},
				"read_response": map[string]any{"type": "boolean", "default": true, "description": "Whether to read and return one response message; requires private-read approval."},
				"timeout_ms":    map[string]any{"type": "integer", "minimum": 100, "maximum": 60000, "default": 5000},
			}, []string{"url", "data_base64"}),
		},
		{
			Name:        "mcpnet_replay_capture",
			Description: "Replay a captured HTTP/HTTPS request to its original destination. Every replay displays a user confirmation; it sends a new network request.",
			InputSchema: obj(map[string]any{
				"capture_id": map[string]any{"type": "integer", "minimum": 0},
				"mode":       map[string]any{"type": "integer", "enum": []int{3, 1, 2}, "default": 3, "description": "3 normal replay, 1 intercept upload, 2 intercept download."},
			}, []string{"capture_id"}),
		},
		{
			Name:        "mcpnet_start_listener",
			Description: "Start the McpNet capture listener (equivalent to 启动监听).",
			InputSchema: obj(map[string]any{}, nil),
		},
		{
			Name:        "mcpnet_stop_listener",
			Description: "Stop the McpNet capture listener and clear the system proxy (equivalent to 停止监听).",
			InputSchema: obj(map[string]any{}, nil),
		},
		{
			Name:        "mcpnet_set_system_proxy",
			Description: "Enable or disable the Windows system IE proxy. Enabling also starts the listener.",
			InputSchema: obj(map[string]any{"enable": map[string]any{"type": "boolean"}}, []string{"enable"}),
		},
		{
			Name:        "mcpnet_command",
			Description: "Execute a McpNet UI-backed command. Use mcpnet_list_commands to discover exact supported command names, argument_fields, permission_category, authorization, and MCP path overrides. The Args object must use those UI-compatible keys and types. File-picker and default-certificate-export commands accept an explicit absolute Args.Path and never show a native dialog through MCP. Commands use private-read, settings, plugin-code, or network-write permissions; unknown commands require per-call confirmation. Changing MCP permissions always requires direct user confirmation.",
			InputSchema: obj(map[string]any{
				"command": map[string]any{"type": "string", "description": "Exact UI command name returned by mcpnet_list_commands, e.g. 启动监听 / 停止监听 / 设置IE代理 / 保存配置"},
				"args":    map[string]any{"type": "object", "description": "Same Args object the UI sends; omit if none."},
			}, []string{"command"}),
		},
		{
			Name:        "mcpnet_list_commands",
			Description: "List UI/backend command names with argument_fields, permission_category, and authorization so an AI client can discover the broader application command surface and construct UI-compatible Args objects. This does not execute commands or expose captured data.",
			InputSchema: obj(map[string]any{}, nil),
		},
		{
			Name:        "mcpnet_help",
			Description: "Show usage and example McpNet commands.",
			InputSchema: obj(map[string]any{}, nil),
		},
	}
}
