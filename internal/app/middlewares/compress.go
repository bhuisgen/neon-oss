// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package middlewares

import (
	"bufio"
	"compress/gzip"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
)

// CompressConfig implements the configuration of the Compress middleware
type CompressConfig struct {
	Level int
}

const (
	compressHeaderContentLength   = "Content-Length"
	compressHeaderContentType     = "Content-Type"
	compressHeaderAcceptEncoding  = "Accept-Encoding"
	compressHeaderContentEncoding = "Content-Encoding"
	compressGzipScheme            = "gzip"
)

// Compress is a middleware to compress server responses
func Compress(config *CompressConfig, next http.Handler) http.Handler {
	if config.Level == 0 {
		return next
	}

	pool := newGzipPool(config)

	fn := func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get(compressHeaderAcceptEncoding), compressGzipScheme) {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set(compressHeaderContentEncoding, compressGzipScheme)
		r.Header.Del(compressHeaderAcceptEncoding)

		writer := pool.Get()
		if writer == nil {
			next.ServeHTTP(w, r)
			return
		}
		writer.Reset(w)

		cw := compressResponseWriter{Writer: writer, ResponseWriter: w}
		next.ServeHTTP(&cw, r)
		if !cw.wroteBody {
			if w.Header().Get(compressHeaderContentEncoding) == compressGzipScheme {
				r.Header.Del(compressHeaderAcceptEncoding)
			}
			writer.Reset(io.Discard)
		}

		writer.Close()
		pool.Put(writer)
	}

	return http.HandlerFunc(fn)
}

// compressResponseWriter implements the compress response writer
type compressResponseWriter struct {
	io.Writer
	http.ResponseWriter
	wroteBody bool
}

// WriteHeader sends an HTTP response header with the provided status code.
func (w *compressResponseWriter) WriteHeader(code int) {
	w.Header().Del(compressHeaderContentLength)
	w.ResponseWriter.WriteHeader(code)
}

// Writes writes the response data
func (w *compressResponseWriter) Write(b []byte) (int, error) {
	if w.Header().Get(compressHeaderContentType) == "" {
		w.Header().Set(compressHeaderContentType, http.DetectContentType(b))
	}
	w.wroteBody = true
	return w.Writer.Write(b)
}

// Flush sends the buffered data
func (w *compressResponseWriter) Flush() {
	w.Writer.(*gzip.Writer).Flush()
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack lets the client take over the connection
func (w *compressResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
}

// Push initiates an HTTP/2 server push
func (w *compressResponseWriter) Push(target string, opts *http.PushOptions) error {
	if p, ok := w.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return http.ErrNotSupported
}

// gzipPool implements a gzip pool
type gzipPool struct {
	pool sync.Pool
}

// newGzipPool creates a new gzip pool
func newGzipPool(config *CompressConfig) *gzipPool {
	return &gzipPool{
		pool: sync.Pool{
			New: func() interface{} {
				w, err := gzip.NewWriterLevel(io.Discard, config.Level)
				if err != nil {
					return nil
				}
				return w
			},
		},
	}
}

// Get selects a writer from the pool
func (p *gzipPool) Get() *gzip.Writer {
	return p.pool.Get().(*gzip.Writer)
}

// Put adds a writer to the pool
func (p *gzipPool) Put(w *gzip.Writer) {
	p.pool.Put(w)
}
