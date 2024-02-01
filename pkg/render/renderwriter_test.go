package render

import (
	"bytes"
	"net/http"
	"reflect"
	"testing"
)

func TestNewRenderWriter(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "default",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			NewRenderWriter()
		})
	}
}

func TestRenderWriterReset(t *testing.T) {
	type fields struct {
		buf         *bytes.Buffer
		header      http.Header
		statusCode  int
		redirect    bool
		redirectURL string
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "default",
			fields: fields{
				buf:    new(bytes.Buffer),
				header: http.Header{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &renderWriter{
				buf:         tt.fields.buf,
				header:      tt.fields.header,
				statusCode:  tt.fields.statusCode,
				redirect:    tt.fields.redirect,
				redirectURL: tt.fields.redirectURL,
			}
			w.Reset()
		})
	}
}

func TestRenderWriterWrite(t *testing.T) {
	type fields struct {
		buf         *bytes.Buffer
		header      http.Header
		statusCode  int
		redirect    bool
		redirectURL string
	}
	type args struct {
		p []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				buf: new(bytes.Buffer),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &renderWriter{
				buf:         tt.fields.buf,
				header:      tt.fields.header,
				statusCode:  tt.fields.statusCode,
				redirect:    tt.fields.redirect,
				redirectURL: tt.fields.redirectURL,
			}
			got, err := w.Write(tt.args.p)
			if (err != nil) != tt.wantErr {
				t.Errorf("renderWriter.Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("renderWriter.Write() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRenderWriterWriteHeader(t *testing.T) {
	type fields struct {
		buf         *bytes.Buffer
		header      http.Header
		statusCode  int
		redirect    bool
		redirectURL string
	}
	type args struct {
		statusCode int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "default",
			args: args{
				statusCode: http.StatusOK,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &renderWriter{
				buf:         tt.fields.buf,
				header:      tt.fields.header,
				statusCode:  tt.fields.statusCode,
				redirect:    tt.fields.redirect,
				redirectURL: tt.fields.redirectURL,
			}
			w.WriteHeader(tt.args.statusCode)
		})
	}
}

func TestRenderWriterWriteRedirect(t *testing.T) {
	type fields struct {
		buf         *bytes.Buffer
		header      http.Header
		statusCode  int
		redirect    bool
		redirectURL string
	}
	type args struct {
		url        string
		statusCode int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "default",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &renderWriter{
				buf:         tt.fields.buf,
				header:      tt.fields.header,
				statusCode:  tt.fields.statusCode,
				redirect:    tt.fields.redirect,
				redirectURL: tt.fields.redirectURL,
			}
			w.WriteRedirect(tt.args.url, tt.args.statusCode)
		})
	}
}

func TestRenderWriterHeader(t *testing.T) {
	type fields struct {
		buf         *bytes.Buffer
		header      http.Header
		statusCode  int
		redirect    bool
		redirectURL string
	}
	tests := []struct {
		name   string
		fields fields
		want   http.Header
	}{
		{
			name: "default",
			fields: fields{
				header: http.Header{
					"test": []string{"test"},
				},
			},
			want: http.Header{
				"test": []string{"test"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &renderWriter{
				buf:         tt.fields.buf,
				header:      tt.fields.header,
				statusCode:  tt.fields.statusCode,
				redirect:    tt.fields.redirect,
				redirectURL: tt.fields.redirectURL,
			}
			if got := w.Header(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("renderWriter.Header() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRenderWriterStatusCode(t *testing.T) {
	type fields struct {
		buf         *bytes.Buffer
		header      http.Header
		statusCode  int
		redirect    bool
		redirectURL string
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name: "default",
			fields: fields{
				statusCode: http.StatusOK,
			},
			want: http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &renderWriter{
				buf:         tt.fields.buf,
				header:      tt.fields.header,
				statusCode:  tt.fields.statusCode,
				redirect:    tt.fields.redirect,
				redirectURL: tt.fields.redirectURL,
			}
			if got := w.StatusCode(); got != tt.want {
				t.Errorf("renderWriter.StatusCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRenderWriterRedirect(t *testing.T) {
	type fields struct {
		buf         *bytes.Buffer
		header      http.Header
		statusCode  int
		redirect    bool
		redirectURL string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "default",
			fields: fields{
				redirect: true,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &renderWriter{
				buf:         tt.fields.buf,
				header:      tt.fields.header,
				statusCode:  tt.fields.statusCode,
				redirect:    tt.fields.redirect,
				redirectURL: tt.fields.redirectURL,
			}
			if got := w.Redirect(); got != tt.want {
				t.Errorf("renderWriter.Redirect() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRenderWriterRedirectURL(t *testing.T) {
	type fields struct {
		buf         *bytes.Buffer
		header      http.Header
		statusCode  int
		redirect    bool
		redirectURL string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "default",
			fields: fields{
				redirectURL: "/redirect",
			},
			want: "/redirect",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &renderWriter{
				buf:         tt.fields.buf,
				header:      tt.fields.header,
				statusCode:  tt.fields.statusCode,
				redirect:    tt.fields.redirect,
				redirectURL: tt.fields.redirectURL,
			}
			if got := w.RedirectURL(); got != tt.want {
				t.Errorf("renderWriter.RedirectURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRenderWriterRender(t *testing.T) {
	type fields struct {
		buf         *bytes.Buffer
		header      http.Header
		statusCode  int
		redirect    bool
		redirectURL string
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "render",
			fields: fields{
				buf:    new(bytes.Buffer),
				header: http.Header{},
			},
		},
		{
			name: "redirect",
			fields: fields{
				buf:         new(bytes.Buffer),
				header:      http.Header{},
				redirect:    true,
				redirectURL: "/redirect",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &renderWriter{
				buf:         tt.fields.buf,
				header:      tt.fields.header,
				statusCode:  tt.fields.statusCode,
				redirect:    tt.fields.redirect,
				redirectURL: tt.fields.redirectURL,
			}
			w.Render()
		})
	}
}
