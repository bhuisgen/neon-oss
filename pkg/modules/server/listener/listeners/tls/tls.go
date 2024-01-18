// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
)

// tlsListener implements the tls listener.
type tlsListener struct {
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

// tlsListenerConfig implements the tls listener configuration.
type tlsListenerConfig struct {
	ListenAddr        *string
	ListenPort        *int
	CAFiles           *[]string
	CertFiles         []string
	KeyFiles          []string
	ClientAuth        *string
	ReadTimeout       *int
	ReadHeaderTimeout *int
	WriteTimeout      *int
	IdleTimeout       *int
}

const (
	tlsModuleID module.ModuleID = "server.listener.tls"
	tlsLogger   string          = "listener[tls]"

	tlsClientAuthNone             string = "none"
	tlsClientAuthRequest          string = "request"
	tlsClientAuthRequire          string = "require"
	tlsClientAuthVerify           string = "verify"
	tlsClientAuthRequireAndVerify string = "requireAndVerify"

	tlsConfigDefaultListenAddr        string = ""
	tlsConfigDefaultListenPort        int    = 443
	tlsConfigDefaultReadTimeout       int    = 60
	tlsConfigDefaultReadHeaderTimeout int    = 10
	tlsConfigDefaultWriteTimeout      int    = 60
	tlsConfigDefaultIdleTimeout       int    = 60
	tlsConfigDefaultClientAuth        string = tlsClientAuthNone
)

// tlsOsOpenFile redirects to os.OpenFile.
func tlsOsOpenFile(name string, flag int, perm fs.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

// tlsOsReadFile redirects to os.ReadFile.
func tlsOsReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// tlsOsClose redirects to os.Close.
func tlsOsClose(f *os.File) error {
	return f.Close()
}

// tlsOsStat redirects to os.Stat.
func tlsOsStat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

// tlsX509CertPoolAppendCertsFromPEM redirects to x509.CertPool.AppendCertsFromPEM.
func tlsX509CertPoolAppendCertsFromPEM(pool *x509.CertPool, pemCerts []byte) bool {
	return pool.AppendCertsFromPEM(pemCerts)
}

// tlsTLSLoadX509KeyPair redirects to tls.LoadX509KeyPair.
func tlsTLSLoadX509KeyPair(certFile string, keyFile string) (tls.Certificate, error) {
	return tls.LoadX509KeyPair(certFile, keyFile)
}

// tlsNetListen redirects to net.Listen.
func tlsNetListen(network string, addr string) (net.Listener, error) {
	return net.Listen(network, addr)
}

// tlsHttpServerServeTLS redirects to http.Server.ServeTLS.
func tlsHttpServerServeTLS(server *http.Server, listener net.Listener, certFile string, keyFile string) error {
	return server.ServeTLS(listener, certFile, keyFile)
}

// tlsHttpServerShutdown redirects to http.Server.Shutdown.
func tlsHttpServerShutdown(server *http.Server, context context.Context) error {
	return server.Shutdown(context)
}

// tlsHttpServerShutdown redirects to http.Server.Close.
func tlsHttpServerClose(server *http.Server) error {
	return server.Close()
}

// init initializes the module.
func init() {
	module.Register(tlsListener{})
}

// ModuleInfo returns the module information.
func (l tlsListener) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: tlsModuleID,
		NewInstance: func() module.Module {
			return &tlsListener{
				osOpenFile:                     tlsOsOpenFile,
				osReadFile:                     tlsOsReadFile,
				osClose:                        tlsOsClose,
				osStat:                         tlsOsStat,
				x509CertPoolAppendCertsFromPEM: tlsX509CertPoolAppendCertsFromPEM,
				tlsLoadX509KeyPair:             tlsTLSLoadX509KeyPair,
				netListen:                      tlsNetListen,
				httpServerServeTLS:             tlsHttpServerServeTLS,
				httpServerShutdown:             tlsHttpServerShutdown,
				httpServerClose:                tlsHttpServerClose,
			}
		},
	}
}

// Check checks the listener configuration.
func (l *tlsListener) Check(config map[string]interface{}) ([]string, error) {
	var report []string

	var c tlsListenerConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		report = append(report, "failed to parse configuration")
		return report, err
	}

	if c.ListenAddr == nil {
		defaultValue := tlsConfigDefaultListenAddr
		c.ListenAddr = &defaultValue
	}
	if c.ListenPort == nil {
		defaultValue := tlsConfigDefaultListenPort
		c.ListenPort = &defaultValue
	}
	if *c.ListenPort < 0 {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%d'", "ListenPort", *c.ListenPort))
	}
	if c.CAFiles != nil {
		for _, item := range *c.CAFiles {
			if item == "" {
				report = append(report, fmt.Sprintf("option '%s', invalid value '%s'", "CAFiles", item))
			} else {
				f, err := l.osOpenFile(item, os.O_RDONLY, 0)
				if err != nil {
					report = append(report, fmt.Sprintf("option '%s', failed to open file '%s'", "CAFiles", item))
				} else {
					l.osClose(f)
					fi, err := l.osStat(item)
					if err != nil {
						report = append(report, fmt.Sprintf("option '%s', failed to stat file '%s'", "CAFiles", item))
					}
					if err == nil && fi.IsDir() {
						report = append(report, fmt.Sprintf("option '%s', '%s' is a directory", "CAFiles", item))
					}
				}
			}
		}
	}
	if len(c.CertFiles) == 0 {
		report = append(report, fmt.Sprintf("option '%s', invalid value", "CertFiles"))
	} else {
		for _, item := range c.CertFiles {
			if item == "" {
				report = append(report, fmt.Sprintf("option '%s', invalid value '%s'", "CertFiles", item))
			} else {
				f, err := l.osOpenFile(item, os.O_RDONLY, 0)
				if err != nil {
					report = append(report, fmt.Sprintf("option '%s', failed to open file '%s'", "CertFiles", item))
				} else {
					l.osClose(f)
					fi, err := l.osStat(item)
					if err != nil {
						report = append(report, fmt.Sprintf("option '%s', failed to stat file '%s'", "CertFiles", item))
					}
					if err == nil && fi.IsDir() {
						report = append(report, fmt.Sprintf("option '%s', '%s' is a directory", "CertFiles", item))
					}
				}
			}
		}
	}
	if len(c.KeyFiles) == 0 || len(c.KeyFiles) != len(c.CertFiles) {
		report = append(report, fmt.Sprintf("option '%s', missing value(s)", "KeyFiles"))
	} else {
		for _, item := range c.KeyFiles {
			if item == "" {
				report = append(report, fmt.Sprintf("option '%s', invalid value '%s'", "KeyFiles", item))
			} else {
				f, err := l.osOpenFile(item, os.O_RDONLY, 0)
				if err != nil {
					report = append(report, fmt.Sprintf("option '%s', failed to open file '%s'", "KeyFiles", item))
				} else {
					l.osClose(f)
					fi, err := l.osStat(item)
					if err != nil {
						report = append(report, fmt.Sprintf("option '%s', failed to stat file '%s'", "KeyFiles", item))
					}
					if err == nil && fi.IsDir() {
						report = append(report, fmt.Sprintf("option '%s', '%s' is a directory", "KeyFiles", item))
					}
				}
			}
		}
	}
	if c.ClientAuth == nil {
		defaultValue := tlsConfigDefaultClientAuth
		c.ClientAuth = &defaultValue
	}
	switch *c.ClientAuth {
	case tlsClientAuthNone:
	case tlsClientAuthRequest:
	case tlsClientAuthRequire:
	case tlsClientAuthVerify:
	case tlsClientAuthRequireAndVerify:
	default:
		report = append(report, fmt.Sprintf("option '%s', invalid value '%s'", "ClientAuth", *c.ClientAuth))
	}
	if c.ReadTimeout == nil {
		defaultValue := tlsConfigDefaultReadTimeout
		c.ReadTimeout = &defaultValue
	}
	if *c.ReadTimeout < 0 {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%d'", "ReadTimeout", *c.ReadTimeout))
	}
	if c.ReadHeaderTimeout == nil {
		defaultValue := tlsConfigDefaultReadHeaderTimeout
		c.ReadHeaderTimeout = &defaultValue
	}
	if *c.ReadHeaderTimeout < 0 {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%d'", "ReadHeaderTimeout", *c.ReadHeaderTimeout))
	}
	if c.WriteTimeout == nil {
		defaultValue := tlsConfigDefaultWriteTimeout
		c.WriteTimeout = &defaultValue
	}
	if *c.WriteTimeout < 0 {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%d'", "WriteTimeout", *c.WriteTimeout))
	}
	if c.IdleTimeout == nil {
		defaultValue := tlsConfigDefaultIdleTimeout
		c.IdleTimeout = &defaultValue
	}
	if *c.IdleTimeout < 0 {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%d'", "IdleTimeout", *c.IdleTimeout))
	}

	if len(report) > 0 {
		return report, errors.New("check failure")
	}

	return nil, nil
}

// Load loads the listener.
func (l *tlsListener) Load(config map[string]interface{}) error {
	var c tlsListenerConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		return err
	}

	l.config = &c
	l.logger = log.New(os.Stderr, fmt.Sprint(tlsLogger, ": "), log.LstdFlags|log.Lmsgprefix)

	if l.config.ListenAddr == nil {
		defaultValue := tlsConfigDefaultListenAddr
		l.config.ListenAddr = &defaultValue
	}
	if l.config.ListenPort == nil {
		defaultValue := tlsConfigDefaultListenPort
		l.config.ListenPort = &defaultValue
	}

	if l.config.ClientAuth == nil {
		defaultValue := tlsConfigDefaultClientAuth
		l.config.ClientAuth = &defaultValue
	}
	if l.config.ReadTimeout == nil {
		defaultValue := tlsConfigDefaultReadTimeout
		l.config.ReadTimeout = &defaultValue
	}
	if l.config.ReadHeaderTimeout == nil {
		defaultValue := tlsConfigDefaultReadHeaderTimeout
		l.config.ReadHeaderTimeout = &defaultValue
	}
	if l.config.WriteTimeout == nil {
		defaultValue := tlsConfigDefaultWriteTimeout
		l.config.WriteTimeout = &defaultValue
	}
	if l.config.IdleTimeout == nil {
		defaultValue := tlsConfigDefaultIdleTimeout
		l.config.IdleTimeout = &defaultValue
	}

	return nil
}

// Register registers the listener.
func (l *tlsListener) Register(listener core.ServerListener) error {
	if len(listener.Listeners()) == 1 {
		l.listener = listener.Listeners()[0]
		return nil
	}

	var err error
	l.listener, err = l.netListen("tcp", fmt.Sprintf("%s:%d", *l.config.ListenAddr, *l.config.ListenPort))
	if err != nil {
		return err
	}

	err = listener.RegisterListener(l.listener)
	if err != nil {
		return err
	}

	return nil
}

// Serve accepts incoming connections.
func (l *tlsListener) Serve(handler http.Handler) error {
	l.server = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", *l.config.ListenAddr, *l.config.ListenPort),
		Handler:           handler,
		ReadTimeout:       time.Duration(*l.config.ReadTimeout) * time.Second,
		ReadHeaderTimeout: time.Duration(*l.config.ReadHeaderTimeout) * time.Second,
		WriteTimeout:      time.Duration(*l.config.WriteTimeout) * time.Second,
		IdleTimeout:       time.Duration(*l.config.IdleTimeout) * time.Second,
		ErrorLog:          l.logger,
	}

	tlsConfig := &tls.Config{}

	if l.config.CAFiles != nil {
		caCertPool := x509.NewCertPool()

		for _, caFile := range *l.config.CAFiles {
			ca, err := l.osReadFile(caFile)
			if err != nil {
				return err
			}
			l.x509CertPoolAppendCertsFromPEM(caCertPool, ca)
		}
		tlsConfig.ClientCAs = caCertPool
	}

	var err error
	tlsConfig.Certificates = make([]tls.Certificate, len(l.config.CertFiles))
	for i := range l.config.CertFiles {
		tlsConfig.Certificates[i], err = l.tlsLoadX509KeyPair(l.config.CertFiles[i], l.config.KeyFiles[i])
		if err != nil {
			return err
		}
	}

	if l.config.ClientAuth != nil {
		switch *l.config.ClientAuth {
		case tlsClientAuthNone:
			tlsConfig.ClientAuth = tls.NoClientCert
		case tlsClientAuthRequest:
			tlsConfig.ClientAuth = tls.RequestClientCert
		case tlsClientAuthVerify:
			tlsConfig.ClientAuth = tls.VerifyClientCertIfGiven
		case tlsClientAuthRequire:
			tlsConfig.ClientAuth = tls.RequestClientCert
		case tlsClientAuthRequireAndVerify:
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		}
	}

	l.server.TLSConfig = tlsConfig

	go func() {
		l.logger.Printf("Listening at https://%s", l.server.Addr)

		err := l.httpServerServeTLS(l.server, l.listener, "", "")
		if err != nil && err != http.ErrServerClosed {
			log.Print(err)
		}
	}()

	return nil
}

// Shutdown shutdowns the listener gracefully.
func (l *tlsListener) Shutdown(ctx context.Context) error {
	err := l.httpServerShutdown(l.server, ctx)
	if err != nil {
		return err
	}

	return nil
}

// Close closes the listener.
func (l *tlsListener) Close() error {
	err := l.httpServerClose(l.server)
	if err != nil {
		return err
	}

	return nil
}

var _ core.ServerListenerModule = (*tlsListener)(nil)
