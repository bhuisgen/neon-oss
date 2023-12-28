// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package core

import (
	"context"
	"net"
	"net/http"
)

// ListenerModule
type ListenerModule interface {
	Module
	Register(listenerRegistry ListenerRegistry) error
	Serve(handler http.Handler) error
	Shutdown(ctx context.Context) error
	Close() error
}

// ListenerRegistry
type ListenerRegistry interface {
	Listeners() []net.Listener
	RegisterListener(listener net.Listener) error
}
