package main

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMCPDirectHTTPRequestRoundTripAndPermissions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("X-Test") != "local" {
			t.Errorf("request method/header = %s/%q", r.Method, r.Header.Get("X-Test"))
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "hello" {
			t.Errorf("request body = %q", body)
		}
		w.Header().Set("X-Reply", "ok")
		_, _ = io.WriteString(w, "world")
	}))
	defer server.Close()

	var categories []string
	s := &mcpServer{confirm: func(title, _ string) bool {
		categories = append(categories, title)
		return true
	}}
	result, err := s.callTool("mcpnet_send_http_request", map[string]any{
		"url": server.URL + "/echo?secret=redacted", "method": "POST",
		"headers":     map[string]any{"X-Test": "local"},
		"body_base64": base64.StdEncoding.EncodeToString([]byte("hello")),
	})
	if err != nil {
		t.Fatal(err)
	}
	got := result.(map[string]any)
	if got["status_code"] != 200 || got["response_body_base64"] != base64.StdEncoding.EncodeToString([]byte("world")) {
		t.Fatalf("unexpected HTTP response: %#v", got)
	}
	if strings.Contains(got["url"].(string), "secret") {
		t.Fatalf("response leaked URL query: %#v", got["url"])
	}
	if len(categories) != 2 {
		t.Fatalf("expected private-read and network-write confirmation, got %v", categories)
	}

	asked := 0
	denied := &mcpServer{confirm: func(string, string) bool { asked++; return false }}
	if _, err := denied.sendHTTPMCPRequest(map[string]any{"url": server.URL, "include_response": false}); err == nil {
		t.Fatal("denied HTTP request unexpectedly succeeded")
	}
	if asked != 1 {
		t.Fatalf("request denial asked %d times, want one network-write confirmation", asked)
	}
}

func TestMCPDirectSocketRequestsRoundTripTCPAndUDP(t *testing.T) {
	t.Run("tcp", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer listener.Close()
		go func() {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			defer conn.Close()
			buf := make([]byte, 5)
			if _, err := io.ReadFull(conn, buf); err == nil {
				_, _ = conn.Write(append([]byte("echo:"), buf...))
			}
		}()
		result, err := (&mcpServer{confirm: func(string, string) bool { return true }}).sendSocketMCPRequest(map[string]any{
			"protocol": "tcp", "address": listener.Addr().String(), "data_base64": base64.StdEncoding.EncodeToString([]byte("hello")), "timeout_ms": 1000,
		})
		if err != nil {
			t.Fatal(err)
		}
		response, err := base64.StdEncoding.DecodeString(result["response_base64"].(string))
		if err != nil || string(response) != "echo:hello" {
			t.Fatalf("TCP response = %q, err=%v", response, err)
		}
	})

	t.Run("udp", func(t *testing.T) {
		server, err := net.ListenPacket("udp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer server.Close()
		go func() {
			buf := make([]byte, 128)
			n, addr, err := server.ReadFrom(buf)
			if err == nil {
				_, _ = server.WriteTo(append([]byte("echo:"), buf[:n]...), addr)
			}
		}()
		result, err := (&mcpServer{confirm: func(string, string) bool { return true }}).sendSocketMCPRequest(map[string]any{
			"protocol": "udp", "address": server.LocalAddr().String(), "data_base64": base64.StdEncoding.EncodeToString([]byte("hello")), "timeout_ms": 1000,
		})
		if err != nil {
			t.Fatal(err)
		}
		response, err := base64.StdEncoding.DecodeString(result["response_base64"].(string))
		if err != nil || string(response) != "echo:hello" {
			t.Fatalf("UDP response = %q, err=%v", response, err)
		}
	})
}

func TestMCPDirectSocketRequestDenialPreventsDial(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	asked := 0
	s := &mcpServer{confirm: func(string, string) bool { asked++; return false }}
	_, err = s.sendSocketMCPRequest(map[string]any{
		"protocol": "tcp", "address": listener.Addr().String(), "data_base64": base64.StdEncoding.EncodeToString([]byte("x")),
	})
	if err == nil || asked != 1 {
		t.Fatalf("denied socket request err=%v confirmations=%d", err, asked)
	}
}

func TestMCPDirectWebSocketRequestRoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/socket" || r.Header.Get("X-Test") != "ws" {
			t.Errorf("WebSocket path/header = %q/%q", r.URL.Path, r.Header.Get("X-Test"))
		}
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			t.Error("HTTP server does not support hijacking")
			return
		}
		connection, rw, err := hijacker.Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		defer connection.Close()
		acceptHash := sha1.Sum([]byte(r.Header.Get("Sec-WebSocket-Key") + websocketGUID))
		fmt.Fprintf(rw, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n", base64.StdEncoding.EncodeToString(acceptHash[:]))
		if err := rw.Flush(); err != nil {
			t.Error(err)
			return
		}
		reader := rw.Reader
		var frameHeader [2]byte
		if _, err := io.ReadFull(reader, frameHeader[:]); err != nil {
			t.Error(err)
			return
		}
		length := int(frameHeader[1] & 0x7f)
		if length >= 126 || frameHeader[0]&0x0f != 1 || frameHeader[1]&0x80 == 0 {
			t.Errorf("unexpected client frame header: %x", frameHeader)
			return
		}
		var mask [4]byte
		if _, err := io.ReadFull(reader, mask[:]); err != nil {
			t.Error(err)
			return
		}
		message := make([]byte, length)
		if _, err := io.ReadFull(reader, message); err != nil {
			t.Error(err)
			return
		}
		for i := range message {
			message[i] ^= mask[i&3]
		}
		if string(message) != "hello" {
			t.Errorf("unmasked client message = %q", message)
		}
		_, _ = connection.Write([]byte{0x01, 2}) // fragmented text message, first frame
		_, _ = io.WriteString(connection, "wo")
		_, _ = connection.Write([]byte{0x89, 1, '!'}) // ping between fragments
		_, _ = connection.Write([]byte{0x80, 3})      // final continuation frame
		_, _ = io.WriteString(connection, "rld")
		if _, err := io.ReadFull(reader, frameHeader[:]); err != nil {
			t.Error(err)
			return
		}
		if frameHeader[0]&0x0f != 10 || frameHeader[1]&0x80 == 0 || frameHeader[1]&0x7f != 1 {
			t.Errorf("unexpected pong frame header: %x", frameHeader)
			return
		}
		if _, err := io.ReadFull(reader, mask[:]); err != nil {
			t.Error(err)
			return
		}
		var pongPayload [1]byte
		if _, err := io.ReadFull(reader, pongPayload[:]); err != nil {
			t.Error(err)
			return
		}
		if pongPayload[0]^mask[0] != '!' {
			t.Errorf("pong payload was not echoed")
		}
	}))
	defer server.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/socket?token=private"
	calls := 0
	s := &mcpServer{confirm: func(string, string) bool { calls++; return true }}
	requestBody, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "tools/call",
		"params": map[string]any{"name": "mcpnet_send_websocket_request", "arguments": map[string]any{
			"url": wsURL, "data_base64": base64.StdEncoding.EncodeToString([]byte("hello")),
			"headers": map[string]any{"X-Test": "ws"}, "timeout_ms": 1000,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(string(requestBody)))
	request.Host = "127.0.0.1:2026"
	request.Header.Set("Content-Type", "application/json")
	s.handleMCP(recorder, request)
	var rpcResponse struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &rpcResponse); err != nil || recorder.Code != http.StatusOK {
		t.Fatalf("WebSocket MCP JSON-RPC status=%d body=%s err=%v", recorder.Code, recorder.Body.String(), err)
	}
	if rpcResponse.Result.IsError || len(rpcResponse.Result.Content) == 0 {
		t.Fatalf("WebSocket MCP tool call failed: %#v", rpcResponse.Result)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(rpcResponse.Result.Content[0].Text), &got); err != nil {
		t.Fatalf("invalid WebSocket MCP tool result %q: %v", rpcResponse.Result.Content[0].Text, err)
	}
	response, err := base64.StdEncoding.DecodeString(got["response_base64"].(string))
	if err != nil || string(response) != "world" || got["upgraded"] != true || got["response_type"] != "text" {
		t.Fatalf("WebSocket result=%#v response=%q err=%v", got, response, err)
	}
	if strings.Contains(got["url"].(string), "private") {
		t.Fatalf("WebSocket response leaked URL query: %q", got["url"])
	}
	if calls != 2 {
		t.Fatalf("expected separate privacy and network confirmations, got %d", calls)
	}
}

func TestMCPDirectWebSocketRejectsHandshakeHeaderOverrides(t *testing.T) {
	s := &mcpServer{confirm: func(string, string) bool { t.Fatal("invalid request prompted for authorization"); return false }}
	_, err := s.sendWebSocketMCPRequest(map[string]any{
		"url": "ws://127.0.0.1:1/", "data_base64": base64.StdEncoding.EncodeToString([]byte("x")),
		"headers": map[string]any{"Sec-WebSocket-Key": "attacker-controlled"},
	})
	if err == nil || !strings.Contains(err.Error(), "managed by the WebSocket handshake") {
		t.Fatalf("reserved handshake header error = %v", err)
	}
}
