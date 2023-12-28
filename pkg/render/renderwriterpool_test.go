// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package render

import (
	"sync"
	"testing"
)

func TestNewRenderWriterPool(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "default",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			NewRenderWriterPool()
		})
	}
}

func TestRenderWriterPoolGet(t *testing.T) {
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
			p := &renderWriterPool{
				pool: sync.Pool{
					New: func() interface{} {
						return NewRenderWriter()
					},
				},
			}
			got := p.Get()
			if (got == nil) != tt.wantNil {
				t.Errorf("renderWriterPool.Get() got = %v, wantNil %v", got, tt.wantNil)
			}
		})
	}
}

func TestRenderWriterPoolPut(t *testing.T) {
	type args struct {
		w RenderWriter
	}
	tests := []struct {
		name    string
		args    args
		wantNil bool
	}{
		{
			name: "default",
			args: args{
				w: NewRenderWriter(),
			},
			wantNil: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &renderWriterPool{
				pool: sync.Pool{
					New: func() interface{} {
						return NewRenderWriter()
					},
				},
			}
			p.Put(tt.args.w)
		})
	}
}
