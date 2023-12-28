// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package render

import (
	"bytes"
	"net/http"
)

// RenderWriter
type RenderWriter interface {
	http.ResponseWriter
	Reset()
	Write(p []byte) (n int, err error)
	WriteHeader(statusCode int)
	Header() http.Header
	StatusCode() int
	WriteRedirect(url string, statusCode int)
	Redirect() bool
	RedirectURL() string
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
	return w.buf.Write(p)
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

// Load loads a render.
func (w *renderWriter) Load(r Render) {
	w.buf = bytes.NewBuffer(r.Body())
	w.header = r.Header().Clone()
	w.statusCode = r.StatusCode()
	w.redirect = r.Redirect()
	w.redirectURL = r.RedirectURL()
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
