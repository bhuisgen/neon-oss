// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package middlewares

import (
	"log"
	"net/http"
	"os"
	"time"
)

type LoggingConfig struct {
	Log     bool
	LogFile string
}

// Logging is a middleware who log all incoming requests
func Logging(config *LoggingConfig, next http.Handler) http.Handler {
	if !config.Log {
		return next
	}

	logger := log.New(os.Stdout, "", log.LstdFlags|log.Lmsgprefix)

	if config.LogFile != "" {
		f, err := os.OpenFile(config.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			log.Fatal(err)
		}
		logger.SetOutput(f)
	}

	fn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := loggingResponseWriter(w)
		next.ServeHTTP(wrapped, r)

		logger.Println(r.Method, r.URL.EscapedPath(), wrapped.status, time.Since(start))
	}

	return http.HandlerFunc(fn)
}

type responseWriter struct {
	http.ResponseWriter

	status      int
	wroteHeader bool
}

func loggingResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w}
}

func (rw *responseWriter) Status() int {
	return rw.status
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}

	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
	rw.wroteHeader = true
}
