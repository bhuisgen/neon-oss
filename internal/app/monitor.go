// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"encoding/json"
	"expvar"
	"log"
	"net/http"
	"net/http/pprof"
	"runtime"
	"time"
)

// monitorStats implements the resources monitor statistics
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

// NewMonitor creates a new resources monitor
func NewMonitor(delay int64) {
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/debug/vars", expvar.Handler())
		mux.HandleFunc("/debug/pprof", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		log.Println(http.ListenAndServe("0.0.0.0:6060", mux))
	}()

	var interval = time.Duration(delay) * time.Second
	var memstats runtime.MemStats
	var goroutine = expvar.NewInt("Goroutine")
	var s monitorStats
	go func() {
		for {
			<-time.After(interval)

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
			log.Printf("Monitor state: %s", string(data))
		}
	}()
}
