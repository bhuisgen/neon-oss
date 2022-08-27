// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// vmPool implements a pool of VMs
type vmPool struct {
	pool        sync.Pool
	count       int32
	maxVMs      int32
	maxSpareVMs int32
	vms         chan struct{}
}

// NewVMPool creates a new pool
func NewVMPool() *vmPool {
	return &vmPool{
		pool: sync.Pool{
			New: func() interface{} {
				return NewVM()
			},
		},
		count:       0,
		maxVMs:      10,
		maxSpareVMs: int32(runtime.NumCPU()),
		vms:         make(chan struct{}, 10),
	}
}

// Get retrieves a VM from the pool
func (p *vmPool) Get() *vm {
	p.vms <- struct{}{}

	vm := p.pool.Get().(*vm)

	atomic.AddInt32(&p.count, 1)

	return vm
}

// Put releases a VM to the pool
func (p *vmPool) Put(v *vm) {
	<-p.vms

	if atomic.LoadInt32(&p.count) > p.maxSpareVMs {
		v.Close()

		return
	}

	p.pool.Put(v)
}
