// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

	"rogchap.com/v8go"
)

// indexRenderer implements the index renderer
type indexRenderer struct {
	Renderer
	next Renderer

	config  *IndexRendererConfig
	logger  *log.Logger
	regexps []*regexp.Regexp
	vmPool  *vmPool
	cache   *cache
	fetcher *fetcher
}

// IndexRendererConfig implements the index renderer configuration
type IndexRendererConfig struct {
	Enable    bool
	HTML      string
	Bundle    string
	Env       string
	Container string
	State     string
	Timeout   int
	Cache     bool
	CacheTTL  int
	Rules     []IndexRule
}

// IndexRule implements a rule
type IndexRule struct {
	Path  string
	State []IndexRuleStateEntry
	Last  bool
}

// IndexRuleStateEntry implements a rule state entry
type IndexRuleStateEntry struct {
	Key      string
	Resource string
	Export   bool
}

const (
	INDEX_LOGGER string = "renderer[index]"
)

// CreateIndexRenderer creates a new index renderer
func CreateIndexRenderer(config *IndexRendererConfig, fetcher *fetcher) (*indexRenderer, error) {
	logger := log.New(os.Stdout, fmt.Sprint(INDEX_LOGGER, ": "), log.LstdFlags|log.Lmsgprefix)

	regexps := []*regexp.Regexp{}
	for _, rule := range config.Rules {
		r, err := regexp.Compile(rule.Path)
		if err != nil {
			return nil, err
		}

		regexps = append(regexps, r)
	}

	return &indexRenderer{
		config:  config,
		logger:  logger,
		regexps: regexps,
		vmPool:  NewVMPool(),
		cache:   NewCache(),
		fetcher: fetcher,
	}, nil
}

// handle implements the renderer handler
func (r *indexRenderer) handle(w http.ResponseWriter, req *http.Request) {
	result, err := r.render(req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte{})

		r.logger.Printf("Render error (url=%s, status=%d)", req.URL.Path, result.Status)

		return
	}

	if result.Redirect {
		http.Redirect(w, req, result.RedirectTarget, result.RedirectStatus)

		r.logger.Printf("Redirect completed (url=%s, status=%d, target=%s)", req.URL.Path, result.RedirectStatus,
			result.RedirectTarget)

		return
	}

	w.WriteHeader(result.Status)
	w.Write(result.Body)

	r.logger.Printf("Render completed (url=%s, status=%d, valid=%t, cache=%t)", req.URL.Path, result.Status, result.Valid,
		result.Cache)
}

// setNext configures the next renderer
func (r *indexRenderer) setNext(renderer Renderer) {
	r.next = renderer
}

// render makes a new render
func (r *indexRenderer) render(req *http.Request) (*Render, error) {
	if r.config.Cache {
		obj := r.cache.Get(req.URL.Path)
		if obj != nil {
			result := obj.(*Render)

			return result, nil
		}
	}

	var vm = r.vmPool.Get()
	defer r.vmPool.Put(vm)

	vm.Configure()

	ctx := v8go.NewContext(vm.isolate)
	defer ctx.Close()

	global := ctx.Global()

	process, err := vm.processObject.NewInstance(ctx)
	if err != nil {
		r.logger.Printf("Failed to create process instance: %s", err)

		return nil, err
	}

	global.Set("process", process)

	env, err := vm.envObject.NewInstance(ctx)
	if err != nil {
		r.logger.Printf("Failed to create env instance: %s", err)

		return nil, err
	}

	env.Set("ENV", r.config.Env)
	process.Set("env", env)

	server, err := vm.serverObject.NewInstance(ctx)
	if err != nil {
		r.logger.Printf("Failed to create server instance: %s", err)

		return nil, err
	}

	server.Set("url", req.URL.Path)

	serverState, err := vm.serverStateObject.NewInstance(ctx)
	if err != nil {
		r.logger.Printf("Failed to create server state instance: %s", err)

		return nil, err
	}

	var valid bool = true
	var clientState map[string]interface{} = make(map[string]interface{})

	for index, rule := range r.config.Rules {
		if _, ok := os.LookupEnv("DEBUG"); ok {
			r.logger.Printf("Index: id=%s, rule=%d, phase=check, path=%s", req.Context().Value(ContextKeyID{}).(string),
				index+1, req.URL.Path)
		}

		m := r.regexps[index].FindStringSubmatch(req.URL.Path)
		if m == nil {
			continue
		}

		if _, ok := os.LookupEnv("DEBUG"); ok {
			r.logger.Printf("Index: id=%s, rule=%d, phase=match, path=%s", req.Context().Value(ContextKeyID{}).(string),
				index+1, req.URL.Path)
		}

		params := make(map[string]string)
		for i, value := range m {
			if i > 0 {
				params[fmt.Sprint(i)] = value
			}
		}
		for i, name := range r.regexps[index].SubexpNames() {
			if i != 0 && name != "" {
				params[name] = m[i]
			}
		}

		if _, ok := os.LookupEnv("DEBUG"); ok {
			r.logger.Printf("Index: id=%s, rule=%d, phase=params, path=%s, params=%s",
				req.Context().Value(ContextKeyID{}).(string), index+1, req.URL.Path, params)
		}

		for _, entry := range rule.State {
			if _, ok := os.LookupEnv("DEBUG"); ok {
				r.logger.Printf("Index: id=%s, rule=%d, phase=state1, path=%s, state_key=%s, state_resource=%s",
					req.Context().Value(ContextKeyID{}).(string), index+1, req.URL.Path, entry.Key, entry.Resource)
			}

			stateKey := replaceParameters(entry.Key, params)
			resourceKey := replaceParameters(entry.Resource, params)

			if _, ok := os.LookupEnv("DEBUG"); ok {
				r.logger.Printf("Index: id=%s, rule=%d, phase=state2, path=%s, state_key=%s, state_resource=%s",
					req.Context().Value(ContextKeyID{}).(string), index+1, req.URL.Path, stateKey, resourceKey)
			}

			serverStateResource, err := vm.serverStateResourceObject.NewInstance(ctx)
			if err != nil {
				r.logger.Printf("Failed to create server state resource instance: %s", err)

				return nil, err
			}

			response, err := r.fetcher.Get(resourceKey)
			if err != nil {
				if r.fetcher.Exists(resourceKey) {
					serverStateResource.Set("loading", true)
				} else {
					serverStateResource.Set("error", "unknown resource")
				}
				serverState.Set(stateKey, serverStateResource)

				valid = false

				continue
			}

			serverStateResource.Set("response", string(response))
			serverState.Set(stateKey, serverStateResource)

			if entry.Export {
				clientState[stateKey] = string(response)
			}
		}

		if r.config.Rules[index].Last {
			if _, ok := os.LookupEnv("DEBUG"); ok {
				r.logger.Printf("Index: id=%s, rule=%d, phase=last, path=%s", req.Context().Value(ContextKeyID{}).(string),
					index+1, req.URL.Path)
			}

			break
		}
	}

	server.Set("state", serverState)
	global.Set("server", server)

	if _, ok := os.LookupEnv("DEBUG"); ok {
		c, _ := json.Marshal(clientState)
		r.logger.Printf("Index: id=%s, path=%s, client_state=%s", req.Context().Value(ContextKeyID{}).(string),
			req.URL.Path, c)
	}

	buf, err := os.ReadFile(r.config.Bundle)
	if err != nil {
		r.logger.Printf("Failed to read bundle file '%s': %s", r.config.Bundle, err)

		return nil, err
	}
	bundle := string(buf)

	worker := func(values chan<- *v8go.Value, errors chan<- error) {
		value, err := ctx.RunScript(bundle, r.config.Bundle)
		if err != nil {
			errors <- err
			return
		}
		values <- value
	}

	values := make(chan *v8go.Value, 1)
	errors := make(chan error, 1)

	go worker(values, errors)
	select {
	case <-values:

	case err := <-errors:
		r.logger.Printf("Failed to execute bundle: %s", err)
		if _, ok := os.LookupEnv("DEBUG"); ok {
			e := err.(*v8go.JSError)
			r.logger.Printf(e.Message)
			r.logger.Printf(e.Location)
			r.logger.Printf(e.StackTrace)
		}
		return nil, err

	case <-time.After(time.Duration(r.config.Timeout) * time.Second):
		vm.isolate.TerminateExecution()
		err := <-errors
		return nil, err
	}

	if *vm.redirect {
		return &Render{
			Redirect:       true,
			RedirectTarget: *vm.redirectURL,
			RedirectStatus: *vm.redirectStatus,
		}, nil
	}

	body, err := r.generateBody(*vm.render, *vm.title, vm.metas, vm.links, vm.scripts, clientState)
	if err != nil {
		r.logger.Printf("Failed to generate body: %s", err)

		return nil, err
	}

	var status int = *vm.status
	if !valid {
		status = http.StatusServiceUnavailable
	}

	result := Render{
		Body:   body,
		Status: status,
		Valid:  valid,
	}

	if result.Valid && r.config.Cache {
		r.cache.Set(req.URL.Path, &result, time.Duration(r.config.CacheTTL)*time.Second)
		result.Cache = true
	}

	return &result, nil
}

// generateBody generates the final HTML body
func (r *indexRenderer) generateBody(render []byte, title string, metas map[string]map[string]string,
	links map[string]map[string]string, scripts map[string]map[string]string, state map[string]interface{}) ([]byte, error) {
	var body []byte
	html, err := os.ReadFile(r.config.HTML)
	if err != nil {
		return nil, err
	}
	body = html

	if title != "" {
		body = bytes.Replace(body,
			[]byte("</head>"),
			[]byte(fmt.Sprintf("<title>%s</title></head>", title)),
			1)
	}

	for id, attributes := range metas {
		var buf = bufferPool.Get().(*bytes.Buffer)
		defer bufferPool.Put(buf)
		buf.Reset()

		for k, v := range attributes {
			buf.Write([]byte(fmt.Sprintf(" %s=\"%s\"", k, v)))
		}
		body = bytes.Replace(body,
			[]byte("</head>"),
			[]byte(fmt.Sprintf("<meta id=\"%s\" name=\"%s\"%s/></head>", id, id, buf.String())),
			1)
	}

	for id, attributes := range links {
		var buf = bufferPool.Get().(*bytes.Buffer)
		defer bufferPool.Put(buf)
		buf.Reset()

		for k, v := range attributes {
			buf.Write([]byte(fmt.Sprintf(" %s=\"%s\"", k, v)))
		}
		body = bytes.Replace(body,
			[]byte("</head>"),
			[]byte(fmt.Sprintf("<link id=\"%s\"%s/></head>", id, buf.String())),
			1)
	}

	for id, attributes := range scripts {
		var buf = bufferPool.Get().(*bytes.Buffer)
		defer bufferPool.Put(buf)
		buf.Reset()

		var content string = ""
		for k, v := range attributes {
			if k == "children" {
				content = v
			}
			buf.Write([]byte(fmt.Sprintf(" %s=\"%s\"", k, v)))
		}
		body = bytes.Replace(body,
			[]byte("</head>"),
			[]byte(fmt.Sprintf("<script id=\"%s\"%s>%s</script></head>", id, buf.String(), content)),
			1)
	}

	body = bytes.Replace(body,
		[]byte(fmt.Sprintf("<div id=\"%s\"></div>", r.config.Container)),
		[]byte(fmt.Sprintf("<div id=\"%s\">%s</div>", r.config.Container, render)),
		1)

	if len(state) > 0 {
		buf, err := json.Marshal(state)
		if err != nil {
			return nil, err
		}
		body = bytes.Replace(body,
			[]byte("</body>"),
			[]byte(fmt.Sprintf("<script id=\"%s\" type=\"application/json\">%s</script></body>", r.config.State, buf)),
			1)
	}

	return body, nil
}
