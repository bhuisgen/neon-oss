// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package middlewares

import (
	"log"
	"net/http"
	"time"
)

// Logging is a middleware who log all incoming requests
func Logging(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := loggingResponseWriter(w)
		next.ServeHTTP(wrapped, r)

		log.Println(r.Method, r.URL.EscapedPath(), wrapped.status, time.Since(start))
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
