package main

import (
	"strings"
	"testing"

	"changeme/MapHash"
	"github.com/qtgolang/McpNet/src/http"
)

func TestMCPAccessSettingsRequireConfirmationByDefault(t *testing.T) {
	settings := normalizeMCPAccessSettings(mcpAccessSettings{})
	if settings.Mode != mcpPermissionConfirm {
		t.Fatalf("default mode = %q, want %q", settings.Mode, mcpPermissionConfirm)
	}
	for _, category := range []string{"private_read", "settings", "plugin_code", "network_write", "generic"} {
		if settings.permits(category) {
			t.Errorf("default policy unexpectedly permits %q", category)
		}
	}
}

func TestMCPAccessSettingsCustomCategoriesAreIndependent(t *testing.T) {
	settings := mcpAccessSettings{Mode: mcpPermissionCustom, AllowPrivateRead: true, AllowPluginCode: true}
	for category, want := range map[string]bool{
		"private_read":  true,
		"plugin_code":   true,
		"settings":      false,
		"network_write": false,
		"generic":       false,
	} {
		if got := settings.permits(category); got != want {
			t.Errorf("permits(%q) = %t, want %t", category, got, want)
		}
	}
}

func TestMCPAccessSettingsUnrestrictedPermitsAllCategories(t *testing.T) {
	settings := mcpAccessSettings{Mode: mcpPermissionUnrestricted}
	for _, category := range []string{"private_read", "settings", "plugin_code", "network_write", "generic", "future_category"} {
		if !settings.permits(category) {
			t.Errorf("unrestricted policy does not permit %q", category)
		}
	}
}

func TestMCPAccessSettingsUnknownModeFallsBackToConfirmation(t *testing.T) {
	settings := normalizeMCPAccessSettings(mcpAccessSettings{Mode: "unexpected", AllowPrivateRead: true})
	if settings.Mode != mcpPermissionConfirm || settings.permits("private_read") {
		t.Fatalf("unknown mode did not fail closed: %+v", settings)
	}
}

func TestMCPInjectedConfirmationControlsAuthorization(t *testing.T) {
	asked := false
	s := &mcpServer{confirm: func(title, message string) bool {
		asked = true
		return false
	}}
	if s.authorize("private_read", "test", "test") {
		t.Fatal("injected denial was overridden by persisted permissions")
	}
	if !asked {
		t.Fatal("expected injected confirmation callback")
	}
}

func TestMCPPluginSaveCannotEnableCodeImplicitly(t *testing.T) {
	authorized := false
	s := &mcpServer{confirm: func(title, message string) bool {
		authorized = true
		return true
	}}
	_, err := s.callTool("mcpnet_save_plugin", map[string]any{
		"manifest_json": `{"id":"test-plugin","name":"Test","enabled":true}`,
		"source":        "package main",
	})
	if err == nil || !strings.Contains(err.Error(), "must keep enabled=false") {
		t.Fatalf("enabled plugin save error = %v, want explicit enablement error", err)
	}
	if authorized {
		t.Fatal("saving a manifest that requests enabled=true should fail before authorization or writing")
	}
}

func TestMCPSensitiveAndMutatingToolsRequireConfirmation(t *testing.T) {
	old := HashMap
	HashMap = MapHash.NewHashMap()
	t.Cleanup(func() { HashMap = old })
	request := &MapHash.Request{Display: true, Way: "HTTP", Method: "GET", URL: "http://example.test/path?token=private"}
	request.Header = http.Header{"Authorization": []string{"secret"}}
	HashMap.SetRequest(5, request)
	HashMap.SetRequest(6, &MapHash.Request{Display: true, Way: "Websocket", SocketData: []*MapHash.UpdateSocketData{{Info: &MapHash.UpdateSocketList{Length: 1}, Body: []byte("x")}}})

	cases := []struct {
		name string
		args map[string]any
	}{
		{"mcpnet_get_capture", map[string]any{"capture_id": 5, "include_body": true}},
		{"mcpnet_get_messages", map[string]any{"capture_id": 6, "include_body": true}},
		{"mcpnet_list_processes", map[string]any{"name": "MiniGameApp.exe"}},
		{"mcpnet_process_control", map[string]any{"action": "load_driver"}},
		{"mcpnet_start_listener", map[string]any{}},
		{"mcpnet_stop_listener", map[string]any{}},
		{"mcpnet_set_system_proxy", map[string]any{"enable": false}},
		{"mcpnet_save_plugin", map[string]any{"manifest_json": `{"id":"x","name":"x","enabled":false}`, "source": "package main"}},
		{"mcpnet_delete_plugin", map[string]any{"plugin_id": "x"}},
		{"mcpnet_plugin_ui_action", map[string]any{"plugin_id": "x", "action": "run"}},
		{"mcpnet_import_plugin", map[string]any{"manifest_json": `{}`, "source": "package main"}},
		{"mcpnet_set_plugin_enabled", map[string]any{"plugin_id": "x", "enabled": true}},
		{"mcpnet_send_session_data", map[string]any{"capture_id": 6, "protocol": "websocket", "direction": "Server", "data_base64": "eA=="}},
		{"mcpnet_send_http_request", map[string]any{"url": "http://127.0.0.1:1/", "include_response": false}},
		{"mcpnet_send_socket_request", map[string]any{"protocol": "tcp", "address": "127.0.0.1:1", "data_base64": "eA==", "read_response": false}},
		{"mcpnet_send_websocket_request", map[string]any{"url": "ws://127.0.0.1:1/", "data_base64": "eA==", "read_response": false}},
		{"mcpnet_replay_capture", map[string]any{"capture_id": 5}},
		{"mcpnet_command", map[string]any{"command": "保存配置", "args": map[string]any{}}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			asked := false
			server := &mcpServer{confirm: func(string, string) bool { asked = true; return false }}
			if _, err := server.callTool(test.name, test.args); err == nil {
				t.Fatal("denied operation unexpectedly succeeded")
			}
			if !asked {
				t.Fatal("operation did not request user confirmation")
			}
		})
	}
}
