package compress

import (
	"errors"
	"log/slog"
	"net/http"
	"testing"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
)

type testCompressMiddlewareServerSite struct {
	err bool
}

func (s testCompressMiddlewareServerSite) Name() string {
	return "test"
}

func (s testCompressMiddlewareServerSite) Listeners() []string {
	return nil
}

func (s testCompressMiddlewareServerSite) Hosts() []string {
	return nil
}

func (s testCompressMiddlewareServerSite) IsDefault() bool {
	return false
}

func (s testCompressMiddlewareServerSite) Store() core.Store {
	return nil
}

func (s testCompressMiddlewareServerSite) Fetcher() core.Fetcher {
	return nil
}

func (s testCompressMiddlewareServerSite) Loader() core.Loader {
	return nil
}

func (s testCompressMiddlewareServerSite) Server() core.Server {
	return nil
}

func (s testCompressMiddlewareServerSite) RegisterMiddleware(
	middleware func(next http.Handler) http.Handler) error {
	if s.err {
		return errors.New("test error")
	}
	return nil
}

func (s testCompressMiddlewareServerSite) RegisterHandler(handler http.Handler) error {
	if s.err {
		return errors.New("test error")
	}
	return nil
}

var _ core.ServerSite = (*testCompressMiddlewareServerSite)(nil)

func TestCompressMiddlewareModuleInfo(t *testing.T) {
	type fields struct {
		config *compressMiddlewareConfig
		logger *slog.Logger
		pool   *gzipPool
	}
	tests := []struct {
		name   string
		fields fields
		want   module.ModuleInfo
	}{
		{
			name: "default",
			want: module.ModuleInfo{
				ID:          compressModuleID,
				NewInstance: func() module.Module { return &compressMiddleware{} },
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := compressMiddleware{
				config: tt.fields.config,
				logger: tt.fields.logger,
				pool:   tt.fields.pool,
			}
			got := m.ModuleInfo()
			if got.ID != tt.want.ID {
				t.Errorf("compressMiddleware.ModuleInfo() = %v, want %v", got.ID, tt.want.ID)
			}
			if instance := got.NewInstance(); instance == nil {
				t.Errorf("compressMiddleware.NewInstance() = %v, want %v", instance, "not nil")
			}
		})
	}
}

func TestCompressMiddlewareInit(t *testing.T) {
	type fields struct {
		config *compressMiddlewareConfig
		logger *slog.Logger
		pool   *gzipPool
	}
	type args struct {
		config map[string]interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "minimal",
			fields: fields{
				logger: slog.Default(),
			},
			args: args{
				config: map[string]interface{}{},
			},
		},
		{
			name: "full",
			fields: fields{
				logger: slog.Default(),
			},
			args: args{
				config: map[string]interface{}{
					"Level": -1,
				},
			},
		},
		{
			name: "invalid config",
			fields: fields{
				logger: slog.Default(),
			},
			args: args{
				config: map[string]interface{}{
					"Level": 100,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &compressMiddleware{
				config: tt.fields.config,
				logger: tt.fields.logger,
				pool:   tt.fields.pool,
			}
			if err := m.Init(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("compressMiddleware.Init() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCompressMiddlewareRegister(t *testing.T) {
	type fields struct {
		config *compressMiddlewareConfig
		logger *slog.Logger
		pool   *gzipPool
	}
	type args struct {
		site core.ServerSite
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "default",
			args: args{
				site: testCompressMiddlewareServerSite{},
			},
		},
		{
			name: "error register",
			args: args{
				site: testCompressMiddlewareServerSite{
					err: true,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &compressMiddleware{
				config: tt.fields.config,
				logger: tt.fields.logger,
				pool:   tt.fields.pool,
			}
			if err := m.Register(tt.args.site); (err != nil) != tt.wantErr {
				t.Errorf("compressMiddleware.Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCompressMiddlewareStart(t *testing.T) {
	type fields struct {
		config *compressMiddlewareConfig
		logger *slog.Logger
		pool   *gzipPool
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &compressMiddleware{
				config: tt.fields.config,
				logger: tt.fields.logger,
				pool:   tt.fields.pool,
			}
			if err := m.Start(); (err != nil) != tt.wantErr {
				t.Errorf("compressMiddleware.Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCompressMiddlewareStop(t *testing.T) {
	type fields struct {
		config *compressMiddlewareConfig
		logger *slog.Logger
		pool   *gzipPool
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &compressMiddleware{
				config: tt.fields.config,
				logger: tt.fields.logger,
				pool:   tt.fields.pool,
			}
			if err := m.Stop(); (err != nil) != tt.wantErr {
				t.Errorf("compressMiddleware.Stop() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCompressMiddlewareHandler(t *testing.T) {
	type fields struct {
		config *compressMiddlewareConfig
		logger *slog.Logger
		pool   *gzipPool
	}
	type args struct {
		next http.Handler
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantNil bool
	}{
		{
			name: "default",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &compressMiddleware{
				config: tt.fields.config,
				logger: tt.fields.logger,
				pool:   tt.fields.pool,
			}
			got := m.Handler(tt.args.next)
			if tt.wantNil && got != nil {
				t.Errorf("compressMiddleware.Handler() = %v, want %v", got, nil)
			}
		})
	}
}
