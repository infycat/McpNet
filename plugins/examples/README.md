# Plugin examples

This directory is the only plugin content intended for the public repository. It contains a deliberately minimal, disabled, read-only starter; it is not a collection of personal protocol plugins.

The application loads **user-installed plugins** from `%USERPROFILE%\McpNet_new\plugins\<plugin-id>\`, not from this repository directory. In the application, use **Program Settings → Plugin Features → Import plugin** to import either a ZIP or a folder containing `main.go` and exactly one JSON manifest (`plugin.json` is recommended). Imported plugins stay disabled until explicitly enabled. You can also copy a reviewed plugin folder into the user-data directory manually.

Do not commit personal finished plugins (including protocol-specific XXTEA/XOR implementations), keys, captures, or user configuration. The repository `.gitignore` allows only this examples directory under `/plugins/`.

The runtime uses Yaegi and offers the `mcpnet` package helpers documented in the project architecture guide. Hooks include `OnHTTPRequest`, `OnHTTPResponse`, `OnWebSocket`, `OnUDP`, `OnURLDisplay`, and `OnPluginAction`. Go hooks are intentionally open-ended: use protocol fields, `regexp`, text, or byte checks as needed; `SetBody`/`SetWireBody` can change traffic and `SetURL` changes display only.

### Optional custom window

Add a `ui` object to `plugin.json` to show an **Open UI** button after enabling the plugin:

```json
{
  "id": "my-tool",
  "name": "My Tool",
  "version": "1.0.0",
  "enabled": false,
  "ui": {
    "title": "My Tool",
    "html": "<!doctype html><html><body><button onclick=\"parent.postMessage({type:'mcpnet-plugin-action',action:'run',payload:{text:'hello'}}, '*')\">Run</button></body></html>"
  }
}
```

The HTML/JavaScript runs in an opaque-origin iframe sandbox; network access and top-level navigation are blocked. UI events are delivered to `OnPluginAction()`. In Go, `mcpnet.Action()` returns `run`, while `mcpnet.Payload()` returns the JSON string `{"text":"hello"}`. Call `mcpnet.SetUIResult("done")` to send a result back; the page can listen for the `mcpnet-plugin-result` message. UI HTML is limited to 256 KiB and each payload/result to 16 KiB. This only isolates the web UI: enabled Go plugin code itself runs locally without a security sandbox. Review and trust plugin source before enabling.
