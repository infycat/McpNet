package main

import (
	"encoding/base64"
	"net/url"
	"testing"

	"changeme/MapHash"
	"github.com/qtgolang/McpNet/src/http"
)

func TestRedactCaptureURLHidesQueryAndUserInfo(t *testing.T) {
	got := redactCaptureURL("https://alice:secret@example.test/path?token=abc&lang=2&apiid=42#frag-secret")
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("redacted URL is invalid: %q: %v", got, err)
	}
	if parsed.User != nil || parsed.Path != "/[redacted]" || parsed.RawQuery != "" || parsed.Fragment != "" {
		t.Fatalf("private URL components were not removed: %q", got)
	}
}

func TestRedactCaptureURLOmitsUnparseableInput(t *testing.T) {
	if got := redactCaptureURL("http://[invalid"); got != "[invalid URL omitted]" {
		t.Fatalf("unexpected invalid URL representation: %q", got)
	}
}

func TestMCPCaptureToolsDefaultToMetadata(t *testing.T) {
	old := HashMap
	HashMap = MapHash.NewHashMap()
	t.Cleanup(func() { HashMap = old })
	r := &MapHash.Request{Display: true, Way: "HTTP", Method: "POST", URL: "http://example.test/submit?token=secret&uid=42", Body: []byte("secret"), SendNum: 6}
	r.Header = http.Header{"Authorization": []string{"secret-token"}}
	HashMap.SetRequest(11, r)
	s := &mcpServer{confirm: func(title, message string) bool { return true }}

	got, err := s.callTool("mcpnet_list_captures", map[string]any{"limit": 10})
	if err != nil {
		t.Fatal(err)
	}
	page := got.(map[string]any)
	if page["count"] != 1 {
		t.Fatalf("expected one capture, got %#v", page)
	}
	row := page["captures"].([]map[string]any)[0]
	if _, ok := row["request_body_base64"]; ok {
		t.Fatal("capture listing exposed a body")
	}
	if _, ok := row["request_headers"]; ok {
		t.Fatal("capture listing exposed headers")
	}
	redactedURL, err := url.Parse(row["url"].(string))
	if err != nil || redactedURL.Path != "/[redacted]" || redactedURL.RawQuery != "" || redactedURL.User != nil {
		t.Fatalf("capture listing did not redact private URL components: %#v", row["url"])
	}
	if _, ok := row["client_ip"]; ok {
		t.Fatal("capture listing exposed client IP without consent")
	}
	if _, ok := row["pid"]; ok {
		t.Fatal("capture listing exposed PID without consent")
	}

	got, err = s.callTool("mcpnet_get_capture", map[string]any{"capture_id": 11})
	if err != nil {
		t.Fatal(err)
	}
	details := got.(map[string]any)
	if _, ok := details["request_body_base64"]; ok {
		t.Fatal("capture details exposed a body without opt-in")
	}
	got, err = s.callTool("mcpnet_get_capture", map[string]any{"capture_id": 11, "include_body": true})
	if err != nil {
		t.Fatal(err)
	}
	details = got.(map[string]any)
	if details["request_body_base64"] != base64.StdEncoding.EncodeToString([]byte("secret")) {
		t.Fatalf("unexpected encoded body: %#v", details)
	}
	if details["request_headers"].(map[string][]string)["Authorization"][0] != "secret-token" {
		t.Fatal("opt-in headers missing")
	}
	if details["url_private"] != "http://example.test/submit?token=secret&uid=42" {
		t.Fatalf("consented response missing original URL: %#v", details["url_private"])
	}
}

func TestMCPPrivateCaptureRequiresUserConsent(t *testing.T) {
	old := HashMap
	HashMap = MapHash.NewHashMap()
	t.Cleanup(func() { HashMap = old })
	HashMap.SetRequest(12, &MapHash.Request{Display: true, Way: "HTTP", URL: "https://example.test/private", Body: []byte("secret")})
	asked := false
	s := &mcpServer{confirm: func(title, message string) bool {
		asked = true
		return false
	}}
	if _, err := s.callTool("mcpnet_get_capture", map[string]any{"capture_id": 12, "include_body": true}); err == nil {
		t.Fatal("expected denied private-data request")
	}
	if !asked {
		t.Fatal("expected direct user consent prompt")
	}
}

func TestMCPToolsExposePluginAndSendWorkflows(t *testing.T) {
	want := map[string]bool{
		"mcpnet_list_plugins":       false,
		"mcpnet_import_plugin":      false,
		"mcpnet_set_plugin_enabled": false,
		"mcpnet_send_session_data":  false,
		"mcpnet_replay_capture":     false,
		"mcpnet_command":            false,
		"mcpnet_list_commands":      false,
	}
	for _, tool := range mcpTools() {
		if _, ok := want[tool.Name]; ok {
			want[tool.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("MCP tool %q is missing", name)
		}
	}
}

func TestMCPMessagePaginationAndPayloadOptIn(t *testing.T) {
	old := HashMap
	HashMap = MapHash.NewHashMap()
	t.Cleanup(func() { HashMap = old })
	r := &MapHash.Request{Display: true, Way: "Websocket", SocketData: []*MapHash.UpdateSocketData{
		{Info: &MapHash.UpdateSocketList{Ico: "上行", WsType: "JSON", Length: 2}, Body: []byte("{}")},
		{Info: &MapHash.UpdateSocketList{Ico: "下行", WsType: "JSON", Length: 2}, Body: []byte("[]")},
	}}
	HashMap.SetRequest(4, r)
	s := &mcpServer{confirm: func(title, message string) bool { return true }}
	got, err := s.callTool("mcpnet_get_messages", map[string]any{"capture_id": 4, "offset": 0, "limit": 1})
	if err != nil {
		t.Fatal(err)
	}
	page := got.(map[string]any)
	if page["total"] != 2 || page["has_more"] != true {
		t.Fatalf("pagination metadata incorrect: %#v", page)
	}
	msg := page["messages"].([]map[string]any)[0]
	if _, ok := msg["payload_base64"]; ok {
		t.Fatal("message payload exposed without opt-in")
	}
	got, err = s.callTool("mcpnet_get_messages", map[string]any{"capture_id": 4, "offset": 1, "limit": 5, "include_body": true})
	if err != nil {
		t.Fatal(err)
	}
	page = got.(map[string]any)
	msg = page["messages"].([]map[string]any)[0]
	if msg["direction"] != "server_to_client" || msg["payload_base64"] != base64.StdEncoding.EncodeToString([]byte("[]")) {
		t.Fatalf("unexpected message: %#v", msg)
	}
}
