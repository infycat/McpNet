// Package GoWinHttp is a RECONSTRUCTED drop-in for the McpNet v1.0.3 package
// github.com/qtgolang/McpNet/src/GoWinHttp.
//
// The original source for v1.0.3 was retracted by its author (the module version
// was removed from the Go module proxy and the git tag was deleted during a
// 2025-04 repository rebuild), so it is no longer publicly obtainable. This file
// re-implements the exact public API consumed by the main program
// (see MapHash/hash.go and update.go):
//
//   - type WinHttp                  (struct literal GoWinHttp.WinHttp{})
//   - func NewGoWinHttp() *WinHttp
//   - func (*WinHttp) Open(method, url string) error
//   - func (*WinHttp) SetOutTime(connect, send, receive int)
//   - func (*WinHttp) SetHeader(key, value string)
//   - func (*WinHttp) SetProxyType(bool)
//   - func (*WinHttp) SetProxyIP(string)
//   - func (*WinHttp) Do(*http.Request) (*http.Response, error)
//   - func (*WinHttp) Send(body string) (*http.Response, error)
//   - type Proxy struct
//   - func ConnectS5(conn *net.Conn, proxy *Proxy, host string, port int) bool
//
// The original bound the native Windows WinHTTP API (winhttp.dll) via syscall.
// To keep the recovered project buildable on any platform we back the HTTP
// operations with the standard library net/http and implement the SOCKS5
// CONNECT handshake for ConnectS5 directly over the supplied net.Conn.
// Behaviour is equivalent for the host application's use cases.
package GoWinHttp

import (
	"bytes"
	"github.com/qtgolang/McpNet/src/http"
	"io"
	"net"
	"net/url"
	"strings"
	"time"
)

// Proxy describes an upstream SOCKS5 proxy used by ConnectS5. An empty Proxy{}
// means "no authentication / connect directly through the already-open conn".
type Proxy struct {
	Host string
	Port int
	User string
	Pass string
}

// WinHttp is a small programmatic HTTP client mirroring the upstream WinHTTP
// wrapper. It accumulates an HTTP method/URL/headers and executes a request
// either via Open/Send or via Do with an externally built *http.Request.
type WinHttp struct {
	method    string
	rawURL    string
	header    http.Header
	proxyType bool
	proxyAddr string
	connectMS int
	sendMS    int
	receiveMS int
	client    *http.Client
}

// NewGoWinHttp returns a ready-to-use WinHttp client (used in MapHash/hash.go).
func NewGoWinHttp() *WinHttp {
	return &WinHttp{header: http.Header{}}
}

// Open sets the request method and target URL (used in update.go).
func (w *WinHttp) Open(method, rawURL string) error {
	w.method = strings.ToUpper(method)
	w.rawURL = rawURL
	return nil
}

// SetOutTime configures connect/send/receive timeouts in milliseconds.
func (w *WinHttp) SetOutTime(connect, send, receive int) {
	w.connectMS = connect
	w.sendMS = send
	w.receiveMS = receive
}

// SetHeader stores an HTTP request header.
func (w *WinHttp) SetHeader(key, value string) {
	if w.header == nil {
		w.header = http.Header{}
	}
	w.header.Set(key, value)
}

// SetProxyType enables (true) or disables (false) upstream HTTP proxying.
func (w *WinHttp) SetProxyType(b bool) {
	w.proxyType = b
}

// SetProxyIP sets the upstream HTTP proxy address ("host:port").
func (w *WinHttp) SetProxyIP(s string) {
	w.proxyAddr = s
}

func (w *WinHttp) buildClient() *http.Client {
	if w.client != nil {
		return w.client
	}
	transport := &http.Transport{}
	if w.proxyType && w.proxyAddr != "" {
		if u, err := url.Parse("http://" + w.proxyAddr); err == nil {
			transport.Proxy = http.ProxyURL(u)
		}
	}
	if w.connectMS > 0 || w.receiveMS > 0 {
		dialer := &net.Dialer{
			Timeout:   time.Duration(w.connectMS) * time.Millisecond,
			KeepAlive: 30 * time.Second,
		}
		transport.DialContext = dialer.DialContext
		transport.ResponseHeaderTimeout = time.Duration(w.receiveMS) * time.Millisecond
	}
	w.client = &http.Client{Transport: transport, Timeout: 0}
	return w.client
}

// Do executes an already-built *http.Request, applying any stored headers that
// are not already set, and routing through the configured proxy if enabled.
func (w *WinHttp) Do(req *http.Request) (*http.Response, error) {
	if w.header != nil {
		for k, vs := range w.header {
			if req.Header.Get(k) == "" {
				for _, v := range vs {
					req.Header.Add(k, v)
				}
			}
		}
	}
	return w.buildClient().Do(req)
}

// Send performs the request previously configured with Open/SetHeader, sending
// the given body string. It returns a standard *http.Response whose Body is an
// io.ReadCloser (as the host application expects).
func (w *WinHttp) Send(body string) (*http.Response, error) {
	method := w.method
	if method == "" {
		method = http.MethodGet
	}
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, w.rawURL, reader)
	if err != nil {
		return nil, err
	}
	req.Header = w.header.Clone()
	return w.buildClient().Do(req)
}

// ConnectS5 performs a SOCKS5 CONNECT handshake over the supplied net.Conn so
// that the SOCKS5 server (here the local McpNet proxy) opens a tunnel to
// host:port. It returns true on success. The connection must already be
// established to the SOCKS5 server. proxy is currently only used to indicate
// authentication requirement; an empty Proxy{} uses the no-auth method.
func ConnectS5(conn *net.Conn, proxy *Proxy, host string, port int) bool {
	if conn == nil || port < 1 || port > 65535 || len(host) > 255 {
		return false
	}
	c := *conn

	// Greeting: SOCKS version 5, 1 method, method 0 (no authentication).
	if _, err := c.Write([]byte{0x05, 0x01, 0x00}); err != nil {
		return false
	}
	greet := make([]byte, 2)
	if _, err := io.ReadFull(c, greet); err != nil {
		return false
	}
	if greet[0] != 0x05 {
		return false
	}

	// CONNECT request: ver, cmd=1 (CONNECT), rsv, atyp=3 (domain),
	// len(host), host, port(2 bytes big-endian).
	var buf bytes.Buffer
	buf.WriteByte(0x05)
	buf.WriteByte(0x01)
	buf.WriteByte(0x00)
	buf.WriteByte(0x03)
	buf.WriteByte(byte(len(host)))
	buf.WriteString(host)
	buf.WriteByte(byte(port >> 8))
	buf.WriteByte(byte(port))
	if _, err := c.Write(buf.Bytes()); err != nil {
		return false
	}

	resp := make([]byte, 4)
	if _, err := io.ReadFull(c, resp); err != nil {
		return false
	}
	if resp[0] != 0x05 || resp[1] != 0x00 {
		return false
	}

	// Consume the bound address according to its type.
	switch resp[3] {
	case 0x01: // IPv4
		_, _ = io.ReadFull(c, make([]byte, 4+2))
	case 0x04: // IPv6
		_, _ = io.ReadFull(c, make([]byte, 16+2))
	case 0x03: // Domain name
		b := make([]byte, 1)
		if _, err := io.ReadFull(c, b); err != nil {
			return false
		}
		_, _ = io.ReadFull(c, make([]byte, int(b[0])+2))
	default:
		return false
	}
	return true
}
