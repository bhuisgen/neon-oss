package compress

import (
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
)

// compressMiddleware implements the compress middleware.
type compressMiddleware struct {
	config *compressMiddlewareConfig
	logger *slog.Logger
	pool   *gzipPool
}

// compressMiddlewareConfig implements the compress middleware configuration.
type compressMiddlewareConfig struct {
	Level *int `mapstructure:"level"`
}

const (
	compressModuleID module.ModuleID = "server.site.middleware.compress"

	compressConfigDefaultLevel int = 0

	compressHeaderContentLength   = "Content-Length"
	compressHeaderContentType     = "Content-Type"
	compressHeaderAcceptEncoding  = "Accept-Encoding"
	compressHeaderContentEncoding = "Content-Encoding"
	compressGzipScheme            = "gzip"
)

// init initializes the module.
func init() {
	module.Register(compressMiddleware{})
}

// ModuleInfo returns the module information.
func (m compressMiddleware) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: compressModuleID,
		NewInstance: func() module.Module {
			return &compressMiddleware{}
		},
	}
}

// Init initializes the middleware.
func (m *compressMiddleware) Init(config map[string]interface{}, logger *slog.Logger) error {
	m.logger = logger

	if err := mapstructure.Decode(config, &m.config); err != nil {
		m.logger.Error("Failed to parse configuration", "err", err)
		return fmt.Errorf("parse config: %v", err)
	}

	var errConfig bool

	if m.config.Level == nil {
		defaultValue := compressConfigDefaultLevel
		m.config.Level = &defaultValue
	}
	if *m.config.Level < -2 || *m.config.Level > 9 {
		m.logger.Error("Invalid value", "option", "Level", "value", *m.config.Level)
		errConfig = true
	}

	if errConfig {
		return errors.New("config")
	}

	m.pool = newGzipPool(&GzipPoolConfig{
		Level: *m.config.Level,
	})

	return nil
}

// Register registers the middleware.
func (m *compressMiddleware) Register(site core.ServerSite) error {
	err := site.RegisterMiddleware(m.Handler)
	if err != nil {
		return fmt.Errorf("register middleware: %v", err)
	}

	return nil
}

// Start starts the middleware.
func (m *compressMiddleware) Start() error {
	return nil
}

// Stop stops the middleware.
func (m *compressMiddleware) Stop() error {
	return nil
}

// Handler implements the middleware handler.
func (m *compressMiddleware) Handler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get(compressHeaderAcceptEncoding), compressGzipScheme) {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set(compressHeaderContentEncoding, compressGzipScheme)
		r.Header.Del(compressHeaderAcceptEncoding)

		writer := m.pool.Get()
		if writer == nil {
			next.ServeHTTP(w, r)
			return
		}
		writer.Reset(w)

		rw := compressResponseWriter{Writer: writer, ResponseWriter: w}
		next.ServeHTTP(&rw, r)
		if !rw.wroteBody {
			if w.Header().Get(compressHeaderContentEncoding) == compressGzipScheme {
				r.Header.Del(compressHeaderAcceptEncoding)
			}
			writer.Reset(io.Discard)
		}

		writer.Close()
		m.pool.Put(writer)
	}

	return http.HandlerFunc(fn)
}

// compressResponseWriter implements the compress response writer.
type compressResponseWriter struct {
	io.Writer
	http.ResponseWriter
	wroteBody bool
}

// WriteHeader sends an HTTP response header with the provided status code.
func (w *compressResponseWriter) WriteHeader(code int) {
	w.Header().Del(compressHeaderContentLength)
	w.ResponseWriter.WriteHeader(code)
}

// Writes writes the response data.
func (w *compressResponseWriter) Write(b []byte) (int, error) {
	if w.Header().Get(compressHeaderContentType) == "" {
		w.Header().Set(compressHeaderContentType, http.DetectContentType(b))
	}
	w.wroteBody = true
	n, err := w.Writer.Write(b)
	if err != nil {
		return n, fmt.Errorf("write: %w", err)
	}
	return n, nil
}

// Flush sends the buffered data.
func (w *compressResponseWriter) Flush() {
	w.Writer.(*gzip.Writer).Flush()
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack lets the client take over the connection.
func (w *compressResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	c, rw, err := w.ResponseWriter.(http.Hijacker).Hijack()
	if err != nil {
		return c, rw, fmt.Errorf("hijack: %w", err)
	}
	return c, rw, nil
}

// Push initiates an HTTP/2 server push.
func (w *compressResponseWriter) Push(target string, opts *http.PushOptions) error {
	if p, ok := w.ResponseWriter.(http.Pusher); ok {
		if err := p.Push(target, opts); err != nil {
			return fmt.Errorf("push: %w", err)
		}
	}
	return http.ErrNotSupported
}

var _ core.ServerSiteMiddlewareModule = (*compressMiddleware)(nil)
