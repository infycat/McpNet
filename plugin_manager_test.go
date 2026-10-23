package main

import (
	"strings"
	"testing"
)

func TestParsePluginManifestAcceptsJSONRepresentations(t *testing.T) {
	const plain = `{"id":"codec","name":"Codec","enabled":true}`
	for name, value := range map[string]any{
		"json string":         plain,
		"escaped json string": strings.ReplaceAll(plain, `"`, `\"`),
		"object":              map[string]any{"id": "codec", "name": "Codec", "enabled": true},
	} {
		t.Run(name, func(t *testing.T) {
			manifest, err := parsePluginManifest(value)
			if err != nil {
				t.Fatalf("parse manifest: %v", err)
			}
			if manifest.ID != "codec" || manifest.Name != "Codec" || !manifest.Enabled {
				t.Fatalf("unexpected manifest: %+v", manifest)
			}
		})
	}
}

func TestDisabledPluginDoesNotRunWebSocketHook(t *testing.T) {
	p := &loadedPlugin{
		Manifest: pluginManifest{ID: "disabled-codec", Enabled: false},
		Source: `package main
import "mcpnet"
func OnWebSocket() { mcpnet.SetDisplayBody([]byte("decoded")) }
`,
	}
	pluginMu.Lock()
	oldList, oldStamp := pluginList, pluginStamp
	pluginList, pluginStamp = []*loadedPlugin{p}, "disabled-codec"
	pluginMu.Unlock()
	t.Cleanup(func() {
		pluginMu.Lock()
		pluginList, pluginStamp = oldList, oldStamp
		pluginMu.Unlock()
	})

	ctx := &pluginCtx{Body: []byte("cipher")}
	if changed := runHook("OnWebSocket", ctx); changed {
		t.Fatal("disabled plugin reported a WebSocket change")
	}
	if p.Interp != nil || ctx.DisplayBodySet || string(ctx.Body) != "cipher" {
		t.Fatalf("disabled plugin still ran: interpreter=%v displaySet=%v body=%q", p.Interp != nil, ctx.DisplayBodySet, ctx.Body)
	}
}

func TestPluginCompileMcpnetPackageImport(t *testing.T) {
	p := &loadedPlugin{Source: `package main
import "mcpnet"

func OnWebSocket() {
	key := []byte("test-key")
	packed, _ := mcpnet.XXTEAPack(mcpnet.Body(), key)
	_, _ = mcpnet.XXTEAUnpack(packed, key)
	mcpnet.SetBody(packed)
	mcpnet.SetURL(mcpnet.URL())
	_ = mcpnet.OpenWindow("Tool", "<button>Run</button>")
	_ = mcpnet.OpenExternalURL("https://example.com/tool")
	_ = mcpnet.Direction()
	_ = mcpnet.Process()
}

func OnUDP() {}
func OnURLDisplay() {}
`}
	if err := p.compile(); err != nil {
		t.Fatalf("compile plugin importing mcpnet: %v", err)
	}
	if !p.hookWS || !p.hookUDP || !p.hookURLView {
		t.Fatalf("plugin hooks not detected: ws=%v udp=%v url=%v", p.hookWS, p.hookUDP, p.hookURLView)
	}
}
