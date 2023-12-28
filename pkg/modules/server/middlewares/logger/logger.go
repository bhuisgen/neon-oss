// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package logger

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
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
	logger     *log.Logger
	reopen     chan os.Signal
	osOpenFile func(name string, flag int, perm fs.FileMode) (*os.File, error)
	osClose    func(*os.File) error
	osStat     func(name string) (fs.FileInfo, error)
}

// loggerMiddlewareConfig implements the logger middleware configuration.
type loggerMiddlewareConfig struct {
	File *string
}

const (
	loggerModuleID module.ModuleID = "server.middleware.logger"
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

// Check checks the middleware configuration.
func (m *loggerMiddleware) Check(config map[string]interface{}) ([]string, error) {
	var report []string

	var c loggerMiddlewareConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		report = append(report, "failed to parse configuration")
		return report, err
	}

	if c.File != nil {
		if *c.File == "" {
			report = append(report, fmt.Sprintf("option '%s', invalid value '%s'", "File", *c.File))
		} else {
			f, err := m.osOpenFile(*c.File, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
			if err != nil {
				report = append(report, fmt.Sprintf("option '%s', failed to open file '%s'", "File", *c.File))
			} else {
				m.osClose(f)
				fi, err := m.osStat(*c.File)
				if err != nil {
					report = append(report, fmt.Sprintf("option '%s', failed to stat file '%s'", "File", *c.File))
				}
				if err == nil && fi.IsDir() {
					report = append(report, fmt.Sprintf("option '%s', '%s' is a directory", "File", *c.File))
				}
			}
		}
	}

	if len(report) > 0 {
		return report, errors.New("check failure")
	}

	return nil, nil
}

// Load loads the middleware.
func (m *loggerMiddleware) Load(config map[string]interface{}) error {
	var c loggerMiddlewareConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		return err
	}

	m.config = &c

	return nil
}

// Register registers the server resources.
func (m *loggerMiddleware) Register(registry core.ServerRegistry) error {
	err := registry.RegisterMiddleware(m.Handler)
	if err != nil {
		return err
	}

	return nil
}

// Start starts the middleware.
func (m *loggerMiddleware) Start(store core.Store, fetcher core.Fetcher) error {
	var logFileWriter LogFileWriter
	if m.config.File != nil {
		w, err := CreateLogFileWriter(*m.config.File, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return err
		}

		logFileWriter = w

		m.reopen = make(chan os.Signal, 1)
		signal.Notify(m.reopen, syscall.SIGUSR1)
		go func() {
			for {
				<-m.reopen

				m.logger.Print("Reopening access log file")

				logFileWriter.Reopen()
			}
		}()
	}

	m.logger = log.New(os.Stdout, "", log.LstdFlags|log.Lmsgprefix)
	if logFileWriter != nil {
		m.logger.SetOutput(logFileWriter)
	}

	return nil
}

// Mount mounts the middleware.
func (m *loggerMiddleware) Mount() error {
	return nil
}

// Unmount unmounts the middleware.
func (m *loggerMiddleware) Unmount() {
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

		m.logger.Println(r.Method, r.URL.EscapedPath(), wrapped.status, time.Since(start))
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

var _ core.ServerMiddlewareModule = (*loggerMiddleware)(nil)
