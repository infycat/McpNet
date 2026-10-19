package main

import "C"
import (
	"github.com/qtgolang/McpNet/src/http"
	_ "github.com/qtgolang/McpNet/src/http/pprof"
)

func init() {
	go func() {
		_ = http.ListenAndServe("0.0.0.0:6001", nil)
	}()
}

func main() {
	Test()
}
