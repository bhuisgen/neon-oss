package render

import "sync"

// RenderWriterPool is the interface of a render writer pool.
type RenderWriterPool interface {
	// Get selects a render writer from the pool.
	Get() RenderWriter
	// Put adds a render writer to the pool.
	Put(w RenderWriter)
}

// renderWriterPool implements a render writer pool.
type renderWriterPool struct {
	pool sync.Pool
}

// NewRenderWriterPool creates a new render writer pool.
func NewRenderWriterPool() *renderWriterPool {
	return &renderWriterPool{
		pool: sync.Pool{
			New: func() interface{} {
				return NewRenderWriter()
			},
		},
	}
}

// Get selects a render writer from the pool.
func (p *renderWriterPool) Get() RenderWriter {
	return p.pool.Get().(RenderWriter)
}

// Put adds a render writer to the pool.
func (p *renderWriterPool) Put(w RenderWriter) {
	w.Reset()
	p.pool.Put(w)
}

var _ RenderWriterPool = (*renderWriterPool)(nil)
