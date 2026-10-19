//go:build !mini
// +build !mini

package Resource

import (
	"embed"
	_ "embed"
	"strings"
)

//go:embed McpNetScriptEdit/assets
var frontendAssets embed.FS

//go:embed McpNetScriptEdit/index.html
var FrontendIndex []byte

//go:embed builtCmdWords.js
var builtCmdWords []byte

func ReadVueFile(name string) ([]byte, error) {
	if strings.Contains(name, "builtCmdWords.js") {
		return builtCmdWords, nil
	}
	fullPath := "McpNetScriptEdit/" + name
	if strings.HasPrefix(name, "/") {
		fullPath = "McpNetScriptEdit" + name
	}
	return frontendAssets.ReadFile(fullPath)
}
