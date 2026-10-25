package main

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestMCPPluginToolsCanManageAndRunPluginWithoutUIBridge(t *testing.T) {
	t.Setenv("USERPROFILE", t.TempDir())
	invalidatePluginCache()
	t.Cleanup(invalidatePluginCache)

	confirmations := 0
	server := &mcpServer{confirm: func(string, string) bool {
		confirmations++
		return true
	}}
	source := "package main\nimport \"mcpnet\"\nfunc OnPluginAction() { mcpnet.SetUIResult(mcpnet.Payload()) }\n"
	manifest := `{"id":"mcp-test-plugin","name":"MCP Test Plugin","version":"1.0.0","author":"test","description":"MCP plugin test","enabled":false,"order":10}`

	saved, err := server.callTool("mcpnet_save_plugin", map[string]any{"manifest_json": manifest, "source": source})
	if err != nil {
		t.Fatalf("save plugin: %v", err)
	}
	if _, leaksLocalPath := saved.(map[string]any)["dir"]; leaksLocalPath {
		t.Fatal("plugin save response exposed the local plugin directory")
	}
	listed, err := server.callTool("mcpnet_list_plugins", map[string]any{})
	if err != nil {
		t.Fatalf("list plugins: %v", err)
	}
	listedJSON, _ := json.Marshal(listed)
	if !strings.Contains(string(listedJSON), `"enabled":false`) || !strings.Contains(string(listedJSON), "mcp-test-plugin") {
		t.Fatalf("saved plugin is missing or not disabled: %s", listedJSON)
	}
	if _, err := server.callTool("mcpnet_get_plugin", map[string]any{"plugin_id": "mcp-test-plugin", "include_source": true}); err != nil {
		t.Fatalf("get plugin source after confirmation: %v", err)
	}
	if _, err := server.callTool("mcpnet_plugin_ui_action", map[string]any{"plugin_id": "mcp-test-plugin", "action": "echo", "payload": "test"}); err == nil || !strings.Contains(err.Error(), "停用") {
		t.Fatalf("disabled plugin action error = %v, want disabled-plugin error", err)
	}
	if _, err := server.callTool("mcpnet_set_plugin_enabled", map[string]any{"plugin_id": "mcp-test-plugin", "enabled": true}); err != nil {
		t.Fatalf("enable plugin: %v", err)
	}
	result, err := server.callTool("mcpnet_plugin_ui_action", map[string]any{"plugin_id": "mcp-test-plugin", "action": "echo", "payload": "test"})
	if err != nil || result != "test" {
		t.Fatalf("plugin UI action = %#v, error=%v; want payload round trip", result, err)
	}
	if _, err := server.callTool("mcpnet_set_plugin_enabled", map[string]any{"plugin_id": "mcp-test-plugin", "enabled": false}); err != nil {
		t.Fatalf("disable plugin: %v", err)
	}
	if _, err := server.callTool("mcpnet_delete_plugin", map[string]any{"plugin_id": "mcp-test-plugin"}); err != nil {
		t.Fatalf("delete plugin: %v", err)
	}

	folderManifest := `{"id":"mcp-folder-plugin","name":"Folder Import","version":"1.0.0","enabled":true}`
	folderSource := "package main\nfunc OnWebSocket() {}\n"
	imported, err := server.callTool("mcpnet_import_plugin", map[string]any{"manifest_json": folderManifest, "source": folderSource})
	if err != nil {
		t.Fatalf("import manifest and source: %v", err)
	}
	if _, leaksLocalPath := imported.(map[string]any)["dir"]; leaksLocalPath {
		t.Fatal("plugin import response exposed the local plugin directory")
	}
	folderPlugin, err := server.callTool("mcpnet_get_plugin", map[string]any{"plugin_id": "mcp-folder-plugin"})
	if err != nil {
		t.Fatalf("get folder-imported plugin: %v", err)
	}
	folderJSON, _ := json.Marshal(folderPlugin)
	if !strings.Contains(string(folderJSON), `"enabled":false`) {
		t.Fatalf("imported folder plugin must remain disabled: %s", folderJSON)
	}

	zipManifest := `{"id":"mcp-zip-plugin","name":"ZIP Import","version":"1.0.0","enabled":true}`
	var zipBuffer bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuffer)
	for name, data := range map[string]string{"plugin.json": zipManifest, "main.go": folderSource} {
		file, err := zipWriter.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write([]byte(data)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := server.callTool("mcpnet_import_plugin", map[string]any{"zip_base64": base64.StdEncoding.EncodeToString(zipBuffer.Bytes())}); err != nil {
		t.Fatalf("import ZIP plugin: %v", err)
	}
	if _, err := server.callTool("mcpnet_delete_plugin", map[string]any{"plugin_id": "mcp-folder-plugin"}); err != nil {
		t.Fatalf("delete folder-imported plugin: %v", err)
	}
	if _, err := server.callTool("mcpnet_delete_plugin", map[string]any{"plugin_id": "mcp-zip-plugin"}); err != nil {
		t.Fatalf("delete ZIP-imported plugin: %v", err)
	}
	if confirmations < 5 {
		t.Fatalf("permission confirmations = %d, want confirms for code read/save/action/enable/delete/import", confirmations)
	}
}
