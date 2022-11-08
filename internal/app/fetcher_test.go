// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestCreateFetcher(t *testing.T) {
	type args struct {
		config *FetcherConfig
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "default",
			args: args{
				config: &FetcherConfig{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CreateFetcher(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateFetcher() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestCreateFetcherWithRequester(t *testing.T) {
	type args struct {
		config         *FetcherConfig
		fetchRequester FetchRequester
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "default",
			args: args{
				config: &FetcherConfig{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := createFetcherWithRequester(tt.args.config, tt.args.fetchRequester)
			if (err != nil) != tt.wantErr {
				t.Errorf("createFetcherWithRequester() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestFetcherInitialize(t *testing.T) {
	tlsCAFile := "ca.pem"
	tlsCertFile := "cert.pem"
	tlsKeyFile := "key.pem"

	type fields struct {
		config                         *FetcherConfig
		logger                         *log.Logger
		requester                      FetchRequester
		resources                      map[string]Resource
		data                           Cache
		osReadFile                     func(name string) ([]byte, error)
		x509CertPoolAppendCertsFromPEM func(pool *x509.CertPool, pemCerts []byte) bool
		tlsLoadX509KeyPair             func(certFile, keyFile string) (tls.Certificate, error)
	}
	type args struct {
		fetchRequester FetchRequester
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
				config: &FetcherConfig{
					Resources: []FetcherResource{
						{
							Name:   "test",
							Method: http.MethodGet,
							URL:    "http://localhost",
						},
					},
				},
				resources:                      make(map[string]Resource),
				osReadFile:                     fetcherOsReadFile,
				x509CertPoolAppendCertsFromPEM: fetcherX509CertPoolAppendCertsFromPEM,
				tlsLoadX509KeyPair:             fetcherTlsLoadX509KeyPair,
			},
		},
		{
			name: "tls",
			fields: fields{
				config: &FetcherConfig{
					RequestTLSCAFile:   &tlsCAFile,
					RequestTLSCertFile: &tlsCertFile,
					RequestTLSKeyFile:  &tlsKeyFile,
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
		},
		{
			name: "tls error read file",
			fields: fields{
				config: &FetcherConfig{
					RequestTLSCAFile:   &tlsCAFile,
					RequestTLSCertFile: &tlsCertFile,
					RequestTLSKeyFile:  &tlsKeyFile,
				},
				osReadFile: func(name string) ([]byte, error) {
					return []byte{}, errors.New("test error")
				},
				x509CertPoolAppendCertsFromPEM: func(pool *x509.CertPool, pemCerts []byte) bool {
					return true
				},
				tlsLoadX509KeyPair: func(certFile, keyFile string) (tls.Certificate, error) {
					return tls.Certificate{}, nil
				},
			},
			wantErr: true,
		},
		{
			name: "tls error append cert",
			fields: fields{
				config: &FetcherConfig{
					RequestTLSCAFile:   &tlsCAFile,
					RequestTLSCertFile: &tlsCertFile,
					RequestTLSKeyFile:  &tlsKeyFile,
				},
				osReadFile: func(name string) ([]byte, error) {
					return []byte{}, nil
				},
				x509CertPoolAppendCertsFromPEM: func(pool *x509.CertPool, pemCerts []byte) bool {
					return false
				},
				tlsLoadX509KeyPair: func(certFile, keyFile string) (tls.Certificate, error) {
					return tls.Certificate{}, nil
				},
			},
			wantErr: true,
		},
		{
			name: "tls error load keypair",
			fields: fields{
				config: &FetcherConfig{
					RequestTLSCAFile:   &tlsCAFile,
					RequestTLSCertFile: &tlsCertFile,
					RequestTLSKeyFile:  &tlsKeyFile,
				},
				osReadFile: func(name string) ([]byte, error) {
					return []byte{}, nil
				},
				x509CertPoolAppendCertsFromPEM: func(pool *x509.CertPool, pemCerts []byte) bool {
					return true
				},
				tlsLoadX509KeyPair: func(certFile, keyFile string) (tls.Certificate, error) {
					return tls.Certificate{}, errors.New("test error")
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fetcher{
				config:                         tt.fields.config,
				logger:                         tt.fields.logger,
				requester:                      tt.fields.requester,
				resources:                      tt.fields.resources,
				resourcesLock:                  sync.RWMutex{},
				data:                           tt.fields.data,
				osReadFile:                     tt.fields.osReadFile,
				x509CertPoolAppendCertsFromPEM: tt.fields.x509CertPoolAppendCertsFromPEM,
				tlsLoadX509KeyPair:             tt.fields.tlsLoadX509KeyPair,
			}
			if err := f.initialize(tt.args.fetchRequester); (err != nil) != tt.wantErr {
				t.Errorf("fetcher.initialize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type testFetcherFetchRequester struct {
	err      bool
	response []byte
}

func (r *testFetcherFetchRequester) Fetch(ctx context.Context, method string, url string, params map[string]string,
	headers map[string]string) ([]byte, error) {
	if r.err {
		return nil, errors.New("test error")
	}
	return r.response, nil
}

func TestFetcherFetch(t *testing.T) {
	type fields struct {
		config    *FetcherConfig
		logger    *log.Logger
		requester FetchRequester
		resources map[string]Resource
		data      Cache
	}
	type args struct {
		ctx  context.Context
		name string
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
				config:    &FetcherConfig{},
				logger:    log.Default(),
				requester: &testFetcherFetchRequester{},
				resources: map[string]Resource{
					"test": {
						Name:   "test",
						Method: http.MethodGet,
						URL:    "http://localhost",
						TTL:    0,
					},
				},
				data: newCache(),
			},
			args: args{
				ctx:  context.Background(),
				name: "test",
			},
		},
		{
			name: "resource not found",
			fields: fields{
				config:    &FetcherConfig{},
				logger:    log.Default(),
				requester: &testFetcherFetchRequester{},
				resources: map[string]Resource{},
				data:      newCache(),
			},
			args: args{
				ctx:  context.Background(),
				name: "test",
			},
			wantErr: true,
		},
		{
			name: "fetch error",
			fields: fields{
				config: &FetcherConfig{},
				logger: log.Default(),
				requester: &testFetcherFetchRequester{
					err: true,
				},
				resources: map[string]Resource{
					"test": {
						Name:   "test",
						Method: http.MethodGet,
						URL:    "http://localhost",
						TTL:    0,
					},
				},
				data: newCache(),
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
			f := &fetcher{
				config:        tt.fields.config,
				logger:        tt.fields.logger,
				requester:     tt.fields.requester,
				resources:     tt.fields.resources,
				resourcesLock: sync.RWMutex{},
				data:          tt.fields.data,
			}
			if err := f.Fetch(tt.args.ctx, tt.args.name); (err != nil) != tt.wantErr {
				t.Errorf("fetcher.Fetch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFetcherExists(t *testing.T) {
	type fields struct {
		config    *FetcherConfig
		logger    *log.Logger
		requester FetchRequester
		resources map[string]Resource
		data      Cache
	}
	type args struct {
		name string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "default",
			fields: fields{
				config: &FetcherConfig{},
				logger: log.Default(),
				requester: &testFetcherFetchRequester{
					err: true,
				},
				resources: map[string]Resource{
					"test": {
						Name:   "test",
						Method: http.MethodGet,
						URL:    "http://localhost",
						TTL:    0,
					},
				},
				data: newCache(),
			},
			args: args{
				name: "test",
			},
			want: true,
		},
		{
			name: "resource not found",
			fields: fields{
				config: &FetcherConfig{},
				logger: log.Default(),
				requester: &testFetcherFetchRequester{
					err: true,
				},
				resources: map[string]Resource{},
				data:      newCache(),
			},
			args: args{
				name: "test",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fetcher{
				config:        tt.fields.config,
				logger:        tt.fields.logger,
				requester:     tt.fields.requester,
				resources:     tt.fields.resources,
				resourcesLock: sync.RWMutex{},
				data:          tt.fields.data,
			}
			if got := f.Exists(tt.args.name); got != tt.want {
				t.Errorf("fetcher.Exists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFetcherGet(t *testing.T) {
	type fields struct {
		config    *FetcherConfig
		logger    *log.Logger
		requester FetchRequester
		resources map[string]Resource
		data      Cache
	}
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				config: &FetcherConfig{},
				logger: log.Default(),
				requester: &testFetcherFetchRequester{
					response: []byte("test"),
				},
				resources: map[string]Resource{
					"test": {
						Name:   "test",
						Method: http.MethodGet,
						URL:    "http://localhost",
						TTL:    0,
					},
				},
				data: &cache{
					objects: map[string]cacheObject{
						"test": {
							Value: []byte("test"),
						},
					},
					lock: sync.RWMutex{},
				},
			},
			args: args{
				name: "test",
			},
			want: []byte("test"),
		},
		{
			name: "resource not found",
			fields: fields{
				config: &FetcherConfig{},
				logger: log.Default(),
				requester: &testFetcherFetchRequester{
					err: true,
				},
				resources: map[string]Resource{},

				data: &cache{
					objects: map[string]cacheObject{},
					lock:    sync.RWMutex{},
				},
			},
			args: args{
				name: "test",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fetcher{
				config:        tt.fields.config,
				logger:        tt.fields.logger,
				requester:     tt.fields.requester,
				resources:     tt.fields.resources,
				resourcesLock: sync.RWMutex{},
				data:          tt.fields.data,
			}
			got, err := f.Get(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("fetcher.Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fetcher.Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFetcherRegister(t *testing.T) {
	type fields struct {
		config    *FetcherConfig
		logger    *log.Logger
		requester FetchRequester
		resources map[string]Resource
		data      Cache
	}
	type args struct {
		r Resource
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "default",
			fields: fields{
				config: &FetcherConfig{},
				logger: log.Default(),
				requester: &testFetcherFetchRequester{
					response: []byte("test"),
				},
				resources: map[string]Resource{},
				data: &cache{
					objects: map[string]cacheObject{},
					lock:    sync.RWMutex{},
				},
			},
			args: args{
				r: Resource{
					Name:   "test",
					Method: http.MethodGet,
					URL:    "http://localhost",
					TTL:    0,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fetcher{
				config:        tt.fields.config,
				logger:        tt.fields.logger,
				requester:     tt.fields.requester,
				resources:     tt.fields.resources,
				resourcesLock: sync.RWMutex{},
				data:          tt.fields.data,
			}
			f.Register(tt.args.r)
		})
	}
}

func TestFetcherUnregister(t *testing.T) {
	type fields struct {
		config    *FetcherConfig
		logger    *log.Logger
		requester FetchRequester
		resources map[string]Resource
		data      Cache
	}
	type args struct {
		name string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "default",
			fields: fields{
				config: &FetcherConfig{},
				logger: log.Default(),
				requester: &testFetcherFetchRequester{
					response: []byte("test"),
				},
				resources: map[string]Resource{},
				data: &cache{
					objects: map[string]cacheObject{},
					lock:    sync.RWMutex{},
				},
			},
			args: args{
				name: "test",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fetcher{
				config:        tt.fields.config,
				logger:        tt.fields.logger,
				requester:     tt.fields.requester,
				resources:     tt.fields.resources,
				resourcesLock: sync.RWMutex{},
				data:          tt.fields.data,
			}
			f.Unregister(tt.args.name)
		})
	}
}

func TestFetcherCreateResourceFromTemplate(t *testing.T) {
	type fields struct {
		config    *FetcherConfig
		logger    *log.Logger
		requester FetchRequester
		resources map[string]Resource
		data      Cache
	}
	type args struct {
		template string
		resource string
		params   map[string]string
		headers  map[string]string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *Resource
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				config: &FetcherConfig{
					Templates: []FetcherTemplate{
						{
							Name:   "template1",
							Method: http.MethodGet,
							URL:    "http://localhost",
							Params: map[string]string{
								"globalParam1": "value1",
							},
							Headers: map[string]string{
								"globalHeader1": "value1",
							},
						},
						{
							Name:   "template2",
							Method: http.MethodGet,
							URL:    "http://localhost",
							Params: map[string]string{
								"globalParam2": "value2",
							},
							Headers: map[string]string{
								"globalHeader2": "value2",
							},
						},
					},
				},
				logger:    log.Default(),
				requester: &testFetcherFetchRequester{},
				resources: map[string]Resource{},
				data: &cache{
					objects: map[string]cacheObject{},
					lock:    sync.RWMutex{},
				},
			},
			args: args{
				template: "template2",
				resource: "resource",
				params: map[string]string{
					"param1": "value1",
				},
				headers: map[string]string{
					"header1": "value1",
				},
			},
			want: &Resource{
				Name:   "resource",
				Method: http.MethodGet,
				URL:    "http://localhost",
				Params: map[string]string{
					"globalParam2": "value2",
					"param1":       "value1",
				},
				Headers: map[string]string{
					"globalHeader2": "value2",
					"header1":       "value1",
				},
			},
		},
		{
			name: "error template not found",
			fields: fields{
				config: &FetcherConfig{
					Templates: []FetcherTemplate{},
				},
				logger:    log.Default(),
				requester: &testFetcherFetchRequester{},
				resources: map[string]Resource{},
				data: &cache{
					objects: map[string]cacheObject{},
					lock:    sync.RWMutex{},
				},
			},
			args: args{
				template: "template",
				resource: "resource",
				params: map[string]string{
					"param1": "value1",
				},
				headers: map[string]string{
					"header1": "value1",
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fetcher{
				config:        tt.fields.config,
				logger:        tt.fields.logger,
				requester:     tt.fields.requester,
				resources:     tt.fields.resources,
				resourcesLock: sync.RWMutex{},
				data:          tt.fields.data,
			}
			got, err := f.CreateResourceFromTemplate(tt.args.template, tt.args.resource, tt.args.params, tt.args.headers)
			if (err != nil) != tt.wantErr {
				t.Errorf("fetcher.CreateResourceFromTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fetcher.CreateResourceFromTemplate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewFetchRequester(t *testing.T) {
	type args struct {
		logger  *log.Logger
		client  *http.Client
		headers map[string]string
		retry   int
		delay   time.Duration
	}
	tests := []struct {
		name    string
		args    args
		wantNil bool
	}{
		{
			name: "default",
			args: args{
				logger: log.Default(),
				client: http.DefaultClient,
				headers: map[string]string{
					"header": "value",
				},
				retry: 3,
				delay: time.Duration(100) * time.Millisecond,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newFetchRequester(tt.args.logger, tt.args.client, tt.args.headers, tt.args.retry, tt.args.delay)
			if (got == nil) != tt.wantNil {
				t.Errorf("newFetchRequester() got = %v, wantNil %v", got, tt.wantNil)
			}
		})
	}
}

type testFetchRequesterRequestResponseBody struct {
	response []byte
	errRead  bool
	errClose bool
}

func (b *testFetchRequesterRequestResponseBody) Read(p []byte) (n int, err error) {
	if b.errRead {
		return 0, errors.New("test error")
	}
	return len(b.response), nil
}

func (b *testFetchRequesterRequestResponseBody) Close() error {
	if b.errClose {
		return errors.New("test error")
	}
	return nil
}

func TestFetchRequesterFetch(t *testing.T) {
	response := []byte("response")
	status := http.StatusOK

	type fields struct {
		logger                    *log.Logger
		client                    *http.Client
		headers                   map[string]string
		retry                     int
		delay                     time.Duration
		httpNewRequestWithContext func(ctx context.Context, method string, url string,
			body io.Reader) (*http.Request, error)
		httpClientDo func(client *http.Client, req *http.Request) (*http.Response, error)
		ioRealAll    func(r io.Reader) ([]byte, error)
	}
	type args struct {
		ctx     context.Context
		method  string
		url     string
		params  map[string]string
		headers map[string]string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				logger: log.Default(),
				client: http.DefaultClient,
				headers: map[string]string{
					"header": "value",
				},
				retry: 3,
				delay: time.Duration(100) * time.Millisecond,
				httpNewRequestWithContext: func(ctx context.Context, method string, u string,
					body io.Reader) (*http.Request, error) {
					return &http.Request{
						Method: method,
						URL:    &url.URL{},
						Body:   nil,
						Header: http.Header{},
					}, nil
				},
				httpClientDo: func(client *http.Client, req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: status,
						Body: &testFetchRequesterRequestResponseBody{
							response: response,
							errRead:  false,
							errClose: false,
						},
					}, nil
				},
				ioRealAll: func(r io.Reader) ([]byte, error) {
					return response, nil
				},
			},
			args: args{
				ctx:    context.Background(),
				method: http.MethodGet,
				url:    "http://localhost",
				params: map[string]string{
					"param1": "value1",
				},
				headers: map[string]string{
					"header1": "value1",
				},
			},
			want: response,
		},
		{
			name: "error new request with context",
			fields: fields{
				logger: log.Default(),
				client: http.DefaultClient,
				headers: map[string]string{
					"header": "value",
				},
				retry: 3,
				delay: time.Duration(100) * time.Millisecond,
				httpNewRequestWithContext: func(ctx context.Context, method string, url string,
					body io.Reader) (*http.Request, error) {
					return nil, errors.New("test error")
				},
				httpClientDo: func(client *http.Client, req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: status,
						Body: &testFetchRequesterRequestResponseBody{
							response: response,
							errRead:  false,
							errClose: false,
						},
					}, nil
				},
				ioRealAll: func(r io.Reader) ([]byte, error) {
					return response, nil
				},
			},
			args: args{
				ctx:    context.Background(),
				method: http.MethodGet,
				url:    "http://localhost",
				params: map[string]string{
					"param1": "value1",
				},
				headers: map[string]string{
					"header1": "value1",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "error do request",
			fields: fields{
				logger: log.Default(),
				client: http.DefaultClient,
				headers: map[string]string{
					"header": "value",
				},
				retry: 3,
				delay: time.Duration(100) * time.Millisecond,
				httpNewRequestWithContext: func(ctx context.Context, method string, u string,
					body io.Reader) (*http.Request, error) {
					return &http.Request{
						Method: method,
						URL:    &url.URL{},
						Body:   nil,
						Header: http.Header{},
					}, nil
				},
				httpClientDo: func(client *http.Client, req *http.Request) (*http.Response, error) {
					return nil, errors.New("test error")
				},
				ioRealAll: func(r io.Reader) ([]byte, error) {
					return response, nil
				},
			},
			args: args{
				ctx:    context.Background(),
				method: http.MethodGet,
				url:    "http://localhost",
				params: map[string]string{
					"param1": "value1",
				},
				headers: map[string]string{
					"header1": "value1",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "error read response",
			fields: fields{
				logger: log.Default(),
				client: http.DefaultClient,
				headers: map[string]string{
					"header": "value",
				},
				retry: 3,
				delay: time.Duration(100) * time.Millisecond,
				httpNewRequestWithContext: func(ctx context.Context, method string, u string,
					body io.Reader) (*http.Request, error) {
					return &http.Request{
						Method: method,
						URL:    &url.URL{},
						Body:   nil,
						Header: http.Header{},
					}, nil
				},
				httpClientDo: func(client *http.Client, req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: status,
						Body: &testFetchRequesterRequestResponseBody{
							response: response,
							errRead:  false,
							errClose: false,
						},
					}, nil
				},
				ioRealAll: func(r io.Reader) ([]byte, error) {
					return []byte{}, errors.New("test error")
				},
			},
			args: args{
				ctx:    context.Background(),
				method: http.MethodGet,
				url:    "http://localhost",
				params: map[string]string{
					"param1": "value1",
				},
				headers: map[string]string{
					"header1": "value1",
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &fetchRequester{
				logger:                    tt.fields.logger,
				client:                    tt.fields.client,
				headers:                   tt.fields.headers,
				retry:                     tt.fields.retry,
				delay:                     tt.fields.delay,
				httpNewRequestWithContext: tt.fields.httpNewRequestWithContext,
				httpClientDo:              tt.fields.httpClientDo,
				ioReadAll:                 tt.fields.ioRealAll,
			}
			got, err := r.Fetch(tt.args.ctx, tt.args.method, tt.args.url, tt.args.params, tt.args.headers)
			if (err != nil) != tt.wantErr {
				t.Errorf("fetchRequester.Fetch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fetchRequester.Fetch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFetchRequesterFetch_Debug(t *testing.T) {
	DEBUG = true
	defer func() {
		DEBUG = false
	}()

	response := []byte("response")
	status := http.StatusOK

	type fields struct {
		logger                    *log.Logger
		client                    *http.Client
		headers                   map[string]string
		retry                     int
		delay                     time.Duration
		httpNewRequestWithContext func(ctx context.Context, method string, url string,
			body io.Reader) (*http.Request, error)
		httpClientDo func(client *http.Client, req *http.Request) (*http.Response, error)
		ioRealAll    func(r io.Reader) ([]byte, error)
	}
	type args struct {
		ctx     context.Context
		method  string
		url     string
		params  map[string]string
		headers map[string]string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				logger: log.Default(),
				client: http.DefaultClient,
				headers: map[string]string{
					"header": "value",
				},
				retry: 3,
				delay: time.Duration(100) * time.Millisecond,
				httpNewRequestWithContext: func(ctx context.Context, method string, u string,
					body io.Reader) (*http.Request, error) {
					return &http.Request{
						Method: method,
						URL:    &url.URL{},
						Body:   nil,
						Header: http.Header{},
					}, nil
				},
				httpClientDo: func(client *http.Client, req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: status,
						Body: &testFetchRequesterRequestResponseBody{
							response: response,
							errRead:  false,
							errClose: false,
						},
					}, nil
				},
				ioRealAll: func(r io.Reader) ([]byte, error) {
					return response, nil
				},
			},
			args: args{
				ctx:    context.Background(),
				method: http.MethodGet,
				url:    "http://localhost",
				params: map[string]string{
					"param1": "value1",
				},
				headers: map[string]string{
					"header1": "value1",
				},
			},
			want: response,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &fetchRequester{
				logger:                    tt.fields.logger,
				client:                    tt.fields.client,
				headers:                   tt.fields.headers,
				retry:                     tt.fields.retry,
				delay:                     tt.fields.delay,
				httpNewRequestWithContext: tt.fields.httpNewRequestWithContext,
				httpClientDo:              tt.fields.httpClientDo,
				ioReadAll:                 tt.fields.ioRealAll,
			}
			got, err := r.Fetch(tt.args.ctx, tt.args.method, tt.args.url, tt.args.params, tt.args.headers)
			if (err != nil) != tt.wantErr {
				t.Errorf("fetchRequester.Fetch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fetchRequester.Fetch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFetchRequesterFetch_ErrorInvalidCode(t *testing.T) {
	response := []byte("response")

	type fields struct {
		logger                    *log.Logger
		client                    *http.Client
		headers                   map[string]string
		retry                     int
		delay                     time.Duration
		httpNewRequestWithContext func(ctx context.Context, method string, url string,
			body io.Reader) (*http.Request, error)
		httpClientDo func(client *http.Client, req *http.Request) (*http.Response, error)
		ioRealAll    func(r io.Reader) ([]byte, error)
	}
	type args struct {
		ctx     context.Context
		method  string
		url     string
		params  map[string]string
		headers map[string]string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "error with invalid status code",
			fields: fields{
				logger: log.Default(),
				client: http.DefaultClient,
				headers: map[string]string{
					"header": "value",
				},
				retry: 3,
				delay: time.Duration(100) * time.Millisecond,
				httpNewRequestWithContext: func(ctx context.Context, method string, u string,
					body io.Reader) (*http.Request, error) {
					return &http.Request{
						Method: method,
						URL:    &url.URL{},
						Body:   nil,
						Header: http.Header{},
					}, nil
				},
				httpClientDo: func(client *http.Client, req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusNotFound,
						Body: &testFetchRequesterRequestResponseBody{
							errRead:  false,
							errClose: false,
						},
					}, nil
				},
				ioRealAll: func(r io.Reader) ([]byte, error) {
					return response, nil
				},
			},
			args: args{
				ctx:    context.Background(),
				method: http.MethodGet,
				url:    "http://localhost",
				params: map[string]string{
					"param1": "value1",
				},
				headers: map[string]string{
					"header1": "value1",
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &fetchRequester{
				logger:                    tt.fields.logger,
				client:                    tt.fields.client,
				headers:                   tt.fields.headers,
				retry:                     tt.fields.retry,
				delay:                     tt.fields.delay,
				httpNewRequestWithContext: tt.fields.httpNewRequestWithContext,
				httpClientDo:              tt.fields.httpClientDo,
				ioReadAll:                 tt.fields.ioRealAll,
			}
			got, err := r.Fetch(tt.args.ctx, tt.args.method, tt.args.url, tt.args.params, tt.args.headers)
			if (err != nil) != tt.wantErr {
				t.Errorf("fetchRequester.Fetch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fetchRequester.Fetch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFetchRequesterFetch_ErrorWithRetrySuccess(t *testing.T) {
	retryCount := 0
	retrySuccess := 2
	response := []byte("response")

	type fields struct {
		logger                    *log.Logger
		client                    *http.Client
		headers                   map[string]string
		retry                     int
		delay                     time.Duration
		httpNewRequestWithContext func(ctx context.Context, method string, url string,
			body io.Reader) (*http.Request, error)
		httpClientDo func(client *http.Client, req *http.Request) (*http.Response, error)
		ioRealAll    func(r io.Reader) ([]byte, error)
	}
	type args struct {
		ctx     context.Context
		method  string
		url     string
		params  map[string]string
		headers map[string]string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "retry with delay",
			fields: fields{
				logger: log.Default(),
				client: http.DefaultClient,
				headers: map[string]string{
					"header": "value",
				},
				retry: 3,
				delay: time.Duration(100) * time.Millisecond,
				httpNewRequestWithContext: func(ctx context.Context, method string, u string,
					body io.Reader) (*http.Request, error) {
					return &http.Request{
						Method: method,
						URL:    &url.URL{},
						Body:   nil,
						Header: http.Header{},
					}, nil
				},
				httpClientDo: func(client *http.Client, req *http.Request) (*http.Response, error) {
					retryCount += 1
					if retryCount >= retrySuccess {
						return &http.Response{
							StatusCode: http.StatusOK,
							Body: &testFetchRequesterRequestResponseBody{
								response: response,
								errRead:  false,
								errClose: false,
							},
						}, nil
					}
					return &http.Response{
						StatusCode: http.StatusInternalServerError,
						Body: &testFetchRequesterRequestResponseBody{
							errRead:  false,
							errClose: false,
						},
					}, nil
				},
				ioRealAll: func(r io.Reader) ([]byte, error) {
					return response, nil
				},
			},
			args: args{
				ctx:    context.Background(),
				method: http.MethodGet,
				url:    "http://localhost",
				params: map[string]string{
					"param1": "value1",
				},
				headers: map[string]string{
					"header1": "value1",
				},
			},
			want:    response,
			wantErr: false,
		},
		{
			name: "retry without delay",
			fields: fields{
				logger: log.Default(),
				client: http.DefaultClient,
				headers: map[string]string{
					"header": "value",
				},
				retry: 3,
				httpNewRequestWithContext: func(ctx context.Context, method string, u string,
					body io.Reader) (*http.Request, error) {
					return &http.Request{
						Method: method,
						URL:    &url.URL{},
						Body:   nil,
						Header: http.Header{},
					}, nil
				},
				httpClientDo: func(client *http.Client, req *http.Request) (*http.Response, error) {
					retryCount += 1
					if retryCount >= retrySuccess {
						return &http.Response{
							StatusCode: http.StatusOK,
							Body: &testFetchRequesterRequestResponseBody{
								response: response,
								errRead:  false,
								errClose: false,
							},
						}, nil
					}
					return &http.Response{
						StatusCode: http.StatusInternalServerError,
						Body: &testFetchRequesterRequestResponseBody{
							errRead:  false,
							errClose: false,
						},
					}, nil
				},
				ioRealAll: func(r io.Reader) ([]byte, error) {
					return response, nil
				},
			},
			args: args{
				ctx:    context.Background(),
				method: http.MethodGet,
				url:    "http://localhost",
				params: map[string]string{
					"param1": "value1",
				},
				headers: map[string]string{
					"header1": "value1",
				},
			},
			want:    response,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &fetchRequester{
				logger:                    tt.fields.logger,
				client:                    tt.fields.client,
				headers:                   tt.fields.headers,
				retry:                     tt.fields.retry,
				delay:                     tt.fields.delay,
				httpNewRequestWithContext: tt.fields.httpNewRequestWithContext,
				httpClientDo:              tt.fields.httpClientDo,
				ioReadAll:                 tt.fields.ioRealAll,
			}
			got, err := r.Fetch(tt.args.ctx, tt.args.method, tt.args.url, tt.args.params, tt.args.headers)
			if (err != nil) != tt.wantErr {
				t.Errorf("fetchRequester.Fetch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fetchRequester.Fetch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFetchRequesterFetch_ErrorWithRetryFailure(t *testing.T) {
	type fields struct {
		logger                    *log.Logger
		client                    *http.Client
		headers                   map[string]string
		retry                     int
		delay                     time.Duration
		httpNewRequestWithContext func(ctx context.Context, method string, url string,
			body io.Reader) (*http.Request, error)
		httpClientDo func(client *http.Client, req *http.Request) (*http.Response, error)
		ioRealAll    func(r io.Reader) ([]byte, error)
	}
	type args struct {
		ctx     context.Context
		method  string
		url     string
		params  map[string]string
		headers map[string]string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "retry with delay",
			fields: fields{
				logger: log.Default(),
				client: http.DefaultClient,
				headers: map[string]string{
					"header": "value",
				},
				retry: 3,
				delay: time.Duration(100) * time.Millisecond,
				httpNewRequestWithContext: func(ctx context.Context, method string, u string,
					body io.Reader) (*http.Request, error) {
					return &http.Request{
						Method: method,
						URL:    &url.URL{},
						Body:   nil,
						Header: http.Header{},
					}, nil
				},
				httpClientDo: func(client *http.Client, req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusInternalServerError,
						Body: &testFetchRequesterRequestResponseBody{
							errRead:  false,
							errClose: false,
						},
					}, nil
				},
				ioRealAll: func(r io.Reader) ([]byte, error) {
					return []byte{}, nil
				},
			},
			args: args{
				ctx:    context.Background(),
				method: http.MethodGet,
				url:    "http://localhost",
				params: map[string]string{
					"param1": "value1",
				},
				headers: map[string]string{
					"header1": "value1",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "retry without delay",
			fields: fields{
				logger: log.Default(),
				client: http.DefaultClient,
				headers: map[string]string{
					"header": "value",
				},
				retry: 3,
				httpNewRequestWithContext: func(ctx context.Context, method string, u string,
					body io.Reader) (*http.Request, error) {
					return &http.Request{
						Method: method,
						URL:    &url.URL{},
						Body:   nil,
						Header: http.Header{},
					}, nil
				},
				httpClientDo: func(client *http.Client, req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusInternalServerError,
						Body: &testFetchRequesterRequestResponseBody{
							errRead:  false,
							errClose: false,
						},
					}, nil
				},
				ioRealAll: func(r io.Reader) ([]byte, error) {
					return []byte{}, nil
				},
			},
			args: args{
				ctx:    context.Background(),
				method: http.MethodGet,
				url:    "http://localhost",
				params: map[string]string{
					"param1": "value1",
				},
				headers: map[string]string{
					"header1": "value1",
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &fetchRequester{
				logger:                    tt.fields.logger,
				client:                    tt.fields.client,
				headers:                   tt.fields.headers,
				retry:                     tt.fields.retry,
				delay:                     tt.fields.delay,
				httpNewRequestWithContext: tt.fields.httpNewRequestWithContext,
				httpClientDo:              tt.fields.httpClientDo,
				ioReadAll:                 tt.fields.ioRealAll,
			}
			got, err := r.Fetch(tt.args.ctx, tt.args.method, tt.args.url, tt.args.params, tt.args.headers)
			if (err != nil) != tt.wantErr {
				t.Errorf("fetchRequester.Fetch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fetchRequester.Fetch() = %v, want %v", got, tt.want)
			}
		})
	}
}
