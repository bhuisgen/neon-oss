// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package core

import (
	"net/http"
)

// ServerHandlerModule
type ServerHandlerModule interface {
	Module
	Register(registry ServerRegistry) error
	Start(store Store, fetcher Fetcher) error
	Mount() error
	Unmount()
	Stop()
}

// ServerMiddlewareModule
type ServerMiddlewareModule interface {
	Module
	Register(registry ServerRegistry) error
	Start(store Store, fetcher Fetcher) error
	Mount() error
	Unmount()
	Stop()
}

// ServerRegistry
type ServerRegistry interface {
	RegisterMiddleware(middleware func(next http.Handler) http.Handler) error
	RegisterHandler(handler http.Handler) error
}
