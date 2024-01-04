// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package memory

import (
	"reflect"
	"testing"
)

func TestQueueLen(t *testing.T) {
	empty := newQueue()
	filled := newQueue()
	item1 := item{
		value:    "test",
		priority: 0,
	}
	filled.Push(&item1)

	tests := []struct {
		name string
		q    queue
		want int
	}{
		{
			name: "empty",
			q:    empty,
			want: 0,
		},
		{
			name: "filled",
			q:    filled,
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.q.Len(); got != tt.want {
				t.Errorf("queue.Len() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQueueLess(t *testing.T) {
	q := newQueue()
	q.Push(&item{
		value:    "test1",
		priority: 1,
	})
	q.Push(&item{
		value:    "test2",
		priority: 2,
	})
	q.Push(&item{
		value:    "test3",
		priority: 0,
	})

	type args struct {
		i int
		j int
	}
	tests := []struct {
		name string
		q    queue
		args args
		want bool
	}{
		{
			name: "true",
			q:    q,
			args: args{
				i: 0,
				j: 1,
			},
			want: true,
		},
		{
			name: "false",
			q:    q,
			args: args{
				i: 1,
				j: 2,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.q.Less(tt.args.i, tt.args.j); got != tt.want {
				t.Errorf("queue.Less() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQueueSwap(t *testing.T) {
	q := newQueue()
	q.Push(&item{
		value:    "test1",
		priority: 1,
	})
	q.Push(&item{
		value:    "test2",
		priority: 2,
	})
	q.Push(&item{
		value:    "test3",
		priority: 0,
	})

	type args struct {
		i int
		j int
	}
	tests := []struct {
		name string
		q    queue
		args args
	}{
		{
			name: "default",
			q:    q,
			args: args{
				i: 1,
				j: 2,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.q.Swap(tt.args.i, tt.args.j)
		})
	}
}

func TestQueuePush(t *testing.T) {
	q := newQueue()
	type args struct {
		x any
	}
	tests := []struct {
		name string
		q    *queue
		args args
	}{
		{
			name: "default",
			q:    &q,
			args: args{
				x: &item{
					value:    "test",
					priority: 1,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.q.Push(tt.args.x)
		})
	}
}

func TestQueuePop(t *testing.T) {
	q := newQueue()
	q.Push(&item{
		value:    "test1",
		priority: 1,
	})

	tests := []struct {
		name string
		q    *queue
		want any
	}{
		{
			name: "default",
			q:    &q,
			want: &item{
				value:    "test1",
				priority: 1,
				index:    -1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.q.Pop(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("queue.Pop() = %v, want %v", got, tt.want)
			}
		})
	}
}
