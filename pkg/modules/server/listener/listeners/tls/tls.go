package tls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
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
	logger                         *slog.Logger
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
	ListenAddr        *string   `mapstructure:"listenAddr"`
	ListenPort        *int      `mapstructure:"listenPort"`
	CAFiles           *[]string `mapstructure:"caFiles"`
	CertFiles         []string  `mapstructure:"certFiles"`
	KeyFiles          []string  `mapstructure:"keyFiles"`
	ClientAuth        *string   `mapstructure:"clientAuth"`
	ReadTimeout       *int      `mapstructure:"readTimeout"`
	ReadHeaderTimeout *int      `mapstructure:"readHeaderTimeout"`
	WriteTimeout      *int      `mapstructure:"writeTimeout"`
	IdleTimeout       *int      `mapstructure:"idleTimeout"`
}

const (
	tlsModuleID module.ModuleID = "server.listener.tls"

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

// Init initializes the listener.
func (l *tlsListener) Init(config map[string]interface{}, logger *slog.Logger) error {
	l.logger = logger

	if err := mapstructure.Decode(config, &l.config); err != nil {
		l.logger.Error("Failed to parse configuration")
		return err
	}

	var errInit bool

	if l.config.ListenAddr == nil {
		defaultValue := tlsConfigDefaultListenAddr
		l.config.ListenAddr = &defaultValue
	}
	if l.config.ListenPort == nil {
		defaultValue := tlsConfigDefaultListenPort
		l.config.ListenPort = &defaultValue
	}
	if *l.config.ListenPort < 0 {
		l.logger.Error("Invalid value", "option", "ListenPort", "value", *l.config.ListenPort)
		errInit = true
	}
	if l.config.CAFiles != nil {
		for _, item := range *l.config.CAFiles {
			if item == "" {
				l.logger.Error("Invalid value", "option", "CAFiles", "value", item)
				errInit = true
				continue
			}
			f, err := l.osOpenFile(item, os.O_RDONLY, 0)
			if err != nil {
				l.logger.Error("Failed to open file", "option", "CAFiles", "value", item)
				errInit = true
				continue
			}
			_ = l.osClose(f)
			fi, err := l.osStat(item)
			if err != nil {
				l.logger.Error("Failed to stat file", "option", "CAFiles", "value", item)
				errInit = true
				continue
			}
			if err == nil && fi.IsDir() {
				l.logger.Error("File is a directory", "option", "CAFiles", "value", item)
				errInit = true
				continue
			}
		}
	}
	if len(l.config.CertFiles) == 0 {
		l.logger.Error("Missing value(s)", "option", "CertFiles")
		errInit = true
	}
	for _, item := range l.config.CertFiles {
		if item == "" {
			l.logger.Error("Invalid value", "option", "CertFiles", "value", item)
			errInit = true
			continue
		}
		f, err := l.osOpenFile(item, os.O_RDONLY, 0)
		if err != nil {
			l.logger.Error("Failed to open file", "option", "CertFiles", "value", item)
			errInit = true
			continue
		}
		_ = l.osClose(f)
		fi, err := l.osStat(item)
		if err != nil {
			l.logger.Error("Failed to stat file", "option", "CertFiles", "value", item)
			errInit = true
			continue
		}
		if err == nil && fi.IsDir() {
			l.logger.Error("File is a directory", "option", "CertFiles", "value", item)
			errInit = true
			continue
		}
	}
	if len(l.config.KeyFiles) == 0 || len(l.config.KeyFiles) != len(l.config.CertFiles) {
		l.logger.Error("Missing value(s)", "option", "KeyFiles")
		errInit = true
	}
	for _, item := range l.config.KeyFiles {
		if item == "" {
			l.logger.Error("Invalid value", "option", "KeyFiles", "value", item)
			errInit = true
			continue
		}
		f, err := l.osOpenFile(item, os.O_RDONLY, 0)
		if err != nil {
			l.logger.Error("Failed to open file", "option", "KeyFiles", "value", item)
			errInit = true
			continue
		}
		_ = l.osClose(f)
		fi, err := l.osStat(item)
		if err != nil {
			l.logger.Error("Failed to stat file", "option", "KeyFiles", "value", item)
			errInit = true
			continue
		}
		if err == nil && fi.IsDir() {
			l.logger.Error("File is a directory", "option", "KeyFiles", "value", item)
			errInit = true
			continue
		}
	}
	if l.config.ClientAuth == nil {
		defaultValue := tlsConfigDefaultClientAuth
		l.config.ClientAuth = &defaultValue
	}
	switch *l.config.ClientAuth {
	case tlsClientAuthNone:
	case tlsClientAuthRequest:
	case tlsClientAuthRequire:
	case tlsClientAuthVerify:
	case tlsClientAuthRequireAndVerify:
	default:
		l.logger.Error("Invalid value", "option", "ClientAuth", "value", *l.config.ClientAuth)
		errInit = true
	}
	if l.config.ReadTimeout == nil {
		defaultValue := tlsConfigDefaultReadTimeout
		l.config.ReadTimeout = &defaultValue
	}
	if *l.config.ReadTimeout < 0 {
		l.logger.Error("Invalid value", "option", "ReadTimeout", "value", *l.config.ReadTimeout)
		errInit = true
	}
	if l.config.ReadHeaderTimeout == nil {
		defaultValue := tlsConfigDefaultReadHeaderTimeout
		l.config.ReadHeaderTimeout = &defaultValue
	}
	if *l.config.ReadHeaderTimeout < 0 {
		l.logger.Error("Invalid value", "option", "ReadHeaderTimeout", "value", *l.config.ReadHeaderTimeout)
		errInit = true
	}
	if l.config.WriteTimeout == nil {
		defaultValue := tlsConfigDefaultWriteTimeout
		l.config.WriteTimeout = &defaultValue
	}
	if *l.config.WriteTimeout < 0 {
		l.logger.Error("Invalid value", "option", "WriteTimeout", "value", *l.config.WriteTimeout)
		errInit = true
	}
	if l.config.IdleTimeout == nil {
		defaultValue := tlsConfigDefaultIdleTimeout
		l.config.IdleTimeout = &defaultValue
	}
	if *l.config.IdleTimeout < 0 {
		l.logger.Error("Invalid value", "option", "IdleTimeout", "value", *l.config.IdleTimeout)
		errInit = true
	}

	if errInit {
		return errors.New("init error")
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
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

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

	tlsConfig.Certificates = make([]tls.Certificate, len(l.config.CertFiles))
	for i := range l.config.CertFiles {
		var err error
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
		l.logger.Info("Starting listener", "addr", l.server.Addr)

		if err := l.httpServerServeTLS(l.server, l.listener, "", ""); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				l.logger.Error("Service error", "err", err)
			}
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
