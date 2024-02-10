package compress

import (
	"compress/gzip"
	"io"
	"sync"
	"testing"
)

func TestNewGzipPool(t *testing.T) {
	type args struct {
		config *GzipPoolConfig
	}
	tests := []struct {
		name    string
		args    args
		wantNil bool
	}{
		{
			name: "default",
			args: args{
				config: &GzipPoolConfig{
					Level: -1,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newGzipPool(tt.args.config)
			if (got == nil) != tt.wantNil {
				t.Errorf("newGzipPool() got = %v, wantNil %v", got, tt.wantNil)
			}
		})
	}
}

func TestGzipPoolGet(t *testing.T) {
	tests := []struct {
		name    string
		wantNil bool
	}{
		{
			name:    "default",
			wantNil: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &gzipPool{
				pool: sync.Pool{
					New: func() interface{} {
						w, err := gzip.NewWriterLevel(io.Discard, 1)
						if err != nil {
							return nil
						}
						return w
					},
				},
			}
			got := p.Get()
			if (got == nil) != tt.wantNil {
				t.Errorf("gzipPool.Get() got = %v, wantNil %v", got, tt.wantNil)
			}
		})
	}
}

func TestGzipPoolPut(t *testing.T) {
	type args struct {
		w *gzip.Writer
	}
	tests := []struct {
		name    string
		args    args
		wantNil bool
	}{
		{
			name: "default",
			args: args{
				w: &gzip.Writer{},
			},
			wantNil: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &gzipPool{
				pool: sync.Pool{
					New: func() interface{} {
						w, err := gzip.NewWriterLevel(io.Discard, 1)
						if err != nil {
							return nil
						}
						return w
					},
				},
			}
			p.Put(tt.args.w)
		})
	}
}
