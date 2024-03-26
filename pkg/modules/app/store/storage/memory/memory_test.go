package memory

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
)

type testMemoryStorageStore struct {
	errLoadResource  bool
	errStoreResource bool
}

func (s testMemoryStorageStore) LoadResource(name string) (*core.Resource, error) {
	if s.errLoadResource {
		return nil, errors.New("test error")
	}
	return &core.Resource{
		Data: [][]byte{[]byte("test")},
		TTL:  0,
	}, nil
}

func (s testMemoryStorageStore) StoreResource(name string, resource *core.Resource) error {
	if s.errStoreResource {
		return errors.New("test error")
	}
	return nil
}

var _ core.Store = (*testMemoryStorageStore)(nil)

type testMemoryStorageCache struct {
	errGet bool
}

func (c testMemoryStorageCache) Get(key string) any {
	if c.errGet {
		return nil
	}

	return &core.Resource{
		Data: [][]byte{[]byte("test")},
		TTL:  0,
	}
}

func (c testMemoryStorageCache) Set(key string, value any) {
}

func (c testMemoryStorageCache) Remove(key string) {
}

func (c testMemoryStorageCache) Clear() {
}

var _ Cache = (*testMemoryStorageCache)(nil)

func TestMemoryStorageModuleInfo(t *testing.T) {
	type fields struct {
		logger  *slog.Logger
		storage Cache
	}
	tests := []struct {
		name   string
		fields fields
		want   module.ModuleInfo
	}{
		{
			name: "default",
			want: module.ModuleInfo{
				ID:          memoryModuleID,
				NewInstance: func() module.Module { return &memoryStorage{} },
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := memoryStorage{
				logger:  tt.fields.logger,
				storage: tt.fields.storage,
			}
			got := s.ModuleInfo()
			if got.ID != tt.want.ID {
				t.Errorf("memoryStorage.ModuleInfo() = %v, want %v", got.ID, tt.want.ID)
			}
			if instance := got.NewInstance(); instance == nil {
				t.Errorf("memoryStorage.NewInstance() = %v, want %v", instance, "not nil")
			}
		})
	}
}

func TestMemoryStorageInit(t *testing.T) {
	type fields struct {
		logger  *slog.Logger
		storage Cache
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
			name: "default",
			args: args{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &memoryStorage{
				logger:  tt.fields.logger,
				storage: tt.fields.storage,
			}
			if err := s.Init(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("memoryStorage.Init() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMemoryStorageLoadResource(t *testing.T) {
	type fields struct {
		logger  *slog.Logger
		storage Cache
	}
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *core.Resource
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				storage: testMemoryStorageCache{},
			},
			args: args{
				name: "test",
			},
		},
		{
			name: "invalid resource name",
			fields: fields{
				storage: testMemoryStorageCache{
					errGet: true,
				},
			},
			args: args{
				name: "invalid",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &memoryStorage{
				logger:  tt.fields.logger,
				storage: tt.fields.storage,
			}
			_, err := s.LoadResource(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("memoryStorage.LoadResource() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMemoryStorageStoreResource(t *testing.T) {
	type fields struct {
		logger  *slog.Logger
		storage Cache
	}
	type args struct {
		name     string
		resource *core.Resource
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
				storage: testMemoryStorageCache{},
			},
			args: args{
				name: "test",
				resource: &core.Resource{
					Data: [][]byte{[]byte("test")},
					TTL:  0,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &memoryStorage{
				logger:  tt.fields.logger,
				storage: tt.fields.storage,
			}
			if err := s.StoreResource(tt.args.name, tt.args.resource); (err != nil) != tt.wantErr {
				t.Errorf("memoryStorage.StoreResource() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
