package local

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
)

// localListener implements the local listener.
type localListener struct {
	config             *localListenerConfig
	logger             *slog.Logger
	listener           net.Listener
	server             *http.Server
	osReadFile         func(name string) ([]byte, error)
	netListen          func(network string, addr string) (net.Listener, error)
	httpServerServe    func(server *http.Server, listener net.Listener) error
	httpServerShutdown func(server *http.Server, context context.Context) error
	httpServerClose    func(server *http.Server) error
}

// localListenerConfig implements the local listener configuration.
type localListenerConfig struct {
	ListenAddr        *string `mapstructure:"listenAddr"`
	ListenPort        *int    `mapstructure:"listenPort"`
	ReadTimeout       *int    `mapstructure:"readTimeout"`
	ReadHeaderTimeout *int    `mapstructure:"readHeaderTimeout"`
	WriteTimeout      *int    `mapstructure:"writeTimeout"`
	IdleTimeout       *int    `mapstructure:"idleTimeout"`
}

const (
	localModuleID module.ModuleID = "server.listener.local"

	localConfigDefaultListenAddr        string = ""
	localConfigDefaultListenPort        int    = 80
	localConfigDefaultReadTimeout       int    = 60
	localConfigDefaultReadHeaderTimeout int    = 10
	localConfigDefaultWriteTimeout      int    = 60
	localConfigDefaultIdleTimeout       int    = 60
)

// localOsReadFile redirects to os.ReadFile.
func localOsReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// localNetListen redirects to net.Listen.
func localNetListen(network string, addr string) (net.Listener, error) {
	return net.Listen(network, addr)
}

// localHttpServerServe redirects to http.Server.Serve.
func localHttpServerServe(server *http.Server, listener net.Listener) error {
	return server.Serve(listener)
}

// localHttpServerShutdown redirects to http.Server.Shutdown.
func localHttpServerShutdown(server *http.Server, context context.Context) error {
	return server.Shutdown(context)
}

// localHttpServerShutdown redirects to http.Server.Close.
func localHttpServerClose(server *http.Server) error {
	return server.Close()
}

// init initializes the module.
func init() {
	module.Register(localListener{})
}

// ModuleInfo returns the module information.
func (l localListener) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: localModuleID,
		NewInstance: func() module.Module {
			return &localListener{
				osReadFile:         localOsReadFile,
				netListen:          localNetListen,
				httpServerServe:    localHttpServerServe,
				httpServerShutdown: localHttpServerShutdown,
				httpServerClose:    localHttpServerClose,
			}
		},
	}
}

// Init initializes the listener.
func (l *localListener) Init(config map[string]interface{}, logger *slog.Logger) error {
	l.logger = logger

	if err := mapstructure.Decode(config, &l.config); err != nil {
		l.logger.Error("Failed to parse configuration", "err", err)
		return fmt.Errorf("parse config: %v", err)
	}

	var errConfig bool

	if l.config.ListenAddr == nil {
		defaultValue := localConfigDefaultListenAddr
		l.config.ListenAddr = &defaultValue
	}
	if l.config.ListenPort == nil {
		defaultValue := localConfigDefaultListenPort
		l.config.ListenPort = &defaultValue
	}
	if *l.config.ListenPort < 0 {
		l.logger.Error("Invalid value", "option", "ListenPort", "value", *l.config.ListenPort)
		errConfig = true
	}
	if l.config.ReadTimeout == nil {
		defaultValue := localConfigDefaultReadTimeout
		l.config.ReadTimeout = &defaultValue
	}
	if *l.config.ReadTimeout < 0 {
		l.logger.Error("Invalid value", "option", "ReadTimeout", "value", *l.config.ReadTimeout)
		errConfig = true
	}
	if l.config.ReadHeaderTimeout == nil {
		defaultValue := localConfigDefaultReadHeaderTimeout
		l.config.ReadHeaderTimeout = &defaultValue
	}
	if *l.config.ReadHeaderTimeout < 0 {
		l.logger.Error("Invalid value", "option", "ReadHeaderTimeout", "value", *l.config.ReadHeaderTimeout)
		errConfig = true
	}
	if l.config.WriteTimeout == nil {
		defaultValue := localConfigDefaultWriteTimeout
		l.config.WriteTimeout = &defaultValue
	}
	if *l.config.WriteTimeout < 0 {
		l.logger.Error("Invalid value", "option", "WriteTimeout", "value", *l.config.WriteTimeout)
		errConfig = true
	}
	if l.config.IdleTimeout == nil {
		defaultValue := localConfigDefaultIdleTimeout
		l.config.IdleTimeout = &defaultValue
	}
	if *l.config.IdleTimeout < 0 {
		l.logger.Error("Invalid value", "option", "IdleTimeout", "value", *l.config.IdleTimeout)
		errConfig = true
	}

	if errConfig {
		return errors.New("config")
	}

	return nil
}

// Register registers the listener.
func (l *localListener) Register(listener core.ServerListener) error {
	if len(listener.Descriptors()) == 1 {
		l.listener = listener.Descriptors()[0]
		return nil
	}

	var err error
	l.listener, err = l.netListen("tcp", fmt.Sprintf("%s:%d", *l.config.ListenAddr, *l.config.ListenPort))
	if err != nil {
		return fmt.Errorf("listen: %v", err)
	}

	if err = listener.RegisterListener(l.listener); err != nil {
		return fmt.Errorf("register listener: %v", err)
	}

	return nil
}

// Serve accepts incoming connections.
func (l *localListener) Serve(handler http.Handler) error {
	l.server = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", *l.config.ListenAddr, *l.config.ListenPort),
		Handler:           handler,
		ReadTimeout:       time.Duration(*l.config.ReadTimeout) * time.Second,
		ReadHeaderTimeout: time.Duration(*l.config.ReadHeaderTimeout) * time.Second,
		WriteTimeout:      time.Duration(*l.config.WriteTimeout) * time.Second,
		IdleTimeout:       time.Duration(*l.config.IdleTimeout) * time.Second,
	}

	go func() {
		l.logger.Info("Starting accepting connections", "addr", l.server.Addr)

		if err := l.httpServerServe(l.server, l.listener); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				l.logger.Error("Service error", "err", err)
			}
		}
	}()

	return nil
}

// Shutdown shutdowns the listener gracefully.
func (l *localListener) Shutdown(ctx context.Context) error {
	if err := l.httpServerShutdown(l.server, ctx); err != nil {
		return fmt.Errorf("shutdown listener: %v", err)
	}

	return nil
}

// Close closes the listener.
func (l *localListener) Close() error {
	if err := l.httpServerClose(l.server); err != nil {
		return fmt.Errorf("close listener: %v", err)
	}

	return nil
}

var _ core.ServerListenerModule = (*localListener)(nil)
