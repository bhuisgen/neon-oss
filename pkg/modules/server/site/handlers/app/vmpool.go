package app

import (
	"runtime"
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
func newVMPool(max int) *vmPool {
	if max < 0 {
		max = runtime.GOMAXPROCS(0)
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
