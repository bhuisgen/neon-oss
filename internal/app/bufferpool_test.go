// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"bytes"
	"sync"
	"testing"
)

func TestNewBufferPool(t *testing.T) {
	tests := []struct {
		name    string
		wantNil bool
	}{
		{
			name: "default",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newBufferPool()
			if (got == nil) != tt.wantNil {
				t.Errorf("newBufferPool() got = %v, wantNil %v", got, tt.wantNil)
			}
		})
	}
}

func TestBufferPoolGet(t *testing.T) {
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
			p := &bufferPool{
				pool: sync.Pool{
					New: func() interface{} {
						return new(bytes.Buffer)
					},
				},
			}
			got := p.Get()
			if (got == nil) != tt.wantNil {
				t.Errorf("bufferPool.Get() got = %v, wantNil %v", got, tt.wantNil)
			}
		})
	}
}

func TestBufferPoolPut(t *testing.T) {
	var b bytes.Buffer

	type args struct {
		b *bytes.Buffer
	}
	tests := []struct {
		name    string
		args    args
		wantNil bool
	}{
		{
			name: "default",
			args: args{
				b: &b,
			},
			wantNil: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &bufferPool{
				pool: sync.Pool{
					New: func() interface{} {
						return new(bytes.Buffer)
					},
				},
			}
			p.Put(tt.args.b)
		})
	}
}
