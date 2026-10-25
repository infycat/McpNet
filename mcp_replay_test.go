package main

import (
	"io"
	"net"
	stdhttp "net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"changeme/MapHash"
	"github.com/qtgolang/McpNet/McpNet"
	"github.com/qtgolang/McpNet/src/http"
)

func TestMCPReplayCaptureReturnsCompletedHTTPResult(t *testing.T) {
	type observedRequest struct {
		method string
		url    string
		body   string
	}
	received := make(chan observedRequest, 1)
	proxy := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		body, _ := io.ReadAll(r.Body)
		received <- observedRequest{method: r.Method, url: r.URL.String(), body: string(body)}
		w.WriteHeader(stdhttp.StatusAccepted)
	}))
	defer proxy.Close()

	_, portText, err := net.SplitHostPort(strings.TrimPrefix(proxy.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatal(err)
	}

	previous := HashMap
	HashMap = MapHash.NewHashMap()
	t.Cleanup(func() { HashMap = previous })
	HashMap.SetRequest(812, &MapHash.Request{
		Display: true,
		Way:     "HTTP",
		Method:  stdhttp.MethodPost,
		URL:     "http://capture-target.invalid/replay?fixture=secret",
		Header:  http.Header{"Content-Type": []string{"text/plain"}},
		Body:    []byte("mcp-replay-fixture"),
	})
	app := &App{App: McpNet.NewMcp().SetPort(port)}
	server := &mcpServer{app: app, confirm: func(string, string) bool { return true }}
	result, err := server.callTool("mcpnet_replay_capture", map[string]any{"capture_id": 812, "mode": 3})
	if err != nil {
		t.Fatalf("mcpnet_replay_capture: %v", err)
	}
	got := result.(map[string]any)
	if got["replayed"] != true || got["status_code"] != stdhttp.StatusAccepted {
		t.Fatalf("replay result = %#v", got)
	}
	select {
	case request := <-received:
		if request.method != stdhttp.MethodPost || request.url != "http://capture-target.invalid/replay?fixture=secret" || request.body != "mcp-replay-fixture" {
			t.Fatalf("replayed request = %#v", request)
		}
	default:
		t.Fatal("MCP reported replay success before the proxy received the request")
	}
}
