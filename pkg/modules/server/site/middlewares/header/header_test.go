package header

import (
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"testing"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
)

type testHeaderMiddlewareServerSite struct {
	err bool
}

func (s testHeaderMiddlewareServerSite) Name() string {
	return "test"
}

func (s testHeaderMiddlewareServerSite) Listeners() []string {
	return nil
}

func (s testHeaderMiddlewareServerSite) Hosts() []string {
	return nil
}

func (s testHeaderMiddlewareServerSite) Store() core.Store {
	return nil
}

func (s testHeaderMiddlewareServerSite) Fetcher() core.Fetcher {
	return nil
}

func (s testHeaderMiddlewareServerSite) Loader() core.Loader {
	return nil
}

func (s testHeaderMiddlewareServerSite) Server() core.Server {
	return nil
}

func (s testHeaderMiddlewareServerSite) RegisterMiddleware(middleware func(next http.Handler) http.Handler) error {
	if s.err {
		return errors.New("test error")
	}
	return nil
}

func (s testHeaderMiddlewareServerSite) RegisterHandler(handler http.Handler) error {
	if s.err {
		return errors.New("test error")
	}
	return nil
}

var _ core.ServerSite = (*testHeaderMiddlewareServerSite)(nil)

func TestHeaderMiddlewareModuleInfo(t *testing.T) {
	type fields struct {
		config  *headerMiddlewareConfig
		logger  *slog.Logger
		regexps []*regexp.Regexp
	}
	tests := []struct {
		name   string
		fields fields
		want   module.ModuleInfo
	}{
		{
			name: "default",
			want: module.ModuleInfo{
				ID:          headerModuleID,
				NewInstance: func() module.Module { return &headerMiddleware{} },
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := headerMiddleware{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				regexps: tt.fields.regexps,
			}
			got := m.ModuleInfo()
			if got.ID != tt.want.ID {
				t.Errorf("headerMiddleware.ModuleInfo() = %v, want %v", got.ID, tt.want.ID)
			}
			if instance := got.NewInstance(); instance == nil {
				t.Errorf("headerMiddleware.NewInstance() = %v, want %v", instance, "not nil")
			}
		})
	}
}

func TestHeaderMiddlewareInit(t *testing.T) {
	type fields struct {
		config  *headerMiddlewareConfig
		logger  *slog.Logger
		regexps []*regexp.Regexp
	}
	type args struct {
		config map[string]interface{}
		logger *slog.Logger
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "minimal",
			args: args{
				config: map[string]interface{}{},
				logger: slog.Default(),
			},
		},
		{
			name: "full",
			args: args{
				config: map[string]interface{}{
					"Rules": []map[string]interface{}{
						{
							"Path": "/.*",
							"Set": map[string]string{
								"test1": "value1",
							},
						},
					},
				},
				logger: slog.Default(),
			},
		},
		{
			name: "invalid values",
			args: args{
				config: map[string]interface{}{
					"Rules": []map[string]interface{}{
						{
							"Path": "",
							"Set": map[string]string{
								"": "",
							},
						},
					},
				},
				logger: slog.Default(),
			},
			wantErr: true,
		},
		{
			name: "invalid regular expression",
			args: args{
				config: map[string]interface{}{
					"Rules": []map[string]interface{}{
						{
							"Path": "(",
							"Set": map[string]string{
								"test": "value",
							},
						},
					},
				},
				logger: slog.Default(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &headerMiddleware{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				regexps: tt.fields.regexps,
			}
			if err := m.Init(tt.args.config, tt.args.logger); (err != nil) != tt.wantErr {
				t.Errorf("headerMiddleware.Init() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHeaderMiddlewareRegister(t *testing.T) {
	type fields struct {
		config  *headerMiddlewareConfig
		logger  *slog.Logger
		regexps []*regexp.Regexp
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
				site: testHeaderMiddlewareServerSite{},
			},
		},
		{
			name: "error register",
			args: args{
				site: testHeaderMiddlewareServerSite{
					err: true,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &headerMiddleware{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				regexps: tt.fields.regexps,
			}
			if err := m.Register(tt.args.site); (err != nil) != tt.wantErr {
				t.Errorf("headerMiddleware.Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHeaderMiddlewareStart(t *testing.T) {
	type fields struct {
		config  *headerMiddlewareConfig
		logger  *slog.Logger
		regexps []*regexp.Regexp
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
			m := &headerMiddleware{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				regexps: tt.fields.regexps,
			}
			if err := m.Start(); (err != nil) != tt.wantErr {
				t.Errorf("headerMiddleware.Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHeaderMiddlewareStop(t *testing.T) {
	type fields struct {
		config  *headerMiddlewareConfig
		logger  *slog.Logger
		regexps []*regexp.Regexp
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
			m := &headerMiddleware{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				regexps: tt.fields.regexps,
			}
			if err := m.Stop(); (err != nil) != tt.wantErr {
				t.Errorf("headerMiddleware.Stop() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHeaderMiddlewareHandler(t *testing.T) {
	type fields struct {
		config  *headerMiddlewareConfig
		logger  *slog.Logger
		regexps []*regexp.Regexp
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
			m := &headerMiddleware{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				regexps: tt.fields.regexps,
			}
			got := m.Handler(tt.args.next)
			if tt.wantNil && got != nil {
				t.Errorf("headerMiddleware.Handler() = %v, want %v", got, nil)
			}
		})
	}
}
