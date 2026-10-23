# Plugin examples

This directory is the only plugin content intended for the public repository. It contains a deliberately minimal, disabled, read-only starter; it is not a collection of personal protocol plugins.

The application loads **user-installed plugins** from `%USERPROFILE%\McpNet_new\plugins\<plugin-id>\`, not from this repository directory. In the application, use **Program Settings → Plugin Features → Import plugin** to import either a ZIP or a folder containing `main.go` and exactly one JSON manifest (`plugin.json` is recommended). Imported plugins stay disabled until explicitly enabled. You can also copy a reviewed plugin folder into the user-data directory manually.

Do not commit personal finished plugins (including protocol-specific XXTEA/XOR implementations), keys, captures, or user configuration. The repository `.gitignore` allows only this examples directory under `/plugins/`.

The runtime uses Yaegi and offers the `mcpnet` package helpers documented in the project architecture guide. Hooks include `OnHTTPRequest`, `OnHTTPResponse`, `OnWebSocket`, `OnUDP`, `OnURLDisplay`, and `OnPluginAction`. Go hooks are intentionally open-ended: use protocol fields, `regexp`, text, or byte checks as needed; `SetBody`/`SetWireBody` can change traffic and `SetURL` changes display only.

### Plugin-owned windows

There is no client-side **Open UI** button. A Go hook decides when to open its own tool window by calling `mcpnet.OpenWindow(title, html)`; use `mcpnet.OpenExternalURL(url)` for an external HTTP(S) page. The client provides a generic in-app window host, while plugin behavior and UI remain plugin-owned. For example:

```go
func OnHTTPRequest() {
    if mcpnet.Host() != "example.com" || mcpnet.Path() != "/tool" {
        return
    }
    html := `<!doctype html><html><body>
      <button onclick="parent.postMessage({type:'mcpnet-plugin-action',action:'run',payload:{text:'hello'}}, '*')">Run</button>
    </body></html>`
    mcpnet.OpenWindow("My tool", html)
}

func OnPluginAction() {
    if mcpnet.Action() == "run" {
        // Read mcpnet.Payload(), perform plugin logic, then:
        mcpnet.SetUIResult("done")
    }
}
```

The HTML/JavaScript runs in an opaque-origin iframe sandbox; network access and top-level navigation are blocked. Enabled Go plugin code itself runs locally without a security sandbox. Review and trust plugin source before enabling.
