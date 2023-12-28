// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package static

import (
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
)

func boolPtr(b bool) *bool {
	return &b
}

type testStaticMiddlewareServerRegistry struct {
	error bool
}

func (r testStaticMiddlewareServerRegistry) RegisterMiddleware(middleware func(next http.Handler) http.Handler) error {
	if r.error {
		return errors.New("test error")
	}
	return nil
}

func (r testStaticMiddlewareServerRegistry) RegisterHandler(handler http.Handler) error {
	if r.error {
		return errors.New("test error")
	}
	return nil
}

var _ core.ServerRegistry = (*testStaticMiddlewareServerRegistry)(nil)

type testStaticMiddlewareFileInfo struct {
	name     string
	size     int64
	fileMode os.FileMode
	modTime  time.Time
	isDir    bool
	sys      any
}

func (fi testStaticMiddlewareFileInfo) Name() string {
	return fi.name
}

func (fi testStaticMiddlewareFileInfo) Size() int64 {
	return fi.size
}

func (fi testStaticMiddlewareFileInfo) Mode() os.FileMode {
	return fi.fileMode
}
func (fi testStaticMiddlewareFileInfo) ModTime() time.Time {
	return fi.modTime
}

func (fi testStaticMiddlewareFileInfo) IsDir() bool {
	return fi.isDir
}

func (fi testStaticMiddlewareFileInfo) Sys() any {
	return fi.sys
}

var _ os.FileInfo = (*testStaticMiddlewareFileInfo)(nil)

func TestStaticMiddlewareModuleInfo(t *testing.T) {
	type fields struct {
		config        *staticMiddlewareConfig
		logger        *log.Logger
		staticFS      StaticFileSystem
		staticHandler http.Handler
		osOpenFile    func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osClose       func(*os.File) error
		osStat        func(name string) (fs.FileInfo, error)
	}
	tests := []struct {
		name   string
		fields fields
		want   module.ModuleInfo
	}{
		{
			name: "default",
			want: module.ModuleInfo{
				ID:          staticModuleID,
				NewInstance: func() module.Module { return &staticMiddleware{} },
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := staticMiddleware{
				config:        tt.fields.config,
				logger:        tt.fields.logger,
				staticFS:      tt.fields.staticFS,
				staticHandler: tt.fields.staticHandler,
				osOpenFile:    tt.fields.osOpenFile,
				osClose:       tt.fields.osClose,
				osStat:        tt.fields.osStat,
			}
			got := m.ModuleInfo()
			if got.ID != tt.want.ID {
				t.Errorf("staticMiddleware.ModuleInfo() = %v, want %v", got.ID, tt.want.ID)
			}
			if instance := got.NewInstance(); instance == nil {
				t.Errorf("staticMiddleware.NewInstance() = %v, want %v", instance, "not nil")
			}
		})
	}
}

func TestStaticMiddlewareCheck(t *testing.T) {
	type fields struct {
		config        *staticMiddlewareConfig
		logger        *log.Logger
		staticFS      StaticFileSystem
		staticHandler http.Handler
		osOpenFile    func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osClose       func(*os.File) error
		osStat        func(name string) (fs.FileInfo, error)
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
			args: args{
				config: map[string]interface{}{
					"Path": "/static",
				},
			},
			fields: fields{
				osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
					return nil, nil
				},
				osClose: func(f *os.File) error {
					return nil
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return testStaticMiddlewareFileInfo{
						isDir: true,
					}, nil
				},
			},
		},
		{
			name: "invalid values",
			args: args{
				config: map[string]interface{}{
					"Path": "",
				},
			},
			want: []string{
				"option 'Path', missing option or value",
			},
			wantErr: true,
		},
		{
			name: "error open file",
			args: args{
				config: map[string]interface{}{
					"Path": "/test",
				},
			},
			fields: fields{
				osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
					return nil, errors.New("test error")
				},
			},
			want: []string{
				"option 'Path', failed to open file '/test'",
			},
			wantErr: true,
		},
		{
			name: "error stat file",
			args: args{
				config: map[string]interface{}{
					"Path": "/test",
				},
			},
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
			want: []string{
				"option 'Path', failed to stat file '/test'",
			},
			wantErr: true,
		},
		{
			name: "error file is not a directory",
			fields: fields{
				osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
					return nil, nil
				},
				osClose: func(f *os.File) error {
					return nil
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return testStaticMiddlewareFileInfo{
						isDir: false,
					}, nil
				},
			},
			args: args{
				config: map[string]interface{}{
					"Path": "dir",
				},
			},
			want: []string{
				"option 'Path', 'dir' is not a directory",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &staticMiddleware{
				config:        tt.fields.config,
				logger:        tt.fields.logger,
				staticFS:      tt.fields.staticFS,
				staticHandler: tt.fields.staticHandler,
				osOpenFile:    tt.fields.osOpenFile,
				osClose:       tt.fields.osClose,
				osStat:        tt.fields.osStat,
			}
			got, err := m.Check(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("staticMiddleware.Check() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("staticMiddleware.Check() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStaticMiddlewareLoad(t *testing.T) {
	type fields struct {
		config        *staticMiddlewareConfig
		logger        *log.Logger
		staticFS      StaticFileSystem
		staticHandler http.Handler
		osOpenFile    func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osClose       func(*os.File) error
		osStat        func(name string) (fs.FileInfo, error)
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
			m := &staticMiddleware{
				config:        tt.fields.config,
				logger:        tt.fields.logger,
				staticFS:      tt.fields.staticFS,
				staticHandler: tt.fields.staticHandler,
				osOpenFile:    tt.fields.osOpenFile,
				osClose:       tt.fields.osClose,
				osStat:        tt.fields.osStat,
			}
			if err := m.Load(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("staticMiddleware.Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStaticMiddlewareRegister(t *testing.T) {
	type fields struct {
		config        *staticMiddlewareConfig
		logger        *log.Logger
		staticFS      StaticFileSystem
		staticHandler http.Handler
		osOpenFile    func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osClose       func(*os.File) error
		osStat        func(name string) (fs.FileInfo, error)
	}
	type args struct {
		registry core.ServerRegistry
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
				registry: testStaticMiddlewareServerRegistry{},
			},
		},
		{
			name: "error register",
			args: args{
				registry: testStaticMiddlewareServerRegistry{
					error: true,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &staticMiddleware{
				config:        tt.fields.config,
				logger:        tt.fields.logger,
				staticFS:      tt.fields.staticFS,
				staticHandler: tt.fields.staticHandler,
				osOpenFile:    tt.fields.osOpenFile,
				osClose:       tt.fields.osClose,
				osStat:        tt.fields.osStat,
			}
			if err := m.Register(tt.args.registry); (err != nil) != tt.wantErr {
				t.Errorf("staticMiddleware.Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStaticMiddlewareStart(t *testing.T) {
	type fields struct {
		config        *staticMiddlewareConfig
		logger        *log.Logger
		staticFS      StaticFileSystem
		staticHandler http.Handler
		osOpenFile    func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osClose       func(*os.File) error
		osStat        func(name string) (fs.FileInfo, error)
	}
	type args struct {
		store   core.Store
		fetcher core.Fetcher
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				config: &staticMiddlewareConfig{
					Path:  "/test",
					Index: boolPtr(false),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &staticMiddleware{
				config:        tt.fields.config,
				logger:        tt.fields.logger,
				staticFS:      tt.fields.staticFS,
				staticHandler: tt.fields.staticHandler,
				osOpenFile:    tt.fields.osOpenFile,
				osClose:       tt.fields.osClose,
				osStat:        tt.fields.osStat,
			}
			if err := m.Start(tt.args.store, tt.args.fetcher); (err != nil) != tt.wantErr {
				t.Errorf("staticMiddleware.Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSstaticMiddlewareMount(t *testing.T) {
	type fields struct {
		config        *staticMiddlewareConfig
		logger        *log.Logger
		staticFS      StaticFileSystem
		staticHandler http.Handler
		osOpenFile    func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osClose       func(*os.File) error
		osStat        func(name string) (fs.FileInfo, error)
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
			m := &staticMiddleware{
				config:        tt.fields.config,
				logger:        tt.fields.logger,
				staticFS:      tt.fields.staticFS,
				staticHandler: tt.fields.staticHandler,
				osOpenFile:    tt.fields.osOpenFile,
				osClose:       tt.fields.osClose,
				osStat:        tt.fields.osStat,
			}
			if err := m.Mount(); (err != nil) != tt.wantErr {
				t.Errorf("staticMiddleware.Mount() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStaticMiddlewareUnmount(t *testing.T) {
	type fields struct {
		config        *staticMiddlewareConfig
		logger        *log.Logger
		staticFS      StaticFileSystem
		staticHandler http.Handler
		osOpenFile    func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osClose       func(*os.File) error
		osStat        func(name string) (fs.FileInfo, error)
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
			m := &staticMiddleware{
				config:        tt.fields.config,
				logger:        tt.fields.logger,
				staticFS:      tt.fields.staticFS,
				staticHandler: tt.fields.staticHandler,
				osOpenFile:    tt.fields.osOpenFile,
				osClose:       tt.fields.osClose,
				osStat:        tt.fields.osStat,
			}
			m.Unmount()
		})
	}
}

func TestStaticMiddlewareStop(t *testing.T) {
	type fields struct {
		config        *staticMiddlewareConfig
		logger        *log.Logger
		staticFS      StaticFileSystem
		staticHandler http.Handler
		osOpenFile    func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osClose       func(*os.File) error
		osStat        func(name string) (fs.FileInfo, error)
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
			m := &staticMiddleware{
				config:        tt.fields.config,
				logger:        tt.fields.logger,
				staticFS:      tt.fields.staticFS,
				staticHandler: tt.fields.staticHandler,
				osOpenFile:    tt.fields.osOpenFile,
				osClose:       tt.fields.osClose,
				osStat:        tt.fields.osStat,
			}
			m.Stop()
		})
	}
}

func TestStaticMiddlewareHandler(t *testing.T) {
	type fields struct {
		config        *staticMiddlewareConfig
		logger        *log.Logger
		staticFS      StaticFileSystem
		staticHandler http.Handler
		osOpenFile    func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osClose       func(*os.File) error
		osStat        func(name string) (fs.FileInfo, error)
	}
	type args struct {
		next http.Handler
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantNil bool
	}{
		{
			name: "default",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &staticMiddleware{
				config:        tt.fields.config,
				logger:        tt.fields.logger,
				staticFS:      tt.fields.staticFS,
				staticHandler: tt.fields.staticHandler,
				osOpenFile:    tt.fields.osOpenFile,
				osClose:       tt.fields.osClose,
				osStat:        tt.fields.osStat,
			}
			got := m.Handler(tt.args.next)
			if tt.wantNil && got != nil {
				t.Errorf("staticMiddleware.Handler() = %v, want %v", got, nil)
			}
		})
	}
}

type testStaticFilesystemFileInfo struct {
	name     string
	size     int64
	fileMode os.FileMode
	modTime  time.Time
	isDir    bool
	sys      any
}

func (fi testStaticFilesystemFileInfo) Name() string {
	return fi.name
}

func (fi testStaticFilesystemFileInfo) Size() int64 {
	return fi.size
}

func (fi testStaticFilesystemFileInfo) Mode() os.FileMode {
	return fi.fileMode
}
func (fi testStaticFilesystemFileInfo) ModTime() time.Time {
	return fi.modTime
}

func (fi testStaticFilesystemFileInfo) IsDir() bool {
	return fi.isDir
}

func (fi testStaticFilesystemFileInfo) Sys() any {
	return fi.sys
}

var _ os.FileInfo = (*testStaticFilesystemFileInfo)(nil)

func TestStaticFileSystemExists(t *testing.T) {
	type fields struct {
		prefix string
		index  bool
		osStat func(name string) (fs.FileInfo, error)
		osOpen func(name string) (*os.File, error)
	}
	type args struct {
		name string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "file",
			fields: fields{
				osStat: func(name string) (fs.FileInfo, error) {
					return testStaticFilesystemFileInfo{}, nil
				},
			},
			args: args{
				name: "/test",
			},
			want: true,
		},
		{
			name: "index file",
			fields: fields{
				index: true,
				osStat: func(name string) (fs.FileInfo, error) {
					return testStaticFilesystemFileInfo{
						isDir: true,
					}, nil
				},
			},
			args: args{
				name: "/",
			},
			want: true,
		},
		{
			name: "error stat file",
			fields: fields{
				osStat: func(name string) (fs.FileInfo, error) {
					return nil, errors.New("test error")
				},
			},
			args: args{
				name: "/",
			},
			want: false,
		},
		{
			name: "error file is directory",
			fields: fields{
				osStat: func(name string) (fs.FileInfo, error) {
					return testStaticFilesystemFileInfo{
						isDir: true,
					}, nil
				},
			},
			args: args{
				name: "/",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := &staticFileSystem{
				prefix: tt.fields.prefix,
				index:  tt.fields.index,
				osStat: tt.fields.osStat,
				osOpen: tt.fields.osOpen,
			}
			if got := fs.Exists(tt.args.name); got != tt.want {
				t.Errorf("staticFileSystem.Exists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStaticFileSystemOpen(t *testing.T) {
	type fields struct {
		prefix string
		index  bool
		osStat func(name string) (fs.FileInfo, error)
		osOpen func(name string) (*os.File, error)
	}
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				osOpen: func(name string) (*os.File, error) {
					return nil, nil
				},
			},
		},
		{
			name: "error open file",
			fields: fields{
				osOpen: func(name string) (*os.File, error) {
					return nil, errors.New("test error")
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := &staticFileSystem{
				prefix: tt.fields.prefix,
				index:  tt.fields.index,
				osStat: tt.fields.osStat,
				osOpen: tt.fields.osOpen,
			}
			_, err := fs.Open(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("staticFileSystem.Open() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}