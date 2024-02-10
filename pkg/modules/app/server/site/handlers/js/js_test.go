package js

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

type testJSHandlerServerSite struct {
	err bool
}

func (s testJSHandlerServerSite) Name() string {
	return "test"
}

func (s testJSHandlerServerSite) Listeners() []string {
	return nil
}

func (s testJSHandlerServerSite) Hosts() []string {
	return nil
}

func (s testJSHandlerServerSite) Store() core.Store {
	return nil
}

func (s testJSHandlerServerSite) Fetcher() core.Fetcher {
	return nil
}

func (s testJSHandlerServerSite) Loader() core.Loader {
	return nil
}

func (s testJSHandlerServerSite) Server() core.Server {
	return nil
}

func (s testJSHandlerServerSite) RegisterMiddleware(middleware func(next http.Handler) http.Handler) error {
	if s.err {
		return errors.New("test error")
	}
	return nil
}

func (s testJSHandlerServerSite) RegisterHandler(handler http.Handler) error {
	if s.err {
		return errors.New("test error")
	}
	return nil
}

var _ core.ServerSite = (*testJSHandlerServerSite)(nil)

type testJSHandlerFileInfo struct {
	name     string
	size     int64
	fileMode os.FileMode
	modTime  time.Time
	isDir    bool
	sys      any
}

func (fi testJSHandlerFileInfo) Name() string {
	return fi.name
}

func (fi testJSHandlerFileInfo) Size() int64 {
	return fi.size
}

func (fi testJSHandlerFileInfo) Mode() os.FileMode {
	return fi.fileMode
}
func (fi testJSHandlerFileInfo) ModTime() time.Time {
	return fi.modTime
}

func (fi testJSHandlerFileInfo) IsDir() bool {
	return fi.isDir
}

func (fi testJSHandlerFileInfo) Sys() any {
	return fi.sys
}

var _ os.FileInfo = (*testJSHandlerFileInfo)(nil)

type testJSHandlerResponseWriter struct {
	header http.Header
}

func (w testJSHandlerResponseWriter) Header() http.Header {
	return w.header
}

func (w testJSHandlerResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (w testJSHandlerResponseWriter) WriteHeader(statusCode int) {
}

var _ http.ResponseWriter = (*testJSHandlerResponseWriter)(nil)

func TestJSHandlerModuleInfo(t *testing.T) {
	type fields struct {
		config      *jsHandlerConfig
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
				ID:          jsModuleID,
				NewInstance: func() module.Module { return &jsHandler{} },
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := jsHandler{
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
				t.Errorf("jsHandler.ModuleInfo() = %v, want %v", got.ID, tt.want.ID)
			}
			if instance := got.NewInstance(); instance == nil {
				t.Errorf("jsHandler.NewInstance() = %v, want %v", instance, "not nil")
			}
		})
	}
}

func TestJSHandlerInit(t *testing.T) {
	type fields struct {
		config      *jsHandlerConfig
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
				logger: slog.Default(),
				osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
					return nil, nil
				},
				osClose: func(f *os.File) error {
					return nil
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return testJSHandlerFileInfo{}, nil
				},
			},
			args: args{
				config: map[string]interface{}{
					"Index":  "index.html",
					"Bundle": "bundle.js",
				},
			},
		},
		{
			name: "full",
			fields: fields{
				logger: slog.Default(),
				osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
					return nil, nil
				},
				osClose: func(f *os.File) error {
					return nil
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return testJSHandlerFileInfo{}, nil
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
			},
		},
		{
			name: "missing options",
			fields: fields{
				logger: slog.Default(),
				osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
					return nil, nil
				},
				osClose: func(f *os.File) error {
					return nil
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return testJSHandlerFileInfo{}, nil
				},
			},
			args: args{
				config: map[string]interface{}{},
			},
			wantErr: true,
		},
		{
			name: "invalid values",
			fields: fields{
				logger: slog.Default(),
				osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
					return nil, nil
				},
				osClose: func(f *os.File) error {
					return nil
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return testJSHandlerFileInfo{}, nil
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
			},
			wantErr: true,
		},
		{
			name: "error open file",
			fields: fields{
				logger: slog.Default(),
				osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
					return nil, errors.New("test error")
				},
				osClose: func(f *os.File) error {
					return nil
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return testJSHandlerFileInfo{}, nil
				},
			},
			args: args{
				config: map[string]interface{}{
					"Index":  "file1",
					"Bundle": "file2",
				},
			},
			wantErr: true,
		},
		{
			name: "error stat file",
			fields: fields{
				logger: slog.Default(),
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
			},
			wantErr: true,
		},
		{
			name: "stat file is directory",
			fields: fields{
				logger: slog.Default(),
				osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
					return nil, nil
				},
				osClose: func(f *os.File) error {
					return nil
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return testJSHandlerFileInfo{
						isDir: true,
					}, nil
				},
			},
			args: args{
				config: map[string]interface{}{
					"Index":  "dir1",
					"Bundle": "dir2",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &jsHandler{
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
			if err := h.Init(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("jsHandler.Init() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJSHandlerRegister(t *testing.T) {
	type fields struct {
		config      *jsHandlerConfig
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
				site: testJSHandlerServerSite{},
			},
		},
		{
			name: "error register",
			args: args{
				site: testJSHandlerServerSite{
					err: true,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &jsHandler{
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
				t.Errorf("jsHandler.Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJSHandlerStart(t *testing.T) {
	type fields struct {
		config      *jsHandlerConfig
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
				config: &jsHandlerConfig{
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
			h := &jsHandler{
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
				t.Errorf("jsHandler.Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJSHandlerStop(t *testing.T) {
	type fields struct {
		config      *jsHandlerConfig
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
			h := &jsHandler{
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
				t.Errorf("jsHandler.Stop() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJSHandlerServeHTTP(t *testing.T) {
	type fields struct {
		config      *jsHandlerConfig
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
				config: &jsHandlerConfig{
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
				site:     testJSHandlerServerSite{},
				osReadFile: func(name string) ([]byte, error) {
					return os.ReadFile(name)
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return os.Stat(name)
				},
			},
			args: args{
				w: testJSHandlerResponseWriter{},
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
			h := &jsHandler{
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
