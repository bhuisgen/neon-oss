// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package neon

import (
	"context"
	"net/http"
	"os"

	"github.com/bhuisgen/neon/pkg/core"
)

// Fetcher
type Fetcher interface {
	Check(config map[string]interface{}) ([]string, error)
	Load(config map[string]interface{}) error
	Fetch(ctx context.Context, name string, provider string, config map[string]interface{}) (*core.Resource, error)
}

// Listener
type Listener interface {
	Check(config map[string]interface{}) ([]string, error)
	Load(config map[string]interface{}) error
	Register(descriptor ListenerDescriptor) error
	Serve() error
	Shutdown(ctx context.Context) error
	Close() error
	Remove() error
	Name() string
	Link(server Server) error
	Unlink(server Server) error
	Descriptor() (ListenerDescriptor, error)
}

// ListenerRouter
type ListenerRouter interface {
	http.Handler
}

// ListenerDescriptor
type ListenerDescriptor interface {
	Files() []*os.File
}

// Loader
type Loader interface {
	Check(config map[string]interface{}) ([]string, error)
	Load(config map[string]interface{}) error
	Start() error
	Stop() error
}

// Server
type Server interface {
	Check(config map[string]interface{}) ([]string, error)
	Load(config map[string]interface{}) error
	Register() error
	Start() error
	Enable() error
	Disable(ctx context.Context) error
	Stop() error
	Remove() error
	Name() string
	Listeners() []string
	Hosts() []string
	Router() (ServerRouter, error)
}

// ServerRouter
type ServerRouter interface {
	Routes() map[string]http.Handler
}

// Store
type Store interface {
	Check(config map[string]interface{}) ([]string, error)
	Load(config map[string]interface{}) error
	Get(name string) (*core.Resource, error)
	Set(name string, resource *core.Resource) error
}
