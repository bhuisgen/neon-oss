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

type testStoreStorageModule struct {
	errInit          bool
	errLoadResource  bool
	errStoreResource bool
}

const (
	testStoreStorageModuleID module.ModuleID = "app.store.storage.test"
)

func (m testStoreStorageModule) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: testStoreStorageModuleID,
		NewInstance: func() module.Module {
			return &testStoreStorageModule{}
		},
	}
}

func (m testStoreStorageModule) Init(config map[string]interface{}) error {
	if m.errInit {
		return errors.New("test error")
	}
	return nil
}

func (m testStoreStorageModule) LoadResource(name string) (*core.Resource, error) {
	if m.errLoadResource {
		return nil, errors.New("test error")
	}
	return &core.Resource{
		Data: [][]byte{[]byte("test")},
		TTL:  0,
	}, nil
}

func (m testStoreStorageModule) StoreResource(name string, resource *core.Resource) error {
	if m.errStoreResource {
		return errors.New("test error")
	}
	return nil
}

var _ core.StoreStorageModule = (*testStoreStorageModule)(nil)

type testFetcherProviderModule struct {
	errInit  bool
	errFetch bool
}

const (
	testFetcherProviderModuleID module.ModuleID = "app.fetcher.provider.test"
)

func (m testFetcherProviderModule) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: testFetcherProviderModuleID,
		NewInstance: func() module.Module {
			return &testFetcherProviderModule{}
		},
	}
}

func (m testFetcherProviderModule) Init(config map[string]interface{}) error {
	if m.errInit {
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
	errInit  bool
	errParse bool
}

const (
	testLoaderProviderModuleID module.ModuleID = "app.loader.parser.test"
)

func (m testLoaderParserModule) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: testLoaderProviderModuleID,
		NewInstance: func() module.Module {
			return &testLoaderParserModule{}
		},
	}
}

func (m testLoaderParserModule) Init(config map[string]interface{}) error {
	if m.errInit {
		return errors.New("test error")
	}
	return nil
}

func (m testLoaderParserModule) Parse(ctx context.Context, store core.Store, fetcher core.Fetcher) error {
	if m.errParse {
		return errors.New("test error")
	}
	return nil
}

var _ core.LoaderParserModule = (*testLoaderParserModule)(nil)

type testServerListenerModule struct {
	errInit     bool
	errRegister bool
	errServe    bool
	errShutdown bool
	errClose    bool
}

const (
	testServerListenerModuleID module.ModuleID = "app.server.listener.test"
)

func (m testServerListenerModule) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: testServerListenerModuleID,
		NewInstance: func() module.Module {
			return &testServerListenerModule{}
		},
	}
}

func (m testServerListenerModule) Init(config map[string]interface{}) error {
	if m.errInit {
		return errors.New("test error")
	}
	return nil
}

func (m testServerListenerModule) Register(listener core.ServerListener) error {
	if m.errRegister {
		return errors.New("test error")
	}
	return nil
}

func (m testServerListenerModule) Serve(handler http.Handler) error {
	if m.errServe {
		return errors.New("test error")
	}
	return nil
}

func (m testServerListenerModule) Shutdown(ctx context.Context) error {
	if m.errShutdown {
		return errors.New("test error")
	}
	return nil
}

func (m testServerListenerModule) Close() error {
	if m.errClose {
		return errors.New("test error")
	}
	return nil
}

var _ core.ServerListenerModule = (*testServerListenerModule)(nil)

type testServerSiteMiddlewareModule struct {
	errInit     bool
	errRegister bool
	errStart    bool
	errStop     bool
}

const (
	testServerSiteMiddlewareModuleID module.ModuleID = "app.server.site.middleware.test"
)

func (m testServerSiteMiddlewareModule) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: testServerSiteMiddlewareModuleID,
		NewInstance: func() module.Module {
			return &testServerSiteMiddlewareModule{}
		},
	}
}

func (m testServerSiteMiddlewareModule) Init(config map[string]interface{}) error {
	if m.errInit {
		return errors.New("test error")
	}
	return nil
}

func (m testServerSiteMiddlewareModule) Register(server core.ServerSite) error {
	if m.errRegister {
		return errors.New("test error")
	}
	return nil
}

func (m testServerSiteMiddlewareModule) Start() error {
	if m.errStart {
		return errors.New("test error")
	}
	return nil
}

func (m testServerSiteMiddlewareModule) Stop() error {
	if m.errStop {
		return errors.New("test error")
	}
	return nil
}

var _ core.ServerSiteHandlerModule = (*testServerSiteMiddlewareModule)(nil)

type testServerSiteHandlerModule struct {
	errInit     bool
	errRegister bool
	errStart    bool
	errStop     bool
}

const (
	testServerSiteHandlerModuleID module.ModuleID = "app.server.site.handler.test"
)

func (m testServerSiteHandlerModule) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: testServerSiteHandlerModuleID,
		NewInstance: func() module.Module {
			return &testServerSiteHandlerModule{}
		},
	}
}

func (m testServerSiteHandlerModule) Init(config map[string]interface{}) error {
	if m.errInit {
		return errors.New("test error")
	}
	return nil
}

func (m testServerSiteHandlerModule) Register(server core.ServerSite) error {
	if m.errRegister {
		return errors.New("test error")
	}
	return nil
}

func (m testServerSiteHandlerModule) Start() error {
	if m.errStart {
		return errors.New("test error")
	}
	return nil
}

func (m testServerSiteHandlerModule) Stop() error {
	if m.errStop {
		return errors.New("test error")
	}
	return nil
}

var _ core.ServerSiteHandlerModule = (*testServerSiteHandlerModule)(nil)

func TestMain(m *testing.M) {
	module.Register(testStoreStorageModule{})
	module.Register(testFetcherProviderModule{})
	module.Register(testLoaderParserModule{})
	module.Register(testServerListenerModule{})
	module.Register(testServerSiteMiddlewareModule{})
	module.Register(testServerSiteHandlerModule{})
	code := m.Run()
	module.Unregister(testStoreStorageModule{})
	module.Unregister(testFetcherProviderModule{})
	module.Unregister(testLoaderParserModule{})
	module.Unregister(testServerListenerModule{})
	module.Unregister(testServerSiteMiddlewareModule{})
	module.Unregister(testServerSiteHandlerModule{})
	os.Exit(code)
}
