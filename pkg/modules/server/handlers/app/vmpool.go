// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"sync"
	"sync/atomic"
)

// VMPool
type VMPool interface {
	Get() VM
	Put(v VM)
}

// vmPool implements a VM pool.
type vmPool struct {
	pool        sync.Pool
	count       int32
	minSpareVMs int32
	maxSpareVMs int32
	vms         chan struct{}
}

// newVMPool creates a new VM pool.
func newVMPool(max int32) *vmPool {
	if max < 0 {
		max = 1
	}

	p := &vmPool{
		pool: sync.Pool{
			New: func() interface{} {
				return newVM()
			},
		},
		count: 0,
		vms:   make(chan struct{}, max),
	}

	return p
}

// Get selects a VM from the pool.
func (p *vmPool) Get() VM {
	p.vms <- struct{}{}

	atomic.AddInt32(&p.count, 1)
	vm := p.pool.Get().(VM)

	return vm
}

// Put adds a VM to the pool.
func (p *vmPool) Put(v VM) {
	<-p.vms

	v.Reset()
	p.pool.Put(v)
}

var _ VMPool = (*vmPool)(nil)
