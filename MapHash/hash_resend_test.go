package MapHash

import (
	"io"
	"net"
	stdhttp "net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/qtgolang/McpNet/src/http"
)

func TestResendHTTPUsesConfiguredLocalProxy(t *testing.T) {
	type observedRequest struct {
		method         string
		url            string
		body           string
		internalClient string
	}
	received := make(chan observedRequest, 1)
	proxy := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		body, _ := io.ReadAll(r.Body)
		received <- observedRequest{method: r.Method, url: r.URL.String(), body: string(body), internalClient: r.Header.Get("McpNetHTTPClient")}
		w.WriteHeader(stdhttp.StatusOK)
		_, _ = io.WriteString(w, "ok")
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
	store := NewHashMap()
	store.SetRequest(731, &Request{
		Way:    "HTTP",
		URL:    "http://capture-target.invalid/replay?fixture=secret",
		Method: stdhttp.MethodPost,
		Header: http.Header{"Content-Type": []string{"text/plain"}},
		Body:   []byte("mcp-replay-probe"),
	})
	statusCode, err := store.ResendHTTPNow(731, 3, port)
	if err != nil {
		t.Fatalf("ResendHTTPNow: %v", err)
	}
	if statusCode != stdhttp.StatusOK {
		t.Fatalf("replay status = %d, want %d", statusCode, stdhttp.StatusOK)
	}

	select {
	case got := <-received:
		if got.method != stdhttp.MethodPost || got.url != "http://capture-target.invalid/replay?fixture=secret" || got.body != "mcp-replay-probe" || got.internalClient != "true" {
			t.Fatalf("proxied replay = %#v", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("resendHttp did not send a request through the configured proxy")
	}
}
