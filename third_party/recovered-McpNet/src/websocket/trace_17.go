//go:build !go1.8
// +build !go1.8

package websocket

import (
	"github.com/qtgolang/McpNet/src/crypto/tls"
	"github.com/qtgolang/McpNet/src/http/httptrace"
)

func doHandshakeWithTrace(trace *httptrace.ClientTrace, tlsConn *tls.Conn, cfg *tls.Config) error {
	return doHandshake(tlsConn, cfg)
}
