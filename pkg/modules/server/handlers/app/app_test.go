// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"errors"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/bhuisgen/neon/pkg/cache"
	"github.com/bhuisgen/neon/pkg/cache/memory"
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

type testAppHandlerServer struct {
	err bool
}

func (s testAppHandlerServer) Name() string {
	return "test"
}

func (s testAppHandlerServer) Listeners() []string {
	return nil
}

func (s testAppHandlerServer) Hosts() []string {
	return nil
}

func (s testAppHandlerServer) Store() core.Store {
	return nil
}

func (s testAppHandlerServer) Fetcher() core.Fetcher {
	return nil
}

func (s testAppHandlerServer) RegisterMiddleware(middleware func(next http.Handler) http.Handler) error {
	if s.err {
		return errors.New("test error")
	}
	return nil
}

func (s testAppHandlerServer) RegisterHandler(handler http.Handler) error {
	if s.err {
		return errors.New("test error")
	}
	return nil
}

var _ core.Server = (*testAppHandlerServer)(nil)

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
		logger      *log.Logger
		regexps     []*regexp.Regexp
		index       []byte
		indexInfo   *time.Time
		bundle      string
		bundleInfo  *time.Time
		rwPool      render.RenderWriterPool
		vmPool      VMPool
		cache       cache.Cache
		server      core.Server
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
				bundle:      tt.fields.bundle,
				bundleInfo:  tt.fields.bundleInfo,
				rwPool:      tt.fields.rwPool,
				vmPool:      tt.fields.vmPool,
				cache:       tt.fields.cache,
				server:      tt.fields.server,
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

func TestAppHandlerCheck(t *testing.T) {
	type fields struct {
		config      *appHandlerConfig
		logger      *log.Logger
		regexps     []*regexp.Regexp
		index       []byte
		indexInfo   *time.Time
		bundle      string
		bundleInfo  *time.Time
		rwPool      render.RenderWriterPool
		vmPool      VMPool
		cache       cache.Cache
		server      core.Server
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
		want    []string
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
					"Index":     "index.html",
					"Bundle":    "bundle.js",
					"Env":       "test",
					"Container": "root",
					"State":     "state",
					"Timeout":   4,
					"MaxVMs":    2,
					"Cache":     true,
					"CacheTTL":  60,
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
			},
			want: []string{
				"option 'Index', missing option or value",
				"option 'Bundle', missing option or value",
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
					"Index":     "",
					"Bundle":    "",
					"Env":       "",
					"Container": "",
					"State":     "",
					"Timeout":   -1,
					"MaxVMs":    -1,
					"CacheTTL":  -1,
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
			want: []string{
				"option 'Index', missing option or value",
				"option 'Bundle', missing option or value",
				"option 'Env', invalid value ''",
				"option 'Container', invalid value ''",
				"option 'State', invalid value ''",
				"option 'Timeout', invalid value '-1'",
				"option 'MaxVMs', invalid value '-1'",
				"option 'CacheTTL', invalid value '-1'",
				"rule option 'Path', missing option or value",
				"rule state option 'Key', missing option or value",
				"rule state option 'Resource', missing option or value",
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
			},
			want: []string{
				"option 'Index', failed to open file 'file1'",
				"option 'Bundle', failed to open file 'file2'",
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
			},
			want: []string{
				"option 'Index', failed to stat file 'file1'",
				"option 'Bundle', failed to stat file 'file2'",
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
			},
			want: []string{
				"option 'Index', 'dir1' is a directory",
				"option 'Bundle', 'dir2' is a directory",
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
				bundle:      tt.fields.bundle,
				bundleInfo:  tt.fields.bundleInfo,
				rwPool:      tt.fields.rwPool,
				vmPool:      tt.fields.vmPool,
				cache:       tt.fields.cache,
				server:      tt.fields.server,
				osOpen:      tt.fields.osOpen,
				osOpenFile:  tt.fields.osOpenFile,
				osReadFile:  tt.fields.osReadFile,
				osClose:     tt.fields.osClose,
				osStat:      tt.fields.osStat,
				jsonMarshal: tt.fields.jsonMarshal,
			}
			got, err := h.Check(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("appHandler.Check() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("appHandler.Check() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAppHandlerLoad(t *testing.T) {
	type fields struct {
		config      *appHandlerConfig
		logger      *log.Logger
		regexps     []*regexp.Regexp
		index       []byte
		indexInfo   *time.Time
		bundle      string
		bundleInfo  *time.Time
		rwPool      render.RenderWriterPool
		vmPool      VMPool
		cache       cache.Cache
		server      core.Server
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
			name: "default",
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
				bundle:      tt.fields.bundle,
				bundleInfo:  tt.fields.bundleInfo,
				rwPool:      tt.fields.rwPool,
				vmPool:      tt.fields.vmPool,
				cache:       tt.fields.cache,
				server:      tt.fields.server,
				osOpen:      tt.fields.osOpen,
				osOpenFile:  tt.fields.osOpenFile,
				osReadFile:  tt.fields.osReadFile,
				osClose:     tt.fields.osClose,
				osStat:      tt.fields.osStat,
				jsonMarshal: tt.fields.jsonMarshal,
			}
			if err := h.Load(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("appHandler.Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAppHandlerRegister(t *testing.T) {
	type fields struct {
		config      *appHandlerConfig
		logger      *log.Logger
		regexps     []*regexp.Regexp
		index       []byte
		indexInfo   *time.Time
		bundle      string
		bundleInfo  *time.Time
		rwPool      render.RenderWriterPool
		vmPool      VMPool
		cache       cache.Cache
		server      core.Server
		osOpen      func(name string) (*os.File, error)
		osOpenFile  func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osReadFile  func(name string) ([]byte, error)
		osClose     func(*os.File) error
		osStat      func(name string) (fs.FileInfo, error)
		jsonMarshal func(v any) ([]byte, error)
	}
	type args struct {
		server core.Server
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
				server: testAppHandlerServer{},
			},
		},
		{
			name: "error register",
			args: args{
				server: testAppHandlerServer{
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
				bundle:      tt.fields.bundle,
				bundleInfo:  tt.fields.bundleInfo,
				rwPool:      tt.fields.rwPool,
				vmPool:      tt.fields.vmPool,
				cache:       tt.fields.cache,
				server:      tt.fields.server,
				osOpen:      tt.fields.osOpen,
				osOpenFile:  tt.fields.osOpenFile,
				osReadFile:  tt.fields.osReadFile,
				osClose:     tt.fields.osClose,
				osStat:      tt.fields.osStat,
				jsonMarshal: tt.fields.jsonMarshal,
			}
			if err := h.Register(tt.args.server); (err != nil) != tt.wantErr {
				t.Errorf("appHandler.Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAppHandlerStart(t *testing.T) {
	type fields struct {
		config      *appHandlerConfig
		logger      *log.Logger
		regexps     []*regexp.Regexp
		index       []byte
		indexInfo   *time.Time
		bundle      string
		bundleInfo  *time.Time
		rwPool      render.RenderWriterPool
		vmPool      VMPool
		cache       cache.Cache
		server      core.Server
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
				logger: log.Default(),
				vmPool: newVMPool(1),
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
				bundle:      tt.fields.bundle,
				bundleInfo:  tt.fields.bundleInfo,
				rwPool:      tt.fields.rwPool,
				vmPool:      tt.fields.vmPool,
				cache:       tt.fields.cache,
				server:      tt.fields.server,
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

func TestAppHandlerMount(t *testing.T) {
	type fields struct {
		config      *appHandlerConfig
		logger      *log.Logger
		regexps     []*regexp.Regexp
		index       []byte
		indexInfo   *time.Time
		bundle      string
		bundleInfo  *time.Time
		rwPool      render.RenderWriterPool
		vmPool      VMPool
		cache       cache.Cache
		server      core.Server
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
				bundle:      tt.fields.bundle,
				bundleInfo:  tt.fields.bundleInfo,
				rwPool:      tt.fields.rwPool,
				vmPool:      tt.fields.vmPool,
				cache:       tt.fields.cache,
				server:      tt.fields.server,
				osOpen:      tt.fields.osOpen,
				osOpenFile:  tt.fields.osOpenFile,
				osReadFile:  tt.fields.osReadFile,
				osClose:     tt.fields.osClose,
				osStat:      tt.fields.osStat,
				jsonMarshal: tt.fields.jsonMarshal,
			}
			if err := h.Mount(); (err != nil) != tt.wantErr {
				t.Errorf("appHandler.Mount() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAppHandlerUnmount(t *testing.T) {
	type fields struct {
		config      *appHandlerConfig
		logger      *log.Logger
		regexps     []*regexp.Regexp
		index       []byte
		indexInfo   *time.Time
		bundle      string
		bundleInfo  *time.Time
		rwPool      render.RenderWriterPool
		vmPool      VMPool
		cache       cache.Cache
		server      core.Server
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
	}{
		{
			name: "default",
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
				bundle:      tt.fields.bundle,
				bundleInfo:  tt.fields.bundleInfo,
				rwPool:      tt.fields.rwPool,
				vmPool:      tt.fields.vmPool,
				cache:       tt.fields.cache,
				server:      tt.fields.server,
				osOpen:      tt.fields.osOpen,
				osOpenFile:  tt.fields.osOpenFile,
				osReadFile:  tt.fields.osReadFile,
				osClose:     tt.fields.osClose,
				osStat:      tt.fields.osStat,
				jsonMarshal: tt.fields.jsonMarshal,
			}
			h.Unmount()
		})
	}
}

func TestAppHandlerStop(t *testing.T) {
	type fields struct {
		config      *appHandlerConfig
		logger      *log.Logger
		regexps     []*regexp.Regexp
		index       []byte
		indexInfo   *time.Time
		bundle      string
		bundleInfo  *time.Time
		rwPool      render.RenderWriterPool
		vmPool      VMPool
		cache       cache.Cache
		server      core.Server
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
	}{
		{
			name: "default",
			fields: fields{
				cache: memory.New(0, 0),
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
				bundle:      tt.fields.bundle,
				bundleInfo:  tt.fields.bundleInfo,
				rwPool:      tt.fields.rwPool,
				vmPool:      tt.fields.vmPool,
				cache:       tt.fields.cache,
				server:      tt.fields.server,
				osOpen:      tt.fields.osOpen,
				osOpenFile:  tt.fields.osOpenFile,
				osReadFile:  tt.fields.osReadFile,
				osClose:     tt.fields.osClose,
				osStat:      tt.fields.osStat,
				jsonMarshal: tt.fields.jsonMarshal,
			}
			h.Stop()
		})
	}
}

func TestAppHandlerServeHTTP(t *testing.T) {
	type fields struct {
		config      *appHandlerConfig
		logger      *log.Logger
		regexps     []*regexp.Regexp
		index       []byte
		indexInfo   *time.Time
		bundle      string
		bundleInfo  *time.Time
		rwPool      render.RenderWriterPool
		vmPool      VMPool
		cache       cache.Cache
		server      core.Server
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
					Index:     "test/default/index.html",
					Bundle:    "test/default/bundle.js",
					Env:       stringPtr("test"),
					Container: stringPtr("root"),
					State:     stringPtr("state"),
					Timeout:   intPtr(4),
					MaxVMs:    intPtr(1),
					Cache:     boolPtr(true),
					CacheTTL:  intPtr(60),
				},
				logger: log.Default(),
				rwPool: render.NewRenderWriterPool(),
				vmPool: newVMPool(1),
				cache:  memory.New(0, 0),
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
				bundle:      tt.fields.bundle,
				bundleInfo:  tt.fields.bundleInfo,
				rwPool:      tt.fields.rwPool,
				vmPool:      tt.fields.vmPool,
				cache:       tt.fields.cache,
				server:      tt.fields.server,
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
