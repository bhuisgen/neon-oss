package redirect

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/log"
	"github.com/bhuisgen/neon/pkg/module"
)

// redirectListener implements the redirect listener.
type redirectListener struct {
	config             *redirectListenerConfig
	logger             *slog.Logger
	listener           net.Listener
	server             *http.Server
	osReadFile         func(name string) ([]byte, error)
	netListen          func(network string, addr string) (net.Listener, error)
	httpServerServe    func(server *http.Server, listener net.Listener) error
	httpServerShutdown func(server *http.Server, context context.Context) error
	httpServerClose    func(server *http.Server) error
}

// redirectListenerConfig implements the redirect listener configuration.
type redirectListenerConfig struct {
	ListenAddr        *string `mapstructure:"listenAddr"`
	ListenPort        *int    `mapstructure:"listenPort"`
	ReadTimeout       *int    `mapstructure:"readTimeout"`
	ReadHeaderTimeout *int    `mapstructure:"readHeaderTimeout"`
	WriteTimeout      *int    `mapstructure:"writeTimeout"`
	IdleTimeout       *int    `mapstructure:"idleTimeout"`
	RedirectPort      *int    `mapstructure:"redirectPort"`
}

const (
	redirectModuleID module.ModuleID = "app.server.listener.redirect"

	redirectConfigDefaultListenAddr        string = ""
	redirectConfigDefaultListenPort        int    = 80
	redirectConfigDefaultReadTimeout       int    = 60
	redirectConfigDefaultReadHeaderTimeout int    = 10
	redirectConfigDefaultWriteTimeout      int    = 60
	redirectConfigDefaultIdleTimeout       int    = 60
)

// redirectOsReadFile redirects to os.ReadFile.
func redirectOsReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// redirectNetListen redirects to net.Listen.
func redirectNetListen(network string, addr string) (net.Listener, error) {
	return net.Listen(network, addr)
}

// redirectHttpServerServe redirects to http.Server.Serve.
func redirectHttpServerServe(server *http.Server, listener net.Listener) error {
	return server.Serve(listener)
}

// redirectHttpServerShutdown redirects to http.Server.Shutdown.
func redirectHttpServerShutdown(server *http.Server, context context.Context) error {
	return server.Shutdown(context)
}

// redirectHttpServerClose redirects to http.Server.Close.
func redirectHttpServerClose(server *http.Server) error {
	return server.Close()
}

// init initializes the package.
func init() {
	module.Register(redirectListener{})
}

// ModuleInfo returns the module information.
func (l redirectListener) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID:           redirectModuleID,
		LoadModule:   func() {},
		UnloadModule: func() {},
		NewInstance: func() module.Module {
			return &redirectListener{
				logger:             slog.New(log.NewHandler(os.Stderr, string(redirectModuleID), nil)),
				osReadFile:         redirectOsReadFile,
				netListen:          redirectNetListen,
				httpServerServe:    redirectHttpServerServe,
				httpServerShutdown: redirectHttpServerShutdown,
				httpServerClose:    redirectHttpServerClose,
			}
		},
	}
}

// Init initializes the listener.
func (l *redirectListener) Init(config map[string]interface{}) error {
	if err := mapstructure.Decode(config, &l.config); err != nil {
		l.logger.Error("Failed to parse configuration", "err", err)
		return fmt.Errorf("parse config: %v", err)
	}

	var errConfig bool

	if l.config.ListenAddr == nil {
		defaultValue := redirectConfigDefaultListenAddr
		l.config.ListenAddr = &defaultValue
	}
	if l.config.ListenPort == nil {
		defaultValue := redirectConfigDefaultListenPort
		l.config.ListenPort = &defaultValue
	}
	if *l.config.ListenPort < 0 {
		l.logger.Error("Invalid value", "option", "ListenPort", "value", *l.config.ListenPort)
		errConfig = true
	}
	if l.config.ReadTimeout == nil {
		defaultValue := redirectConfigDefaultReadTimeout
		l.config.ReadTimeout = &defaultValue
	}
	if *l.config.ReadTimeout < 0 {
		l.logger.Error("Invalid value", "option", "ReadTimeout", "value", *l.config.ReadTimeout)
		errConfig = true
	}
	if l.config.ReadHeaderTimeout == nil {
		defaultValue := redirectConfigDefaultReadHeaderTimeout
		l.config.ReadHeaderTimeout = &defaultValue
	}
	if *l.config.ReadHeaderTimeout < 0 {
		l.logger.Error("Invalid value", "option", "ReadHeaderTimeout", "value", *l.config.ReadHeaderTimeout)
		errConfig = true
	}
	if l.config.WriteTimeout == nil {
		defaultValue := redirectConfigDefaultWriteTimeout
		l.config.WriteTimeout = &defaultValue
	}
	if *l.config.WriteTimeout < 0 {
		l.logger.Error("Invalid value", "option", "WriteTimeout", "value", *l.config.WriteTimeout)
		errConfig = true
	}
	if l.config.IdleTimeout == nil {
		defaultValue := redirectConfigDefaultIdleTimeout
		l.config.IdleTimeout = &defaultValue
	}
	if *l.config.IdleTimeout < 0 {
		l.logger.Error("Invalid value", "option", "IdleTimeout", "value", *l.config.IdleTimeout)
		errConfig = true
	}
	if l.config.RedirectPort != nil && *l.config.RedirectPort < 0 {
		l.logger.Error("Invalid value", "option", "RedirectPort", "value", *l.config.RedirectPort)
		errConfig = true
	}

	if errConfig {
		return errors.New("config")
	}

	return nil
}

// Register registers the listener.
func (l *redirectListener) Register(listener core.ServerListener) error {
	listeners := listener.Listeners()
	if len(listeners) == 1 {
		l.listener = listeners[0]
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
func (l *redirectListener) Serve(handler http.Handler) error {
	l.server = &http.Server{
		Addr:                         fmt.Sprintf("%s:%d", *l.config.ListenAddr, *l.config.ListenPort),
		Handler:                      http.HandlerFunc(l.redirectHandler),
		ReadTimeout:                  time.Duration(*l.config.ReadTimeout) * time.Second,
		ReadHeaderTimeout:            time.Duration(*l.config.ReadHeaderTimeout) * time.Second,
		WriteTimeout:                 time.Duration(*l.config.WriteTimeout) * time.Second,
		IdleTimeout:                  time.Duration(*l.config.IdleTimeout) * time.Second,
		DisableGeneralOptionsHandler: true,
	}

	go func() {
		l.logger.Info("Starting listener", "addr", l.server.Addr)

		if err := l.httpServerServe(l.server, l.listener); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				l.logger.Error("Service error", "err", err)
			}
		}
	}()

	return nil
}

// Shutdown shutdowns the listener gracefully.
func (l *redirectListener) Shutdown(ctx context.Context) error {
	if err := l.httpServerShutdown(l.server, ctx); err != nil {
		return fmt.Errorf("shutdown listener: %v", err)
	}

	return nil
}

// Close closes the listener.
func (l *redirectListener) Close() error {
	if err := l.httpServerClose(l.server); err != nil {
		return fmt.Errorf("close listener: %v", err)
	}

	return nil
}

// redirectHandler is the redirect handler.
func (l *redirectListener) redirectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "HEAD" {
		http.Error(w, "Use HTTPS", http.StatusBadRequest)
		return
	}

	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}

	var target string
	if l.config.RedirectPort == nil {
		target = "https://" + host + r.URL.RequestURI()
	} else {
		target = "https://" + net.JoinHostPort(host, strconv.Itoa(*l.config.RedirectPort)) + r.URL.RequestURI()
	}

	http.Redirect(w, r, target, http.StatusFound)
}

var _ core.ServerListenerModule = (*redirectListener)(nil)
