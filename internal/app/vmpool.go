// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"sync"
)

// VMPool
type VMPool interface {
	Get() VM
	Put(v VM)
}

// vmPool implements a VM pool
type vmPool struct {
	pool sync.Pool
	vms  chan struct{}
}

// newVMPool creates a new VM pool
func newVMPool(max int32) *vmPool {
	if max < 1 {
		max = 1
	}

	return &vmPool{
		pool: sync.Pool{
			New: func() interface{} {
				return newVM()
			},
		},
		vms: make(chan struct{}, max),
	}
}

// Get selects a VM from the pool
func (p *vmPool) Get() VM {
	p.vms <- struct{}{}

	vm := p.pool.Get().(VM)

	return vm
}

// Put adds a VM to the pool
func (p *vmPool) Put(v VM) {
	<-p.vms

	v.Reset()
	p.pool.Put(v)
}
