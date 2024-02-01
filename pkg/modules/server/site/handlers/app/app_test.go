package app

import (
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
	"github.com/bhuisgen/neon/pkg/render"
)

func boolPtr(b bool) *bool {
	return &b
}

func intPtr(i int) *int {
	return &i
}

func bytePtr(b []byte) *[]byte {
	return &b
}

func stringPtr(s string) *string {
	return &s
}

type testAppHandlerServerSite struct {
	err bool
}

func (s testAppHandlerServerSite) Name() string {
	return "test"
}

func (s testAppHandlerServerSite) Listeners() []string {
	return nil
}

func (s testAppHandlerServerSite) Hosts() []string {
	return nil
}

func (s testAppHandlerServerSite) Store() core.Store {
	return nil
}

func (s testAppHandlerServerSite) Fetcher() core.Fetcher {
	return nil
}

func (s testAppHandlerServerSite) Loader() core.Loader {
	return nil
}

func (s testAppHandlerServerSite) Server() core.Server {
	return nil
}

func (s testAppHandlerServerSite) RegisterMiddleware(middleware func(next http.Handler) http.Handler) error {
	if s.err {
		return errors.New("test error")
	}
	return nil
}

func (s testAppHandlerServerSite) RegisterHandler(handler http.Handler) error {
	if s.err {
		return errors.New("test error")
	}
	return nil
}

var _ core.ServerSite = (*testAppHandlerServerSite)(nil)

type testAppHandlerFileInfo struct {
	name     string
	size     int64
	fileMode os.FileMode
	modTime  time.Time
	isDir    bool
	sys      any
}

func (fi testAppHandlerFileInfo) Name() string {
	return fi.name
}

func (fi testAppHandlerFileInfo) Size() int64 {
	return fi.size
}

func (fi testAppHandlerFileInfo) Mode() os.FileMode {
	return fi.fileMode
}
func (fi testAppHandlerFileInfo) ModTime() time.Time {
	return fi.modTime
}

func (fi testAppHandlerFileInfo) IsDir() bool {
	return fi.isDir
}

func (fi testAppHandlerFileInfo) Sys() any {
	return fi.sys
}

var _ os.FileInfo = (*testAppHandlerFileInfo)(nil)

type testAppHandlerResponseWriter struct {
	header http.Header
}

func (w testAppHandlerResponseWriter) Header() http.Header {
	return w.header
}

func (w testAppHandlerResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (w testAppHandlerResponseWriter) WriteHeader(statusCode int) {
}

var _ http.ResponseWriter = (*testAppHandlerResponseWriter)(nil)

func TestAppHandlerModuleInfo(t *testing.T) {
	type fields struct {
		config      *appHandlerConfig
		logger      *slog.Logger
		regexps     []*regexp.Regexp
		index       []byte
		indexInfo   *time.Time
		muIndex     *sync.RWMutex
		bundle      string
		bundleInfo  *time.Time
		muBundle    *sync.RWMutex
		rwPool      render.RenderWriterPool
		vmPool      VMPool
		cache       Cache
		site        core.ServerSite
		osOpen      func(name string) (*os.File, error)
		osOpenFile  func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osReadFile  func(name string) ([]byte, error)
		osClose     func(*os.File) error
		osStat      func(name string) (fs.FileInfo, error)
		jsonMarshal func(v any) ([]byte, error)
	}
	tests := []struct {
		name   string
		fields fields
		want   module.ModuleInfo
	}{
		{
			name: "default",
			want: module.ModuleInfo{
				ID:          appModuleID,
				NewInstance: func() module.Module { return &appHandler{} },
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := appHandler{
				config:      tt.fields.config,
				logger:      tt.fields.logger,
				regexps:     tt.fields.regexps,
				index:       tt.fields.index,
				indexInfo:   tt.fields.indexInfo,
				muIndex:     tt.fields.muIndex,
				bundle:      tt.fields.bundle,
				bundleInfo:  tt.fields.bundleInfo,
				muBundle:    tt.fields.muBundle,
				rwPool:      tt.fields.rwPool,
				vmPool:      tt.fields.vmPool,
				cache:       tt.fields.cache,
				site:        tt.fields.site,
				osOpen:      tt.fields.osOpen,
				osOpenFile:  tt.fields.osOpenFile,
				osReadFile:  tt.fields.osReadFile,
				osClose:     tt.fields.osClose,
				osStat:      tt.fields.osStat,
				jsonMarshal: tt.fields.jsonMarshal,
			}
			got := h.ModuleInfo()
			if got.ID != tt.want.ID {
				t.Errorf("appHandler.ModuleInfo() = %v, want %v", got.ID, tt.want.ID)
			}
			if instance := got.NewInstance(); instance == nil {
				t.Errorf("appHandler.NewInstance() = %v, want %v", instance, "not nil")
			}
		})
	}
}

func TestAppHandlerInit(t *testing.T) {
	type fields struct {
		config      *appHandlerConfig
		logger      *slog.Logger
		regexps     []*regexp.Regexp
		index       []byte
		indexInfo   *time.Time
		muIndex     *sync.RWMutex
		bundle      string
		bundleInfo  *time.Time
		muBundle    *sync.RWMutex
		rwPool      render.RenderWriterPool
		vmPool      VMPool
		cache       Cache
		site        core.ServerSite
		osOpen      func(name string) (*os.File, error)
		osOpenFile  func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osReadFile  func(name string) ([]byte, error)
		osClose     func(*os.File) error
		osStat      func(name string) (fs.FileInfo, error)
		jsonMarshal func(v any) ([]byte, error)
	}
	type args struct {
		config map[string]interface{}
		logger *slog.Logger
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "minimal",
			fields: fields{
				osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
					return nil, nil
				},
				osClose: func(f *os.File) error {
					return nil
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return testAppHandlerFileInfo{}, nil
				},
			},
			args: args{
				config: map[string]interface{}{
					"Index":  "index.html",
					"Bundle": "bundle.js",
				},
				logger: slog.Default(),
			},
		},
		{
			name: "full",
			fields: fields{
				osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
					return nil, nil
				},
				osClose: func(f *os.File) error {
					return nil
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return testAppHandlerFileInfo{}, nil
				},
			},
			args: args{
				config: map[string]interface{}{
					"Index":         "index.html",
					"Bundle":        "bundle.js",
					"Env":           "test",
					"Container":     "root",
					"State":         "state",
					"Timeout":       200,
					"MaxVMs":        2,
					"Cache":         true,
					"CacheTTL":      60,
					"CacheMaxItems": 100,
					"Rules": []map[string]interface{}{
						{
							"Path": "/",
							"State:": []map[string]interface{}{
								{
									"Key":      "test",
									"Resource": "test",
								},
							},
							"Last": true,
						},
					},
				},
				logger: slog.Default(),
			},
		},
		{
			name: "missing options",
			fields: fields{
				osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
					return nil, nil
				},
				osClose: func(f *os.File) error {
					return nil
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return testAppHandlerFileInfo{}, nil
				},
			},
			args: args{
				config: map[string]interface{}{},
				logger: slog.Default(),
			},
			wantErr: true,
		},
		{
			name: "invalid values",
			fields: fields{
				osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
					return nil, nil
				},
				osClose: func(f *os.File) error {
					return nil
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return testAppHandlerFileInfo{}, nil
				},
			},
			args: args{
				config: map[string]interface{}{
					"Index":         "",
					"Bundle":        "",
					"Env":           "",
					"Container":     "",
					"State":         "",
					"Timeout":       -1,
					"MaxVMs":        -1,
					"CacheTTL":      -1,
					"CacheMaxItems": -1,
					"Rules": []map[string]interface{}{
						{
							"Path": "",
							"State": []map[string]interface{}{
								{
									"Key":      "",
									"Resource": "",
								},
							},
						},
					},
				},
				logger: slog.Default(),
			},
			wantErr: true,
		},
		{
			name: "error open file",
			fields: fields{
				osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
					return nil, errors.New("test error")
				},
				osClose: func(f *os.File) error {
					return nil
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return testAppHandlerFileInfo{}, nil
				},
			},
			args: args{
				config: map[string]interface{}{
					"Index":  "file1",
					"Bundle": "file2",
				},
				logger: slog.Default(),
			},
			wantErr: true,
		},
		{
			name: "error stat file",
			fields: fields{
				osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
					return nil, nil
				},
				osClose: func(f *os.File) error {
					return nil
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return nil, errors.New("test error")
				},
			},
			args: args{
				config: map[string]interface{}{
					"Index":  "file1",
					"Bundle": "file2",
				},
				logger: slog.Default(),
			},
			wantErr: true,
		},
		{
			name: "stat file is directory",
			fields: fields{
				osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
					return nil, nil
				},
				osClose: func(f *os.File) error {
					return nil
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return testAppHandlerFileInfo{
						isDir: true,
					}, nil
				},
			},
			args: args{
				config: map[string]interface{}{
					"Index":  "dir1",
					"Bundle": "dir2",
				},
				logger: slog.Default(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &appHandler{
				config:      tt.fields.config,
				logger:      tt.fields.logger,
				regexps:     tt.fields.regexps,
				index:       tt.fields.index,
				indexInfo:   tt.fields.indexInfo,
				muIndex:     tt.fields.muIndex,
				bundle:      tt.fields.bundle,
				bundleInfo:  tt.fields.bundleInfo,
				muBundle:    tt.fields.muBundle,
				rwPool:      tt.fields.rwPool,
				vmPool:      tt.fields.vmPool,
				cache:       tt.fields.cache,
				site:        tt.fields.site,
				osOpen:      tt.fields.osOpen,
				osOpenFile:  tt.fields.osOpenFile,
				osReadFile:  tt.fields.osReadFile,
				osClose:     tt.fields.osClose,
				osStat:      tt.fields.osStat,
				jsonMarshal: tt.fields.jsonMarshal,
			}
			if err := h.Init(tt.args.config, tt.args.logger); (err != nil) != tt.wantErr {
				t.Errorf("appHandler.Init() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAppHandlerRegister(t *testing.T) {
	type fields struct {
		config      *appHandlerConfig
		logger      *slog.Logger
		regexps     []*regexp.Regexp
		index       []byte
		indexInfo   *time.Time
		muIndex     *sync.RWMutex
		bundle      string
		bundleInfo  *time.Time
		muBundle    *sync.RWMutex
		rwPool      render.RenderWriterPool
		vmPool      VMPool
		cache       Cache
		site        core.ServerSite
		osOpen      func(name string) (*os.File, error)
		osOpenFile  func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osReadFile  func(name string) ([]byte, error)
		osClose     func(*os.File) error
		osStat      func(name string) (fs.FileInfo, error)
		jsonMarshal func(v any) ([]byte, error)
	}
	type args struct {
		site core.ServerSite
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "default",
			args: args{
				site: testAppHandlerServerSite{},
			},
		},
		{
			name: "error register",
			args: args{
				site: testAppHandlerServerSite{
					err: true,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &appHandler{
				config:      tt.fields.config,
				logger:      tt.fields.logger,
				regexps:     tt.fields.regexps,
				index:       tt.fields.index,
				indexInfo:   tt.fields.indexInfo,
				muIndex:     tt.fields.muIndex,
				bundle:      tt.fields.bundle,
				bundleInfo:  tt.fields.bundleInfo,
				muBundle:    tt.fields.muBundle,
				rwPool:      tt.fields.rwPool,
				vmPool:      tt.fields.vmPool,
				cache:       tt.fields.cache,
				site:        tt.fields.site,
				osOpen:      tt.fields.osOpen,
				osOpenFile:  tt.fields.osOpenFile,
				osReadFile:  tt.fields.osReadFile,
				osClose:     tt.fields.osClose,
				osStat:      tt.fields.osStat,
				jsonMarshal: tt.fields.jsonMarshal,
			}
			if err := h.Register(tt.args.site); (err != nil) != tt.wantErr {
				t.Errorf("appHandler.Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAppHandlerStart(t *testing.T) {
	type fields struct {
		config      *appHandlerConfig
		logger      *slog.Logger
		regexps     []*regexp.Regexp
		index       []byte
		indexInfo   *time.Time
		muIndex     *sync.RWMutex
		bundle      string
		bundleInfo  *time.Time
		muBundle    *sync.RWMutex
		rwPool      render.RenderWriterPool
		vmPool      VMPool
		cache       Cache
		site        core.ServerSite
		osOpen      func(name string) (*os.File, error)
		osOpenFile  func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osReadFile  func(name string) ([]byte, error)
		osClose     func(*os.File) error
		osStat      func(name string) (fs.FileInfo, error)
		jsonMarshal func(v any) ([]byte, error)
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				config: &appHandlerConfig{
					Index:  "test/default/index.html",
					Bundle: "test/default/bundle.js",
					MaxVMs: intPtr(1),
				},
				logger:   slog.Default(),
				muIndex:  &sync.RWMutex{},
				muBundle: &sync.RWMutex{},
				vmPool:   newVMPool(1),
				osReadFile: func(name string) ([]byte, error) {
					return os.ReadFile(name)
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return os.Stat(name)
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &appHandler{
				config:      tt.fields.config,
				logger:      tt.fields.logger,
				regexps:     tt.fields.regexps,
				index:       tt.fields.index,
				indexInfo:   tt.fields.indexInfo,
				muIndex:     tt.fields.muIndex,
				bundle:      tt.fields.bundle,
				bundleInfo:  tt.fields.bundleInfo,
				muBundle:    tt.fields.muBundle,
				rwPool:      tt.fields.rwPool,
				vmPool:      tt.fields.vmPool,
				cache:       tt.fields.cache,
				site:        tt.fields.site,
				osOpen:      tt.fields.osOpen,
				osOpenFile:  tt.fields.osOpenFile,
				osReadFile:  tt.fields.osReadFile,
				osClose:     tt.fields.osClose,
				osStat:      tt.fields.osStat,
				jsonMarshal: tt.fields.jsonMarshal,
			}
			if err := h.Start(); (err != nil) != tt.wantErr {
				t.Errorf("appHandler.Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAppHandlerStop(t *testing.T) {
	type fields struct {
		config      *appHandlerConfig
		logger      *slog.Logger
		regexps     []*regexp.Regexp
		index       []byte
		indexInfo   *time.Time
		muIndex     *sync.RWMutex
		bundle      string
		bundleInfo  *time.Time
		muBundle    *sync.RWMutex
		rwPool      render.RenderWriterPool
		vmPool      VMPool
		cache       Cache
		site        core.ServerSite
		osOpen      func(name string) (*os.File, error)
		osOpenFile  func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osReadFile  func(name string) ([]byte, error)
		osClose     func(*os.File) error
		osStat      func(name string) (fs.FileInfo, error)
		jsonMarshal func(v any) ([]byte, error)
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				muIndex:  &sync.RWMutex{},
				muBundle: &sync.RWMutex{},
				cache:    newCache(1),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &appHandler{
				config:      tt.fields.config,
				logger:      tt.fields.logger,
				regexps:     tt.fields.regexps,
				index:       tt.fields.index,
				indexInfo:   tt.fields.indexInfo,
				muIndex:     tt.fields.muIndex,
				bundle:      tt.fields.bundle,
				bundleInfo:  tt.fields.bundleInfo,
				muBundle:    tt.fields.muBundle,
				rwPool:      tt.fields.rwPool,
				vmPool:      tt.fields.vmPool,
				cache:       tt.fields.cache,
				site:        tt.fields.site,
				osOpen:      tt.fields.osOpen,
				osOpenFile:  tt.fields.osOpenFile,
				osReadFile:  tt.fields.osReadFile,
				osClose:     tt.fields.osClose,
				osStat:      tt.fields.osStat,
				jsonMarshal: tt.fields.jsonMarshal,
			}
			if err := h.Stop(); (err != nil) != tt.wantErr {
				t.Errorf("appHandler.Stop() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAppHandlerServeHTTP(t *testing.T) {
	type fields struct {
		config      *appHandlerConfig
		logger      *slog.Logger
		regexps     []*regexp.Regexp
		index       []byte
		indexInfo   *time.Time
		muIndex     *sync.RWMutex
		bundle      string
		bundleInfo  *time.Time
		muBundle    *sync.RWMutex
		rwPool      render.RenderWriterPool
		vmPool      VMPool
		cache       Cache
		site        core.ServerSite
		osOpen      func(name string) (*os.File, error)
		osOpenFile  func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osReadFile  func(name string) ([]byte, error)
		osClose     func(*os.File) error
		osStat      func(name string) (fs.FileInfo, error)
		jsonMarshal func(v any) ([]byte, error)
	}
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "default",
			fields: fields{
				config: &appHandlerConfig{
					Index:         "test/default/index.html",
					Bundle:        "test/default/bundle.js",
					Env:           stringPtr("test"),
					Container:     stringPtr("root"),
					State:         stringPtr("state"),
					Timeout:       intPtr(200),
					MaxVMs:        intPtr(1),
					Cache:         boolPtr(true),
					CacheTTL:      intPtr(60),
					CacheMaxItems: intPtr(100),
				},
				logger:   slog.Default(),
				muIndex:  &sync.RWMutex{},
				muBundle: &sync.RWMutex{},
				rwPool:   render.NewRenderWriterPool(),
				vmPool:   newVMPool(1),
				cache:    newCache(1),
				site:     testAppHandlerServerSite{},
				osReadFile: func(name string) ([]byte, error) {
					return os.ReadFile(name)
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return os.Stat(name)
				},
			},
			args: args{
				w: testAppHandlerResponseWriter{},
				r: &http.Request{
					URL: &url.URL{
						Path: "/test",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &appHandler{
				config:      tt.fields.config,
				logger:      tt.fields.logger,
				regexps:     tt.fields.regexps,
				index:       tt.fields.index,
				indexInfo:   tt.fields.indexInfo,
				muIndex:     tt.fields.muIndex,
				bundle:      tt.fields.bundle,
				bundleInfo:  tt.fields.bundleInfo,
				muBundle:    tt.fields.muBundle,
				rwPool:      tt.fields.rwPool,
				vmPool:      tt.fields.vmPool,
				cache:       tt.fields.cache,
				site:        tt.fields.site,
				osOpen:      tt.fields.osOpen,
				osOpenFile:  tt.fields.osOpenFile,
				osReadFile:  tt.fields.osReadFile,
				osClose:     tt.fields.osClose,
				osStat:      tt.fields.osStat,
				jsonMarshal: tt.fields.jsonMarshal,
			}
			h.ServeHTTP(tt.args.w, tt.args.r)
		})
	}
}
