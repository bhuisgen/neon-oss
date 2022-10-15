// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"context"
	"encoding/json"
	"expvar"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime"
	"time"
)

// monitor implements a metrics monitor
type monitor struct {
	config     *MonitorConfig
	logger     *log.Logger
	stopLogger chan struct{}
	stopTracer chan struct{}
}

// MonitorConfig implements the monitor configuration
type MonitorConfig struct {
	Delay  int
	Writer io.Writer
}

// monitorStats implements the monitor statistics
type monitorStats struct {
	NumGoroutine int
	Alloc        uint64
	TotalAlloc   uint64
	Sys          uint64
	Mallocs      uint64
	Frees        uint64
	LiveObjects  uint64
	PauseTotalNs uint64
	NumGC        uint32
}

const (
	monitorLogger string = "monitor"
)

// NewMonitor creates a new resources monitor
func NewMonitor(config *MonitorConfig) *monitor {
	return &monitor{
		config:     config,
		logger:     log.New(os.Stderr, fmt.Sprint(monitorLogger, ": "), log.LstdFlags|log.Lmsgprefix),
		stopLogger: make(chan struct{}),
		stopTracer: make(chan struct{}),
	}
}

// Start starts the monitor
func (m *monitor) Start() {
	go func() {
		var memstats runtime.MemStats
		var goroutine = expvar.NewInt("Goroutine")
		var s monitorStats
		ticker := time.NewTicker(time.Duration(m.config.Delay) * time.Second)

		for {
			select {
			case <-ticker.C:
				runtime.ReadMemStats(&memstats)

				s.NumGoroutine = runtime.NumGoroutine()
				goroutine.Set(int64(s.NumGoroutine))

				s.Alloc = memstats.Alloc
				s.TotalAlloc = memstats.TotalAlloc
				s.Sys = memstats.Sys
				s.Frees = memstats.Frees
				s.Mallocs = memstats.Mallocs
				s.LiveObjects = s.Mallocs - s.Frees
				s.PauseTotalNs = memstats.PauseTotalNs
				s.NumGC = memstats.NumGC

				data, _ := json.Marshal(s)
				m.logger.Printf("Status: %s", string(data))

			case <-m.stopLogger:
				ticker.Stop()

				return
			}
		}
	}()

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/debug/vars", expvar.Handler())
		mux.HandleFunc("/debug/pprof", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

		tracer := http.Server{
			Addr:    "0.0.0.0:6060",
			Handler: mux,
		}

		go tracer.ListenAndServe()

		<-m.stopTracer
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer func() {
			cancel()
		}()

		tracer.Shutdown(ctx)
	}()
}

// Stop stops the monitor
func (m *monitor) Stop(ctx context.Context) {
	m.stopLogger <- struct{}{}
	m.stopTracer <- struct{}{}
}
