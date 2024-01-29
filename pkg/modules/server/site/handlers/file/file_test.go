package file

import (
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"os"
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

type testFileHandlerServerSite struct {
	err bool
}

func (s testFileHandlerServerSite) Name() string {
	return "test"
}

func (s testFileHandlerServerSite) Listeners() []string {
	return nil
}

func (s testFileHandlerServerSite) Hosts() []string {
	return nil
}

func (s testFileHandlerServerSite) Store() core.Store {
	return nil
}

func (s testFileHandlerServerSite) Fetcher() core.Fetcher {
	return nil
}

func (s testFileHandlerServerSite) Loader() core.Loader {
	return nil
}

func (s testFileHandlerServerSite) Server() core.Server {
	return nil
}

func (s testFileHandlerServerSite) RegisterMiddleware(middleware func(next http.Handler) http.Handler) error {
	if s.err {
		return errors.New("test error")
	}
	return nil
}

func (s testFileHandlerServerSite) RegisterHandler(handler http.Handler) error {
	if s.err {
		return errors.New("test error")
	}
	return nil
}

var _ core.ServerSite = (*testFileHandlerServerSite)(nil)

type testFileHandlerFileInfo struct {
	name     string
	size     int64
	fileMode os.FileMode
	modTime  time.Time
	isDir    bool
	sys      any
}

func (fi testFileHandlerFileInfo) Name() string {
	return fi.name
}

func (fi testFileHandlerFileInfo) Size() int64 {
	return fi.size
}

func (fi testFileHandlerFileInfo) Mode() os.FileMode {
	return fi.fileMode
}
func (fi testFileHandlerFileInfo) ModTime() time.Time {
	return fi.modTime
}

func (fi testFileHandlerFileInfo) IsDir() bool {
	return fi.isDir
}

func (fi testFileHandlerFileInfo) Sys() any {
	return fi.sys
}

var _ os.FileInfo = (*testFileHandlerFileInfo)(nil)

type testFileHandlerResponseWriter struct {
	header http.Header
}

func (w testFileHandlerResponseWriter) Header() http.Header {
	return w.header
}

func (w testFileHandlerResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (w testFileHandlerResponseWriter) WriteHeader(statusCode int) {
}

var _ http.ResponseWriter = (*testFileHandlerResponseWriter)(nil)

func TestFileHandlerModuleInfo(t *testing.T) {
	type fields struct {
		config     *fileHandlerConfig
		logger     *slog.Logger
		file       []byte
		fileInfo   *time.Time
		muFile     *sync.RWMutex
		rwPool     render.RenderWriterPool
		cache      *fileHandlerCache
		muCache    *sync.RWMutex
		osOpenFile func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osReadFile func(name string) ([]byte, error)
		osClose    func(*os.File) error
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
				ID:          fileModuleID,
				NewInstance: func() module.Module { return &fileHandler{} },
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := fileHandler{
				config:     tt.fields.config,
				logger:     tt.fields.logger,
				file:       tt.fields.file,
				fileInfo:   tt.fields.fileInfo,
				muFile:     tt.fields.muFile,
				rwPool:     tt.fields.rwPool,
				cache:      tt.fields.cache,
				muCache:    tt.fields.muCache,
				osOpenFile: tt.fields.osOpenFile,
				osReadFile: tt.fields.osReadFile,
				osClose:    tt.fields.osClose,
				osStat:     tt.fields.osStat,
			}
			got := h.ModuleInfo()
			if got.ID != tt.want.ID {
				t.Errorf("fileHandler.ModuleInfo() = %v, want %v", got.ID, tt.want.ID)
			}
			if instance := got.NewInstance(); instance == nil {
				t.Errorf("fileHandler.NewInstance() = %v, want %v", instance, "not nil")
			}
		})
	}
}

func TestFileHandlerInit(t *testing.T) {
	type fields struct {
		config     *fileHandlerConfig
		logger     *slog.Logger
		file       []byte
		fileInfo   *time.Time
		muFile     *sync.RWMutex
		rwPool     render.RenderWriterPool
		cache      *fileHandlerCache
		muCache    *sync.RWMutex
		osOpenFile func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osReadFile func(name string) ([]byte, error)
		osClose    func(*os.File) error
		osStat     func(name string) (fs.FileInfo, error)
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
					return testFileHandlerFileInfo{}, nil
				},
			},
			args: args{
				config: map[string]interface{}{
					"Path": "file",
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
					return testFileHandlerFileInfo{}, nil
				},
			},
			args: args{
				config: map[string]interface{}{
					"Path":       "file",
					"StatusCode": 404,
					"Cache":      true,
					"CacheTTL":   60,
				},
				logger: slog.Default(),
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
					return testFileHandlerFileInfo{}, nil
				},
			},
			args: args{
				config: map[string]interface{}{
					"Path":       "",
					"StatusCode": -1,
					"CacheTTL":   -1,
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
					return testFileHandlerFileInfo{}, nil
				},
			},
			args: args{
				config: map[string]interface{}{
					"Path": "file",
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
					"Path": "file",
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
					return testFileHandlerFileInfo{
						isDir: true,
					}, nil
				},
			},
			args: args{
				config: map[string]interface{}{
					"Path": "dir",
				},
				logger: slog.Default(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &fileHandler{
				config:     tt.fields.config,
				logger:     tt.fields.logger,
				file:       tt.fields.file,
				fileInfo:   tt.fields.fileInfo,
				muFile:     tt.fields.muFile,
				rwPool:     tt.fields.rwPool,
				cache:      tt.fields.cache,
				muCache:    tt.fields.muCache,
				osOpenFile: tt.fields.osOpenFile,
				osReadFile: tt.fields.osReadFile,
				osClose:    tt.fields.osClose,
				osStat:     tt.fields.osStat,
			}
			if err := h.Init(tt.args.config, tt.args.logger); (err != nil) != tt.wantErr {
				t.Errorf("fileHandler.Init() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFileHandlerRegister(t *testing.T) {
	type fields struct {
		config     *fileHandlerConfig
		logger     *slog.Logger
		file       []byte
		fileInfo   *time.Time
		muFile     *sync.RWMutex
		rwPool     render.RenderWriterPool
		cache      *fileHandlerCache
		muCache    *sync.RWMutex
		osOpenFile func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osReadFile func(name string) ([]byte, error)
		osClose    func(*os.File) error
		osStat     func(name string) (fs.FileInfo, error)
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
				site: testFileHandlerServerSite{},
			},
		},
		{
			name: "error register",
			args: args{
				site: testFileHandlerServerSite{
					err: true,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &fileHandler{
				config:     tt.fields.config,
				logger:     tt.fields.logger,
				file:       tt.fields.file,
				fileInfo:   tt.fields.fileInfo,
				muFile:     tt.fields.muFile,
				rwPool:     tt.fields.rwPool,
				cache:      tt.fields.cache,
				muCache:    tt.fields.muCache,
				osOpenFile: tt.fields.osOpenFile,
				osReadFile: tt.fields.osReadFile,
				osClose:    tt.fields.osClose,
				osStat:     tt.fields.osStat,
			}
			if err := h.Register(tt.args.site); (err != nil) != tt.wantErr {
				t.Errorf("fileHandler.Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFileHandlerStart(t *testing.T) {
	type fields struct {
		config     *fileHandlerConfig
		logger     *slog.Logger
		file       []byte
		fileInfo   *time.Time
		muFile     *sync.RWMutex
		rwPool     render.RenderWriterPool
		cache      *fileHandlerCache
		muCache    *sync.RWMutex
		osOpenFile func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osReadFile func(name string) ([]byte, error)
		osClose    func(*os.File) error
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
				config: &fileHandlerConfig{
					Path: "test",
				},
				logger: slog.Default(),
				muFile: &sync.RWMutex{},
				osStat: func(name string) (fs.FileInfo, error) {
					return testFileHandlerFileInfo{}, nil
				},
				osReadFile: func(name string) ([]byte, error) {
					return []byte("test"), nil
				},
			},
		},
		{
			name: "error read",
			fields: fields{
				config: &fileHandlerConfig{
					Path: "test",
				},
				logger: slog.Default(),
				muFile: &sync.RWMutex{},
				osStat: func(name string) (fs.FileInfo, error) {
					return nil, errors.New("test error")
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &fileHandler{
				config:     tt.fields.config,
				logger:     tt.fields.logger,
				file:       tt.fields.file,
				fileInfo:   tt.fields.fileInfo,
				muFile:     tt.fields.muFile,
				rwPool:     tt.fields.rwPool,
				cache:      tt.fields.cache,
				muCache:    tt.fields.muCache,
				osOpenFile: tt.fields.osOpenFile,
				osReadFile: tt.fields.osReadFile,
				osClose:    tt.fields.osClose,
				osStat:     tt.fields.osStat,
			}
			if err := h.Start(); (err != nil) != tt.wantErr {
				t.Errorf("fileHandler.Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFileHandlerStop(t *testing.T) {
	type fields struct {
		config     *fileHandlerConfig
		logger     *slog.Logger
		file       []byte
		fileInfo   *time.Time
		muFile     *sync.RWMutex
		rwPool     render.RenderWriterPool
		cache      *fileHandlerCache
		muCache    *sync.RWMutex
		osOpenFile func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osReadFile func(name string) ([]byte, error)
		osClose    func(*os.File) error
		osStat     func(name string) (fs.FileInfo, error)
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "default",
			fields: fields{
				muFile:  &sync.RWMutex{},
				muCache: &sync.RWMutex{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &fileHandler{
				config:     tt.fields.config,
				logger:     tt.fields.logger,
				file:       tt.fields.file,
				fileInfo:   tt.fields.fileInfo,
				muFile:     tt.fields.muFile,
				rwPool:     tt.fields.rwPool,
				cache:      tt.fields.cache,
				muCache:    tt.fields.muCache,
				osOpenFile: tt.fields.osOpenFile,
				osReadFile: tt.fields.osReadFile,
				osClose:    tt.fields.osClose,
				osStat:     tt.fields.osStat,
			}
			h.Stop()
		})
	}
}

func TestFileHandlerServeHTTP(t *testing.T) {
	type fields struct {
		config     *fileHandlerConfig
		logger     *slog.Logger
		file       []byte
		fileInfo   *time.Time
		muFile     *sync.RWMutex
		rwPool     render.RenderWriterPool
		cache      *fileHandlerCache
		muCache    *sync.RWMutex
		osOpenFile func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osReadFile func(name string) ([]byte, error)
		osClose    func(*os.File) error
		osStat     func(name string) (fs.FileInfo, error)
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
				config: &fileHandlerConfig{
					Path:       "test",
					StatusCode: intPtr(404),
					Cache:      boolPtr(true),
					CacheTTL:   intPtr(60),
				},
				logger:  slog.Default(),
				muFile:  &sync.RWMutex{},
				rwPool:  render.NewRenderWriterPool(),
				muCache: &sync.RWMutex{},
				osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
					return nil, nil
				},
				osReadFile: func(name string) ([]byte, error) {
					return []byte("test"), nil
				},
				osClose: func(f *os.File) error {
					return nil
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return testFileHandlerFileInfo{}, nil
				},
			},
			args: args{
				w: testFileHandlerResponseWriter{},
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
			h := &fileHandler{
				config:     tt.fields.config,
				logger:     tt.fields.logger,
				file:       tt.fields.file,
				fileInfo:   tt.fields.fileInfo,
				muFile:     tt.fields.muFile,
				rwPool:     tt.fields.rwPool,
				cache:      tt.fields.cache,
				muCache:    tt.fields.muCache,
				osOpenFile: tt.fields.osOpenFile,
				osClose:    tt.fields.osClose,
				osReadFile: tt.fields.osReadFile,
				osStat:     tt.fields.osStat,
			}
			h.ServeHTTP(tt.args.w, tt.args.r)
		})
	}
}
