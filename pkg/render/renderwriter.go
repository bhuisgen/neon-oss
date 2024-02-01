package render

import (
	"bytes"
	"fmt"
	"net/http"
)

// RenderWriter is the interface of a render writer.
type RenderWriter interface {
	// Reset resets the buffer.
	Reset()
	// Write writes bytes into the response body.
	Write(p []byte) (n int, err error)
	// WriteHeader writes the HTTP response header.
	WriteHeader(statusCode int)
	// WriteRedirect writes a HTTP redirect.
	WriteRedirect(url string, statusCode int)
	// Header returns the HTTP response headers.
	Header() http.Header
	// StatusCode returns the HTTP response status code.
	StatusCode() int
	// Redirect returns the redirect flag.
	Redirect() bool
	// RedirectURL returns the redirect URL.
	RedirectURL() string
	// Render returns a render.
	Render() Render
}

// renderWriter implements a render writer.
type renderWriter struct {
	buf         *bytes.Buffer
	header      http.Header
	statusCode  int
	redirect    bool
	redirectURL string
}

// NewRenderWriter creates a new render writer.
func NewRenderWriter() *renderWriter {
	w := new(renderWriter)
	w.buf = new(bytes.Buffer)
	w.header = http.Header{}
	w.statusCode = http.StatusOK

	return w
}

// Reset resets the buffer.
func (w *renderWriter) Reset() {
	w.buf.Reset()
	w.header = http.Header{}
	w.statusCode = http.StatusOK
	w.redirect = false
	w.redirectURL = ""
}

// Write writes bytes into the response body.
func (w *renderWriter) Write(p []byte) (int, error) {
	n, err := w.buf.Write(p)
	if err != nil {
		return n, fmt.Errorf("write buffer: %w", err)
	}
	return n, nil
}

// WriteHeader writes the HTTP response header.
func (w *renderWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

// WriteRedirect writes a HTTP redirect.
func (w *renderWriter) WriteRedirect(url string, statusCode int) {
	w.redirect = true
	w.redirectURL = url
	w.statusCode = statusCode
}

// Header returns the HTTP response headers.
func (w *renderWriter) Header() http.Header {
	return w.header
}

// StatusCode returns the HTTP response status code.
func (w *renderWriter) StatusCode() int {
	return w.statusCode
}

// Redirect returns the redirect flag.
func (w *renderWriter) Redirect() bool {
	return w.redirect
}

// RedirectURL returns the redirect URL.
func (w *renderWriter) RedirectURL() string {
	return w.redirectURL
}

// Render returns a render.
func (w *renderWriter) Render() Render {
	r := new(render)

	if !w.redirect {
		r.body = make([]byte, w.buf.Len())
		copy(r.body, w.buf.Bytes())
	}
	r.header = w.header.Clone()
	r.statusCode = w.statusCode
	if w.redirect {
		r.redirect = true
		r.redirectURL = w.redirectURL
	}

	return r
}

var _ RenderWriter = (*renderWriter)(nil)
