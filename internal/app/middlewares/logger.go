// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package middlewares

import (
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// LoggerConfig implements the configuration of the logger middleware
type LoggerConfig struct {
	Log    bool
	Writer io.Writer
}

// Logger is a middleware to log all incoming requests
func Logger(config *LoggerConfig, next http.Handler) http.Handler {
	if !config.Log {
		return next
	}

	logger := log.New(os.Stdout, "", log.LstdFlags|log.Lmsgprefix)
	if config.Writer != nil {
		logger.SetOutput(config.Writer)
	}

	fn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := loggerResponseWriter{w, false, 200}
		next.ServeHTTP(&wrapped, r)

		logger.Println(r.Method, r.URL.EscapedPath(), wrapped.status, time.Since(start))
	}

	return http.HandlerFunc(fn)
}

// loggerResponseWriter implements the logging response writer
type loggerResponseWriter struct {
	http.ResponseWriter
	wroteHeader bool
	status      int
}

// WriteHeader sends an HTTP response header with the provided status code.
func (rw *loggerResponseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
	rw.wroteHeader = true
}
