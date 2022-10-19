// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package middlewares

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestCompress(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://test", nil)
	req.Header.Set(compressHeaderAcceptEncoding, compressGzipScheme)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	compress := Compress(&CompressConfig{Level: -1}, next)
	compress.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestCompress_Disabled(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://test", nil)
	req.Header.Set(compressHeaderAcceptEncoding, compressGzipScheme)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	compress := Compress(&CompressConfig{}, next)
	compress.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestCompress_NoAcceptEncodingHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://test", nil)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	compress := Compress(&CompressConfig{Level: -1}, next)
	compress.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestNewGzipPool(t *testing.T) {
	type args struct {
		config *CompressConfig
	}
	tests := []struct {
		name    string
		args    args
		wantNil bool
	}{
		{
			name: "default",
			args: args{
				config: &CompressConfig{
					Level: -1,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newGzipPool(tt.args.config)
			if (got == nil) != tt.wantNil {
				t.Errorf("newGzipPool() got = %v, wantNil %v", got, tt.wantNil)
			}
		})
	}
}

func TestGzipPoolGet(t *testing.T) {
	tests := []struct {
		name    string
		wantNil bool
	}{
		{
			name:    "default",
			wantNil: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &gzipPool{
				pool: sync.Pool{
					New: func() interface{} {
						w, err := gzip.NewWriterLevel(io.Discard, 1)
						if err != nil {
							return nil
						}
						return w
					},
				},
			}
			got := p.Get()
			if (got == nil) != tt.wantNil {
				t.Errorf("gzipPool.Get() got = %v, wantNil %v", got, tt.wantNil)
			}
		})
	}
}

func TestGzipPoolPut(t *testing.T) {
	type args struct {
		w *gzip.Writer
	}
	tests := []struct {
		name    string
		args    args
		wantNil bool
	}{
		{
			name: "default",
			args: args{
				w: &gzip.Writer{},
			},
			wantNil: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &gzipPool{
				pool: sync.Pool{
					New: func() interface{} {
						w, err := gzip.NewWriterLevel(io.Discard, 1)
						if err != nil {
							return nil
						}
						return w
					},
				},
			}
			p.Put(tt.args.w)
		})
	}
}
