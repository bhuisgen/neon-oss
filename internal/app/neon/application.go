// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package neon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/bhuisgen/neon/pkg/core"
)

// application implements the application.
type application struct {
	config  *config
	logger  *log.Logger
	state   *applicationState
	store   *store
	fetcher *fetcher
	loader  *loader
	server  *server
}

// applicationState implements the application state.
type applicationState struct {
	serverListenersDescriptors map[string]ServerListenerDescriptor
}

const (
	applicationLogger string = "app"
)

// NewApplication creates a new application.
func NewApplication(config *config) *application {
	return &application{
		config: config,
		logger: log.New(os.Stderr, fmt.Sprint(applicationLogger, ": "), log.LstdFlags|log.Lmsgprefix),
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
	a.logger.Printf("%s version %s, commit %s", Name, Version, Commit)

	if core.DEBUG {
		a.logger.Print("debug enabled")
	}

	a.state = &applicationState{}

	if _, ok := os.LookupEnv(childEnvKey); ok {
		if err := a.child(); err != nil {
			return err
		}
	}

	a.store = newStore()
	a.fetcher = newFetcher()
	a.loader = newLoader(a.store, a.fetcher)
	a.server = newServer(a.store, a.fetcher, a.loader)

	if err := a.startStore(a.store, a.config.Store.Config); err != nil {
		return err
	}
	if err := a.startFetcher(a.fetcher, a.config.Fetcher.Config); err != nil {
		return err
	}
	if err := a.startLoader(a.loader, a.config.Loader.Config); err != nil {
		return err
	}
	if err := a.startServer(a.server, a.config.Server.Config); err != nil {
		return err
	}

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGQUIT)
	reload := make(chan os.Signal, 1)
	signal.Notify(reload, syscall.SIGHUP)

	for {
		select {
		case <-exit:
			a.logger.Print("Signal SIGTERM received, stopping instance")
			if err := a.stop(); err != nil {
				a.logger.Printf("Failed to stop the instance: %s", err)
			}

		case <-shutdown:
			a.logger.Print("Signal SIGQUIT received, stopping instance gracefully")
			if err := a.shutdown(); err != nil {
				a.logger.Printf("Failed to shutdown the instance: %s", err)
			}

		case <-reload:
			a.logger.Print("Signal SIGHUP received, reloading instance")
			if err := a.reload(); err != nil {
				a.logger.Printf("Failed to reload instance: %s", err)
				continue
			}
		}

		break
	}

	signal.Stop(exit)
	signal.Stop(shutdown)
	signal.Stop(reload)

	a.logger.Print("Instance terminated")

	return nil
}

// stop stops the application.
func (a *application) stop() error {
	if err := a.stopServer(a.server); err != nil {
		return err
	}
	if a.loader != nil {
		if err := a.stopLoader(a.loader); err != nil {
			return err
		}
	}
	if err := a.stopFetcher(a.fetcher); err != nil {
		return err
	}
	if err := a.stopStore(a.store); err != nil {
		return err
	}

	return nil
}

// shutdown stops the application gracefully.
func (a *application) shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := a.shutdownServer(ctx, a.server); err != nil {
		return err
	}
	if a.loader != nil {
		if err := a.stopLoader(a.loader); err != nil {
			return err
		}
	}
	if err := a.stopFetcher(a.fetcher); err != nil {
		return err
	}
	if err := a.stopStore(a.store); err != nil {
		return err
	}

	return nil
}

// reload reloads the instance.
func (a *application) reload() error {
	ch := make(chan string)
	defer close(ch)
	errCh := make(chan error)
	defer close(errCh)

	go a.listenChild(ch, errCh)
	for {
		select {
		case event := <-ch:
			switch event {
			case "init":
				exe, err := os.Executable()
				if err != nil {
					return err
				}
				env := os.Environ()
				env = append(env, childEnvKey+"=1")
				files := []*os.File{
					os.Stdin,
					os.Stdout,
					os.Stderr,
				}
				for _, listener := range a.server.state.listenersMap {
					descriptor, err := listener.Descriptor()
					if err != nil {
						return err
					}
					files = append(files, descriptor.Files()...)
				}

				if _, err := os.StartProcess(exe, os.Args, &os.ProcAttr{
					Dir:   filepath.Dir(exe),
					Env:   env,
					Files: files,
					Sys:   &syscall.SysProcAttr{},
				}); err != nil {
					return err
				}

				a.logger.Print("child process started, waiting for connection")

			case "done":
				a.logger.Print("child process ready, stopping parent process")

				return nil
			}

		case err := <-errCh:
			return err
		}
	}
}

// checkStore checks the store configuration.
func (a *application) checkStore(store Store, config map[string]interface{}) error {
	if err := store.Init(config); err != nil {
		return err
	}

	return nil
}

// startStore starts the store.
func (a *application) startStore(store Store, config map[string]interface{}) error {
	if err := store.Init(config); err != nil {
		return err
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
		return err
	}

	return nil
}

// startFetcher starts the fetcher.
func (a *application) startFetcher(fetcher Fetcher, config map[string]interface{}) error {
	if err := fetcher.Init(config); err != nil {
		return err
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
		return err
	}

	return nil
}

// startLoader starts the loader.
func (a *application) startLoader(loader Loader, config map[string]interface{}) error {
	if err := loader.Init(config); err != nil {
		return err
	}
	if err := loader.Start(); err != nil {
		return err
	}

	return nil
}

// stopLoader stops the loader.
func (a *application) stopLoader(loader Loader) error {
	if err := loader.Stop(); err != nil {
		return err
	}

	return nil
}

// checkServer checks the server configuration.
func (a *application) checkServer(server Server, config map[string]interface{}) error {
	if err := server.Init(config); err != nil {
		return err
	}

	return nil
}

// startServer starts the server.
func (a *application) startServer(server Server, config map[string]interface{}) error {
	if err := server.Init(config); err != nil {
		return err
	}
	if err := server.Register(a.state.serverListenersDescriptors); err != nil {
		return err
	}
	if err := server.Start(); err != nil {
		return err
	}

	return nil
}

// stopServer stops the server.
func (a *application) stopServer(server Server) error {
	if err := server.Stop(); err != nil {
		return err
	}

	return nil
}

// shutdownServer shutdowns the server gracefully.
func (a *application) shutdownServer(ctx context.Context, server Server) error {
	if err := server.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}

// childHelloResponse implements the hello message response.
type childHelloResponse struct {
	Listeners []struct {
		Name  string   `json:"name"`
		Files []string `json:"files"`
	} `json:"listeners"`
}

const (
	childSocketFile    string = "neon.sock"
	childSocketTimeout int    = 5
	childEnvKey        string = "CHILD"
	childMessageHello  string = "hello"
	childMessageReady  string = "ready"
)

// listenChild listens for child connection and messages.
func (a *application) listenChild(ch chan<- string, errorCh chan<- error) {
	l, err := net.Listen("unix", childSocketFile)
	if err != nil {
		errorCh <- err
		return
	}
	defer func() {
		l.Close()
		os.Remove(childSocketFile)
	}()

	ch <- "init"

	c, err := a.acceptChild(l)
	if err != nil {
		errorCh <- err
		return
	}

	var done bool
	b := make([]byte, 1024)
	for {
		n, err := c.Read(b)
		if err != nil {
			errorCh <- err
			return
		}

		msg := string(b[0:n])

		switch msg {
		case childMessageHello:
			response := childHelloResponse{}

			for _, listener := range a.server.state.listenersMap {
				helloListener := struct {
					Name  string   `json:"name"`
					Files []string `json:"files"`
				}{
					Name: listener.Name(),
				}

				descriptor, err := listener.Descriptor()
				if err != nil {
					errorCh <- err
					return
				}

				for _, file := range descriptor.Files() {
					helloListener.Files = append(helloListener.Files, file.Name())
				}

				response.Listeners = append(response.Listeners, helloListener)
			}

			data, err := json.Marshal(response)
			if err != nil {
				errorCh <- err
				return
			}

			if _, err := c.Write(data); err != nil {
				errorCh <- err
				return
			}

		case childMessageReady:
			if err := a.shutdown(); err != nil {
				errorCh <- err
				return
			}

			if _, err := c.Write([]byte("ok")); err != nil {
				errorCh <- err
				return
			}

			done = true
		}

		if done {
			ch <- "done"
			break
		}
	}
}

// acceptChild accepts child connection.
func (a *application) acceptChild(l net.Listener) (net.Conn, error) {
	var c net.Conn
	var err error
	ch := make(chan error, 1)

	go func() {
		defer close(ch)

		c, err = l.Accept()
		ch <- err
	}()

	select {
	case err := <-ch:
		if err != nil {
			return nil, err
		}

	case <-time.After(time.Duration(childSocketTimeout) * time.Second):
		return nil, errors.New("accept timeout")
	}

	return c, err
}

// child connects and send messages to the parent process.
func (a *application) child() error {
	c, err := net.Dial("unix", childSocketFile)
	if err != nil {
		return err
	}
	defer c.Close()

	var data []byte
	wg := sync.WaitGroup{}
	readResponse := func() {
		defer wg.Done()

		b := make([]byte, 1024)
		n, err := c.Read(b[:])
		if err != nil {
			return
		}
		data = b[0:n]
	}

	wg.Add(1)
	go readResponse()

	if _, err := c.Write([]byte(childMessageHello)); err != nil {
		return err
	}

	wg.Wait()

	if len(data) == 0 {
		return errors.New("no server response")
	}

	var response childHelloResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return err
	}

	var fdsIndex int = 3
	a.state.serverListenersDescriptors = make(map[string]ServerListenerDescriptor, len(response.Listeners))
	for _, listener := range response.Listeners {
		descriptor := newServerListenerDescriptor()
		for _, file := range listener.Files {
			descriptor.addFile(os.NewFile(uintptr(fdsIndex), file))
			fdsIndex++
		}
		a.state.serverListenersDescriptors[listener.Name] = descriptor
	}

	wg.Add(1)
	go readResponse()

	if _, err := c.Write([]byte(childMessageReady)); err != nil {
		return err
	}

	wg.Wait()

	return nil
}

var _ (Application) = (*application)(nil)
