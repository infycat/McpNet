//go:build mini
// +build mini

package McpNet

import (
	"github.com/qtgolang/McpNet/src/http"
)

// 是否是用户自定义脚本编辑请求
func (s *proxyRequest) isUserScriptCodeEditRequest(request *http.Request) bool {
	return false
}

// SetScriptCode 设置脚本代码
func (s *Mcp) SetScriptCode(code string) string {
	return "no"
}

// SetScriptPage 设置脚本页面
func (s *Mcp) SetScriptPage(Page string) string {
	return "no"
}
