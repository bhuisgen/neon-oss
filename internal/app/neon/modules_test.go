// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package neon

import (
	"context"
	"errors"
	"net/http"
	"os"
	"testing"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
)

type testListenerModule struct {
	errCheck    bool
	errLoad     bool
	errRegister bool
	errServe    bool
	errShutdown bool
	errClose    bool
}

const (
	testListenerModuleID module.ModuleID = "listener.test"
)

func (m testListenerModule) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: testListenerModuleID,
		NewInstance: func() module.Module {
			return &testListenerModule{}
		},
	}
}

func (m testListenerModule) Check(config map[string]interface{}) ([]string, error) {
	if m.errCheck {
		return []string{"test"}, errors.New("test error")
	}
	return nil, nil
}

func (m testListenerModule) Load(config map[string]interface{}) error {
	if m.errLoad {
		return errors.New("test error")
	}
	return nil
}

func (m testListenerModule) Register(listener core.Listener) error {
	if m.errRegister {
		return errors.New("test error")
	}
	return nil
}

func (m testListenerModule) Serve(handler http.Handler) error {
	if m.errServe {
		return errors.New("test error")
	}
	return nil
}

func (m testListenerModule) Shutdown(ctx context.Context) error {
	if m.errShutdown {
		return errors.New("test error")
	}
	return nil
}

func (m testListenerModule) Close() error {
	if m.errClose {
		return errors.New("test error")
	}
	return nil
}

var _ core.ListenerModule = (*testListenerModule)(nil)

type testServerMiddlewareModule struct {
	errCheck    bool
	errLoad     bool
	errRegister bool
	errStart    bool
	errMount    bool
}

const (
	testServerMiddlewareModuleID module.ModuleID = "server.middleware.test"
)

func (m testServerMiddlewareModule) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: testServerMiddlewareModuleID,
		NewInstance: func() module.Module {
			return &testServerMiddlewareModule{}
		},
	}
}

func (m testServerMiddlewareModule) Check(config map[string]interface{}) ([]string, error) {
	if m.errCheck {
		return []string{"test"}, errors.New("test error")
	}
	return nil, nil
}

func (m testServerMiddlewareModule) Load(config map[string]interface{}) error {
	if m.errLoad {
		return errors.New("test error")
	}
	return nil
}

func (m testServerMiddlewareModule) Register(server core.Server) error {
	if m.errRegister {
		return errors.New("test error")
	}
	return nil
}

func (m testServerMiddlewareModule) Start() error {
	if m.errStart {
		return errors.New("test error")
	}
	return nil
}

func (m testServerMiddlewareModule) Mount() error {
	if m.errMount {
		return errors.New("test error")
	}
	return nil
}

func (m testServerMiddlewareModule) Unmount() {
}

func (m testServerMiddlewareModule) Stop() {
}

var _ core.ServerHandlerModule = (*testServerMiddlewareModule)(nil)

type testServerHandlerModule struct {
	errCheck    bool
	errLoad     bool
	errRegister bool
	errStart    bool
	errMount    bool
}

const (
	testServerHandlerModuleID module.ModuleID = "server.handler.test"
)

func (m testServerHandlerModule) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: testServerHandlerModuleID,
		NewInstance: func() module.Module {
			return &testServerHandlerModule{}
		},
	}
}

func (m testServerHandlerModule) Check(config map[string]interface{}) ([]string, error) {
	if m.errCheck {
		return []string{"test"}, errors.New("test error")
	}
	return nil, nil
}

func (m testServerHandlerModule) Load(config map[string]interface{}) error {
	if m.errLoad {
		return errors.New("test error")
	}
	return nil
}

func (m testServerHandlerModule) Register(server core.Server) error {
	if m.errRegister {
		return errors.New("test error")
	}
	return nil
}

func (m testServerHandlerModule) Start() error {
	if m.errStart {
		return errors.New("test error")
	}
	return nil
}

func (m testServerHandlerModule) Mount() error {
	if m.errMount {
		return errors.New("test error")
	}
	return nil
}

func (m testServerHandlerModule) Unmount() {
}

func (m testServerHandlerModule) Stop() {
}

var _ core.ServerHandlerModule = (*testServerHandlerModule)(nil)

type testFetcherProviderModule struct {
	errCheck bool
	errLoad  bool
	errFetch bool
}

const (
	testFetcherProviderModuleID module.ModuleID = "fetcher.provider.test"
)

func (m testFetcherProviderModule) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: testFetcherProviderModuleID,
		NewInstance: func() module.Module {
			return &testFetcherProviderModule{}
		},
	}
}

func (m testFetcherProviderModule) Check(config map[string]interface{}) ([]string, error) {
	if m.errCheck {
		return []string{"test"}, errors.New("test error")
	}
	return nil, nil
}

func (m testFetcherProviderModule) Load(config map[string]interface{}) error {
	if m.errLoad {
		return errors.New("test error")
	}
	return nil
}

func (m testFetcherProviderModule) Fetch(ctx context.Context, name string, config map[string]interface{}) (
	*core.Resource, error) {
	if m.errFetch {
		return nil, errors.New("test error")
	}
	return &core.Resource{
		Data: [][]byte{[]byte("test")},
		TTL:  0,
	}, nil
}

var _ core.FetcherProviderModule = (*testFetcherProviderModule)(nil)

type testLoaderParserModule struct {
	errCheck   bool
	errLoad    bool
	errExecute bool
}

const (
	testLoaderProviderModuleID module.ModuleID = "loader.parser.test"
)

func (m testLoaderParserModule) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: testLoaderProviderModuleID,
		NewInstance: func() module.Module {
			return &testLoaderParserModule{}
		},
	}
}

func (m testLoaderParserModule) Check(config map[string]interface{}) ([]string, error) {
	if m.errCheck {
		return []string{"test"}, errors.New("test error")
	}
	return nil, nil
}

func (m testLoaderParserModule) Load(config map[string]interface{}) error {
	if m.errLoad {
		return errors.New("test error")
	}
	return nil
}

func (m testLoaderParserModule) Parse(ctx context.Context, store core.Store, fetcher core.Fetcher) error {
	if m.errExecute {
		return errors.New("test error")
	}
	return nil
}

var _ core.LoaderParserModule = (*testLoaderParserModule)(nil)

func TestMain(m *testing.M) {
	module.Register(testListenerModule{})
	module.Register(testServerMiddlewareModule{})
	module.Register(testServerHandlerModule{})
	module.Register(testFetcherProviderModule{})
	module.Register(testLoaderParserModule{})
	code := m.Run()
	module.Unregister(testListenerModule{})
	module.Unregister(testServerMiddlewareModule{})
	module.Unregister(testServerHandlerModule{})
	module.Unregister(testFetcherProviderModule{})
	module.Unregister(testLoaderParserModule{})
	os.Exit(code)
}
