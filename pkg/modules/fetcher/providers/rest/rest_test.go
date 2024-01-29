// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rest

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/bhuisgen/neon/pkg/core"
)

type testRestProviderFileInfo struct {
	name     string
	size     int64
	fileMode os.FileMode
	modTime  time.Time
	isDir    bool
	sys      any
}

func (fi testRestProviderFileInfo) Name() string {
	return fi.name
}

func (fi testRestProviderFileInfo) Size() int64 {
	return fi.size
}

func (fi testRestProviderFileInfo) Mode() os.FileMode {
	return fi.fileMode
}
func (fi testRestProviderFileInfo) ModTime() time.Time {
	return fi.modTime
}

func (fi testRestProviderFileInfo) IsDir() bool {
	return fi.isDir
}

func (fi testRestProviderFileInfo) Sys() any {
	return fi.sys
}

var _ os.FileInfo = (*testRestProviderFileInfo)(nil)

func TestRestProviderInit(t *testing.T) {
	type fields struct {
		config                         *restProviderConfig
		logger                         *slog.Logger
		client                         http.Client
		osOpenFile                     func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osReadFile                     func(name string) ([]byte, error)
		osClose                        func(f *os.File) error
		osStat                         func(name string) (fs.FileInfo, error)
		x509CertPoolAppendCertsFromPEM func(pool *x509.CertPool, pemCerts []byte) bool
		tlsLoadX509KeyPair             func(certFile, keyFile string) (tls.Certificate, error)
		httpNewRequestWithContext      func(ctx context.Context, method string, url string, body io.Reader) (*http.Request, error)
		httpClientDo                   func(client *http.Client, req *http.Request) (*http.Response, error)
		ioReadAll                      func(r io.Reader) ([]byte, error)
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
			fields: fields{
				osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
					return nil, nil
				},
				osClose: func(f *os.File) error {
					return nil
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return testRestProviderFileInfo{}, nil
				},
			},
			args: args{
				config: map[string]interface{}{},
				logger: slog.Default(),
			},
		},
		{
			name: "full",
			fields: fields{
				osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
					return nil, nil
				},
				osClose: func(f *os.File) error {
					return nil
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return testRestProviderFileInfo{}, nil
				},
				osReadFile: func(name string) ([]byte, error) {
					return []byte{}, nil
				},
				x509CertPoolAppendCertsFromPEM: func(pool *x509.CertPool, pemCerts []byte) bool {
					return true
				},
				tlsLoadX509KeyPair: func(certFile, keyFile string) (tls.Certificate, error) {
					return tls.Certificate{}, nil
				},
			},
			args: args{
				config: map[string]interface{}{
					"TLSCAFiles":          []string{"ca.pem"},
					"TLSCertFiles":        []string{"cert.pem"},
					"TLSKeyFiles":         []string{"key.pem"},
					"Timeout":             15,
					"MaxConnsPerHost":     100,
					"MaxIdleConns":        100,
					"MaxIdleConnsPerHost": 4,
					"IdleConnTimeout":     60,
					"Retry":               3,
					"RetryDelay":          1,
					"Headers:": map[string]string{
						"header": "value",
					},
					"Params": map[string]string{
						"header": "value",
					},
				},
			},
		},
		{
			name: "invalid values",
			fields: fields{
				osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
					return nil, nil
				},
				osClose: func(f *os.File) error {
					return nil
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return testRestProviderFileInfo{}, nil
				},
			},
			args: args{
				config: map[string]interface{}{
					"TLSCAFiles":          []string{""},
					"TLSCertFiles":        []string{""},
					"TLSKeyFiles":         []string{""},
					"Timeout":             -1,
					"MaxConnsPerHost":     -1,
					"MaxIdleConns":        -1,
					"MaxIdleConnsPerHost": -1,
					"IdleConnTimeout":     -1,
					"Retry":               -1,
					"RetryDelay":          -1,
					"Headers": map[string]string{
						"": "",
					},
					"params": map[string]string{
						"": "",
					},
				},
				logger: slog.Default(),
			},
			wantErr: true,
		},
		{
			name: "error open file",
			fields: fields{
				osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
					return nil, errors.New("test error")
				},
				osClose: func(f *os.File) error {
					return nil
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return testRestProviderFileInfo{}, nil
				},
			},
			args: args{
				config: map[string]interface{}{
					"TLSCAFiles":   []string{"ca.pem"},
					"TLSCertFiles": []string{"cert.pem"},
					"TLSKeyFiles":  []string{"key.pem"},
				},
				logger: slog.Default(),
			},
			wantErr: true,
		},
		{
			name: "error stat file",
			fields: fields{
				osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
					return nil, nil
				},
				osClose: func(f *os.File) error {
					return nil
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return nil, errors.New("test error")
				},
			},
			args: args{
				config: map[string]interface{}{
					"TLSCAFiles":   []string{"ca.pem"},
					"TLSCertFiles": []string{"cert.pem"},
					"TLSKeyFiles":  []string{"key.pem"},
				},
				logger: slog.Default(),
			},
			wantErr: true,
		},
		{
			name: "error file is directory",
			fields: fields{
				osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
					return nil, nil
				},
				osClose: func(f *os.File) error {
					return nil
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return testRestProviderFileInfo{
						isDir: true,
					}, nil
				},
			},
			args: args{
				config: map[string]interface{}{
					"TLSCAFiles":   []string{"ca.pem"},
					"TLSCertFiles": []string{"cert.pem"},
					"TLSKeyFiles":  []string{"key.pem"},
				},
				logger: slog.Default(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &restProvider{
				config:                         tt.fields.config,
				logger:                         tt.fields.logger,
				client:                         tt.fields.client,
				osOpenFile:                     tt.fields.osOpenFile,
				osReadFile:                     tt.fields.osReadFile,
				osClose:                        tt.fields.osClose,
				osStat:                         tt.fields.osStat,
				x509CertPoolAppendCertsFromPEM: tt.fields.x509CertPoolAppendCertsFromPEM,
				tlsLoadX509KeyPair:             tt.fields.tlsLoadX509KeyPair,
				httpNewRequestWithContext:      tt.fields.httpNewRequestWithContext,
				httpClientDo:                   tt.fields.httpClientDo,
				ioReadAll:                      tt.fields.ioReadAll,
			}
			if err := p.Init(tt.args.config, tt.args.logger); (err != nil) != tt.wantErr {
				t.Errorf("restProvider.Init() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestRestProviderFetch(t *testing.T) {
	type fields struct {
		config                         *restProviderConfig
		logger                         *slog.Logger
		client                         http.Client
		osOpenFile                     func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osReadFile                     func(name string) ([]byte, error)
		osClose                        func(f *os.File) error
		osStat                         func(name string) (fs.FileInfo, error)
		x509CertPoolAppendCertsFromPEM func(pool *x509.CertPool, pemCerts []byte) bool
		tlsLoadX509KeyPair             func(certFile, keyFile string) (tls.Certificate, error)
		httpNewRequestWithContext      func(ctx context.Context, method string, url string, body io.Reader) (*http.Request, error)
		httpClientDo                   func(client *http.Client, req *http.Request) (*http.Response, error)
		ioReadAll                      func(r io.Reader) ([]byte, error)
	}
	type args struct {
		ctx    context.Context
		name   string
		config map[string]interface{}
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
				config: &restProviderConfig{
					Headers: map[string]string{
						"global": "value",
					},
					Params: map[string]string{
						"global": "value",
					},
				},
				logger:                    slog.Default(),
				httpNewRequestWithContext: restHttpNewRequestWithContext,
				httpClientDo: func(client *http.Client, req *http.Request) (*http.Response, error) {
					return &http.Response{
						Body:       http.NoBody,
						StatusCode: http.StatusOK,
					}, nil
				},
				ioReadAll: func(r io.Reader) ([]byte, error) {
					return nil, nil
				},
			},
			args: args{
				ctx:  context.Background(),
				name: "test",
			},
		},
		{
			name: "error http create request",
			fields: fields{
				config: &restProviderConfig{},
				logger: slog.Default(),
				httpNewRequestWithContext: func(ctx context.Context, method, url string, body io.Reader) (*http.Request,
					error) {
					return nil, errors.New("test error")
				},
			},
			args: args{
				ctx:  context.Background(),
				name: "test",
			},
			wantErr: true,
		},
		{
			name: "error http client do",
			fields: fields{
				config: &restProviderConfig{},
				logger: slog.Default(),
				httpNewRequestWithContext: func(ctx context.Context, method, u string, body io.Reader) (*http.Request,
					error) {
					return &http.Request{
						Method: http.MethodGet,
						URL:    &url.URL{},
					}, nil
				},
				httpClientDo: func(client *http.Client, req *http.Request) (*http.Response, error) {
					return nil, errors.New("test error")
				},
			},
			args: args{
				ctx:  context.Background(),
				name: "test",
			},
			wantErr: true,
		},
		{
			name: "error io read all",
			fields: fields{
				config: &restProviderConfig{},
				logger: slog.Default(),
				httpNewRequestWithContext: func(ctx context.Context, method, u string, body io.Reader) (*http.Request,
					error) {
					return &http.Request{
						Method: http.MethodGet,
						URL:    &url.URL{},
					}, nil
				},
				httpClientDo: func(client *http.Client, req *http.Request) (*http.Response, error) {
					return &http.Response{
						Body: http.NoBody,
					}, nil
				},
				ioReadAll: func(r io.Reader) ([]byte, error) {
					return nil, errors.New("test error")
				},
			},
			args: args{
				ctx:  context.Background(),
				name: "test",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &restProvider{
				config:                         tt.fields.config,
				logger:                         tt.fields.logger,
				client:                         tt.fields.client,
				osOpenFile:                     tt.fields.osOpenFile,
				osReadFile:                     tt.fields.osReadFile,
				osClose:                        tt.fields.osClose,
				osStat:                         tt.fields.osStat,
				x509CertPoolAppendCertsFromPEM: tt.fields.x509CertPoolAppendCertsFromPEM,
				tlsLoadX509KeyPair:             tt.fields.tlsLoadX509KeyPair,
				httpNewRequestWithContext:      tt.fields.httpNewRequestWithContext,
				httpClientDo:                   tt.fields.httpClientDo,
				ioReadAll:                      tt.fields.ioReadAll,
			}
			_, err := p.Fetch(tt.args.ctx, tt.args.name, tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("restProvider.Fetch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestParseLinkNextFromHeader(t *testing.T) {
	type args struct {
		headers http.Header
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "default",
			args: args{
				headers: http.Header{
					"Link": []string{"<http://test?limit=10>, <http://test?offset=10&limit=10>; rel=\"next\""},
				},
			},
			want: "http://test?offset=10&limit=10",
		},
		{
			name: "multiple link headers",
			args: args{
				headers: http.Header{
					"Link": []string{"<http://test?limit=10>", "<http://test?offset=10&limit=10>; rel=\"next\""},
				},
			},
			want: "http://test?offset=10&limit=10",
		},
		{
			name: "empty link if missing header",
			args: args{
				headers: http.Header{},
			},
			want: "",
		},
		{
			name: "empty link if missing next",
			args: args{
				headers: http.Header{
					"Link": []string{"<http://test?limit=10>", "<http://test?offset=10&limit=10>; rel=\"prev\""},
				},
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseLinkNextFromHeader(tt.args.headers); got != tt.want {
				t.Errorf("parseLinkNextFromHeader() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseLinkNextFromBody(t *testing.T) {
	type args struct {
		body   []byte
		filter string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "default",
			args: args{
				body:   []byte("{\"links\":{\"next\":\"http://test?offset=10&limit=10\"}}"),
				filter: "$.links.next",
			},
			want: "http://test?offset=10&limit=10",
		},
		{
			name: "invalid json",
			args: args{
				body:   []byte(""),
				filter: "$.links.next",
			},
			want: "",
		},
		{
			name: "missing link",
			args: args{
				body:   []byte("{}"),
				filter: "$.links.next",
			},
			want: "",
		},
		{
			name: "link is null",
			args: args{
				body:   []byte("{\"links\":{\"next\":null}}"),
				filter: "$.links.next",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseLinkNextFromBody(tt.args.body, tt.args.filter); got != tt.want {
				t.Errorf("parseLinkNextFromBody() = %v, want %v", got, tt.want)
			}
		})
	}
}
