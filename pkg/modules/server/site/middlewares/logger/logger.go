// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package logger

import (
	"errors"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
)

// loggerMiddleware implements the logger middleware.
type loggerMiddleware struct {
	config     *loggerMiddlewareConfig
	logger     *slog.Logger
	log        *log.Logger
	reopen     chan os.Signal
	osOpenFile func(name string, flag int, perm fs.FileMode) (*os.File, error)
	osClose    func(*os.File) error
	osStat     func(name string) (fs.FileInfo, error)
}

// loggerMiddlewareConfig implements the logger middleware configuration.
type loggerMiddlewareConfig struct {
	File *string `mapstructure:"file"`
}

const (
	loggerModuleID module.ModuleID = "server.site.middleware.logger"
)

// loggerOsOpenFile redirects to os.OpenFile.
func loggerOsOpenFile(name string, flag int, perm fs.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

// loggerOsClose redirects to os.Close.
func loggerOsClose(f *os.File) error {
	return f.Close()
}

// loggerOsStat redirects to os.Stat.
func loggerOsStat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

// init initializes the module.
func init() {
	module.Register(loggerMiddleware{})
}

// ModuleInfo returns the module information.
func (m loggerMiddleware) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: loggerModuleID,
		NewInstance: func() module.Module {
			return &loggerMiddleware{
				osOpenFile: loggerOsOpenFile,
				osClose:    loggerOsClose,
				osStat:     loggerOsStat,
			}
		},
	}
}

// Init initializes the middleware.
func (m *loggerMiddleware) Init(config map[string]interface{}, logger *slog.Logger) error {
	m.logger = logger

	if err := mapstructure.Decode(config, &m.config); err != nil {
		m.logger.Error("Failed to parse configuration")
		return err
	}

	var errInit bool

	if m.config.File != nil {
		if *m.config.File == "" {
			m.logger.Error("Invalid value", "option", "File", "value", *m.config.File)
			errInit = true
		} else {
			f, err := m.osOpenFile(*m.config.File, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
			if err != nil {
				m.logger.Error("failed to open file", "option", "File", "value", *m.config.File)
				errInit = true
			} else {
				_ = m.osClose(f)
				fi, err := m.osStat(*m.config.File)
				if err != nil {
					m.logger.Error("failed to stat file", "option", "File", "value", *m.config.File)
					errInit = true
				}
				if err == nil && fi.IsDir() {
					m.logger.Error("File is a directory", "option", "File", "value", *m.config.File)
					errInit = true
				}
			}
		}
	}

	if errInit {
		return errors.New("init error")
	}

	return nil
}

// Register registers the middleware.
func (m *loggerMiddleware) Register(site core.ServerSite) error {
	err := site.RegisterMiddleware(m.Handler)
	if err != nil {
		return err
	}

	return nil
}

// Start starts the middleware.
func (m *loggerMiddleware) Start() error {
	var logFileWriter LogFileWriter
	if m.config.File != nil {
		w, err := CreateLogFileWriter(*m.config.File, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			return err
		}

		logFileWriter = w

		m.reopen = make(chan os.Signal, 1)
		signal.Notify(m.reopen, syscall.SIGUSR1)
		go func() {
			for {
				<-m.reopen
				m.logger.Info("Reopening log file")

				if err := logFileWriter.Reopen(); err != nil {
					m.logger.Error("Failed to reopen file")
					return
				}
			}
		}()
	}

	m.log = log.New(os.Stdout, "", log.LstdFlags|log.Lmsgprefix)
	if logFileWriter != nil {
		m.log.SetOutput(logFileWriter)
	}

	return nil
}

// Stop stops the middleware.
func (m *loggerMiddleware) Stop() {
	if m.config.File != nil {
		signal.Stop(m.reopen)
	}
}

// Handler implements the middleware handler.
func (m *loggerMiddleware) Handler(next http.Handler) http.Handler {
	f := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := loggerResponseWriter{w, false, http.StatusOK}

		next.ServeHTTP(&wrapped, r)

		m.log.Println(r.Method, r.URL.EscapedPath(), wrapped.status, time.Since(start))
	}

	return http.HandlerFunc(f)
}

// loggerResponseWriter implements the logging response writer.
type loggerResponseWriter struct {
	http.ResponseWriter
	wroteHeader bool
	status      int
}

// WriteHeader sends an HTTP response header with the provided status code.
func (w *loggerResponseWriter) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}
	w.status = code
	w.ResponseWriter.WriteHeader(code)
	w.wroteHeader = true
}

var _ core.ServerSiteMiddlewareModule = (*loggerMiddleware)(nil)
