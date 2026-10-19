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
	if parsed.User.Username() != "REDACTED" {
		t.Fatalf("userinfo was not redacted: %q", got)
	}
	for key, values := range parsed.Query() {
		for _, value := range values {
			if value != "REDACTED" {
				t.Fatalf("query value leaked for %q: %q", key, value)
			}
		}
	}
	if parsed.Fragment != "REDACTED" {
		t.Fatalf("fragment was not redacted: %q", got)
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
	s := &mcpServer{}

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
	if row["url"] != "http://example.test/submit?token=REDACTED&uid=REDACTED" {
		t.Fatalf("capture listing did not redact URL values: %#v", row["url"])
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
	s := &mcpServer{}
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
