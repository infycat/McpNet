package main

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

// ---------------------------------------------------------------------------
// Plugin manager
//
// A plugin is a directory under <data>/plugins containing:
//   plugin.json  {id,name,version,author,description,enabled,order}
//   main.go      Go source evaluated with yaegi
//
// The source may implement any of these entry points:
//   func OnHTTPRequest()
//   func OnHTTPResponse()
//   func OnWebSocket()
//   func OnUDP()
//   func OnURLDisplay()
//
// While a hook runs, the `mcpnet` package exposes the current context
// (direction, host, path, url, body, pid, process) plus helpers.
// ---------------------------------------------------------------------------

type pluginManifest struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Author      string `json:"author"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	Order       int    `json:"order"`
}

func parsePluginManifest(value any) (pluginManifest, error) {
	var manifest pluginManifest
	var raw []byte
	switch v := value.(type) {
	case string:
		raw = []byte(v)
	case nil:
		return manifest, fmt.Errorf("插件清單為空")
	default:
		var err error
		raw, err = json.Marshal(v)
		if err != nil {
			return manifest, err
		}
	}
	if err := json.Unmarshal(raw, &manifest); err == nil {
		return manifest, nil
	}

	// SyJson.GetData stringifies string values and leaves their JSON escaping
	// intact. Accept that legacy representation as well as raw JSON/object data.
	if unescaped, err := strconv.Unquote(`"` + string(raw) + `"`); err == nil {
		if err := json.Unmarshal([]byte(unescaped), &manifest); err == nil {
			return manifest, nil
		}
	}
	return manifest, fmt.Errorf("插件清單不是有效的 JSON")
}

type loadedPlugin struct {
	Manifest pluginManifest
	Dir      string
	Source   string
	Interp   *interp.Interpreter
	Err      string

	hookHTTPReq  bool
	hookHTTPResp bool
	hookWS       bool
	hookUDP      bool
	hookURLView  bool
	hookUIAction bool
}

var (
	pluginMu    sync.Mutex
	hookMu      sync.Mutex
	pluginList  []*loadedPlugin
	pluginStamp string
)

// ---- runtime context shared with plugin hooks ----

type pluginCtx struct {
	PluginID       string
	Direction      string
	Host           string
	Path           string
	URL            string
	Method         string
	Body           []byte
	WireBody       []byte
	WireBodySet    bool
	DisplayBody    []byte
	DisplayBodySet bool
	PID            int
	Process        string
	Changed        bool
	Action         string
	Payload        string
	UIResult       string
}

var (
	ctxMu  sync.Mutex
	curCtx *pluginCtx
)

func pluginDataDir() string {
	return filepath.Join(mcpDataDir(), "plugins")
}

func pluginManifestPath(dir string) string { return filepath.Join(dir, "plugin.json") }
func pluginSourcePath(dir string) string   { return filepath.Join(dir, "main.go") }

func ensurePluginDir() error {
	if err := os.MkdirAll(pluginDataDir(), 0700); err != nil {
		return err
	}
	migrateLegacyPlugins()
	return nil
}

// migrateLegacyPlugins converts the flat *.json files written by the earlier
// codec build into proper plugin directories so nothing a user already had
// disappears from the list.
func migrateLegacyPlugins() {
	dir := pluginDataDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			continue
		}
		flat := filepath.Join(dir, e.Name())
		bs, err := os.ReadFile(flat)
		if err != nil {
			continue
		}
		var legacy struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			Type      string `json:"type"`
			Codec     string `json:"codec"`
			Enabled   bool   `json:"enabled"`
			Host      string `json:"host"`
			Path      string `json:"path"`
			Process   string `json:"process"`
			Key       string `json:"key"`
			Direction string `json:"direction"`
			Order     int    `json:"order"`
		}
		if json.Unmarshal(bs, &legacy) != nil {
			continue
		}
		id := legacy.ID
		if id == "" {
			id = strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		}
		id = sanitizePluginID(id)
		target := filepath.Join(dir, id)
		if err := os.MkdirAll(target, 0700); err != nil {
			continue
		}
		name := legacy.Name
		if name == "" {
			name = id
		}
		m := pluginManifest{
			ID: id, Name: name, Version: "1.0.0", Enabled: legacy.Enabled,
			Order:       legacy.Order,
			Description: fmt.Sprintf("%s · %s", legacy.Type, legacy.Codec),
		}
		mb, _ := json.MarshalIndent(m, "", "  ")
		if err := os.WriteFile(pluginManifestPath(target), mb, 0600); err != nil {
			continue
		}
		if _, err := os.Stat(pluginSourcePath(target)); err != nil {
			_ = os.WriteFile(pluginSourcePath(target), []byte(defaultPluginSource(name)), 0600)
		}
		_ = os.Remove(flat)
	}
}

func defaultPluginSource(name string) string {
	return "package main\n\nimport \"mcpnet\"\n\n// " + name + "\n// 可用入口：OnHTTPRequest / OnHTTPResponse / OnWebSocket / OnUDP / OnURLDisplay\nfunc OnWebSocket() {\n\tif mcpnet.Direction() == \"in\" {\n\t\tmcpnet.Log(\"hit: \" + mcpnet.Host() + mcpnet.Path())\n\t}\n}\n"
}

// scanPlugins reads plugin.json + main.go from disk. Interpreters are built
// lazily on first use so a broken plugin cannot stop the app from starting.
func scanPlugins() []*loadedPlugin {
	pluginMu.Lock()
	defer pluginMu.Unlock()
	dir := pluginDataDir()
	_ = ensurePluginDir()
	entries, _ := os.ReadDir(dir)
	var out []*loadedPlugin
	var stamp strings.Builder
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pdir := filepath.Join(dir, e.Name())
		bs, err := os.ReadFile(pluginManifestPath(pdir))
		if err != nil {
			continue
		}
		var m pluginManifest
		if json.Unmarshal(bs, &m) != nil {
			continue
		}
		if m.ID == "" {
			m.ID = e.Name()
		}
		if m.Name == "" {
			m.Name = m.ID
		}
		src, _ := os.ReadFile(pluginSourcePath(pdir))
		stamp.WriteString(m.ID)
		stamp.WriteString(fmt.Sprintf("%v%d%d", m.Enabled, len(src), m.Order))
		out = append(out, &loadedPlugin{
			Manifest: m,
			Dir:      pdir,
			Source:   string(src),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Manifest.Order != out[j].Manifest.Order {
			return out[i].Manifest.Order < out[j].Manifest.Order
		}
		return out[i].Manifest.Name < out[j].Manifest.Name
	})
	pluginStamp = stamp.String()
	pluginList = out
	return out
}

func plugins() []*loadedPlugin {
	pluginMu.Lock()
	cached := pluginList
	pluginMu.Unlock()
	if cached == nil {
		return scanPlugins()
	}
	return cached
}

func invalidatePluginCache() {
	pluginMu.Lock()
	pluginList = nil
	pluginStamp = ""
	pluginMu.Unlock()
}

// compile builds the yaegi interpreter for a plugin and detects its hooks.
func (p *loadedPlugin) compile() error {
	if p.Interp != nil {
		return nil
	}
	if strings.TrimSpace(p.Source) == "" {
		return fmt.Errorf("插件源码为空")
	}
	i := interp.New(interp.Options{})
	i.Use(stdlib.Symbols)
	if err := i.Use(pluginExports()); err != nil {
		return fmt.Errorf("register mcpnet plugin API: %w", err)
	}
	src := p.Source
	if !strings.Contains(src, "package ") {
		src = "package main\n" + src
	}
	if _, err := i.Eval(src); err != nil {
		return err
	}
	for _, h := range []struct {
		name string
		flag *bool
	}{
		{"OnHTTPRequest", &p.hookHTTPReq},
		{"OnHTTPResponse", &p.hookHTTPResp},
		{"OnWebSocket", &p.hookWS},
		{"OnUDP", &p.hookUDP},
		{"OnURLDisplay", &p.hookURLView},
		{"OnPluginAction", &p.hookUIAction},
	} {
		if v, err := i.Eval(h.name); err == nil && v.IsValid() {
			*h.flag = true
		}
	}
	p.Interp = i
	return nil
}

func (p *loadedPlugin) call(hook string) {
	if p.Interp == nil {
		return
	}
	v, err := p.Interp.Eval(hook)
	if err != nil || !v.IsValid() {
		return
	}
	fn, ok := v.Interface().(func())
	if !ok {
		return
	}
	defer func() { _ = recover() }()
	fn()
}

// runHook executes hook across enabled plugins. Returns true when any plugin
// changed the body.
func runHook(hook string, ctx *pluginCtx) bool {
	// The interpreter and the context-backed mcpnet API are not safe for two
	// captured connections to execute concurrently. Keep each hook pipeline
	// bound to its own context for the full duration of plugin execution.
	hookMu.Lock()
	defer hookMu.Unlock()
	ctxMu.Lock()
	curCtx = ctx
	ctxMu.Unlock()
	changed := false
	for _, p := range plugins() {
		ctxMu.Lock()
		ctx.PluginID = p.Manifest.ID
		ctxMu.Unlock()
		if !p.Manifest.Enabled {
			continue
		}
		switch hook {
		case "OnHTTPRequest":
			if !p.hookHTTPReq {
				if err := p.compile(); err != nil {
					p.Err = err.Error()
					continue
				}
				if !p.hookHTTPReq {
					continue
				}
			}
		case "OnHTTPResponse":
			if !p.hookHTTPResp {
				if err := p.compile(); err != nil {
					p.Err = err.Error()
					continue
				}
				if !p.hookHTTPResp {
					continue
				}
			}
		case "OnWebSocket":
			if !p.hookWS {
				if err := p.compile(); err != nil {
					p.Err = err.Error()
					continue
				}
				if !p.hookWS {
					continue
				}
			}
		case "OnUDP":
			if !p.hookUDP {
				if err := p.compile(); err != nil {
					p.Err = err.Error()
					continue
				}
				if !p.hookUDP {
					continue
				}
			}
		case "OnURLDisplay":
			if !p.hookURLView {
				if err := p.compile(); err != nil {
					p.Err = err.Error()
					continue
				}
				if !p.hookURLView {
					continue
				}
			}
		}
		before := len(ctx.Body)
		p.call(hook)
		if ctx.Changed || len(ctx.Body) != before {
			changed = true
		}
		ctx.Changed = false
	}
	ctxMu.Lock()
	curCtx = nil
	ctxMu.Unlock()
	return changed
}

// ---- manifest CRUD ----

func writePluginManifest(m pluginManifest, source *string) error {
	if strings.TrimSpace(m.ID) == "" {
		m.ID = fmt.Sprintf("plugin-%d", time.Now().UnixNano())
	}
	if strings.TrimSpace(m.Name) == "" {
		m.Name = m.ID
	}
	dir := filepath.Join(pluginDataDir(), sanitizePluginID(m.ID))
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	bs, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(pluginManifestPath(dir), bs, 0600); err != nil {
		return err
	}
	if source != nil {
		if err := os.WriteFile(pluginSourcePath(dir), []byte(*source), 0600); err != nil {
			return err
		}
	}
	invalidatePluginCache()
	return nil
}

// Plugin settings are persisted while holding hookMu so once a disable call
// returns, no in-flight hook can continue using the previously enabled copy.
func savePluginManifest(m pluginManifest, source *string) error {
	hookMu.Lock()
	defer hookMu.Unlock()
	return writePluginManifest(m, source)
}

func setPluginEnabled(id string, enabled bool) error {
	hookMu.Lock()
	defer hookMu.Unlock()

	dir := filepath.Join(pluginDataDir(), sanitizePluginID(id))
	bs, err := os.ReadFile(pluginManifestPath(dir))
	if err != nil {
		return err
	}
	var manifest pluginManifest
	if err := json.Unmarshal(bs, &manifest); err != nil {
		return fmt.Errorf("解析插件清單失敗：%w", err)
	}
	if manifest.ID == "" {
		manifest.ID = id
	}
	manifest.Enabled = enabled
	return writePluginManifest(manifest, nil)
}

func deletePluginDir(id string) error {
	hookMu.Lock()
	defer hookMu.Unlock()
	dir := filepath.Join(pluginDataDir(), sanitizePluginID(id))
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	invalidatePluginCache()
	return nil
}

func sanitizePluginID(id string) string {
	out := strings.TrimSpace(id)
	for _, r := range []string{"\\", "/", ":", "*", "?", "\"", "<", ">", "|", ".."} {
		out = strings.ReplaceAll(out, r, "_")
	}
	if out == "" {
		out = "plugin"
	}
	return out
}

func pluginInfo() map[string]any {
	list := plugins()
	items := make([]map[string]any, 0, len(list))
	for _, p := range list {
		items = append(items, map[string]any{
			"id":          p.Manifest.ID,
			"name":        p.Manifest.Name,
			"version":     p.Manifest.Version,
			"author":      p.Manifest.Author,
			"description": p.Manifest.Description,
			"enabled":     p.Manifest.Enabled,
			"order":       p.Manifest.Order,
			"error":       p.Err,
		})
	}
	return map[string]any{
		"dir":     pluginDataDir(),
		"plugins": items,
	}
}

func runPluginUIAction(id, action, payload string) (string, error) {
	if len(action) > 128 || strings.TrimSpace(action) == "" {
		return "", fmt.Errorf("插件動作名稱無效")
	}
	if len(payload) > 16<<10 {
		return "", fmt.Errorf("插件介面資料不可超過 16 KB")
	}
	hookMu.Lock()
	defer hookMu.Unlock()
	for _, p := range plugins() {
		if p.Manifest.ID != id {
			continue
		}
		if !p.Manifest.Enabled {
			return "", fmt.Errorf("插件已停用")
		}
		if !p.hookUIAction {
			if err := p.compile(); err != nil {
				p.Err = err.Error()
				return "", fmt.Errorf("插件載入失敗：%w", err)
			}
		}
		if !p.hookUIAction {
			return "", fmt.Errorf("插件未實作 OnPluginAction")
		}
		ctx := &pluginCtx{PluginID: id, Action: action, Payload: payload}
		ctxMu.Lock()
		curCtx = ctx
		ctxMu.Unlock()
		p.call("OnPluginAction")
		ctxMu.Lock()
		result := ctx.UIResult
		curCtx = nil
		ctxMu.Unlock()
		return result, nil
	}
	return "", fmt.Errorf("找不到插件")
}

func pluginSourceOf(id string) string {
	for _, p := range plugins() {
		if p.Manifest.ID == id {
			return p.Source
		}
	}
	return ""
}

func pluginManifestOf(id string) pluginManifest {
	for _, p := range plugins() {
		if p.Manifest.ID == id {
			return p.Manifest
		}
	}
	return pluginManifest{}
}

// ---- API exported to plugin source (package mcpnet) ----

func pluginExports() map[string]map[string]reflect.Value {
	return map[string]map[string]reflect.Value{
		// Yaegi keys exported packages by import path/package name. A bare
		// "mcpnet" is treated as a missing package name, so source importing
		// "mcpnet" falls through to GOPATH lookup and fails in packaged builds.
		"mcpnet/mcpnet": {
			"Direction":       reflect.ValueOf(apiDirection),
			"Action":          reflect.ValueOf(apiAction),
			"Payload":         reflect.ValueOf(apiPayload),
			"SetUIResult":     reflect.ValueOf(apiSetUIResult),
			"OpenWindow":      reflect.ValueOf(apiOpenWindow),
			"OpenExternalURL": reflect.ValueOf(apiOpenExternalURL),
			"Host":            reflect.ValueOf(apiHost),
			"Path":            reflect.ValueOf(apiPath),
			"URL":             reflect.ValueOf(apiURL),
			"Method":          reflect.ValueOf(apiMethod),
			"Body":            reflect.ValueOf(apiBody),
			"SetBody":         reflect.ValueOf(apiSetBody),
			"SetWireBody":     reflect.ValueOf(apiSetWireBody),
			"DisplayBody":     reflect.ValueOf(apiDisplayBody),
			"SetDisplayBody":  reflect.ValueOf(apiSetDisplayBody),
			"SetURL":          reflect.ValueOf(apiSetURL),
			"PID":             reflect.ValueOf(apiPID),
			"Process":         reflect.ValueOf(apiProcess),
			"Log":             reflect.ValueOf(apiLog),
			"HexEncode":       reflect.ValueOf(apiHexEncode),
			"HexDecode":       reflect.ValueOf(apiHexDecode),
			"Base64Encode":    reflect.ValueOf(apiBase64Encode),
			"Base64Decode":    reflect.ValueOf(apiBase64Decode),
			"Xor":             reflect.ValueOf(apiXor),
			"XXTEAEncrypt":    reflect.ValueOf(apiXXTEAEncrypt),
			"XXTEADecrypt":    reflect.ValueOf(apiXXTEADecrypt),
			"XXTEAPack":       reflect.ValueOf(apiXXTEAPack),
			"XXTEAUnpack":     reflect.ValueOf(apiXXTEAUnpack),
		},
	}
}

func apiAction() string {
	ctxMu.Lock()
	defer ctxMu.Unlock()
	if curCtx == nil {
		return ""
	}
	return curCtx.Action
}

func apiPayload() string {
	ctxMu.Lock()
	defer ctxMu.Unlock()
	if curCtx == nil {
		return ""
	}
	return curCtx.Payload
}

func apiSetUIResult(value string) {
	ctxMu.Lock()
	defer ctxMu.Unlock()
	if curCtx == nil || len(value) > 16<<10 {
		return
	}
	curCtx.UIResult = value
}

// OpenWindow lets trusted Go plugins provide their own in-app tool window.
// It is only available while one of the plugin hooks is running.
func apiOpenWindow(title, html string) bool {
	ctxMu.Lock()
	if curCtx == nil || curCtx.PluginID == "" || strings.TrimSpace(html) == "" || len(title) > 128 || len(html) > 256<<10 {
		ctxMu.Unlock()
		return false
	}
	id := curCtx.PluginID
	ctxMu.Unlock()
	CallJs("插件開啟介面", map[string]string{"Id": id, "Title": title, "HTML": html})
	return true
}

// OpenExternalURL opens an HTTP(S) URL in the user's default browser.
func apiOpenExternalURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" || len(raw) > 4096 {
		return false
	}
	ctxMu.Lock()
	allowed := curCtx != nil && curCtx.PluginID != ""
	ctxMu.Unlock()
	if !allowed {
		return false
	}
	CallJs("插件開啟外部網址", map[string]string{"URL": parsed.String()})
	return true
}

func apiDirection() string {
	ctxMu.Lock()
	defer ctxMu.Unlock()
	if curCtx == nil {
		return ""
	}
	return curCtx.Direction
}
func apiHost() string {
	ctxMu.Lock()
	defer ctxMu.Unlock()
	if curCtx == nil {
		return ""
	}
	return curCtx.Host
}
func apiPath() string {
	ctxMu.Lock()
	defer ctxMu.Unlock()
	if curCtx == nil {
		return ""
	}
	return curCtx.Path
}
func apiURL() string {
	ctxMu.Lock()
	defer ctxMu.Unlock()
	if curCtx == nil {
		return ""
	}
	return curCtx.URL
}
func apiMethod() string {
	ctxMu.Lock()
	defer ctxMu.Unlock()
	if curCtx == nil {
		return ""
	}
	return curCtx.Method
}
func apiPID() int {
	ctxMu.Lock()
	defer ctxMu.Unlock()
	if curCtx == nil {
		return 0
	}
	return curCtx.PID
}
func apiProcess() string {
	ctxMu.Lock()
	defer ctxMu.Unlock()
	if curCtx == nil {
		return ""
	}
	return curCtx.Process
}

func apiBody() []byte {
	ctxMu.Lock()
	defer ctxMu.Unlock()
	if curCtx == nil {
		return nil
	}
	out := make([]byte, len(curCtx.Body))
	copy(out, curCtx.Body)
	return out
}

func apiSetBody(b []byte) {
	ctxMu.Lock()
	defer ctxMu.Unlock()
	if curCtx == nil {
		return
	}
	curCtx.Body = append([]byte(nil), b...)
	curCtx.Changed = true
}

// SetWireBody explicitly opts a plugin into changing bytes sent to the peer.
// This is separate from display decoding, which should normally use
// SetDisplayBody.
func apiSetWireBody(b []byte) {
	ctxMu.Lock()
	defer ctxMu.Unlock()
	if curCtx == nil {
		return
	}
	curCtx.WireBody = append([]byte{}, b...)
	curCtx.WireBodySet = true
	curCtx.Changed = true
}

func apiDisplayBody() []byte {
	ctxMu.Lock()
	defer ctxMu.Unlock()
	if curCtx == nil {
		return nil
	}
	return append([]byte(nil), curCtx.DisplayBody...)
}

// SetDisplayBody changes only the body shown in McpNet. It never alters the
// bytes sent to the peer; use SetBody in an outgoing hook to change wire data.
func apiSetDisplayBody(b []byte) {
	ctxMu.Lock()
	defer ctxMu.Unlock()
	if curCtx == nil {
		return
	}
	curCtx.DisplayBody = append([]byte{}, b...)
	curCtx.DisplayBodySet = true
	curCtx.Changed = true
}

func apiSetURL(raw string) {
	ctxMu.Lock()
	defer ctxMu.Unlock()
	if curCtx == nil {
		return
	}
	curCtx.URL = raw
	curCtx.Changed = true
}

func apiLog(s string) {}

func apiHexEncode(b []byte) string { return hex.EncodeToString(b) }
func apiHexDecode(s string) []byte {
	s = strings.ReplaceAll(strings.ReplaceAll(s, " ", ""), "\n", "")
	b, _ := hex.DecodeString(s)
	return b
}
func apiBase64Encode(b []byte) string { return base64.StdEncoding.EncodeToString(b) }
func apiBase64Decode(s string) []byte {
	b, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(s))
	return b
}
func apiXor(b []byte, key []byte) []byte {
	if len(key) == 0 {
		return b
	}
	out := make([]byte, len(b))
	for i := range b {
		out[i] = b[i] ^ key[i%len(key)]
	}
	return out
}
func apiXXTEAEncrypt(b []byte, key []byte) []byte { return xxteaEncrypt(b, key) }
func apiXXTEADecrypt(b []byte, key []byte) []byte { return xxteaDecrypt(b, key) }

func apiXXTEAPack(v any, key []byte) ([]byte, error)   { return xxteaPack(v, key) }
func apiXXTEAUnpack(b []byte, key []byte) (any, error) { return xxteaUnpack(b, key) }

// displayURL allows an installed local plugin to provide a display-only URL
// transform. The underlying request URL is never changed.
func displayURL(raw string) string {
	ctx := &pluginCtx{URL: raw}
	if runHook("OnURLDisplay", ctx) && ctx.URL != "" {
		return ctx.URL
	}
	return raw
}
