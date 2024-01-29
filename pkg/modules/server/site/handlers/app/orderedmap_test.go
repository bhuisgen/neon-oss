package app

import (
	"reflect"
	"sync"
	"testing"
)

func TestNewOrderedMap(t *testing.T) {
	tests := []struct {
		name    string
		wantNil bool
	}{
		{
			name: "default",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newOrderedMap()
			if (got == nil) != tt.wantNil {
				t.Errorf("NewOrderedMap() got = %v, wantNil %v", got, tt.wantNil)
			}
		})
	}
}

func TestOrderedMapKeys(t *testing.T) {
	type fields struct {
		keys []interface{}
		data map[interface{}]interface{}
	}
	tests := []struct {
		name   string
		fields fields
		want   []interface{}
	}{
		{
			name: "default",
			fields: fields{
				keys: []interface{}{"key3", "key2", "key1"},
				data: map[interface{}]interface{}{
					"key1": "value3",
					"key2": "value2",
					"key3": "value1",
				},
			},
			want: []interface{}{"key3", "key2", "key1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &orderedMap{
				keys: tt.fields.keys,
				data: tt.fields.data,
				mu:   sync.RWMutex{},
			}
			if got := m.Keys(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("orderedMap.Keys() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrderedMapGet(t *testing.T) {
	type fields struct {
		keys []interface{}
		data map[interface{}]interface{}
	}
	type args struct {
		key string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   interface{}
		want1  bool
	}{
		{
			name: "default",
			fields: fields{
				keys: []interface{}{"key"},
				data: map[interface{}]interface{}{
					"key": "value",
				},
			},
			args: args{
				key: "key",
			},
			want:  "value",
			want1: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &orderedMap{
				keys: tt.fields.keys,
				data: tt.fields.data,
				mu:   sync.RWMutex{},
			}
			got, got1 := m.Get(tt.args.key)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("orderedMap.Get() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("orderedMap.Get() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestOrderedMapSet(t *testing.T) {
	type fields struct {
		keys []interface{}
		data map[interface{}]interface{}
	}
	type args struct {
		key   string
		value interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "default",
			fields: fields{
				keys: []interface{}{},
				data: map[interface{}]interface{}{},
			},
			args: args{
				key:   "key",
				value: "value",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &orderedMap{
				keys: tt.fields.keys,
				data: tt.fields.data,
				mu:   sync.RWMutex{},
			}
			m.Set(tt.args.key, tt.args.value)
		})
	}
}

func TestOrderedMapRemove(t *testing.T) {
	type fields struct {
		keys []interface{}
		data map[interface{}]interface{}
	}
	type args struct {
		key string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "default",
			fields: fields{
				keys: []interface{}{"key"},
				data: map[interface{}]interface{}{
					"key": "value",
				},
			},
			args: args{
				key: "key",
			},
		},
		{
			name: "invalid key",
			fields: fields{
				keys: []interface{}{"key"},
				data: map[interface{}]interface{}{
					"key": "value",
				},
			},
			args: args{
				key: "invalid",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &orderedMap{
				keys: tt.fields.keys,
				data: tt.fields.data,
				mu:   sync.RWMutex{},
			}
			m.Remove(tt.args.key)
		})
	}
}

func TestOrderedMapClear(t *testing.T) {
	type fields struct {
		keys []interface{}
		data map[interface{}]interface{}
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "default",
			fields: fields{
				keys: []interface{}{"key"},
				data: map[interface{}]interface{}{
					"key": "value",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &orderedMap{
				keys: tt.fields.keys,
				data: tt.fields.data,
				mu:   sync.RWMutex{},
			}
			m.Clear()
		})
	}
}
