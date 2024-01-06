// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package core

import (
	"context"
	"net"
	"net/http"
)

// Listener
type Listener interface {
	Name() string
	RegisterListener(listener net.Listener) error
	Listeners() []net.Listener
}

// ListenerModule
type ListenerModule interface {
	Module
	Register(listener Listener) error
	Serve(handler http.Handler) error
	Shutdown(ctx context.Context) error
	Close() error
}
