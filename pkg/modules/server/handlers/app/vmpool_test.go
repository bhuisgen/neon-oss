// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestNewVMPool(t *testing.T) {
	type args struct {
		max int32
	}
	tests := []struct {
		name    string
		args    args
		wantNil bool
	}{
		{
			name: "default",
			args: args{
				max: 1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newVMPool(tt.args.max)
			if (got == nil) != tt.wantNil {
				t.Errorf("newVMPool() got = %v, wantNil %v", got, tt.wantNil)
			}
		})
	}
}

func TestVMPoolGet(t *testing.T) {
	type fields struct {
		maxVMs      int32
		minSpareVMs int32
		maxSpareVMs int32
	}
	tests := []struct {
		name    string
		fields  fields
		wantNil bool
	}{
		{
			name: "default",
			fields: fields{
				maxVMs:      1,
				minSpareVMs: int32(1),
				maxSpareVMs: int32(1),
			},
			wantNil: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &vmPool{
				pool: sync.Pool{
					New: func() interface{} {
						return newVM()
					},
				},
				count:       0,
				minSpareVMs: tt.fields.minSpareVMs,
				maxSpareVMs: tt.fields.maxSpareVMs,
				vms:         make(chan struct{}, tt.fields.maxVMs),
			}
			got := p.Get()
			if (got == nil) != tt.wantNil {
				t.Errorf("vmPool.Get() got = %v, wantNil %v", got, tt.wantNil)
			}
		})
	}
}

func TestVMPoolPut(t *testing.T) {
	type fields struct {
		maxVMs      int32
		minSpareVMs int32
		maxSpareVMs int32
	}
	type args struct {
		v VM
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "default",
			fields: fields{
				maxVMs:      1,
				minSpareVMs: int32(1),
				maxSpareVMs: int32(1),
			},
			args: args{
				v: &vm{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &vmPool{
				pool: sync.Pool{
					New: func() interface{} {
						return newVM()
					},
				},
				count:       0,
				minSpareVMs: tt.fields.minSpareVMs,
				maxSpareVMs: tt.fields.maxSpareVMs,
				vms:         make(chan struct{}, 1),
			}
			p.vms <- struct{}{}
			p.Put(tt.args.v)
		})
	}
}

func TestVMPool_MaxVMs(t *testing.T) {
	p := newVMPool(1)

	var wg sync.WaitGroup
	vm := make(chan VM, 1)
	get := false
	put := false

	vm1 := p.Get()
	wg.Add(2)
	go func() {
		vm2 := p.Get()
		get = true
		vm <- vm2
		wg.Done()
	}()
	go func() {
		vm2 := <-vm
		p.Put(vm2)
		put = true
		wg.Done()
	}()

	if get == true {
		t.Error("failed to get vm")
	}

	p.Put(vm1)
	wg.Wait()

	if put != true {
		t.Error("failed to put vm")
	}
}

func BenchmarkNewVMPool(b *testing.B) {
	for n := 0; n < b.N; n++ {
		p := newVMPool(4)

		worker := func(wg *sync.WaitGroup) {
			defer wg.Done()

			vm := p.Get()
			vm.Configure(&vmConfig{
				Env:     "test",
				Request: &http.Request{},
				State:   nil,
			})
			vm.Execute("test", "(() => { const t = 2 + 2; })();", 4*time.Second)
			p.Put(vm)
		}

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go worker(&wg)
		}

		wg.Wait()
	}
}
