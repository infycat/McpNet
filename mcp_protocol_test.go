package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qtgolang/McpNet/public"
)

func TestEveryAdvertisedMCPToolHasAReachableHandler(t *testing.T) {
	t.Setenv("USERPROFILE", t.TempDir())
	invalidatePluginCache()
	t.Cleanup(invalidatePluginCache)
	server := &mcpServer{app: &App{}}
	seen := make(map[string]bool)
	for _, tool := range mcpTools() {
		if tool.Name == "" || tool.Description == "" || seen[tool.Name] {
			t.Errorf("invalid or duplicate advertised MCP tool: %#v", tool)
		}
		seen[tool.Name] = true
		if tool.InputSchema["type"] != "object" {
			t.Errorf("tool %q input schema type = %v, want object", tool.Name, tool.InputSchema["type"])
		}
		properties, ok := tool.InputSchema["properties"].(map[string]any)
		if !ok {
			t.Errorf("tool %q has no properties object", tool.Name)
		}
		if required, ok := tool.InputSchema["required"].([]string); ok {
			for _, key := range required {
				if _, exists := properties[key]; !exists {
					t.Errorf("tool %q requires undocumented property %q", tool.Name, key)
				}
			}
		}
		_, err := server.callTool(tool.Name, map[string]any{})
		if err != nil && strings.Contains(err.Error(), "unknown tool") {
			t.Errorf("advertised tool %q has no handler: %v", tool.Name, err)
		}
	}
}

func TestMCPUIFileDialogCommandsUseExplicitPathsWithoutNativeDialogs(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "input.dat")
	if err := os.WriteFile(input, []byte("private sample"), 0600); err != nil {
		t.Fatal(err)
	}
	confirmations := 0
	server := &mcpServer{app: &App{}, confirm: func(string, string) bool {
		confirmations++
		return true
	}}
	call := func(command, path string) (map[string]any, error) {
		body, err := json.Marshal(map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "tools/call",
			"params": map[string]any{"name": "mcpnet_command", "arguments": map[string]any{
				"command": command,
				"args":    map[string]any{"Path": path},
			}},
		})
		if err != nil {
			t.Fatal(err)
		}
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(string(body)))
		request.Host = "127.0.0.1:2026"
		request.Header.Set("Content-Type", "application/json")
		server.handleMCP(recorder, request)
		if recorder.Code != http.StatusOK {
			return nil, fmt.Errorf("MCP HTTP status %d: %s", recorder.Code, recorder.Body.String())
		}
		var response struct {
			Result struct {
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
				IsError bool `json:"isError"`
			} `json:"result"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			return nil, err
		}
		if response.Result.IsError {
			return nil, fmt.Errorf("MCP command %q failed: %s", command, response.Result.Content[0].Text)
		}
		var result map[string]any
		if err := json.Unmarshal([]byte(response.Result.Content[0].Text), &result); err != nil {
			return nil, err
		}
		return result, nil
	}

	selected, err := call("选择文件", input)
	if err != nil {
		t.Fatalf("select explicit file path: %v", err)
	}
	if selected["result"] != input || selected["native_dialog_skipped"] != true {
		t.Fatalf("file selection result = %#v", selected)
	}
	output := filepath.Join(root, "output.syn")
	saved, err := call("保存文件对话框", output)
	if err != nil {
		t.Fatalf("select explicit output path: %v", err)
	}
	if saved["result"] != output || saved["native_dialog_skipped"] != true {
		t.Fatalf("save path result = %#v", saved)
	}
	if _, err := call("选择文件", ""); err == nil || !strings.Contains(err.Error(), "Args.Path") {
		t.Fatalf("missing explicit path error = %v", err)
	}
	if _, err := call("选择文件", filepath.Join(root, "missing.dat")); err == nil {
		t.Fatal("nonexistent input path should fail")
	}
	if _, err := call("选择文件", root); err == nil || !strings.Contains(err.Error(), "directory") {
		t.Fatalf("input directory error = %v", err)
	}
	if _, err := call("保存文件对话框", root); err == nil || !strings.Contains(err.Error(), "directory") {
		t.Fatalf("output directory error = %v", err)
	}
	certificateBase := filepath.Join(root, "default-ca")
	certificate, err := call("导出默认证书", certificateBase)
	if err != nil {
		t.Fatalf("export default certificate: %v", err)
	}
	certificatePath := certificateBase + ".cer"
	if certificate["result"] != certificatePath || certificate["native_dialog_skipped"] != true {
		t.Fatalf("certificate export result = %#v", certificate)
	}
	certificateBytes, err := os.ReadFile(certificatePath)
	if err != nil || string(certificateBytes) != public.RootCa {
		t.Fatalf("exported certificate contents mismatch, read error=%v", err)
	}
	if _, err := call("导出默认证书", certificateBase); err == nil {
		t.Fatal("certificate export must refuse to overwrite an existing file")
	}
	if _, err := call("导出默认证书", filepath.Join(root, "missing", "certificate.cer")); err == nil {
		t.Fatal("certificate export must reject a missing output directory")
	}
	if confirmations != 9 {
		t.Fatalf("file path operations confirmed %d times, want 9", confirmations)
	}
	pathOverrides := map[string]bool{}
	for _, command := range mcpUICommandCatalog() {
		pathOverrides[command.Name] = command.PathOverride
	}
	for _, name := range []string{"选择文件", "保存文件对话框", "导出默认证书"} {
		if !pathOverrides[name] {
			t.Errorf("%q does not advertise its MCP path override", name)
		}
	}
}

func TestMCPHTTPProtocolAdvertisesAndCallsCommandDiscovery(t *testing.T) {
	server := &mcpServer{}
	call := func(body string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
		request.Host = "127.0.0.1:2026"
		request.Header.Set("Content-Type", "application/json")
		server.handleMCP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("MCP HTTP status = %d, body=%s", recorder.Code, recorder.Body.String())
		}
		return recorder
	}

	var listed struct {
		Result struct {
			Tools []mcpTool `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(call(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`).Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	toolNames := make(map[string]bool, len(listed.Result.Tools))
	toolDescriptions := make(map[string]string, len(listed.Result.Tools))
	for _, tool := range listed.Result.Tools {
		toolNames[tool.Name] = true
		toolDescriptions[tool.Name] = tool.Description
	}
	for _, name := range []string{"mcpnet_command", "mcpnet_list_commands", "mcpnet_send_session_data", "mcpnet_send_http_request", "mcpnet_send_socket_request", "mcpnet_send_websocket_request", "mcpnet_get_capture", "mcpnet_get_plugin", "mcpnet_save_plugin", "mcpnet_delete_plugin", "mcpnet_plugin_ui_action"} {
		if !toolNames[name] {
			t.Errorf("tools/list omitted %q", name)
		}
	}
	if !strings.Contains(toolDescriptions["mcpnet_command"], "MCP path overrides") {
		t.Error("mcpnet_command description omits MCP path overrides")
	}

	var called struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
	}
	if err := json.Unmarshal(call(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"mcpnet_list_commands","arguments":{}}}`).Body.Bytes(), &called); err != nil {
		t.Fatal(err)
	}
	if called.Result.IsError || len(called.Result.Content) == 0 {
		t.Fatalf("command discovery failed: %#v", called.Result)
	}
	var catalog struct {
		Commands []mcpUICommand `json:"commands"`
	}
	if err := json.Unmarshal([]byte(called.Result.Content[0].Text), &catalog); err != nil {
		t.Fatal(err)
	}
	if len(catalog.Commands) < 70 {
		t.Fatalf("command discovery returned only %d commands", len(catalog.Commands))
	}
	pathOverrides := make(map[string]mcpUICommand)
	for _, command := range catalog.Commands {
		if command.PermissionCategory == "" || command.Authorization == "" {
			t.Errorf("command discovery omitted permission metadata for %q: %#v", command.Name, command)
		}
		if command.PathOverride {
			pathOverrides[command.Name] = command
		}
	}
	for _, name := range []string{"选择文件", "保存文件对话框", "导出默认证书"} {
		command, ok := pathOverrides[name]
		if !ok {
			t.Errorf("JSON-RPC command directory omitted the path override for %q", name)
			continue
		}
		wantCategory := "private_read"
		if name == "导出默认证书" {
			wantCategory = "settings"
		}
		if command.PermissionCategory != wantCategory {
			t.Errorf("path override %q permission category = %q, want %s", name, command.PermissionCategory, wantCategory)
		}
	}
}

func TestMCPHTTPPluginWorkflowRoundTripsOverJSONRPC(t *testing.T) {
	t.Setenv("USERPROFILE", t.TempDir())
	invalidatePluginCache()
	t.Cleanup(invalidatePluginCache)
	server := &mcpServer{confirm: func(string, string) bool { return true }}
	call := func(body string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
		request.Host = "127.0.0.1:2026"
		request.Header.Set("Content-Type", "application/json")
		server.handleMCP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("MCP HTTP status = %d, body=%s", recorder.Code, recorder.Body.String())
		}
		return recorder
	}
	toolCall := func(id int, name string, args map[string]any) (string, error) {
		body, err := json.Marshal(map[string]any{
			"jsonrpc": "2.0", "id": id, "method": "tools/call",
			"params": map[string]any{"name": name, "arguments": args},
		})
		if err != nil {
			t.Fatal(err)
		}
		var response struct {
			Result struct {
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
				IsError bool `json:"isError"`
			} `json:"result"`
		}
		if err := json.Unmarshal(call(string(body)).Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if len(response.Result.Content) == 0 {
			t.Fatalf("tool %q returned no text content", name)
		}
		if response.Result.IsError {
			return "", fmt.Errorf("%s", response.Result.Content[0].Text)
		}
		return response.Result.Content[0].Text, nil
	}

	manifest := `{"id":"mcp-http-test","name":"MCP HTTP Test","version":"1.0.0","enabled":false}`
	source := "package main\nimport \"mcpnet\"\nfunc OnPluginAction() { mcpnet.SetUIResult(mcpnet.Payload()) }\n"
	saved, err := toolCall(1, "mcpnet_save_plugin", map[string]any{"manifest_json": manifest, "source": source})
	if err != nil {
		t.Fatalf("save plugin over JSON-RPC: %v", err)
	}
	if strings.Contains(saved, "USERPROFILE") || strings.Contains(saved, `"dir"`) {
		t.Fatalf("plugin save response leaked a local path: %s", saved)
	}
	if _, err := toolCall(2, "mcpnet_get_plugin", map[string]any{"plugin_id": "mcp-http-test", "include_source": true}); err != nil {
		t.Fatalf("read plugin/source over JSON-RPC: %v", err)
	}
	if _, err := toolCall(3, "mcpnet_set_plugin_enabled", map[string]any{"plugin_id": "mcp-http-test", "enabled": true}); err != nil {
		t.Fatalf("enable plugin over JSON-RPC: %v", err)
	}
	result, err := toolCall(4, "mcpnet_plugin_ui_action", map[string]any{"plugin_id": "mcp-http-test", "action": "echo", "payload": "round-trip"})
	if err != nil || result != "round-trip" {
		t.Fatalf("plugin UI action result=%q error=%v", result, err)
	}
	if _, err := toolCall(5, "mcpnet_set_plugin_enabled", map[string]any{"plugin_id": "mcp-http-test", "enabled": false}); err != nil {
		t.Fatalf("disable plugin over JSON-RPC: %v", err)
	}
	if _, err := toolCall(6, "mcpnet_delete_plugin", map[string]any{"plugin_id": "mcp-http-test"}); err != nil {
		t.Fatalf("delete plugin over JSON-RPC: %v", err)
	}
}
