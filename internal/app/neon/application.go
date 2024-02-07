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
	"syscall"
	"time"
)

// application implements the application.
type application struct {
	config  *config
	logger  *slog.Logger
	state   *applicationState
	store   *store
	fetcher *fetcher
	loader  *loader
	server  *server
}

// applicationState implements the application state.
type applicationState struct {
	serverListeners map[string][]net.Listener
}

const (
	applicationLogger string = "app"
)

// NewApplication creates a new application.
func NewApplication(config *config) *application {
	if _, ok := os.LookupEnv("DEBUG"); ok {
		DEBUG = true
	}
	if v, ok := os.LookupEnv("CHILD_SOCKET"); ok {
		CHILD_SOCKET = v
	}

	if DEBUG {
		programLevel.Set(slog.LevelDebug)
	}

	return &application{
		config: config,
		logger: slog.New(NewLogHandler(os.Stderr, applicationLogger, nil)),
	}
}

// Check checks the instance configuration.
func (a *application) Check() error {
	a.store = newStore()
	a.fetcher = newFetcher()
	a.loader = newLoader(a.store, a.fetcher)
	a.server = newServer(a.store, a.fetcher, a.loader)

	var errCheck bool

	if err := a.checkStore(a.store, a.config.Store.Config); err != nil {
		errCheck = true
	}
	if err := a.checkFetcher(a.fetcher, a.config.Fetcher.Config); err != nil {
		errCheck = true
	}
	if err := a.checkLoader(a.loader, a.config.Loader.Config); err != nil {
		errCheck = true
	}
	if err := a.checkServer(a.server, a.config.Server.Config); err != nil {
		errCheck = true
	}

	if errCheck {
		return errors.New("check failure")
	}

	return nil
}

// Serve executes the instance.
func (a *application) Serve() error {
	a.logger.Info(fmt.Sprintf("%s version %s, commit %s", Name, Version, Commit))

	a.logger.Info("Starting instance")

	if DEBUG {
		a.logger.Warn("Debug enabled")
	}

	a.state = &applicationState{}

	if key, ok := os.LookupEnv(childEnvKey); ok {
		if err := a.child(key); err != nil {
			a.logger.Error("Failed to execute child", "err", err)
			return fmt.Errorf("execute child: %v", err)
		}
	}

	a.store = newStore()
	a.fetcher = newFetcher()
	a.loader = newLoader(a.store, a.fetcher)
	a.server = newServer(a.store, a.fetcher, a.loader)

	if err := a.startStore(a.store, a.config.Store.Config); err != nil {
		a.logger.Error("Failed to start store", "err", err)
		return fmt.Errorf("start store: %v", err)
	}
	if err := a.startFetcher(a.fetcher, a.config.Fetcher.Config); err != nil {
		a.logger.Error("Failed to start fetcher", "err", err)
		return fmt.Errorf("start fetcher: %v", err)
	}
	if err := a.startLoader(a.loader, a.config.Loader.Config); err != nil {
		a.logger.Error("Failed to start loader", "err", err)
		return fmt.Errorf("start loader: %v", err)
	}
	if err := a.startServer(a.server, a.config.Server.Config); err != nil {
		a.logger.Error("Failed to start server", "err", err)
		return fmt.Errorf("start server: %v", err)
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
		case <-exit:
			a.logger.Info("Signal SIGINT/SIGTERM received, stopping instance")
			if err := a.stop(); err != nil {
				a.logger.Error("stop instance", "err", err)
				continue
			}

		case <-shutdown:
			a.logger.Info("Signal SIGQUIT received, shutting down instance gracefully")
			if err := a.shutdown(); err != nil {
				a.logger.Error("shutdown instance", "err", err)
				continue
			}

		case <-reload:
			a.logger.Info("Signal SIGHUP received, reloading instance")
			if err := a.reload(); err != nil {
				a.logger.Error("reload instance", "err", err)
				continue
			}
		}

		break
	}

	signal.Stop(exit)
	signal.Stop(shutdown)
	signal.Stop(reload)

	a.logger.Info("Instance terminated")

	return nil
}

// stop stops the instance.
func (a *application) stop() error {
	if err := a.stopServer(a.server); err != nil {
		a.logger.Error("Failed to stop server", "err", err)
		return fmt.Errorf("stop server: %v", err)
	}
	if a.loader != nil {
		if err := a.stopLoader(a.loader); err != nil {
			a.logger.Error("Failed to stop loader", "err", err)
			return fmt.Errorf("stop loader: %v", err)
		}
	}
	if err := a.stopFetcher(a.fetcher); err != nil {
		a.logger.Error("Failed to stop fetcher", "err", err)
		return fmt.Errorf("stop fetcher: %v", err)
	}
	if err := a.stopStore(a.store); err != nil {
		a.logger.Error("Failed to stop store", "err", err)
		return fmt.Errorf("stop store: %v", err)
	}

	return nil
}

// shutdown stops the instance gracefully.
func (a *application) shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := a.shutdownServer(ctx, a.server); err != nil {
		a.logger.Error("Failed to shutdown server", "err", err)
		return fmt.Errorf("shutdown server: %v", err)
	}
	if a.loader != nil {
		if err := a.stopLoader(a.loader); err != nil {
			a.logger.Error("Failed to stop loader", "err", err)
			return fmt.Errorf("stop loader: %v", err)
		}
	}
	if err := a.stopFetcher(a.fetcher); err != nil {
		a.logger.Error("Failed to stop fetcher", "err", err)
		return fmt.Errorf("stop fetcher: %v", err)
	}
	if err := a.stopStore(a.store); err != nil {
		a.logger.Error("Failed to stop store", "err", err)
		return fmt.Errorf("stop store: %v", err)
	}

	return nil
}

// reload reloads the instance.
func (a *application) reload() error {
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
				for _, listener := range a.server.state.listenersMap {
					listeners, err := listener.Listeners()
					if err != nil {
						return fmt.Errorf("get listener descriptor: %w", err)
					}
					for _, l := range listeners {
						switch v := l.(type) {
						case *net.TCPListener:
							file, err := v.File()
							if err != nil {
								return fmt.Errorf("get listener file: %w", err)
							}
							files = append(files, file)
						case *net.UnixListener:
							file, err := v.File()
							if err != nil {
								return fmt.Errorf("get listener file: %w", err)
							}
							files = append(files, file)
						default:
							return errors.New("invalid listener type")
						}
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

				if err := a.shutdown(); err != nil {
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

// checkStore checks the store configuration.
func (a *application) checkStore(store Store, config map[string]interface{}) error {
	if err := store.Init(config); err != nil {
		return fmt.Errorf("init store: %w", err)
	}

	return nil
}

// startStore starts the store.
func (a *application) startStore(store Store, config map[string]interface{}) error {
	if err := store.Init(config); err != nil {
		return fmt.Errorf("init store: %w", err)
	}

	return nil
}

// stopStore stops the store.
func (a *application) stopStore(store Store) error {
	return nil
}

// checkFetcher checks the fetcher configuration.
func (a *application) checkFetcher(fetcher Fetcher, config map[string]interface{}) error {
	if err := fetcher.Init(config); err != nil {
		return fmt.Errorf("init fetcher: %w", err)
	}

	return nil
}

// startFetcher starts the fetcher.
func (a *application) startFetcher(fetcher Fetcher, config map[string]interface{}) error {
	if err := fetcher.Init(config); err != nil {
		return fmt.Errorf("init fetcher: %w", err)
	}

	return nil
}

// stopFetcher stops the fetcher.
func (a *application) stopFetcher(fetcher Fetcher) error {
	return nil
}

// checkLoader checks the loader configuration.
func (a *application) checkLoader(loader Loader, config map[string]interface{}) error {
	if err := loader.Init(config); err != nil {
		return fmt.Errorf("init loader: %w", err)
	}

	return nil
}

// startLoader starts the loader.
func (a *application) startLoader(loader Loader, config map[string]interface{}) error {
	if err := loader.Init(config); err != nil {
		return fmt.Errorf("init loader: %w", err)
	}
	if err := loader.Start(); err != nil {
		return fmt.Errorf("start loader: %w", err)
	}

	return nil
}

// stopLoader stops the loader.
func (a *application) stopLoader(loader Loader) error {
	if err := loader.Stop(); err != nil {
		return fmt.Errorf("stop loader: %w", err)
	}

	return nil
}

// checkServer checks the server configuration.
func (a *application) checkServer(server Server, config map[string]interface{}) error {
	if err := server.Init(config); err != nil {
		return fmt.Errorf("init server: %w", err)
	}

	return nil
}

// startServer starts the server.
func (a *application) startServer(server Server, config map[string]interface{}) error {
	if err := server.Init(config); err != nil {
		return fmt.Errorf("init server: %w", err)
	}
	if err := server.Register(a.state.serverListeners); err != nil {
		return fmt.Errorf("register server: %w", err)
	}
	if err := server.Start(); err != nil {
		return fmt.Errorf("start server: %w", err)
	}

	return nil
}

// stopServer stops the server.
func (a *application) stopServer(server Server) error {
	if err := server.Stop(); err != nil {
		return fmt.Errorf("stop server: %w", err)
	}

	return nil
}

// shutdownServer shutdowns the server gracefully.
func (a *application) shutdownServer(ctx context.Context, server Server) error {
	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	return nil
}

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

const (
	childTimeout int    = 5
	childEnvKey  string = "CHILD"

	childCommandHello  string = "HELLO"
	childCommandReload string = "RELOAD"
	childCommandReady  string = "READY"

	childResultOK    string = "OK"
	childResultError string = "ERROR"
)

// listenChild listens for child connections.
func (a *application) listenChild(key string, ch chan<- string, errCh chan<- error, stop <-chan struct{}) {
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
func (a *application) handleChild(conn net.Conn, key string, ch chan<- string, errorCh chan<- error) {
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
			for _, listener := range a.server.state.listenersMap {
				helloListener := struct {
					Name  string   `json:"name"`
					Files []string `json:"files"`
				}{
					Name: listener.Name(),
				}

				listeners, err := listener.Listeners()
				if err != nil {
					errorCh <- fmt.Errorf("get listeners: %w", err)
					return
				}
				for _, l := range listeners {
					switch v := l.(type) {
					case *net.TCPListener:
						file, err := v.File()
						if err != nil {
							errorCh <- fmt.Errorf("get listener file: %w", err)
							return
						}
						helloListener.Files = append(helloListener.Files, file.Name())
					case *net.UnixListener:
						file, err := v.File()
						if err != nil {
							errorCh <- fmt.Errorf("get listener file: %w", err)
							return
						}
						helloListener.Files = append(helloListener.Files, file.Name())
					default:
						errorCh <- errors.New("invalid listener type")
						return
					}
				}
				response.Listeners = append(response.Listeners, helloListener)
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
func (a *application) child(key string) error {
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
				Key: os.Getenv(childEnvKey),
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
			a.state.serverListeners = make(map[string][]net.Listener, len(response.Listeners))
			for _, listener := range response.Listeners {
				for _, file := range listener.Files {
					f := os.NewFile(uintptr(fdsIndex), file)
					fdsIndex++
					l, err := net.FileListener(f)
					if err != nil {
						return fmt.Errorf("copy network listener: %w", err)
					}
					a.state.serverListeners[listener.Name] = append(a.state.serverListeners[listener.Name], l)
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
		return fmt.Errorf("not ready")
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

var _ (Application) = (*application)(nil)
