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
}

// applicationState implements the application state.
type applicationState struct {
	listeners           []Listener
	listenersMap        map[string]Listener
	listenersConfig     map[string]map[string]interface{}
	listenersDescriptor map[string]ListenerDescriptor
	servers             []Server
	serversMap          map[string]Server
	serversConfig       map[string]map[string]interface{}
	serverListeners     map[string][]Listener
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

	var listenersMap map[string]*listener = make(map[string]*listener)
	var listenersConfigMap map[string]map[string]interface{} = make(map[string]map[string]interface{})
	for _, configListener := range a.config.Listeners {
		listener := newListener(configListener.Name)

		listenersMap[configListener.Name] = listener
		listenersConfigMap[configListener.Name] = configListener.Config
	}

	var serversMap map[string]*server = make(map[string]*server)
	var serversConfigMap map[string]map[string]interface{} = make(map[string]map[string]interface{})
	for _, configServer := range a.config.Servers {
		server := newServer(configServer.Name, a.store, a.fetcher)

		serversMap[configServer.Name] = server
		serversConfigMap[configServer.Name] = configServer.Config
	}

	var report []string

	r, err := a.checkStore(a.store, a.config.Store.Config)
	if err != nil {
		report = append(report, r...)
	}

	r, err = a.checkFetcher(a.fetcher, a.config.Fetcher.Config)
	if err != nil {
		report = append(report, r...)
	}

	r, err = a.checkLoader(a.loader, a.config.Loader.Config)
	if err != nil {
		report = append(report, r...)
	}

	if len(a.config.Listeners) == 0 {
		report = append(report, "no listener defined")
	}
	for name, listener := range listenersMap {
		r, err := a.checkListener(listener, listenersConfigMap[name])
		if err != nil {
			report = append(report, r...)
		}
	}

	if len(a.config.Servers) == 0 {
		report = append(report, "no server defined")
	}
	for id, server := range serversMap {
		r, err := a.checkServer(server, serversConfigMap[id])
		if err != nil {
			report = append(report, r...)
		}
	}

	if len(report) > 0 {
		for _, l := range report {
			a.logger.Print(l)
		}
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

	a.state = &applicationState{
		listenersMap:        make(map[string]Listener),
		listenersConfig:     make(map[string]map[string]interface{}),
		listenersDescriptor: make(map[string]ListenerDescriptor),
		serversMap:          make(map[string]Server),
		serversConfig:       make(map[string]map[string]interface{}),
		serverListeners:     make(map[string][]Listener),
	}

	if _, ok := os.LookupEnv(childEnvKey); ok {
		err := a.child()
		if err != nil {
			return err
		}
	}

	a.store = newStore()
	a.fetcher = newFetcher()
	a.loader = newLoader(a.store, a.fetcher)

	if len(a.config.Listeners) == 0 {
		return errors.New("invalid configuration")
	}
	for _, configListener := range a.config.Listeners {
		listener := newListener(configListener.Name)

		a.state.listeners = append(a.state.listeners, listener)
		a.state.listenersMap[configListener.Name] = listener
		a.state.listenersConfig[configListener.Name] = configListener.Config
	}

	if len(a.config.Servers) == 0 {
		return errors.New("invalid configuration")
	}
	for _, configServer := range a.config.Servers {
		server := newServer(configServer.Name, a.store, a.fetcher)

		a.state.servers = append(a.state.servers, server)
		a.state.serversMap[configServer.Name] = server
		a.state.serversConfig[configServer.Name] = configServer.Config
	}

	err := a.startStore(a.store, a.config.Store.Config)
	if err != nil {
		return err
	}

	err = a.startFetcher(a.fetcher, a.config.Fetcher.Config)
	if err != nil {
		return err
	}

	err = a.startLoader(a.loader, a.config.Loader.Config)
	if err != nil {
		return err
	}

	for name, listener := range a.state.listenersMap {
		err := a.startListener(listener, a.state.listenersConfig[name])
		if err != nil {
			return err
		}
	}

	for id, server := range a.state.serversMap {
		err := a.startServer(server, a.state.serversConfig[id])
		if err != nil {
			return err
		}

		err = a.linkServer(server)
		if err != nil {
			return err
		}
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
			log.Print("Signal SIGTERM received, stopping instance")

			err := a.stop()
			if err != nil {
				return err
			}

		case <-shutdown:
			log.Print("Signal SIGQUIT received, starting instance shutdown")

			err := a.shutdown()
			if err != nil {
				log.Printf("Failed to shutdown the instance: %s", err)
			}

		case <-reload:
			log.Print("Signal SIGHUP received, reloading instance")

			err := a.reload()
			if err != nil {
				log.Printf("Failed to reload instance: %s", err)

				continue
			}
		}

		break
	}

	signal.Stop(exit)
	signal.Stop(shutdown)
	signal.Stop(reload)

	return nil
}

// stop stops the application.
func (a *application) stop() error {
	for _, server := range a.state.servers {
		err := a.stopServer(server)
		if err != nil {
			return err
		}

		err = a.unlinkServer(server)
		if err != nil {
			return err
		}

		err = a.removeServer(server)
		if err != nil {
			return err
		}
	}

	for _, listener := range a.state.listeners {
		err := a.stopListener(listener)
		if err != nil {
			return err
		}

		err = a.removeListener(listener)
		if err != nil {
			return err
		}
	}

	if a.loader != nil {
		err := a.stopLoader(a.loader)
		if err != nil {
			return err
		}
	}

	err := a.stopFetcher(a.fetcher)
	if err != nil {
		return err
	}

	err = a.stopStore(a.store)
	if err != nil {
		return err
	}

	return nil
}

// shutdown stops the application gracefully.
func (a *application) shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer func() {
		cancel()
	}()

	for _, server := range a.state.servers {
		err := a.shutdownServer(ctx, server)
		if err != nil {
			return err
		}

		err = a.stopServer(server)
		if err != nil {
			return err
		}

		err = a.removeServer(server)
		if err != nil {
			return err
		}
	}

	for _, listener := range a.state.listeners {
		err := a.shutdownListener(ctx, listener)
		if err != nil {
			return err
		}

		err = a.stopListener(listener)
		if err != nil {
			return err
		}

		err = a.removeListener(listener)
		if err != nil {
			return err
		}
	}

	if a.loader != nil {
		err := a.stopLoader(a.loader)
		if err != nil {
			return err
		}
	}

	err := a.stopFetcher(a.fetcher)
	if err != nil {
		return err
	}

	err = a.stopStore(a.store)
	if err != nil {
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
				for _, listener := range a.state.listeners {
					descriptor, err := listener.Descriptor()
					if err != nil {
						return err
					}
					files = append(files, descriptor.Files()...)
				}

				_, err = os.StartProcess(exe, os.Args, &os.ProcAttr{
					Dir:   filepath.Dir(exe),
					Env:   env,
					Files: files,
					Sys:   &syscall.SysProcAttr{},
				})
				if err != nil {
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

// startStore starts the store.
func (a *application) startStore(store Store, config map[string]interface{}) error {
	_, err := store.Check(config)
	if err != nil {
		return err
	}

	err = store.Load(config)
	if err != nil {
		return err
	}

	return nil
}

// stopStore stops the store.
func (a *application) stopStore(store Store) error {
	return nil
}

// checkStore checks the store configuration.
func (a *application) checkStore(store Store, config map[string]interface{}) ([]string, error) {
	r, err := store.Check(config)
	if err != nil {
		return r, err
	}

	return nil, nil
}

// startFetcher starts the fetcher.
func (a *application) startFetcher(fetcher Fetcher, config map[string]interface{}) error {
	_, err := fetcher.Check(config)
	if err != nil {
		return err
	}

	err = fetcher.Load(config)
	if err != nil {
		return err
	}

	return nil
}

// stopFetcher stops the fetcher.
func (a *application) stopFetcher(fetcher Fetcher) error {
	return nil
}

// checkFetcher checks the fetcher configuration.
func (a *application) checkFetcher(fetcher Fetcher, config map[string]interface{}) ([]string, error) {
	r, err := fetcher.Check(config)
	if err != nil {
		return r, err
	}

	return nil, nil
}

// startLoader starts the loader.
func (a *application) startLoader(loader Loader, config map[string]interface{}) error {
	_, err := loader.Check(config)
	if err != nil {
		return err
	}

	err = loader.Load(config)
	if err != nil {
		return err
	}

	err = loader.Start()
	if err != nil {
		return err
	}

	return nil
}

// stopLoader stops the loader.
func (a *application) stopLoader(loader Loader) error {
	err := loader.Stop()
	if err != nil {
		return err
	}

	return nil
}

// checkLoader checks the loader configuration.
func (a *application) checkLoader(loader Loader, config map[string]interface{}) ([]string, error) {
	r, err := loader.Check(config)
	if err != nil {
		return r, err
	}

	return nil, nil
}

// startListener starts the listener.
func (a *application) startListener(listener Listener, config map[string]interface{}) error {
	_, err := listener.Check(config)
	if err != nil {
		return err
	}

	err = listener.Load(config)
	if err != nil {
		return err
	}

	err = listener.Register(a.state.listenersDescriptor[listener.Name()])
	if err != nil {
		return err
	}

	err = listener.Serve()
	if err != nil {
		return err
	}

	return nil
}

// stopListener stops the listener.
func (a *application) stopListener(listener Listener) error {
	err := listener.Close()
	if err != nil {
		return err
	}

	return nil
}

// shutdownListener shutdowns gracefully the listener.
func (a *application) shutdownListener(ctx context.Context, listener Listener) error {
	err := listener.Shutdown(ctx)
	if err != nil {
		return err
	}

	err = listener.Close()
	if err != nil {
		return err
	}

	return nil
}

// removeListener removes the listener.
func (a *application) removeListener(listener Listener) error {
	err := listener.Remove()
	if err != nil {
		return err
	}

	return nil
}

// checkListener checks the listener configuration.
func (a *application) checkListener(listener Listener, config map[string]interface{}) ([]string, error) {
	r, err := listener.Check(config)
	if err != nil {
		return r, err
	}

	return nil, nil
}

// startServer starts the server.
func (a *application) startServer(server Server, config map[string]interface{}) error {
	_, err := server.Check(config)
	if err != nil {
		return err
	}

	err = server.Load(config)
	if err != nil {
		return err
	}

	err = server.Register()
	if err != nil {
		return err
	}

	err = server.Start()
	if err != nil {
		return err
	}

	err = server.Enable()
	if err != nil {
		return err
	}

	return nil
}

// stopServer stops the server.
func (a *application) stopServer(server Server) error {
	err := server.Stop()
	if err != nil {
		return err
	}

	return nil
}

// shutdownServer shutdowns gracefully the server.
func (a *application) shutdownServer(ctx context.Context, server Server) error {
	err := server.Disable(ctx)
	if err != nil {
		return err
	}

	return nil
}

// removeServer removes the server.
func (a *application) removeServer(server Server) error {
	err := server.Remove()
	if err != nil {
		return err
	}

	return nil
}

// linkServer links the server to its listeners.
func (a *application) linkServer(server Server) error {
	for _, listenerName := range server.Listeners() {
		if _, ok := a.state.listenersMap[listenerName]; !ok {
			return errors.New("listener not found")
		}
		if _, ok := a.state.serverListeners[server.Name()]; ok {
			for _, listener := range a.state.serverListeners[server.Name()] {
				if listener.Name() == listenerName {
					return errors.New("server already linked to listener")
				}
			}
		}

		a.state.listenersMap[listenerName].Link(server)

		a.state.serverListeners[server.Name()] = append(a.state.serverListeners[server.Name()],
			a.state.listenersMap[listenerName])
	}

	return nil
}

// unlinkServer unlinks the server from its listeners.
func (a *application) unlinkServer(server Server) error {
	if _, ok := a.state.serverListeners[server.Name()]; !ok {
		return errors.New("server not linked")
	}

	for _, listener := range a.state.serverListeners[server.Name()] {
		listener.Unlink(server)
	}
	delete(a.state.serverListeners, server.Name())

	return nil
}

// checkServer checks the server configuration.
func (a *application) checkServer(server Server, config map[string]interface{}) ([]string, error) {
	r, err := server.Check(config)
	if err != nil {
		return r, err
	}

	return nil, nil
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

			for _, listener := range a.state.listeners {
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

			_, err = c.Write(data)
			if err != nil {
				errorCh <- err
				return
			}

		case childMessageReady:
			err = a.shutdown()
			if err != nil {
				errorCh <- err
				return
			}

			_, err = c.Write([]byte("ok"))
			if err != nil {
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
	case err = <-ch:
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
	defer func() {
		c.Close()
	}()

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

	_, err = c.Write([]byte(childMessageHello))
	if err != nil {
		return err
	}

	wg.Wait()

	if len(data) == 0 {
		return errors.New("no server response")
	}

	var response childHelloResponse
	err = json.Unmarshal(data, &response)
	if err != nil {
		return err
	}

	var index int = 3
	for _, listener := range response.Listeners {
		descriptor := newListenerDescriptor()
		for _, file := range listener.Files {
			descriptor.addFile(os.NewFile(uintptr(index), file))
			index++
		}
		a.state.listenersDescriptor[listener.Name] = descriptor
	}

	wg.Add(1)
	go readResponse()

	_, err = c.Write([]byte(childMessageReady))
	if err != nil {
		return err
	}

	wg.Wait()

	return nil
}
