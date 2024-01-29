package raw

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/bhuisgen/neon/pkg/core"
)

type testRawParserStore struct {
	errGet bool
	errSet bool
}

func (s *testRawParserStore) LoadResource(name string) (*core.Resource, error) {
	if s.errGet {
		return nil, errors.New("test error")
	}
	return nil, nil
}

func (s *testRawParserStore) StoreResource(name string, resource *core.Resource) error {
	if s.errSet {
		return errors.New("test error")
	}
	return nil
}

var _ core.Store = (*testRawParserStore)(nil)

type testRawParserFetcher struct {
	errFetch bool
}

func (f *testRawParserFetcher) Fetch(ctx context.Context, name string, provider string, config map[string]interface{}) (
	*core.Resource, error) {
	if f.errFetch {
		return nil, errors.New("test error")
	}
	return &core.Resource{}, nil
}

var _ core.Fetcher = (*testRawParserFetcher)(nil)

func TestRawParserInit(t *testing.T) {
	type fields struct {
		config *rawParserConfig
		logger *slog.Logger
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
				config: map[string]interface{}{
					"resource": map[string]map[string]interface{}{
						"name": {
							"provider": map[string]interface{}{},
						},
					},
				},
				logger: slog.Default(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &rawParser{
				config: tt.fields.config,
				logger: tt.fields.logger,
			}
			if err := p.Init(tt.args.config, tt.args.logger); (err != nil) != tt.wantErr {
				t.Errorf("rawParser.Init() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRawParserParse(t *testing.T) {
	type fields struct {
		config *rawParserConfig
		logger *slog.Logger
	}
	type args struct {
		ctx     context.Context
		store   core.Store
		fetcher core.Fetcher
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				config: &rawParserConfig{
					Resource: map[string]map[string]interface{}{
						"test": {
							"provider": map[string]interface{}{},
						},
					},
				},
				logger: slog.Default(),
			},
			args: args{
				ctx:     context.Background(),
				store:   &testRawParserStore{},
				fetcher: &testRawParserFetcher{},
			},
		},
		{
			name: "invalid resource name",
			fields: fields{
				config: &rawParserConfig{
					Resource: map[string]map[string]interface{}{
						"": {
							"provider": map[string]interface{}{},
						},
					},
				},
				logger: slog.Default(),
			},
			args: args{
				ctx:     context.Background(),
				store:   &testRawParserStore{},
				fetcher: &testRawParserFetcher{},
			},
			wantErr: true,
		},
		{
			name: "invalid provider name",
			fields: fields{
				config: &rawParserConfig{
					Resource: map[string]map[string]interface{}{
						"name": {
							"": map[string]interface{}{},
						},
					},
				},
				logger: slog.Default(),
			},
			args: args{
				ctx:     context.Background(),
				store:   &testRawParserStore{},
				fetcher: &testRawParserFetcher{},
			},
			wantErr: true,
		},
		{
			name: "error fetch",
			fields: fields{
				config: &rawParserConfig{
					Resource: map[string]map[string]interface{}{
						"test": {
							"provider": map[string]interface{}{},
						},
					},
				},
				logger: slog.Default(),
			},
			args: args{
				ctx:   context.Background(),
				store: &testRawParserStore{},
				fetcher: &testRawParserFetcher{
					errFetch: true,
				},
			},
			wantErr: true,
		},
		{
			name: "error store",
			fields: fields{
				config: &rawParserConfig{
					Resource: map[string]map[string]interface{}{
						"test": {
							"provider": map[string]interface{}{},
						},
					},
				},
				logger: slog.Default(),
			},
			args: args{
				ctx: context.Background(),
				store: &testRawParserStore{
					errSet: true},
				fetcher: &testRawParserFetcher{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &rawParser{
				config: tt.fields.config,
				logger: tt.fields.logger,
			}
			if err := p.Parse(tt.args.ctx, tt.args.store, tt.args.fetcher); (err != nil) != tt.wantErr {
				t.Errorf("rawParser.Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
