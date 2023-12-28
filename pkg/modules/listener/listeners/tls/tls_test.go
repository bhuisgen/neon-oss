// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
)

func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}

type testTLSListenerServerRegistry struct {
	error bool
}

func (r testTLSListenerServerRegistry) Listeners() []net.Listener {
	return nil
}

func (r testTLSListenerServerRegistry) RegisterListener(listener net.Listener) error {
	if r.error {
		return errors.New("test error")
	}
	return nil
}

var _ core.ListenerRegistry = (*testTLSListenerServerRegistry)(nil)

type testTLSListenerFileInfo struct {
	name     string
	size     int64
	fileMode os.FileMode
	modTime  time.Time
	isDir    bool
	sys      any
}

func (fi testTLSListenerFileInfo) Name() string {
	return fi.name
}

func (fi testTLSListenerFileInfo) Size() int64 {
	return fi.size
}

func (fi testTLSListenerFileInfo) Mode() os.FileMode {
	return fi.fileMode
}
func (fi testTLSListenerFileInfo) ModTime() time.Time {
	return fi.modTime
}

func (fi testTLSListenerFileInfo) IsDir() bool {
	return fi.isDir
}

func (fi testTLSListenerFileInfo) Sys() any {
	return fi.sys
}

var _ os.FileInfo = (*testTLSListenerFileInfo)(nil)

func TestTLSListenerModuleInfo(t *testing.T) {
	type fields struct {
		config                         *tlsListenerConfig
		logger                         *log.Logger
		listener                       net.Listener
		server                         *http.Server
		osOpenFile                     func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osReadFile                     func(name string) ([]byte, error)
		osClose                        func(f *os.File) error
		osStat                         func(name string) (fs.FileInfo, error)
		x509CertPoolAppendCertsFromPEM func(pool *x509.CertPool, pemCerts []byte) bool
		tlsLoadX509KeyPair             func(certFile string, keyFile string) (tls.Certificate, error)
		netListen                      func(network string, addr string) (net.Listener, error)
		httpServerServeTLS             func(server *http.Server, listener net.Listener, certFile string, keyFile string) error
		httpServerShutdown             func(server *http.Server, context context.Context) error
		httpServerClose                func(server *http.Server) error
	}
	tests := []struct {
		name   string
		fields fields
		want   module.ModuleInfo
	}{
		{
			name: "default",
			want: module.ModuleInfo{
				ID:          tlsModuleID,
				NewInstance: func() module.Module { return &tlsListener{} },
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := tlsListener{
				config:                         tt.fields.config,
				logger:                         tt.fields.logger,
				listener:                       tt.fields.listener,
				server:                         tt.fields.server,
				osOpenFile:                     tt.fields.osOpenFile,
				osReadFile:                     tt.fields.osReadFile,
				osClose:                        tt.fields.osClose,
				osStat:                         tt.fields.osStat,
				x509CertPoolAppendCertsFromPEM: tt.fields.x509CertPoolAppendCertsFromPEM,
				tlsLoadX509KeyPair:             tt.fields.tlsLoadX509KeyPair,
				netListen:                      tt.fields.netListen,
				httpServerServeTLS:             tt.fields.httpServerServeTLS,
				httpServerShutdown:             tt.fields.httpServerShutdown,
				httpServerClose:                tt.fields.httpServerClose,
			}
			got := l.ModuleInfo()
			if got.ID != tt.want.ID {
				t.Errorf("tlsListener.ModuleInfo() = %v, want %v", got.ID, tt.want.ID)
			}
			if instance := got.NewInstance(); instance == nil {
				t.Errorf("tlsListener.NewInstance() = %v, want %v", instance, "not nil")
			}
		})
	}
}

func TestTLSListenerCheck(t *testing.T) {
	type fields struct {
		config                         *tlsListenerConfig
		logger                         *log.Logger
		listener                       net.Listener
		server                         *http.Server
		osOpenFile                     func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osReadFile                     func(name string) ([]byte, error)
		osClose                        func(f *os.File) error
		osStat                         func(name string) (fs.FileInfo, error)
		x509CertPoolAppendCertsFromPEM func(pool *x509.CertPool, pemCerts []byte) bool
		tlsLoadX509KeyPair             func(certFile string, keyFile string) (tls.Certificate, error)
		netListen                      func(network string, addr string) (net.Listener, error)
		httpServerServeTLS             func(server *http.Server, listener net.Listener, certFile string, keyFile string) error
		httpServerShutdown             func(server *http.Server, context context.Context) error
		httpServerClose                func(server *http.Server) error
	}
	type args struct {
		config map[string]interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []string
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
					return testTLSListenerFileInfo{}, nil
				},
			},
			args: args{
				config: map[string]interface{}{
					"CertFiles": []string{"cert.pem"},
					"KeyFiles":  []string{"key.pem"},
				},
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
					return testTLSListenerFileInfo{}, nil
				},
			},
			args: args{
				config: map[string]interface{}{
					"ListenAddr":        "0.0.0.0",
					"ListenPort":        443,
					"CAFiles":           []string{"ca.pem"},
					"CertFiles":         []string{"cert.pem"},
					"KeyFiles":          []string{"key.pem"},
					"ClientAuth":        "requireAndVerify",
					"ReadTimeout":       30,
					"ReadHeaderTimeout": 4,
					"WriteTimeout":      30,
					"IdleTimeout":       60,
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
					return testTLSListenerFileInfo{}, nil
				},
			},
			args: args{
				config: map[string]interface{}{
					"ListenAddr":        "",
					"ListenPort":        -1,
					"CAFiles":           []string{""},
					"CertFiles":         []string{""},
					"KeyFiles":          []string{""},
					"ClientAuth":        "",
					"ReadTimeout":       -1,
					"ReadHeaderTimeout": -1,
					"WriteTimeout":      -1,
					"IdleTimeout":       -1,
				},
			},
			want: []string{
				"option 'ListenPort', invalid value '-1'",
				"option 'CAFiles', invalid value ''",
				"option 'CertFiles', invalid value ''",
				"option 'KeyFiles', invalid value ''",
				"option 'ClientAuth', invalid value ''",
				"option 'ReadTimeout', invalid value '-1'",
				"option 'ReadHeaderTimeout', invalid value '-1'",
				"option 'WriteTimeout', invalid value '-1'",
				"option 'IdleTimeout', invalid value '-1'",
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
					return testTLSListenerFileInfo{}, nil
				},
			},
			args: args{
				config: map[string]interface{}{
					"CAFiles":   []string{"ca.pem"},
					"CertFiles": []string{"cert.pem"},
					"KeyFiles":  []string{"key.pem"},
				},
			},
			want: []string{
				"option 'CAFiles', failed to open file 'ca.pem'",
				"option 'CertFiles', failed to open file 'cert.pem'",
				"option 'KeyFiles', failed to open file 'key.pem'",
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
					"CAFiles":   []string{"ca.pem"},
					"CertFiles": []string{"cert.pem"},
					"KeyFiles":  []string{"key.pem"},
				},
			},
			want: []string{
				"option 'CAFiles', failed to stat file 'ca.pem'",
				"option 'CertFiles', failed to stat file 'cert.pem'",
				"option 'KeyFiles', failed to stat file 'key.pem'",
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
					return testTLSListenerFileInfo{
						isDir: true,
					}, nil
				},
			},
			args: args{
				config: map[string]interface{}{
					"CAFiles":   []string{"ca.pem"},
					"CertFiles": []string{"cert.pem"},
					"KeyFiles":  []string{"key.pem"},
				},
			},
			want: []string{
				"option 'CAFiles', 'ca.pem' is a directory",
				"option 'CertFiles', 'cert.pem' is a directory",
				"option 'KeyFiles', 'key.pem' is a directory",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &tlsListener{
				config:                         tt.fields.config,
				logger:                         tt.fields.logger,
				listener:                       tt.fields.listener,
				server:                         tt.fields.server,
				osOpenFile:                     tt.fields.osOpenFile,
				osReadFile:                     tt.fields.osReadFile,
				osClose:                        tt.fields.osClose,
				osStat:                         tt.fields.osStat,
				x509CertPoolAppendCertsFromPEM: tt.fields.x509CertPoolAppendCertsFromPEM,
				tlsLoadX509KeyPair:             tt.fields.tlsLoadX509KeyPair,
				netListen:                      tt.fields.netListen,
				httpServerServeTLS:             tt.fields.httpServerServeTLS,
				httpServerShutdown:             tt.fields.httpServerShutdown,
				httpServerClose:                tt.fields.httpServerClose,
			}
			got, err := l.Check(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("tlsListener.Check() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("tlsListener.Check() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTLSListenerLoad(t *testing.T) {
	type fields struct {
		config                         *tlsListenerConfig
		logger                         *log.Logger
		listener                       net.Listener
		server                         *http.Server
		osOpenFile                     func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osReadFile                     func(name string) ([]byte, error)
		osClose                        func(f *os.File) error
		osStat                         func(name string) (fs.FileInfo, error)
		x509CertPoolAppendCertsFromPEM func(pool *x509.CertPool, pemCerts []byte) bool
		tlsLoadX509KeyPair             func(certFile string, keyFile string) (tls.Certificate, error)
		netListen                      func(network string, addr string) (net.Listener, error)
		httpServerServeTLS             func(server *http.Server, listener net.Listener, certFile string, keyFile string) error
		httpServerShutdown             func(server *http.Server, context context.Context) error
		httpServerClose                func(server *http.Server) error
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &tlsListener{
				config:                         tt.fields.config,
				logger:                         tt.fields.logger,
				listener:                       tt.fields.listener,
				server:                         tt.fields.server,
				osOpenFile:                     tt.fields.osOpenFile,
				osReadFile:                     tt.fields.osReadFile,
				osClose:                        tt.fields.osClose,
				osStat:                         tt.fields.osStat,
				x509CertPoolAppendCertsFromPEM: tt.fields.x509CertPoolAppendCertsFromPEM,
				tlsLoadX509KeyPair:             tt.fields.tlsLoadX509KeyPair,
				netListen:                      tt.fields.netListen,
				httpServerServeTLS:             tt.fields.httpServerServeTLS,
				httpServerShutdown:             tt.fields.httpServerShutdown,
				httpServerClose:                tt.fields.httpServerClose,
			}
			if err := l.Load(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("tlsListener.Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTLSListenerRegister(t *testing.T) {
	type fields struct {
		config                         *tlsListenerConfig
		logger                         *log.Logger
		listener                       net.Listener
		server                         *http.Server
		osOpenFile                     func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osReadFile                     func(name string) ([]byte, error)
		osClose                        func(f *os.File) error
		osStat                         func(name string) (fs.FileInfo, error)
		x509CertPoolAppendCertsFromPEM func(pool *x509.CertPool, pemCerts []byte) bool
		tlsLoadX509KeyPair             func(certFile string, keyFile string) (tls.Certificate, error)
		netListen                      func(network string, addr string) (net.Listener, error)
		httpServerServeTLS             func(server *http.Server, listener net.Listener, certFile string, keyFile string) error
		httpServerShutdown             func(server *http.Server, context context.Context) error
		httpServerClose                func(server *http.Server) error
	}
	type args struct {
		registry core.ListenerRegistry
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
				config: &tlsListenerConfig{
					ListenAddr: stringPtr(tlsConfigDefaultListenAddr),
					ListenPort: intPtr(tlsConfigDefaultListenPort),
				},
				netListen: func(network, addr string) (net.Listener, error) {
					return nil, nil
				},
			},
			args: args{
				registry: testTLSListenerServerRegistry{},
			},
		},
		{
			name: "error listen",
			fields: fields{
				config: &tlsListenerConfig{
					ListenAddr: stringPtr(tlsConfigDefaultListenAddr),
					ListenPort: intPtr(tlsConfigDefaultListenPort),
				},
				netListen: func(network, addr string) (net.Listener, error) {
					return nil, errors.New("test error")
				},
			},
			args: args{
				registry: testTLSListenerServerRegistry{},
			},
			wantErr: true,
		},
		{
			name: "error register",
			fields: fields{
				config: &tlsListenerConfig{
					ListenAddr: stringPtr(tlsConfigDefaultListenAddr),
					ListenPort: intPtr(tlsConfigDefaultListenPort),
				},
				netListen: func(network, addr string) (net.Listener, error) {
					return nil, nil
				},
			},
			args: args{
				registry: testTLSListenerServerRegistry{
					error: true,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &tlsListener{
				config:                         tt.fields.config,
				logger:                         tt.fields.logger,
				listener:                       tt.fields.listener,
				server:                         tt.fields.server,
				osOpenFile:                     tt.fields.osOpenFile,
				osReadFile:                     tt.fields.osReadFile,
				osClose:                        tt.fields.osClose,
				osStat:                         tt.fields.osStat,
				x509CertPoolAppendCertsFromPEM: tt.fields.x509CertPoolAppendCertsFromPEM,
				tlsLoadX509KeyPair:             tt.fields.tlsLoadX509KeyPair,
				netListen:                      tt.fields.netListen,
				httpServerServeTLS:             tt.fields.httpServerServeTLS,
				httpServerShutdown:             tt.fields.httpServerShutdown,
				httpServerClose:                tt.fields.httpServerClose,
			}
			if err := l.Register(tt.args.registry); (err != nil) != tt.wantErr {
				t.Errorf("tlsListener.Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTLSListenerServe(t *testing.T) {
	type fields struct {
		config                         *tlsListenerConfig
		logger                         *log.Logger
		listener                       net.Listener
		server                         *http.Server
		osOpenFile                     func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osReadFile                     func(name string) ([]byte, error)
		osClose                        func(f *os.File) error
		osStat                         func(name string) (fs.FileInfo, error)
		x509CertPoolAppendCertsFromPEM func(pool *x509.CertPool, pemCerts []byte) bool
		tlsLoadX509KeyPair             func(certFile string, keyFile string) (tls.Certificate, error)
		netListen                      func(network string, addr string) (net.Listener, error)
		httpServerServeTLS             func(server *http.Server, listener net.Listener, certFile string, keyFile string) error
		httpServerShutdown             func(server *http.Server, context context.Context) error
		httpServerClose                func(server *http.Server) error
	}
	type args struct {
		handler http.Handler
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
				config: &tlsListenerConfig{
					ListenAddr:        stringPtr(tlsConfigDefaultListenAddr),
					ListenPort:        intPtr(tlsConfigDefaultListenPort),
					ReadTimeout:       intPtr(30),
					ReadHeaderTimeout: intPtr(4),
					WriteTimeout:      intPtr(30),
					IdleTimeout:       intPtr(60),
				},
				logger: log.Default(),
				httpServerServeTLS: func(server *http.Server, listener net.Listener, certFile, keyFile string) error {
					return nil
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &tlsListener{
				config:                         tt.fields.config,
				logger:                         tt.fields.logger,
				listener:                       tt.fields.listener,
				server:                         tt.fields.server,
				osOpenFile:                     tt.fields.osOpenFile,
				osReadFile:                     tt.fields.osReadFile,
				osClose:                        tt.fields.osClose,
				osStat:                         tt.fields.osStat,
				x509CertPoolAppendCertsFromPEM: tt.fields.x509CertPoolAppendCertsFromPEM,
				tlsLoadX509KeyPair:             tt.fields.tlsLoadX509KeyPair,
				netListen:                      tt.fields.netListen,
				httpServerServeTLS:             tt.fields.httpServerServeTLS,
				httpServerShutdown:             tt.fields.httpServerShutdown,
				httpServerClose:                tt.fields.httpServerClose,
			}
			if err := l.Serve(tt.args.handler); (err != nil) != tt.wantErr {
				t.Errorf("tlsListener.Serve() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTLSListenerShutdown(t *testing.T) {
	type fields struct {
		config                         *tlsListenerConfig
		logger                         *log.Logger
		listener                       net.Listener
		server                         *http.Server
		osOpenFile                     func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osReadFile                     func(name string) ([]byte, error)
		osClose                        func(f *os.File) error
		osStat                         func(name string) (fs.FileInfo, error)
		x509CertPoolAppendCertsFromPEM func(pool *x509.CertPool, pemCerts []byte) bool
		tlsLoadX509KeyPair             func(certFile string, keyFile string) (tls.Certificate, error)
		netListen                      func(network string, addr string) (net.Listener, error)
		httpServerServeTLS             func(server *http.Server, listener net.Listener, certFile string, keyFile string) error
		httpServerShutdown             func(server *http.Server, context context.Context) error
		httpServerClose                func(server *http.Server) error
	}
	type args struct {
		ctx context.Context
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
				httpServerShutdown: func(server *http.Server, context context.Context) error {
					return nil
				},
			},
		},
		{
			name: "error shutdown",
			fields: fields{
				httpServerShutdown: func(server *http.Server, context context.Context) error {
					return errors.New("test error")
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &tlsListener{
				config:                         tt.fields.config,
				logger:                         tt.fields.logger,
				listener:                       tt.fields.listener,
				server:                         tt.fields.server,
				osOpenFile:                     tt.fields.osOpenFile,
				osReadFile:                     tt.fields.osReadFile,
				osClose:                        tt.fields.osClose,
				osStat:                         tt.fields.osStat,
				x509CertPoolAppendCertsFromPEM: tt.fields.x509CertPoolAppendCertsFromPEM,
				tlsLoadX509KeyPair:             tt.fields.tlsLoadX509KeyPair,
				netListen:                      tt.fields.netListen,
				httpServerServeTLS:             tt.fields.httpServerServeTLS,
				httpServerShutdown:             tt.fields.httpServerShutdown,
				httpServerClose:                tt.fields.httpServerClose,
			}
			if err := l.Shutdown(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("tlsListener.Shutdown() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTLSListenerClose(t *testing.T) {
	type fields struct {
		config                         *tlsListenerConfig
		logger                         *log.Logger
		listener                       net.Listener
		server                         *http.Server
		osOpenFile                     func(name string, flag int, perm fs.FileMode) (*os.File, error)
		osReadFile                     func(name string) ([]byte, error)
		osClose                        func(f *os.File) error
		osStat                         func(name string) (fs.FileInfo, error)
		x509CertPoolAppendCertsFromPEM func(pool *x509.CertPool, pemCerts []byte) bool
		tlsLoadX509KeyPair             func(certFile string, keyFile string) (tls.Certificate, error)
		netListen                      func(network string, addr string) (net.Listener, error)
		httpServerServeTLS             func(server *http.Server, listener net.Listener, certFile string, keyFile string) error
		httpServerShutdown             func(server *http.Server, context context.Context) error
		httpServerClose                func(server *http.Server) error
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				httpServerClose: func(server *http.Server) error {
					return nil
				},
			},
		},
		{
			name: "error close",
			fields: fields{
				httpServerClose: func(server *http.Server) error {
					return errors.New("test error")
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &tlsListener{
				config:                         tt.fields.config,
				logger:                         tt.fields.logger,
				listener:                       tt.fields.listener,
				server:                         tt.fields.server,
				osOpenFile:                     tt.fields.osOpenFile,
				osReadFile:                     tt.fields.osReadFile,
				osClose:                        tt.fields.osClose,
				osStat:                         tt.fields.osStat,
				x509CertPoolAppendCertsFromPEM: tt.fields.x509CertPoolAppendCertsFromPEM,
				tlsLoadX509KeyPair:             tt.fields.tlsLoadX509KeyPair,
				netListen:                      tt.fields.netListen,
				httpServerServeTLS:             tt.fields.httpServerServeTLS,
				httpServerShutdown:             tt.fields.httpServerShutdown,
				httpServerClose:                tt.fields.httpServerClose,
			}
			if err := l.Close(); (err != nil) != tt.wantErr {
				t.Errorf("tlsListener.Close() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
