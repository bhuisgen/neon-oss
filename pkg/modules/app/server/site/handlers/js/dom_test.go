package js

import (
	"reflect"
	"testing"
)

func TestNewDOMElement(t *testing.T) {
	type args struct {
		id string
	}
	tests := []struct {
		name    string
		args    args
		wantNil bool
	}{
		{
			name: "default",
			args: args{
				id: "test",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newDOMElement(tt.args.id)
			if (got == nil) != tt.wantNil {
				t.Errorf("newDOMElement() got = %v, wantNil %v", got, tt.wantNil)
			}
		})
	}
}

func TestDOMElementId(t *testing.T) {
	type fields struct {
		id string
		m  map[string]string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "default",
			fields: fields{
				id: "test",
				m:  map[string]string{},
			},
			want: "test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &domElement{
				id: tt.fields.id,
				m:  tt.fields.m,
			}
			if got := e.Id(); got != tt.want {
				t.Errorf("domElement.Id() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDOMElementAttributes(t *testing.T) {
	type fields struct {
		id string
		m  map[string]string
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "default",
			fields: fields{
				id: "test",
				m: map[string]string{
					"key1": "value1",
					"key2": "value2",
					"key3": "value3",
				},
			},
			want: []string{"key1", "key2", "key3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &domElement{
				id: tt.fields.id,
				m:  tt.fields.m,
			}
			got := e.Attributes()
			for _, e := range tt.want {
				var found bool
				for _, g := range got {
					if g == e {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("domElement.Attributes() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestDOMElementGetAttribute(t *testing.T) {
	type fields struct {
		id string
		m  map[string]string
	}
	type args struct {
		name string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "default",
			fields: fields{
				id: "test",
				m: map[string]string{
					"key": "value",
				},
			},
			args: args{
				name: "key",
			},
			want: "value",
		},
		{
			name: "invalid key",
			fields: fields{
				id: "test",
				m: map[string]string{
					"key": "value",
				},
			},
			args: args{
				name: "invalid key",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &domElement{
				id: tt.fields.id,
				m:  tt.fields.m,
			}
			if got := e.GetAttribute(tt.args.name); got != tt.want {
				t.Errorf("domElement.GetAttribute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDOMElementSetAttribute(t *testing.T) {
	type fields struct {
		id string
		m  map[string]string
	}
	type args struct {
		key   string
		value string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "default",
			fields: fields{
				id: "test",
				m: map[string]string{
					"key": "value",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &domElement{
				id: tt.fields.id,
				m:  tt.fields.m,
			}
			e.SetAttribute(tt.args.key, tt.args.value)
		})
	}
}

func TestNewDOMElementList(t *testing.T) {
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
			got := newDOMElementList()
			if (got == nil) != tt.wantNil {
				t.Errorf("newDOMElementList() got = %v, wantNil %v", got, tt.wantNil)
			}
		})
	}
}

func TestDOMElementListIds(t *testing.T) {
	type fields struct {
		ids []string
		m   map[string]*domElement
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "default",
			fields: fields{
				ids: []string{"test"},
				m:   map[string]*domElement{},
			},
			want: []string{"test"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &domElementList{
				ids: tt.fields.ids,
				m:   tt.fields.m,
			}
			if got := l.Ids(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("domElementList.Ids() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDOMElementListGet(t *testing.T) {
	test := newDOMElement("test")

	type fields struct {
		ids []string
		m   map[string]*domElement
	}
	type args struct {
		id string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *domElement
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				ids: []string{"test"},
				m: map[string]*domElement{
					"test": test,
				},
			},
			args: args{
				id: "test",
			},
			want: test,
		},
		{
			name: "invalid id",
			fields: fields{
				ids: []string{"test"},
				m: map[string]*domElement{
					"test": test,
				},
			},
			args: args{
				id: "invalid id",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &domElementList{
				ids: tt.fields.ids,
				m:   tt.fields.m,
			}
			got, err := l.Get(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("domElementList.Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("domElementList.Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDOMElementListSet(t *testing.T) {
	test := newDOMElement("test")

	type fields struct {
		ids []string
		m   map[string]*domElement
	}
	type args struct {
		e *domElement
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "default",
			fields: fields{
				ids: []string{},
				m:   map[string]*domElement{},
			},
			args: args{
				e: test,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &domElementList{
				ids: tt.fields.ids,
				m:   tt.fields.m,
			}
			l.Set(tt.args.e)
		})
	}
}
