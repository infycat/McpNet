package main

import (
	"changeme/MapHash"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultMCPPort = 2026
	mcpServerName  = "mcpnet"
)

type mcpServer struct {
	app *App

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
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"ok":true,"name":"mcpnet","version":"1.2.0","port":%d}`, defaultMCPPort)
	})
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

func (s *mcpServer) handleMCP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Mcp-Session-Id")
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

	body, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
	w.Header().Set("Mcp-Session-Id", fmt.Sprintf("mcpnet-%d", time.Now().UnixNano()))
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
	if req.Method == "" {
		return mcpError(req.ID, -32600, "invalid request"), false
	}
	notification := len(req.ID) == 0 || string(req.ID) == "null"

	switch req.Method {
	case "initialize":
		return mcpResult(req.ID, map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "mcpnet",
				"version": "1.2.0",
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
		"capture_id": c.ID, "protocol": c.Protocol, "method": c.Method, "url": c.URL,
		"pid": c.PID, "client_ip": c.ClientIP, "status_code": c.StatusCode,
		"request_time": c.RequestAt, "response_time": c.ResponseAt,
		"request_body_length": len(c.RequestBody), "response_body_length": len(c.ResponseBody),
		"sent_bytes": c.SentBytes, "received_bytes": c.RecvBytes, "message_count": len(c.Messages),
	}
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
		info := s.app.mcpInfo()
		return map[string]any{
			"listener_running": running,
			"system_proxy":     proxy,
			"mcp":              info,
		}, nil
	case "mcpnet_start_listener":
		if err := s.app.startListener(); err != nil {
			return nil, err
		}
		running, proxy := s.app.listenerStatus()
		return map[string]any{"listener_running": running, "system_proxy": proxy}, nil
	case "mcpnet_stop_listener":
		s.app.stopListener()
		running, proxy := s.app.listenerStatus()
		return map[string]any{"listener_running": running, "system_proxy": proxy}, nil
	case "mcpnet_set_system_proxy":
		enable, _ := args["enable"].(bool)
		ok := s.app.setSystemProxy(enable)
		running, proxy := s.app.listenerStatus()
		return map[string]any{"ok": ok, "listener_running": running, "system_proxy": proxy}, nil
	case "mcpnet_command":
		command, _ := args["command"].(string)
		if strings.TrimSpace(command) == "" {
			return nil, fmt.Errorf("command is required")
		}
		var cmdArgs any
		if v, ok := args["args"]; ok {
			cmdArgs = v
		}
		res := s.app.Do(map[string]any{"Command": command, "Args": cmdArgs})
		return map[string]any{"command": command, "result": res}, nil
	case "mcpnet_help":
		return map[string]any{
			"usage": "Use mcpnet_command with any command accepted by the McpNet UI.",
			"examples": []string{
				"启动监听", "停止监听", "监听状态", "设置IE代理",
				"加载配置", "保存配置", "获取运行端口", "获取内网IP",
			},
			"capture_tools": []string{"mcpnet_list_captures", "mcpnet_get_capture", "mcpnet_get_messages"},
			"note":          "Capture listing is metadata-only. Headers and payloads are excluded unless include_body is explicitly true. The generic UI command tool can change app state.",
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
			out["request_headers"] = c.RequestHead
			out["response_headers"] = c.ResponseHead
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
			Description: "List captured HTTP/HTTPS, WebSocket, TCP, and UDP sessions. Returns metadata only; no headers or payloads.",
			InputSchema: obj(map[string]any{"after_id": map[string]any{"type": "integer", "minimum": 0}, "limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 200}, "protocol": map[string]any{"type": "string", "description": "Optional HTTP, WebSocket, TCP, or UDP filter."}}, nil),
		},
		{
			Name:        "mcpnet_get_capture",
			Description: "Get one captured session's metadata. Set include_body=true to include headers and request/response payloads (base64, capped at 64 KiB each); these may contain credentials or personal data.",
			InputSchema: obj(map[string]any{"capture_id": map[string]any{"type": "integer", "minimum": 0}, "include_body": map[string]any{"type": "boolean", "default": false}}, []string{"capture_id"}),
		},
		{
			Name:        "mcpnet_get_messages",
			Description: "List WebSocket/TCP/UDP messages for a captured session. Payloads are omitted unless include_body=true; then one message per call is returned with a Base64 payload capped at 64 KiB. Payloads may contain private data.",
			InputSchema: obj(map[string]any{"capture_id": map[string]any{"type": "integer", "minimum": 0}, "offset": map[string]any{"type": "integer", "minimum": 0}, "limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 200}, "include_body": map[string]any{"type": "boolean", "default": false}}, []string{"capture_id"}),
		},
		{
			Name:        "mcpnet_status",
			Description: "Return listener/system-proxy/MCP status for McpNet.",
			InputSchema: obj(map[string]any{}, nil),
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
			Description: "Execute any McpNet UI command. This exposes the same command dispatcher used by the frontend, including capture start/stop, config load/save, rules, hosts, scripts and request tools.",
			InputSchema: obj(map[string]any{
				"command": map[string]any{"type": "string", "description": "UI command name, e.g. 启动监听 / 停止监听 / 设置IE代理 / 保存配置"},
				"args":    map[string]any{"type": "object", "description": "Same Args object the UI sends; omit if none."},
			}, []string{"command"}),
		},
		{
			Name:        "mcpnet_help",
			Description: "Show usage and example McpNet commands.",
			InputSchema: obj(map[string]any{}, nil),
		},
	}
}
