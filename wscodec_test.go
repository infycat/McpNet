package main

import (
	"bytes"
	"testing"
)

func installTestPlugin(t *testing.T, source string) {
	t.Helper()
	pluginMu.Lock()
	oldList, oldStamp := pluginList, pluginStamp
	pluginList = []*loadedPlugin{{
		Manifest: pluginManifest{ID: "codec-test", Name: "codec test", Enabled: true},
		Source:   source,
	}}
	pluginStamp = "codec-test"
	pluginMu.Unlock()
	t.Cleanup(func() {
		pluginMu.Lock()
		pluginList, pluginStamp = oldList, oldStamp
		pluginMu.Unlock()
	})
}

func TestWebSocketCodecDirectionsAndDisplayDecode(t *testing.T) {
	installTestPlugin(t, `package main
import "mcpnet"
func OnWebSocket() {
	if mcpnet.Direction() == "in" {
		mcpnet.SetDisplayBody([]byte("{\"decoded\":true}"))
	} else if mcpnet.Direction() == "out" {
		mcpnet.SetBody([]byte("wire-bytes"))
	}
}`)

	cipher := []byte("cipher")
	decoded, messageType, ok := wsDecodeMessage("wss://example.test/socket", 2, cipher)
	if !ok || messageType != 1 || string(decoded) != `{"decoded":true}` {
		t.Fatalf("outgoing frame not decoded for display: body=%q type=%d ok=%v", decoded, messageType, ok)
	}

	decoded, messageType, ok = wsDecodeMessage("wss://example.test/socket", 3, cipher)
	if !ok || messageType != 1 || string(decoded) != `{"decoded":true}` {
		t.Fatalf("incoming frame not decoded for display: body=%q type=%d ok=%v", decoded, messageType, ok)
	}

	encoded, messageType, ok := wsEncodeMessage("wss://example.test/socket", 2, []byte(`{"input":true}`))
	if !ok || messageType != 2 || string(encoded) != "wire-bytes" {
		t.Fatalf("outgoing JSON was not encoded: body=%q type=%d ok=%v", encoded, messageType, ok)
	}
}

func TestWebSocketCodecWithoutScriptStillProcessesPlugins(t *testing.T) {
	installTestPlugin(t, `package main
import "mcpnet"
func OnWebSocket() {
	if mcpnet.Direction() == "in" {
		mcpnet.SetBody([]byte("decoded-json"))
	} else if mcpnet.Direction() == "out" {
		mcpnet.SetBody([]byte("encoded-binary"))
	}
}`)

	wire, display, messageType, changed := wsProcessWithoutScript("wss://example.test/socket", 3, []byte("cipher"))
	if changed || string(wire) != "cipher" || string(display) != "decoded-json" || messageType != 0 {
		t.Fatalf("incoming decode without JS failed: wire=%q display=%q type=%d changed=%v", wire, display, messageType, changed)
	}

	wire, display, messageType, changed = wsProcessWithoutScript("wss://example.test/socket", 2, []byte(`{"input":true}`))
	if !changed || string(wire) != "encoded-binary" || string(display) != `{"input":true}` || messageType != 2 {
		t.Fatalf("outgoing encode without JS failed: wire=%q display=%q type=%d changed=%v", wire, display, messageType, changed)
	}

	wire, display, messageType, changed = wsProcessWithoutScript("wss://example.test/socket", 2, []byte("cipher"))
	if changed || string(wire) != "cipher" || string(display) != "decoded-json" || messageType != 0 {
		t.Fatalf("outgoing decode changed the wire frame: wire=%q display=%q type=%d changed=%v", wire, display, messageType, changed)
	}
}

func TestWebSocketDisplayDecodeDoesNotRewriteUnchangedIncomingFrame(t *testing.T) {
	wire := []byte{0xC8, 0xA2, 0x55, 0xCC}
	decoded := []byte(`{"message":"readable"}`)
	if got := wsKeepWireBodyForDisplayDecode(wire, decoded, decoded); !bytes.Equal(got, wire) {
		t.Fatalf("display-only decode rewrote the incoming wire frame: got=%x want=%x", got, wire)
	}
	modified := []byte(`{"message":"intentionally modified"}`)
	if got := wsKeepWireBodyForDisplayDecode(wire, decoded, modified); !bytes.Equal(got, modified) {
		t.Fatalf("explicit script modification was not preserved: got=%q want=%q", got, modified)
	}
}
