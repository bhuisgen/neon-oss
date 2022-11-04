// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"reflect"
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
			got := newDataMap()
			if (got == nil) != tt.wantNil {
				t.Errorf("NewOrderedMap() got = %v, wantNil %v", got, tt.wantNil)
			}
		})
	}
}

func TestOrderedMapGet(t *testing.T) {
	type fields struct {
		keys []string
		data map[string]interface{}
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
				keys: []string{"key"},
				data: map[string]interface{}{
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
			m := &dataMap{
				keys: tt.fields.keys,
				data: tt.fields.data,
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
		keys []string
		data map[string]interface{}
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
				keys: []string{},
				data: map[string]interface{}{},
			},
			args: args{
				key:   "key",
				value: "value",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &dataMap{
				keys: tt.fields.keys,
				data: tt.fields.data,
			}
			m.Set(tt.args.key, tt.args.value)
		})
	}
}

func TestOrderedMapDelete(t *testing.T) {
	type fields struct {
		keys []string
		data map[string]interface{}
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
				keys: []string{"key"},
				data: map[string]interface{}{
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
				keys: []string{"key"},
				data: map[string]interface{}{
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
			m := &dataMap{
				keys: tt.fields.keys,
				data: tt.fields.data,
			}
			m.Delete(tt.args.key)
		})
	}
}

func TestOrderedMapKeys(t *testing.T) {
	type fields struct {
		keys []string
		data map[string]interface{}
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "default",
			fields: fields{
				keys: []string{"key1", "key2", "key3"},
				data: map[string]interface{}{
					"key3": "value3",
					"key2": "value2",
					"key1": "value1",
				},
			},
			want: []string{"key1", "key2", "key3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &dataMap{
				keys: tt.fields.keys,
				data: tt.fields.data,
			}
			if got := m.Keys(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("orderedMap.Keys() = %v, want %v", got, tt.want)
			}
		})
	}
}
