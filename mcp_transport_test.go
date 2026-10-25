package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func mcpTestPost(body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	request.Host = "127.0.0.1:2026"
	request.Header.Set("Content-Type", "application/json")
	(&mcpServer{}).handleMCP(recorder, request)
	return recorder
}

func TestMCPTransportRejectsNonLoopbackHostAndOrigin(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		origins []string
	}{
		{name: "remote host", host: "attacker.example:2026"},
		{name: "malformed host port", host: "localhost:attacker"},
		{name: "remote origin", host: "127.0.0.1:2026", origins: []string{"https://attacker.example"}},
		{name: "malformed origin port", host: "127.0.0.1:2026", origins: []string{"http://localhost:attacker"}},
		{name: "lookalike localhost", host: "127.0.0.1:2026", origins: []string{"http://localhost.attacker.example"}},
		{name: "null origin", host: "127.0.0.1:2026", origins: []string{"null"}},
		{name: "multiple origins", host: "127.0.0.1:2026", origins: []string{"http://localhost", "https://attacker.example"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping","params":{}}`))
			request.Host = test.host
			for _, origin := range test.origins {
				request.Header.Add("Origin", origin)
			}
			(&mcpServer{}).handleMCP(recorder, request)
			if recorder.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusForbidden, recorder.Body.String())
			}
			if recorder.Header().Get("Access-Control-Allow-Origin") != "" {
				t.Fatal("rejected origin received a CORS allow-origin header")
			}
		})
	}
}

func TestMCPTransportPreflightValidatesOriginAndAdvertisesVersionHeader(t *testing.T) {
	tests := []struct {
		name       string
		origin     string
		wantStatus int
	}{
		{name: "local origin", origin: "http://localhost:5173", wantStatus: http.StatusNoContent},
		{name: "remote origin", origin: "https://attacker.example", wantStatus: http.StatusForbidden},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodOptions, "/mcp", nil)
			request.Host = "127.0.0.1:2026"
			request.Header.Set("Origin", test.origin)
			request.Header.Set("Access-Control-Request-Method", "POST")
			request.Header.Set("Access-Control-Request-Headers", "content-type,mcp-protocol-version")
			(&mcpServer{}).handleMCP(recorder, request)
			if recorder.Code != test.wantStatus {
				t.Fatalf("preflight status = %d, want %d; body=%s", recorder.Code, test.wantStatus, recorder.Body.String())
			}
			if test.wantStatus == http.StatusForbidden {
				if recorder.Header().Get("Access-Control-Allow-Origin") != "" {
					t.Fatal("rejected preflight origin received a CORS allow-origin header")
				}
				return
			}
			if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != test.origin {
				t.Fatalf("preflight allow-origin = %q, want %q", got, test.origin)
			}
			if got := recorder.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(strings.ToLower(got), "mcp-protocol-version") {
				t.Fatalf("preflight allow-headers omits MCP-Protocol-Version: %q", got)
			}
		})
	}
}

func TestMCPTransportAllowsNativeAndLoopbackOriginClients(t *testing.T) {
	tests := []struct {
		name   string
		host   string
		origin string
	}{
		{name: "native client", host: "127.0.0.1:2026"},
		{name: "localhost browser", host: "localhost:2026", origin: "http://localhost:5173"},
		{name: "IPv6 loopback browser", host: "[::1]:2026", origin: "http://[::1]:5173"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping","params":{}}`))
			request.Host = test.host
			if test.origin != "" {
				request.Header.Set("Origin", test.origin)
			}
			(&mcpServer{}).handleMCP(recorder, request)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
			}
			if test.origin == "" {
				if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "" {
					t.Fatalf("native client unexpectedly received CORS origin %q", got)
				}
			} else if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != test.origin {
				t.Fatalf("allow-origin = %q, want exact loopback origin %q", got, test.origin)
			}
			if strings.Contains(recorder.Header().Get("Access-Control-Allow-Origin"), "*") {
				t.Fatal("wildcard CORS origin is not allowed")
			}
		})
	}
}

func TestMCPHealthEndpointRejectsRebindingHostAndOrigin(t *testing.T) {
	tests := []struct {
		name   string
		host   string
		origin string
		status int
	}{
		{name: "loopback health client", host: "127.0.0.1:2026", status: http.StatusOK},
		{name: "remote Host", host: "attacker.example:2026", status: http.StatusForbidden},
		{name: "remote Origin", host: "127.0.0.1:2026", origin: "https://attacker.example", status: http.StatusForbidden},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/health", nil)
			request.Host = test.host
			if test.origin != "" {
				request.Header.Set("Origin", test.origin)
			}
			mcpHealthHandler()(recorder, request)
			if recorder.Code != test.status {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, test.status, recorder.Body.String())
			}
		})
	}
}

func TestMCPProtocolVersionNegotiationAndValidation(t *testing.T) {
	for _, version := range []string{"2024-11-05", "2025-03-26", "2025-06-18", mcpCurrentProtocol, "unknown"} {
		t.Run(version, func(t *testing.T) {
			body, _ := json.Marshal(map[string]any{
				"jsonrpc": "2.0", "id": 1, "method": "initialize",
				"params": map[string]any{"protocolVersion": version, "capabilities": map[string]any{}, "clientInfo": map[string]any{"name": "test", "version": "1"}},
			})
			recorder := mcpTestPost(string(body))
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, body=%s", recorder.Code, recorder.Body.String())
			}
			var response struct {
				Result struct {
					ProtocolVersion string `json:"protocolVersion"`
				} `json:"result"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			want := version
			if !supportedMCPProtocol(version) {
				want = mcpCurrentProtocol
			}
			if response.Result.ProtocolVersion != want {
				t.Fatalf("negotiated version = %q, want %q", response.Result.ProtocolVersion, want)
			}
			if got := recorder.Header().Get("Mcp-Session-Id"); got != "" {
				t.Fatalf("stateless MCP endpoint must not issue a session id, got %q", got)
			}
		})
	}

	badVersion := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":2,"method":"ping","params":{}}`))
	request.Host = "127.0.0.1:2026"
	request.Header.Set("MCP-Protocol-Version", "future-unsupported")
	(&mcpServer{}).handleMCP(badVersion, request)
	if badVersion.Code != http.StatusBadRequest {
		t.Fatalf("unsupported protocol header status = %d, want %d", badVersion.Code, http.StatusBadRequest)
	}

	badJSONRPC := mcpTestPost(`{"jsonrpc":"1.0","id":2,"method":"ping","params":{}}`)
	if badJSONRPC.Code != http.StatusOK || !strings.Contains(badJSONRPC.Body.String(), `"code":-32600`) {
		t.Fatalf("invalid JSON-RPC version response = %d %s", badJSONRPC.Code, badJSONRPC.Body.String())
	}
}
