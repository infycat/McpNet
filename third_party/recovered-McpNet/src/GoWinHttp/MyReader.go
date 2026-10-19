package GoWinHttp

import "io"

// MyReader wraps an io.Reader and satisfies io.ReadCloser. The upstream
// GoWinHttp package used a custom reader (MyReader.go) to adapt the native
// WinHTTP response stream into something the standard library http.Response
// Body interface could consume. With the reconstructed net/http backend the
// response Body is already a proper io.ReadCloser, but MyReader is retained
// for API/structural compatibility.
type MyReader struct {
	r io.Reader
}

// NewMyReader wraps r.
func NewMyReader(r io.Reader) *MyReader {
	return &MyReader{r: r}
}

// Read implements io.Reader.
func (m *MyReader) Read(p []byte) (int, error) {
	if m.r == nil {
		return 0, io.EOF
	}
	return m.r.Read(p)
}

// Close closes the underlying reader if it is an io.Closer.
func (m *MyReader) Close() error {
	if c, ok := m.r.(io.Closer); ok {
		return c.Close()
	}
	return nil
}
