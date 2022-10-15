// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"bytes"
	"sync"
)

// BufferPool
type BufferPool interface {
	Get() *bytes.Buffer
	Put(b *bytes.Buffer)
}

// bufferPool implements a buffer pool
type bufferPool struct {
	pool sync.Pool
}

// newBufferPool creates a new buffer pool
func newBufferPool() *bufferPool {
	return &bufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}
}

// Get selects a buffer from the pool
func (p *bufferPool) Get() *bytes.Buffer {
	return p.pool.Get().(*bytes.Buffer)
}

// Put adds a buffer to the pool
func (p *bufferPool) Put(b *bytes.Buffer) {
	b.Reset()
	p.pool.Put(b)
}
