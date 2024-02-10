package compress

import (
	"compress/gzip"
	"io"
	"sync"
)

// gzipPool implements a gzip pool.
type gzipPool struct {
	pool sync.Pool
}

// GzipPoolConfig implements the gzip pool configuration.
type GzipPoolConfig struct {
	Level int
}

// newGzipPool creates a new gzip pool.
func newGzipPool(config *GzipPoolConfig) *gzipPool {
	return &gzipPool{
		pool: sync.Pool{
			New: func() interface{} {
				w, err := gzip.NewWriterLevel(io.Discard, config.Level)
				if err != nil {
					return nil
				}
				return w
			},
		},
	}
}

// Get selects a writer from the pool.
func (p *gzipPool) Get() *gzip.Writer {
	return p.pool.Get().(*gzip.Writer)
}

// Put adds a writer to the pool.
func (p *gzipPool) Put(w *gzip.Writer) {
	p.pool.Put(w)
}
