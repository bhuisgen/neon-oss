package neon

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/log"
	"github.com/bhuisgen/neon/pkg/module"
)

// app implements the app module.
type app struct {
	config *appConfig
	logger *slog.Logger
	state  *appState
}

// appConfig implements the app configuration.
type appConfig struct {
	Store   map[string]interface{}
	Fetcher map[string]interface{}
	Loader  map[string]interface{}
	Server  map[string]interface{}
}

// appState implements the app state.
type appState struct {
	listeners map[string][]net.Listener
	store     Store
	fetcher   Fetcher
	loader    Loader
	server    Server
	mediator  *appMediator
}

const (
	appModuleID module.ModuleID = "app"
)

// ModuleInfo returns the module information.
func (a app) ModuleInfo() module.ModuleInfo {
	module.Register(store{})
	module.Register(fetcher{})
	module.Register(loader{})
	module.Register(server{})

	return module.ModuleInfo{
		ID:           appModuleID,
		LoadModule:   func() {},
		UnloadModule: func() {},
		NewInstance: func() module.Module {
			return &app{
				logger: slog.New(log.NewHandler(os.Stderr, string(appModuleID), nil)),
				state:  &appState{},
			}
		},
	}
}

// Init initializes the app.
func (a *app) Init(config map[string]interface{}) error {
	a.logger.Debug("Initializing app")

	if config == nil {
		a.config = &appConfig{}
	} else {
		if err := mapstructure.Decode(config, &a.config); err != nil {
			a.logger.Error("Failed to parse configuration", "err", err)
			return fmt.Errorf("parse config: %w", err)
		}
	}

	storeModuleInfo, err := module.Lookup("app.store")
	if err != nil {
		return fmt.Errorf("lookup module %s: %w", "app.store", err)
	}
	store, ok := storeModuleInfo.NewInstance().(Store)
	if !ok {
		return errors.New("invalid store")
	}

	fetcherModuleInfo, err := module.Lookup("app.fetcher")
	if err != nil {
		return fmt.Errorf("lookup module %s: %w", "app.fetcher", err)
	}
	fetcher, ok := fetcherModuleInfo.NewInstance().(Fetcher)
	if !ok {
		return errors.New("invalid fetcher")
	}

	loaderModuleInfo, err := module.Lookup("app.loader")
	if err != nil {
		return fmt.Errorf("lookup module %s: %w", "app.loader", err)
	}
	loader, ok := loaderModuleInfo.NewInstance().(Loader)
	if !ok {
		return errors.New("invalid loader")
	}

	serverModuleInfo, err := module.Lookup("app.server")
	if err != nil {
		return fmt.Errorf("lookup module %s: %w", "app.server", err)
	}
	server, ok := serverModuleInfo.NewInstance().(Server)
	if !ok {
		return errors.New("invalid server")
	}

	a.state.store = store
	a.state.fetcher = fetcher
	a.state.loader = loader
	a.state.server = server
	a.state.mediator = newAppMediator(a)

	return nil
}

// Check checks the instance configuration.
func (a *app) Check() error {
	var errCheck bool

	if err := a.state.store.Init(a.config.Store); err != nil {
		errCheck = true
	}
	if err := a.state.fetcher.Init(a.config.Fetcher); err != nil {
		errCheck = true
	}
	if err := a.state.loader.Init(a.config.Loader); err != nil {
		errCheck = true
	}
	if err := a.state.server.Init(a.config.Server); err != nil {
		errCheck = true
	}

	if errCheck {
		return errors.New("check failure")
	}

	return nil
}

// Serve executes the instance.
func (a *app) Serve(ctx context.Context) error {

	module.Load()

	if DEBUG {
		a.logger.Warn("Debug enabled")
	}

	if key, ok := os.LookupEnv(childEnvKey); ok {
		if err := a.child(key); err != nil {
			a.logger.Error("Failed to execute child", "err", err)
			return fmt.Errorf("execute child: %v", err)
		}
	}

	a.logger.Info("Starting instance")

	if err := a.start(ctx); err != nil {
		a.logger.Error("Failed to start instance", "err", err)
		return fmt.Errorf("start instance: %v", err)
	}

	a.logger.Info("Instance ready")

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGQUIT)
	reload := make(chan os.Signal, 1)
	signal.Notify(reload, syscall.SIGHUP)

	for {
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				a.logger.Error("Context error", "err", err)
				return fmt.Errorf("context: %v", err)
			}
		case <-exit:
			a.logger.Info("Signal SIGINT/SIGTERM received, stopping instance")
			if err := a.stop(); err != nil {
				a.logger.Error("Failed to stop instance", "err", err)
				continue
			}
		case <-shutdown:
			a.logger.Info("Signal SIGQUIT received, shutting down instance gracefully")
			if err := a.shutdown(ctx); err != nil {
				a.logger.Error("Failed to shutdown instance", "err", err)
				continue
			}
		case <-reload:
			a.logger.Info("Signal SIGHUP received, reloading instance")
			if err := a.reload(ctx); err != nil {
				a.logger.Error("Failed to reload instance", "err", err)
				continue
			}
		}
		break
	}

	signal.Stop(exit)
	signal.Stop(shutdown)
	signal.Stop(reload)

	module.Unload()

	a.logger.Info("Instance terminated")

	return nil
}

// start starts the instance.
func (a *app) start(ctx context.Context) error {
	if err := a.state.store.Init(a.config.Store); err != nil {
		a.logger.Error("Failed to init store", "err", err)
		return fmt.Errorf("init store: %v", err)
	}
	if err := a.state.store.Register(a.state.mediator); err != nil {
		a.logger.Error("Failed to register store", "err", err)
		return fmt.Errorf("register store: %v", err)
	}

	if err := a.state.fetcher.Init(a.config.Fetcher); err != nil {
		return fmt.Errorf("init fetcher: %w", err)
	}
	if err := a.state.fetcher.Register(a.state.mediator); err != nil {
		a.logger.Error("Failed to register fetcher", "err", err)
		return fmt.Errorf("register fetcher: %v", err)
	}

	if err := a.state.loader.Init(a.config.Loader); err != nil {
		a.logger.Error("Failed to init loader", "err", err)
		return fmt.Errorf("init loader: %v", err)
	}
	if err := a.state.loader.Register(a.state.mediator); err != nil {
		a.logger.Error("Failed to register loader", "err", err)
		return fmt.Errorf("register loader: %v", err)
	}
	if err := a.state.loader.Start(ctx); err != nil {
		a.logger.Error("Failed to start loader", "err", err)
		return fmt.Errorf("start loader: %v", err)
	}

	if err := a.state.server.Init(a.config.Server); err != nil {
		a.logger.Error("Failed to init server", "err", err)
		return fmt.Errorf("init server: %v", err)
	}
	if err := a.state.server.Register(a.state.mediator); err != nil {
		a.logger.Error("Failed to register server", "err", err)
		return fmt.Errorf("register server: %v", err)
	}
	if err := a.state.server.Start(ctx); err != nil {
		a.logger.Error("Failed to start server", "err", err)
		return fmt.Errorf("start server: %v", err)
	}

	return nil
}

// stop stops the instance.
func (a *app) stop() error {
	if err := a.state.server.Stop(); err != nil {
		a.logger.Error("Failed to stop server", "err", err)
		return fmt.Errorf("stop server: %w", err)
	}
	if a.state.loader != nil {
		if err := a.state.loader.Stop(); err != nil {
			a.logger.Error("Failed to stop loader", "err", err)
			return fmt.Errorf("stop loader: %v", err)
		}
	}

	return nil
}

// shutdown stops the instance gracefully.
func (a *app) shutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	if err := a.state.server.Shutdown(ctx); err != nil {
		a.logger.Error("Failed to shutdown server", "err", err)
		return fmt.Errorf("shutdown server: %w", err)
	}
	if a.state.loader != nil {
		if err := a.state.loader.Stop(); err != nil {
			a.logger.Error("Failed to stop loader", "err", err)
			return fmt.Errorf("stop loader: %v", err)
		}
	}

	return nil
}

// reload reloads the instance.
func (a *app) reload(ctx context.Context) error {
	ch := make(chan string)
	errCh := make(chan error)
	stop := make(chan struct{})
	defer func() {
		close(ch)
		close(errCh)
		close(stop)
	}()

	key, err := generateKey()
	if err != nil {
		return fmt.Errorf("generate key: %w", err)
	}

	go a.listenChild(key, ch, errCh, stop)
	for {
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				a.logger.Error("Context error", "err", err)
				return fmt.Errorf("context: %v", err)
			}
		case event := <-ch:
			switch event {
			case "init":
				exe, err := os.Executable()
				if err != nil {
					return fmt.Errorf("get executable: %w", err)
				}
				env := os.Environ()
				for index, item := range env {
					if strings.HasPrefix(item, childEnvKey+"=") {
						env = append(env[:index], env[index+1:]...)
						break
					}
				}
				env = append(env, childEnvKey+"="+key)
				files := []*os.File{
					os.Stdin,
					os.Stdout,
					os.Stderr,
				}
				listeners, err := a.state.server.Listeners()
				if err != nil {
					return fmt.Errorf("get listeners: %w", err)
				}
				for _, listener := range listeners {
					for _, l := range listener {
						file, err := getListenerFile(l)
						if err != nil {
							return fmt.Errorf("get listener file: %w", err)
						}
						files = append(files, file)
					}
				}

				if _, err := os.StartProcess(exe, os.Args, &os.ProcAttr{
					Dir:   filepath.Dir(exe),
					Env:   env,
					Files: files,
					Sys:   &syscall.SysProcAttr{},
				}); err != nil {
					return fmt.Errorf("start process: %w", err)
				}
				a.logger.Info("Child process started, waiting for connection")

			case "done":
				stop <- struct{}{}
				a.logger.Info("Child process ready, stopping instance")

				if err := a.shutdown(ctx); err != nil {
					a.logger.Error("Shutdown error", "err", err)
					return fmt.Errorf("shutdown: %w", err)
				}

				return nil
			}

		case err := <-errCh:
			a.logger.Error("Child error", "err", err)
			return fmt.Errorf("reload: %w", err)
		}
	}
}

const (
	childTimeout int    = 5
	childEnvKey  string = "CHILD"

	childCommandHello  string = "HELLO"
	childCommandReload string = "RELOAD"
	childCommandReady  string = "READY"

	childResultOK    string = "OK"
	childResultError string = "ERROR"
)

// childReloadRequest implements the reload request message.
type childReloadRequest struct {
	Key string `json:"key"`
}

// childReloadResponse implements the reload response message.
type childReloadResponse struct {
	Listeners []struct {
		Name  string   `json:"name"`
		Files []string `json:"files"`
	} `json:"listeners"`
}

// listenChild listens for child connections.
func (a *app) listenChild(key string, ch chan<- string, errCh chan<- error, stop <-chan struct{}) {
	ln, err := net.Listen("unix", CHILD_SOCKET)
	if err != nil {
		errCh <- fmt.Errorf("listen: %w", err)
		return
	}
	defer func() {
		_ = ln.Close()
		_ = os.Remove(CHILD_SOCKET)
	}()

	ch <- "init"

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go a.handleChild(conn, key, ch, errCh)
		}
	}()

	select {
	case <-stop:
		return
	case <-time.After(time.Duration(childTimeout) * time.Second):
		errCh <- errors.New("timeout")
	}
}

// handleChild handles a child connection.
func (a *app) handleChild(conn net.Conn, key string, ch chan<- string, errorCh chan<- error) {
	s := bufio.NewScanner(conn)

	var hello bool
	var reload bool

	for s.Scan() {
		cmd, data, _ := bytes.Cut(s.Bytes(), []byte(":"))
		switch string(cmd) {
		case childCommandHello:
			if _, err := fmt.Fprintf(conn, "%s\r\n", childCommandHello); err != nil {
				errorCh <- fmt.Errorf("write: %w", err)
				return
			}
			hello = true

		case childCommandReload:
			if !hello {
				if _, err := fmt.Fprintf(conn, "%s\r\n", childResultError); err != nil {
					errorCh <- fmt.Errorf("write: %w", err)
					return
				}
				return
			}

			var msg childReloadRequest
			if err := json.Unmarshal(data, &msg); err != nil {
				a.logger.Debug("Failed to unmarshal data", "err", err)
				continue
			}

			if msg.Key != key {
				a.logger.Debug("Invalid key from child", "parent", key, "child", msg.Key)
				if _, err := fmt.Fprintf(conn, "%s:%s\r\n", childResultError, "invalid key"); err != nil {
					errorCh <- fmt.Errorf("write: %w", err)
					return
				}
				return
			}

			response := childReloadResponse{}
			listeners, err := a.state.server.Listeners()
			if err != nil {
				errorCh <- fmt.Errorf("get listeners: %w", err)
				return
			}
			for name, listener := range listeners {
				childListener := struct {
					Name  string   `json:"name"`
					Files []string `json:"files"`
				}{
					Name: name,
				}
				for _, l := range listener {
					file, err := getListenerFile(l)
					if err != nil {
						errorCh <- fmt.Errorf("get listener file: %w", err)
					}
					childListener.Files = append(childListener.Files, file.Name())
				}
				response.Listeners = append(response.Listeners, childListener)
			}
			data, err := json.Marshal(response)
			if err != nil {
				errorCh <- fmt.Errorf("marshal reload response: %w", err)
				return
			}
			if _, err := fmt.Fprintf(conn, "%s %s\r\n", childResultOK, data); err != nil {
				errorCh <- fmt.Errorf("write: %w", err)
				return
			}
			reload = true

		case childCommandReady:
			if !hello || !reload {
				if _, err := fmt.Fprintf(conn, "%s\r\n", childResultError); err != nil {
					errorCh <- fmt.Errorf("write: %w", err)
					return
				}
				return
			}

			if _, err := fmt.Fprintf(conn, "%s %s\r\n", childResultOK, data); err != nil {
				errorCh <- fmt.Errorf("write: %w", err)
				return
			}
			ch <- "done"

		default:
			if _, err := fmt.Fprintf(conn, "%s\r\n", childResultError); err != nil {
				errorCh <- fmt.Errorf("write: %w", err)
				return
			}
		}
	}
	if err := s.Err(); err != nil {
		errorCh <- fmt.Errorf("read: %w", err)
		return
	}
}

// child handles the connection of the child process to the parent process.
func (a *app) child(key string) error {
	conn, err := net.Dial("unix", CHILD_SOCKET)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer conn.Close()

	if _, err := fmt.Fprintf(conn, "%s\r\n", childCommandHello); err != nil {
		return fmt.Errorf("write: %v", err)
	}

	var ready bool

	s := bufio.NewScanner(conn)
	for s.Scan() {
		cmd, _, _ := bytes.Cut(s.Bytes(), []byte(":"))
		switch string(cmd) {
		case childCommandHello:
			request, err := json.Marshal(childReloadRequest{
				Key: key,
			})
			if err != nil {
				return errors.New("encode reload request")
			}
			if _, err := fmt.Fprintf(conn, "%s:%s\r\n", childCommandReload, request); err != nil {
				return fmt.Errorf("send reload: %w", err)
			}

			if !s.Scan() {
				break
			}
			reload, reloadData, _ := bytes.Cut(s.Bytes(), []byte(" "))
			if string(reload) != childResultOK {
				return errors.New("reload")
			}
			var response childReloadResponse
			if err := json.Unmarshal(reloadData, &response); err != nil {
				return fmt.Errorf("decode reload response: %w", err)
			}
			var fdsIndex int = 3
			a.state.listeners = make(map[string][]net.Listener, len(response.Listeners))
			for _, listener := range response.Listeners {
				for _, file := range listener.Files {
					f := os.NewFile(uintptr(fdsIndex), file)
					fdsIndex++
					l, err := net.FileListener(f)
					if err != nil {
						return fmt.Errorf("copy network listener: %w", err)
					}
					a.state.listeners[listener.Name] = append(a.state.listeners[listener.Name], l)
				}
			}
			if _, err := fmt.Fprintf(conn, "%s\r\n", childCommandReady); err != nil {
				return fmt.Errorf("send ready: %w", err)
			}

			if !s.Scan() {
				break
			}
			ready, _, _ := bytes.Cut(s.Bytes(), []byte(" "))
			if string(ready) != childResultOK {
				return errors.New("ready")
			}

			return nil

		default:
		}
	}
	if err := s.Err(); err != nil {
		return fmt.Errorf("read: %w", err)
	}

	if !ready {
		return errors.New("not ready")
	}

	return nil
}

// generateKey generates a random secret key for the instance reloading.
func generateKey() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("rand: %w", err)
	}
	return base64.StdEncoding.EncodeToString([]byte(b)), nil
}

// getListenerFile returns the file of a listener.
func getListenerFile(listener net.Listener) (*os.File, error) {
	switch v := listener.(type) {
	case *net.TCPListener:
		file, err := v.File()
		if err != nil {
			return nil, fmt.Errorf("get file descriptor: %w", err)
		}
		return file, nil
	case *net.UnixListener:
		file, err := v.File()
		if err != nil {
			return nil, fmt.Errorf("get file descriptor: %w", err)
		}
		return file, nil
	default:
		return nil, errors.New("invalid listener type")
	}
}

var _ App = (*app)(nil)

// appMediator implements the app mediator.
type appMediator struct {
	app *app
	mu  sync.RWMutex
}

// newAppMediator creates a new mediator.
func newAppMediator(app *app) *appMediator {
	return &appMediator{
		app: app,
	}
}

// Returns the store.
func (m *appMediator) Store() core.Store {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.app.state.store
}

// Returns the fetcher.
func (m *appMediator) Fetcher() core.Fetcher {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.app.state.fetcher
}

// Returns the loader.
func (m *appMediator) Loader() core.Loader {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.app.state.loader
}

// Returns the server.
func (m *appMediator) Server() core.Server {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.app.state.server
}

// Returns the network listeners.
func (m *appMediator) Listeners() map[string][]net.Listener {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.app.state.listeners
}

// RegisterListeners registers the network listeners.
func (m *appMediator) RegisterListeners(listeners map[string][]net.Listener) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.app.state.listeners = listeners

	return nil
}

var _ core.App = (*appMediator)(nil)
