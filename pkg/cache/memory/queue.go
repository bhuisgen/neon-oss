// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package memory

import "container/heap"

// queue implements a priority queue.
type queue []*item

// item implements an item of the queue.
type item struct {
	value    any
	priority int
	index    int
}

// newQueue creates a new queue.
func newQueue() queue {
	q := make(queue, 0)
	heap.Init(&q)
	return q
}

// Len returns the length of the queue.
func (q queue) Len() int {
	return len(q)
}

// Less compares the priority of two queued items.
func (q queue) Less(i int, j int) bool {
	return q[i].priority < q[j].priority
}

// Swap swaps two items into the queue.
func (q queue) Swap(i int, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].index = i
	q[j].index = j
}

// Push pushes a new item in the queue.
func (q *queue) Push(x any) {
	n := len(*q)
	item := x.(*item)
	item.index = n
	*q = append(*q, item)
}

// Pop pops an item from the queue.
func (q *queue) Pop() any {
	old := *q
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*q = old[0 : n-1]
	return item
}

var _ heap.Interface = (*queue)(nil)
