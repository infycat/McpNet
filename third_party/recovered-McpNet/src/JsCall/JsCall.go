// Package JsCall is a RECONSTRUCTED drop-in for the McpNet v1.0.3 package
// github.com/qtgolang/McpNet/src/JsCall.
//
// As with src/GoWinHttp, the original v1.0.3 source was retracted by its author
// and is no longer publicly obtainable. The only symbol the main
// program consumes from this package is ToGBK (see process.go), which converts
// a UTF-8 string to its GBK byte representation (used when passing process
// names to the Windows GUI layer).
package JsCall

import (
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// ToGBK converts the given UTF-8 string to GBK encoding. On conversion error
// the original string is returned unchanged so callers remain safe.
func ToGBK(s string) string {
	if s == "" {
		return s
	}
	out, _, err := transform.String(simplifiedchinese.GBK.NewEncoder(), s)
	if err != nil {
		return s
	}
	return out
}
