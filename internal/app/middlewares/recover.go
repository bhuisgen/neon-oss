// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package middlewares

import (
	"io"
	"log"
	"net/http"
	"os"
	"runtime/debug"
)

// RecoverConfig implements the configuration of the recover middleware
type RecoverConfig struct {
	Writer io.Writer
	Debug  bool
}

// Recover is a middleware who catch all errors and recovers them
func Recover(config *RecoverConfig, next http.Handler) http.Handler {
	logger := log.New(os.Stderr, "", log.LstdFlags|log.Lmsgprefix)
	if config.Writer != nil {
		logger.SetOutput(config.Writer)
	}

	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			err := recover()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				if config.Debug {
					logger.Printf("%s, %s", err, debug.Stack())
				}
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
