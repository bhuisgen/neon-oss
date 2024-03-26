package js

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/bhuisgen/neon/pkg/core"
)

type testVMAPIServerSite struct {
	name      string
	listeners []string
	hosts     []string
	isDefault bool
}

func (t *testVMAPIServerSite) Name() string {
	return t.name
}

func (t *testVMAPIServerSite) Listeners() []string {
	return t.listeners
}

func (t *testVMAPIServerSite) Hosts() []string {
	return t.hosts
}

func (t *testVMAPIServerSite) IsDefault() bool {
	return t.isDefault
}

func (t *testVMAPIServerSite) Server() core.Server {
	return nil
}

func (t *testVMAPIServerSite) Store() core.Store {
	return nil
}

func (t *testVMAPIServerSite) RegisterMiddleware(middleware func(next http.Handler) http.Handler) error {
	return nil
}

func (t *testVMAPIServerSite) RegisterHandler(handler http.Handler) error {
	return nil
}

var _ core.ServerSite = (*testVMAPIServerSite)(nil)

func TestVMAPIServerSite(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Errorf("failed to create request: %s", err)
	}

	type args struct {
		name    string
		config  vmConfig
		code    []byte
		timeout time.Duration
	}
	tests := []struct {
		name    string
		args    args
		want    *vmResult
		wantErr bool
	}{
		{
			name: "name",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   bytePtr([]byte(`{}`)),
					Site: &testVMAPIServerSite{
						name: "test",
					},
				},
				code:    []byte(`(() => { if (server.site.name() !== "test") throw Error(); })();`),
				timeout: 4 * time.Second,
			},
			want: &vmResult{},
		},
		{
			name: "listeners",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   bytePtr([]byte(`{}`)),
					Site: &testVMAPIServerSite{
						listeners: []string{"listener1", "listener2"},
					},
				},
				code: []byte(`
(() => {
  if (JSON.stringify(server.site.listeners()) !== JSON.stringify(["listener1","listener2"])) {
    throw Error();
  }
})();
`),
				timeout: 4 * time.Second,
			},
			want: &vmResult{},
		},
		{
			name: "hosts",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   bytePtr([]byte(`{}`)),
					Site: &testVMAPIServerSite{
						hosts: []string{"host1", "host2"},
					},
				},
				code: []byte(`
(() => {
  if (JSON.stringify(server.site.hosts()) !== JSON.stringify(["host1","host2"])) {
    throw Error();
  }
})();
`),
				timeout: 4 * time.Second,
			},
			want: &vmResult{},
		},
		{
			name: "isDefault",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   bytePtr([]byte(`{}`)),
					Site: &testVMAPIServerSite{
						isDefault: true,
					},
				},
				code: []byte(`
(() => {
  if (server.site.isDefault() !== true) {
    throw Error();
  }
})();
`),
				timeout: 4 * time.Second,
			},
			want: &vmResult{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := newVM()
			if err != nil {
				t.Fatal()
			}
			got, err := v.Execute(tt.args.config, tt.args.name, tt.args.code, tt.args.timeout)
			if (err != nil) != tt.wantErr {
				t.Errorf("vm.Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("vm.Execute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVMAPIServerHandler(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Errorf("failed to create request: %s", err)
	}

	type args struct {
		name    string
		config  vmConfig
		code    []byte
		timeout time.Duration
	}
	tests := []struct {
		name    string
		args    args
		want    *vmResult
		wantErr bool
	}{
		{
			name: "state",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   bytePtr([]byte(`{"test":"value"}`)),
				},
				code:    []byte(`(() => { if (server.handler.state().test !== "value") throw Error(); })();`),
				timeout: 4 * time.Second,
			},
			want: &vmResult{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := newVM()
			if err != nil {
				t.Fatal()
			}
			got, err := v.Execute(tt.args.config, tt.args.name, tt.args.code, tt.args.timeout)
			if (err != nil) != tt.wantErr {
				t.Errorf("vm.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("vm.Execute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVMAPIServerRequest(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Errorf("failed to create request: %s", err)
	}

	type args struct {
		name    string
		config  vmConfig
		code    []byte
		timeout time.Duration
	}
	tests := []struct {
		name    string
		args    args
		want    *vmResult
		wantErr bool
	}{
		{
			name: "method",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   bytePtr([]byte(`{}`)),
				},
				code:    []byte(`(() => { server.request.method(); })();`),
				timeout: 4 * time.Second,
			},
			want: &vmResult{},
		},
		{
			name: "proto",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   bytePtr([]byte(`{}`)),
				},
				code:    []byte(`(() => { server.request.proto(); })();`),
				timeout: 4 * time.Second,
			},
			want: &vmResult{},
		},
		{
			name: "proto major",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   bytePtr([]byte(`{}`)),
				},
				code:    []byte(`(() => { server.request.protoMajor(); })();`),
				timeout: 4 * time.Second,
			},
			want: &vmResult{},
		},
		{
			name: "proto minor",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   bytePtr([]byte(`{}`)),
				},
				code:    []byte(`(() => { server.request.protoMinor(); })();`),
				timeout: 4 * time.Second,
			},
			want: &vmResult{},
		},
		{
			name: "remote addr",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   bytePtr([]byte(`{}`)),
				},
				code:    []byte(`(() => { server.request.remoteAddr(); })();`),
				timeout: 4 * time.Second,
			},
			want: &vmResult{},
		},
		{
			name: "host",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   bytePtr([]byte(`{}`)),
				},
				code:    []byte(`(() => { server.request.host(); })();`),
				timeout: 4 * time.Second,
			},
			want: &vmResult{},
		},
		{
			name: "path",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   bytePtr([]byte(`{}`)),
				},
				code:    []byte(`(() => { server.request.path(); })();`),
				timeout: 4 * time.Second,
			},
			want: &vmResult{},
		},
		{
			name: "query",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   bytePtr([]byte(`{}`)),
				},
				code:    []byte(`(() => { server.request.query(); })();`),
				timeout: 4 * time.Second,
			},
			want: &vmResult{},
		},
		{
			name: "header",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   bytePtr([]byte(`{}`)),
				},
				code:    []byte(`(() => { server.request.headers(); })();`),
				timeout: 4 * time.Second,
			},
			want: &vmResult{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := newVM()
			if err != nil {
				t.Fatal()
			}
			got, err := v.Execute(tt.args.config, tt.args.name, tt.args.code, tt.args.timeout)
			if (err != nil) != tt.wantErr {
				t.Errorf("vm.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("vm.Execute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVMAPIServerResponse(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Errorf("failed to create request: %s", err)
	}

	meta := newDOMElement("test")
	meta.SetAttribute("k1", "v1")
	meta.SetAttribute("k2", "v2")
	meta.SetAttribute("k3", "v3")
	metas := newDOMElementList()
	metas.Set(meta)

	link := newDOMElement("test")
	link.SetAttribute("k1", "v1")
	link.SetAttribute("k2", "v2")
	link.SetAttribute("k3", "v3")
	links := newDOMElementList()
	links.Set(link)

	script := newDOMElement("test")
	script.SetAttribute("k1", "v1")
	script.SetAttribute("k2", "v2")
	script.SetAttribute("k3", "v3")
	scripts := newDOMElementList()
	scripts.Set(script)

	type args struct {
		name    string
		config  vmConfig
		code    []byte
		timeout time.Duration
	}
	tests := []struct {
		name    string
		args    args
		want    *vmResult
		wantErr bool
	}{
		{
			name: "render without status code",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   bytePtr([]byte(`{}`)),
				},
				code:    []byte(`(() => { server.response.render("test"); })();`),
				timeout: 4 * time.Second,
			},
			want: &vmResult{
				Render: bytePtr([]byte("test")),
				Status: intPtr(http.StatusOK),
			},
		},
		{
			name: "render with status code",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   bytePtr([]byte(`{}`)),
				},
				code:    []byte(`(() => { server.response.render("test", 200); })();`),
				timeout: 4 * time.Second,
			},
			want: &vmResult{
				Render: bytePtr([]byte("test")),
				Status: intPtr(http.StatusOK),
			},
		},
		{
			name: "render with invalid status code",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   bytePtr([]byte(`{}`)),
				},
				code:    []byte(`(() => { server.response.render("test", 9999); })();`),
				timeout: 4 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "redirect without status code",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   bytePtr([]byte(`{}`)),
				},
				code:    []byte(`(() => { server.response.redirect("http://test"); })();`),
				timeout: 4 * time.Second,
			},
			want: &vmResult{
				Redirect:       boolPtr(true),
				RedirectURL:    stringPtr("http://test"),
				RedirectStatus: intPtr(http.StatusFound),
			},
		},
		{
			name: "redirect with status code",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   bytePtr([]byte(`{}`)),
				},
				code:    []byte(`(() => { server.response.redirect("http://test", 303); })();`),
				timeout: 4 * time.Second,
			},
			want: &vmResult{
				Redirect:       boolPtr(true),
				RedirectURL:    stringPtr("http://test"),
				RedirectStatus: intPtr(http.StatusSeeOther),
			},
		},
		{
			name: "redirect with invalid status code",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   bytePtr([]byte(`{}`)),
				},
				code:    []byte(`(() => { server.response.redirect("http://test", 999); })();`),
				timeout: 4 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "set header",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   bytePtr([]byte(`{}`)),
				},
				code:    []byte(`(() => { server.response.setHeader("key", "value"); })();`),
				timeout: 4 * time.Second,
			},
			want: &vmResult{
				Headers: map[string][]string{
					"key": {"value"},
				},
			},
		},
		{
			name: "set title",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   bytePtr([]byte(`{}`)),
				},
				code:    []byte(`(() => { server.response.setTitle("test"); })();`),
				timeout: 4 * time.Second,
			},
			want: &vmResult{
				Title: stringPtr("test"),
			},
		},
		{
			name: "set meta",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   bytePtr([]byte(`{}`)),
				},
				code:    []byte(`(() => { server.response.setMeta("test", new Map([["k1", "v1"],["k2", "v2"],["k3", "v3"]])); })();`),
				timeout: 4 * time.Second,
			},
			want: &vmResult{
				Metas: metas,
			},
		},
		{
			name: "set link",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   bytePtr([]byte(`{}`)),
				},
				code:    []byte(`(() => { server.response.setLink("test", new Map([["k1", "v1"],["k2", "v2"],["k3", "v3"]])); })();`),
				timeout: 4 * time.Second,
			},
			want: &vmResult{
				Links: links,
			},
		},
		{
			name: "set script",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   bytePtr([]byte(`{}`)),
				},
				code:    []byte(`(() => { server.response.setScript("test", new Map([["k1", "v1"],["k2", "v2"],["k3", "v3"]])); })();`),
				timeout: 4 * time.Second,
			},
			want: &vmResult{
				Scripts: scripts,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := newVM()
			if err != nil {
				t.Fatal()
			}
			got, err := v.Execute(tt.args.config, tt.args.name, tt.args.code, tt.args.timeout)
			if (err != nil) != tt.wantErr {
				t.Errorf("vm.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("vm.Execute() = %v, want %v", got, tt.want)
			}
		})
	}
}
