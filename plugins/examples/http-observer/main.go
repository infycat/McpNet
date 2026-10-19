//go:build plugin_example

package main

import "mcpnet"

// OnHTTPRequest is a read-only example. It intentionally does not call
// SetBody, so it does not alter the captured request.
func OnHTTPRequest() {
	_ = mcpnet.Method()
	_ = mcpnet.URL()
	_ = mcpnet.Host()
}
