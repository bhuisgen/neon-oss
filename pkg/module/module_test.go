package module

import (
	"reflect"
	"testing"
)

type testModule struct{}

func (m testModule) ModuleInfo() ModuleInfo {
	return ModuleInfo{
		ID: "test",
		NewInstance: func() Module {
			return new(testModule)
		},
	}
}

var _ Module = (*testModule)(nil)

func TestRegisterModule(t *testing.T) {
	type args struct {
		module Module
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "default",
			args: args{
				module: testModule{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Register(tt.args.module)
		})
	}
}

func TestGetModule(t *testing.T) {
	modulesLock.Lock()
	modules = map[ModuleID]ModuleInfo{
		"test": {ID: "test"},
	}
	modulesLock.Unlock()

	type args struct {
		id ModuleID
	}
	tests := []struct {
		name    string
		args    args
		want    ModuleInfo
		wantErr bool
	}{
		{
			name: "default",
			args: args{
				id: "test",
			},
			want: ModuleInfo{
				ID: "test",
			},
		},
		{
			name: "error unknown module",
			args: args{
				id: "unknown",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Lookup(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetModule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetModule() = %v, want %v", got, tt.want)
			}
		})
	}
}
