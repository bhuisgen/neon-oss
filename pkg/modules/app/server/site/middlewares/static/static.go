package static

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/log"
	"github.com/bhuisgen/neon/pkg/module"
)

// staticMiddleware implements the static middleware.
type staticMiddleware struct {
	config        *staticMiddlewareConfig
	logger        *slog.Logger
	staticFS      StaticFileSystem
	staticHandler http.Handler
	osOpenFile    func(name string, flag int, perm fs.FileMode) (*os.File, error)
	osClose       func(*os.File) error
	osStat        func(name string) (fs.FileInfo, error)
}

// staticMiddlewareConfig implements the static middleware configuration.
type staticMiddlewareConfig struct {
	Path  string `mapstructure:"path"`
	Index *bool  `mapstructure:"index"`
}

const (
	staticModuleID module.ModuleID = "app.server.site.middleware.static"

	staticConfigDefaultIndex bool = false
)

// staticOsOpenFile redirects to os.OpenFile.
func staticOsOpenFile(name string, flag int, perm fs.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

// staticOsClose redirects to os.Close.
func staticOsClose(f *os.File) error {
	return f.Close()
}

// staticOsStat redirects to os.Stat.
func staticOsStat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

// init initializes the package.
func init() {
	module.Register(staticMiddleware{})
}

// ModuleInfo returns the module information.
func (m staticMiddleware) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID:           staticModuleID,
		LoadModule:   func() {},
		UnloadModule: func() {},
		NewInstance: func() module.Module {
			return &staticMiddleware{
				logger:     slog.New(log.NewHandler(os.Stderr, string(staticModuleID), nil)),
				osOpenFile: staticOsOpenFile,
				osClose:    staticOsClose,
				osStat:     staticOsStat,
			}
		},
	}
}

// Init initializes the middleware.
func (m *staticMiddleware) Init(config map[string]interface{}) error {
	if err := mapstructure.Decode(config, &m.config); err != nil {
		m.logger.Error("Failed to parse configuration", "err", err)
		return fmt.Errorf("parse config: %v", err)
	}

	var errConfig bool

	if m.config.Path == "" {
		m.logger.Error("Missing option or value", "option", "Path")
		errConfig = true
	} else {
		f, err := m.osOpenFile(m.config.Path, os.O_RDONLY, 0)
		if err != nil {
			m.logger.Error("Failed to open file", "option", "Path", "value", m.config.Path)
			errConfig = true
		} else {
			_ = m.osClose(f)
			fi, err := m.osStat(m.config.Path)
			if err != nil {
				m.logger.Error("Failed to stat file", "option", "Path", "value", m.config.Path)
				errConfig = true
			}
			if err == nil && !fi.IsDir() {
				m.logger.Error("File is not a directory", "option", "Path", "value", m.config.Path)
				errConfig = true
			}
		}
	}
	if m.config.Index == nil {
		defaultValue := staticConfigDefaultIndex
		m.config.Index = &defaultValue
	}

	if errConfig {
		return errors.New("config")
	}

	return nil
}

// Register registers the middleware.
func (m *staticMiddleware) Register(site core.ServerSite) error {
	if err := site.RegisterMiddleware(m.Handler); err != nil {
		return fmt.Errorf("register middleware: %v", err)
	}

	return nil
}

// Start starts the middleware.
func (m *staticMiddleware) Start() error {
	path, err := filepath.Abs(m.config.Path)
	if err != nil {
		return fmt.Errorf("resolve absolute path: %v", err)
	}

	m.staticFS = &staticFileSystem{
		prefix: path,
		index:  *m.config.Index,
		osStat: staticFileSystemOsStat,
		osOpen: staticFilesystemOsOpen,
	}
	m.staticHandler = http.FileServer(m.staticFS)

	return nil
}

// Stop stops the middleware.
func (m *staticMiddleware) Stop() error {
	return nil
}

// Handler implements the middleware handler.
func (m *staticMiddleware) Handler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if !m.staticFS.Exists(r.URL.Path) {
			next.ServeHTTP(w, r)

			return
		}

		m.staticHandler.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

// StaticFileSystem
type StaticFileSystem interface {
	Exists(name string) bool
	Open(name string) (http.File, error)
}

// staticFileSystem implements the default static filesystem.
type staticFileSystem struct {
	prefix string
	index  bool
	osStat func(name string) (fs.FileInfo, error)
	osOpen func(name string) (*os.File, error)
}

// staticFileSystemOsStat redirects to os.Stat.
func staticFileSystemOsStat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

// staticFilesystemOsOpen redirects to os.Open.
func staticFilesystemOsOpen(name string) (*os.File, error) {
	return os.Open(name)
}

// Exists checks if a file or an index exists.
func (fs *staticFileSystem) Exists(name string) bool {
	if !strings.HasPrefix(name, "/") {
		name = "/" + name
	}

	name = filepath.Join(fs.prefix, name)
	s, err := fs.osStat(name)
	if err != nil {
		return false
	}

	if s.IsDir() {
		if !fs.index {
			return false
		}
		name = filepath.Join(name, "index.html")
		_, err = fs.osStat(name)
	}

	return err == nil
}

// Open implements FileSystem using os.Open, opening files for reading rooted.
// and relative to the directory d.
func (fs *staticFileSystem) Open(name string) (http.File, error) {
	if filepath.Separator != '/' && strings.ContainsRune(name, filepath.Separator) {
		return nil, errors.New("invalid character in file path")
	}
	dir := fs.prefix
	if dir == "" {
		dir = "."
	}
	fullName := filepath.Join(dir, filepath.FromSlash(path.Clean("/"+name)))
	f, err := fs.osOpen(fullName)
	if err != nil {
		return nil, err
	}
	return f, nil
}

var _ core.ServerSiteMiddlewareModule = (*staticMiddleware)(nil)
