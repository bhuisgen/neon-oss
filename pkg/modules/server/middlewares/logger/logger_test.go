// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package logger

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

type testLoggerMiddlewareServer struct {
	err bool
}

func (s testLoggerMiddlewareServer) Name() string {
	return "test"
}

func (s testLoggerMiddlewareServer) Listeners() []string {
	return nil
}

func (s testLoggerMiddlewareServer) Hosts() []string {
	return nil
}

func (s testLoggerMiddlewareServer) Store() core.Store {
	return nil
}

func (s testLoggerMiddlewareServer) Fetcher() core.Fetcher {
	return nil
}

func (s testLoggerMiddlewareServer) RegisterMiddleware(middleware func(next http.Handler) http.Handler) error {
	if s.err {
		return errors.New("test error")
	}
	return nil
}

func (s testLoggerMiddlewareServer) RegisterHandler(handler http.Handler) error {
	if s.err {
		return errors.New("test error")
	}
	return nil
}

var _ core.Server = (*testLoggerMiddlewareServer)(nil)

type testLoggerMiddlewareFileInfo struct {
	name     string
	size     int64
	fileMode os.FileMode
	modTime  time.Time
	isDir    bool
	sys      any
}

func (fi testLoggerMiddlewareFileInfo) Name() string {
	return fi.name
}

func (fi testLoggerMiddlewareFileInfo) Size() int64 {
	return fi.size
}

func (fi testLoggerMiddlewareFileInfo) Mode() os.FileMode {
	return fi.fileMode
}
func (fi testLoggerMiddlewareFileInfo) ModTime() time.Time {
	return fi.modTime
}

func (fi testLoggerMiddlewareFileInfo) IsDir() bool {
	return fi.isDir
}

func (fi testLoggerMiddlewareFileInfo) Sys() any {
	return fi.sys
}

var _ os.FileInfo = (*testLoggerMiddlewareFileInfo)(nil)

func TestLoggerMiddlewareModuleInfo(t *testing.T) {
	type fields struct {
		config     *loggerMiddlewareConfig
		logger     *log.Logger
		reopen     chan os.Signal
		osOpenFile func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osClose    func(f *os.File) error
		osStat     func(name string) (fs.FileInfo, error)
	}
	tests := []struct {
		name   string
		fields fields
		want   module.ModuleInfo
	}{
		{
			name: "default",
			want: module.ModuleInfo{
				ID:          loggerModuleID,
				NewInstance: func() module.Module { return &loggerMiddleware{} },
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := loggerMiddleware{
				config:     tt.fields.config,
				logger:     tt.fields.logger,
				reopen:     tt.fields.reopen,
				osOpenFile: tt.fields.osOpenFile,
				osClose:    tt.fields.osClose,
				osStat:     tt.fields.osStat,
			}
			got := m.ModuleInfo()
			if got.ID != tt.want.ID {
				t.Errorf("loggerMiddleware.ModuleInfo() = %v, want %v", got.ID, tt.want.ID)
			}
			if instance := got.NewInstance(); instance == nil {
				t.Errorf("loggerMiddleware.NewInstance() = %v, want %v", instance, "not nil")
			}
		})
	}
}

func TestLoggerMiddlewareCheck(t *testing.T) {
	type fields struct {
		config     *loggerMiddlewareConfig
		logger     *log.Logger
		reopen     chan os.Signal
		osOpenFile func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osClose    func(f *os.File) error
		osStat     func(name string) (fs.FileInfo, error)
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
					return testLoggerMiddlewareFileInfo{}, nil
				},
			},
			args: args{},
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
					return testLoggerMiddlewareFileInfo{}, nil
				},
			},
			args: args{
				config: map[string]interface{}{
					"File": "access.log",
				},
			},
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
					return testLoggerMiddlewareFileInfo{}, nil
				},
			},
			args: args{
				config: map[string]interface{}{
					"File": "",
				},
			},
			want: []string{
				"option 'File', invalid value ''",
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
					return testLoggerMiddlewareFileInfo{}, nil
				},
			},
			args: args{
				config: map[string]interface{}{
					"File": "access.log",
				},
			},
			want: []string{
				"option 'File', failed to open file 'access.log'",
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
					"File": "access.log",
				},
			},
			want: []string{
				"option 'File', failed to stat file 'access.log'",
			},
			wantErr: true,
		},
		{
			name: "error file is directory",
			fields: fields{
				osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
					return nil, nil
				},
				osClose: func(f *os.File) error {
					return nil
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return testLoggerMiddlewareFileInfo{
						isDir: true,
					}, nil
				},
			},
			args: args{
				config: map[string]interface{}{
					"File": "dir",
				},
			},
			want: []string{
				"option 'File', 'dir' is a directory",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &loggerMiddleware{
				config:     tt.fields.config,
				logger:     tt.fields.logger,
				reopen:     tt.fields.reopen,
				osOpenFile: tt.fields.osOpenFile,
				osClose:    tt.fields.osClose,
				osStat:     tt.fields.osStat,
			}
			got, err := m.Check(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("loggerMiddleware.Check() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("loggerMiddleware.Check() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoggerMiddlewareLoad(t *testing.T) {
	type fields struct {
		config     *loggerMiddlewareConfig
		logger     *log.Logger
		reopen     chan os.Signal
		osOpenFile func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osClose    func(f *os.File) error
		osStat     func(name string) (fs.FileInfo, error)
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
			m := &loggerMiddleware{
				config:     tt.fields.config,
				logger:     tt.fields.logger,
				reopen:     tt.fields.reopen,
				osOpenFile: tt.fields.osOpenFile,
				osClose:    tt.fields.osClose,
				osStat:     tt.fields.osStat,
			}
			if err := m.Load(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("loggerMiddleware.Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoggerMiddlewareRegister(t *testing.T) {
	type fields struct {
		config     *loggerMiddlewareConfig
		logger     *log.Logger
		reopen     chan os.Signal
		osOpenFile func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osClose    func(f *os.File) error
		osStat     func(name string) (fs.FileInfo, error)
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
				server: testLoggerMiddlewareServer{},
			},
		},
		{
			name: "error register",
			args: args{
				server: testLoggerMiddlewareServer{
					err: true,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &loggerMiddleware{
				config:     tt.fields.config,
				logger:     tt.fields.logger,
				reopen:     tt.fields.reopen,
				osOpenFile: tt.fields.osOpenFile,
				osClose:    tt.fields.osClose,
				osStat:     tt.fields.osStat,
			}
			if err := m.Register(tt.args.server); (err != nil) != tt.wantErr {
				t.Errorf("loggerMiddleware.Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoggerMiddlewareStart(t *testing.T) {
	type fields struct {
		config     *loggerMiddlewareConfig
		logger     *log.Logger
		reopen     chan os.Signal
		osOpenFile func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osClose    func(f *os.File) error
		osStat     func(name string) (fs.FileInfo, error)
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				config: &loggerMiddlewareConfig{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &loggerMiddleware{
				config:     tt.fields.config,
				logger:     tt.fields.logger,
				reopen:     tt.fields.reopen,
				osOpenFile: tt.fields.osOpenFile,
				osClose:    tt.fields.osClose,
				osStat:     tt.fields.osStat,
			}
			if err := m.Start(); (err != nil) != tt.wantErr {
				t.Errorf("loggerMiddleware.Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoggerMiddlewareMount(t *testing.T) {
	type fields struct {
		config     *loggerMiddlewareConfig
		logger     *log.Logger
		reopen     chan os.Signal
		osOpenFile func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osClose    func(f *os.File) error
		osStat     func(name string) (fs.FileInfo, error)
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
			m := &loggerMiddleware{
				config:     tt.fields.config,
				logger:     tt.fields.logger,
				reopen:     tt.fields.reopen,
				osOpenFile: tt.fields.osOpenFile,
				osClose:    tt.fields.osClose,
				osStat:     tt.fields.osStat,
			}
			if err := m.Mount(); (err != nil) != tt.wantErr {
				t.Errorf("loggerMiddleware.Mount() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoggerMiddlewareUnmount(t *testing.T) {
	type fields struct {
		config     *loggerMiddlewareConfig
		logger     *log.Logger
		reopen     chan os.Signal
		osOpenFile func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osClose    func(f *os.File) error
		osStat     func(name string) (fs.FileInfo, error)
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
			m := &loggerMiddleware{
				config:     tt.fields.config,
				logger:     tt.fields.logger,
				reopen:     tt.fields.reopen,
				osOpenFile: tt.fields.osOpenFile,
				osClose:    tt.fields.osClose,
				osStat:     tt.fields.osStat,
			}
			m.Unmount()
		})
	}
}

func TestLoggerMiddlewareStop(t *testing.T) {
	type fields struct {
		config     *loggerMiddlewareConfig
		logger     *log.Logger
		reopen     chan os.Signal
		osOpenFile func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osClose    func(f *os.File) error
		osStat     func(name string) (fs.FileInfo, error)
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "default",
			fields: fields{
				config: &loggerMiddlewareConfig{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &loggerMiddleware{
				config:     tt.fields.config,
				logger:     tt.fields.logger,
				reopen:     tt.fields.reopen,
				osOpenFile: tt.fields.osOpenFile,
				osClose:    tt.fields.osClose,
				osStat:     tt.fields.osStat,
			}
			m.Stop()
		})
	}
}

func TestLoggerMiddlewareHandler(t *testing.T) {
	type fields struct {
		config     *loggerMiddlewareConfig
		logger     *log.Logger
		reopen     chan os.Signal
		osOpenFile func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osClose    func(f *os.File) error
		osStat     func(name string) (fs.FileInfo, error)
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
			m := &loggerMiddleware{
				config:     tt.fields.config,
				logger:     tt.fields.logger,
				reopen:     tt.fields.reopen,
				osOpenFile: tt.fields.osOpenFile,
				osClose:    tt.fields.osClose,
				osStat:     tt.fields.osStat,
			}
			got := m.Handler(tt.args.next)
			if tt.wantNil && got != nil {
				t.Errorf("loggerMiddleware.Handler() = %v, want %v", got, nil)
			}
		})
	}
}
