package js

import (
	"log/slog"
	"net/http"
	"reflect"
	"testing"
	"time"

	"rogchap.com/v8go"
)

func TestVMAPIServerHandler(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	isolate := v8go.NewIsolate()
	defer isolate.Dispose()
	context := v8go.NewContext(isolate)
	defer context.Close()

	req, err := http.NewRequest(http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Errorf("failed to create request: %s", err)
	}

	type args struct {
		name    string
		config  vmConfig
		source  string
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
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.handler.state(); })();`,
				timeout: time.Duration(1) * time.Second,
			},
			want: &vmResult{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := newVM()
			c := tt.args.config
			if err := v.Configure(&c, slog.Default()); err != nil {
				t.Errorf("failed to configure VM: %s", err)
			}
			got, err := v.Execute(tt.args.name, tt.args.source, tt.args.timeout)
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
	if testing.Short() {
		t.SkipNow()
	}

	isolate := v8go.NewIsolate()
	defer isolate.Dispose()
	context := v8go.NewContext(isolate)
	defer context.Close()

	req, err := http.NewRequest(http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Errorf("failed to create request: %s", err)
	}

	type args struct {
		name    string
		config  vmConfig
		source  string
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
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.request.method(); })();`,
				timeout: time.Duration(1) * time.Second,
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
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.request.proto(); })();`,
				timeout: time.Duration(1) * time.Second,
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
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.request.protoMajor(); })();`,
				timeout: time.Duration(1) * time.Second,
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
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.request.protoMinor(); })();`,
				timeout: time.Duration(1) * time.Second,
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
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.request.remoteAddr(); })();`,
				timeout: time.Duration(1) * time.Second,
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
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.request.host(); })();`,
				timeout: time.Duration(1) * time.Second,
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
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.request.path(); })();`,
				timeout: time.Duration(1) * time.Second,
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
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.request.query(); })();`,
				timeout: time.Duration(1) * time.Second,
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
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.request.headers(); })();`,
				timeout: time.Duration(1) * time.Second,
			},
			want: &vmResult{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := newVM()
			c := tt.args.config
			if err := v.Configure(&c, slog.Default()); err != nil {
				t.Errorf("failed to configure VM: %s", err)
			}
			got, err := v.Execute(tt.args.name, tt.args.source, tt.args.timeout)
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
	if testing.Short() {
		t.SkipNow()
	}

	isolate := v8go.NewIsolate()
	defer isolate.Dispose()
	context := v8go.NewContext(isolate)
	defer context.Close()

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
		source  string
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
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.response.render("test"); })();`,
				timeout: time.Duration(1) * time.Second,
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
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.response.render("test", 200); })();`,
				timeout: time.Duration(1) * time.Second,
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
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.response.render("test", 9999); })();`,
				timeout: time.Duration(1) * time.Second,
			},
			want: &vmResult{
				Render: bytePtr([]byte("test")),
				Status: intPtr(http.StatusInternalServerError),
			},
		},
		{
			name: "redirect without status code",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.response.redirect("http://test"); })();`,
				timeout: time.Duration(1) * time.Second,
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
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.response.redirect("http://test", 303); })();`,
				timeout: time.Duration(1) * time.Second,
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
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.response.redirect("http://test", 999); })();`,
				timeout: time.Duration(1) * time.Second,
			},
			want: &vmResult{
				Redirect:       boolPtr(true),
				RedirectURL:    stringPtr("http://test"),
				RedirectStatus: intPtr(http.StatusInternalServerError),
			},
		},
		{
			name: "set header",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.response.setHeader("key", "value"); })();`,
				timeout: time.Duration(1) * time.Second,
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
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.response.setTitle("test"); })();`,
				timeout: time.Duration(1) * time.Second,
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
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.response.setMeta("test", new Map([["k1", "v1"],["k2", "v2"],["k3", "v3"]])); })();`,
				timeout: time.Duration(1) * time.Second,
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
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.response.setLink("test", new Map([["k1", "v1"],["k2", "v2"],["k3", "v3"]])); })();`,
				timeout: time.Duration(1) * time.Second,
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
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.response.setScript("test", new Map([["k1", "v1"],["k2", "v2"],["k3", "v3"]])); })();`,
				timeout: time.Duration(1) * time.Second,
			},
			want: &vmResult{
				Scripts: scripts,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := newVM()
			c := tt.args.config
			if err := v.Configure(&c, slog.Default()); err != nil {
				t.Errorf("failed to configure VM: %s", err)
			}
			got, err := v.Execute(tt.args.name, tt.args.source, tt.args.timeout)
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
