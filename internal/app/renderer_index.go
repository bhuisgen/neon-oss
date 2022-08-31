// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"bytes"
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

	config *IndexRendererConfig
	logger *log.Logger
	routes []struct {
		matcher *regexp.Regexp
		params  []string
	}
	vmPool  *vmPool
	cache   *cache
	fetcher *fetcher
}

// IndexRendererConfig implements the index renderer configuration
type IndexRendererConfig struct {
	Enable   bool
	HTML     string
	Bundle   string
	Env      string
	Timeout  int
	Cache    bool
	CacheTTL int
	Routes   []IndexRoute
}

// IndexRoute implements a route
type IndexRoute struct {
	Path string
	Data []IndexRouteData
}

// IndexRouteData implements a route data
type IndexRouteData struct {
	Name     string
	Resource string
}

var (
	regexpRouteParameters = regexp.MustCompile(`\:(.[^\:/]+)`)
)

// CreateIndexRenderer creates a new index renderer
func CreateIndexRenderer(config *IndexRendererConfig, fetcher *fetcher) (*indexRenderer, error) {
	routes := []struct {
		matcher *regexp.Regexp
		params  []string
	}{}

	for _, route := range config.Routes {
		pattern := fmt.Sprintf("^%s/?$", regexpRouteParameters.ReplaceAllString(route.Path, "([^/]+)"))
		matcher, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}

		params := []string{}
		routeParams := regexpRouteParameters.FindAllString(route.Path, -1)
		for _, param := range routeParams {
			params = append(params, param)
		}

		routes = append(routes, struct {
			matcher *regexp.Regexp
			params  []string
		}{
			matcher: matcher,
			params:  params,
		})
	}

	return &indexRenderer{
		config:  config,
		logger:  log.Default(),
		routes:  routes,
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

	serverData, err := vm.serverDataObject.NewInstance(ctx)
	if err != nil {
		r.logger.Printf("Failed to create server data instance: %s", err)

		return nil, err
	}

	var valid bool = true

	for index, route := range r.config.Routes {
		params := r.routes[index].matcher.FindStringSubmatch(req.URL.Path)
		if params == nil {
			continue
		}
		for _, data := range route.Data {
			key := ReplaceParameters(data.Resource, r.routes[index].params, params[1:])

			serverDataResource, err := vm.serverDataResourceObject.NewInstance(ctx)
			if err != nil {
				r.logger.Printf("Failed to create server data resource instance: %s", err)

				return nil, err
			}

			response, err := r.fetcher.Get(key)
			if err != nil {
				serverDataResource.Set("loading", true)
				serverDataResource.Set("response", "{}")
				serverData.Set(data.Name, serverDataResource)
			} else {
				serverDataResource.Set("loading", false)
				serverDataResource.Set("response", string(response))
				serverData.Set(data.Name, serverDataResource)
			}
		}
		break
	}

	server.Set("data", serverData)
	global.Set("server", server)

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

	body, err := r.generateBody(*vm.render, *vm.title, vm.metas, vm.links, vm.scripts)
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
		Cache:  r.config.Cache,
	}

	if result.Valid && result.Cache {
		r.cache.Set(req.URL.Path, &result, time.Duration(r.config.CacheTTL)*time.Second)
	}

	return &result, nil
}

// generateBody generates the final HTML body
func (r *indexRenderer) generateBody(render []byte, title string, metas map[string]map[string]string,
	links map[string]map[string]string, scripts map[string]map[string]string) ([]byte, error) {
	var body []byte
	html, err := os.ReadFile(r.config.HTML)
	if err != nil {
		return nil, err
	}
	body = html

	if title != "" {
		body = bytes.Replace(body,
			[]byte("</head>\n"),
			[]byte(fmt.Sprintf("<title>%s</title>\n</head>\n", title)),
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
			[]byte("</head>\n"),
			[]byte(fmt.Sprintf("<meta name=\"%s\"%s/>\n</head>\n", id, buf.String())),
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
			[]byte("</head>\n"),
			[]byte(fmt.Sprintf("<link id=\"%s\"%s/>\n</head>\n", id, buf.String())),
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
			[]byte("</head>\n"),
			[]byte(fmt.Sprintf("<script id=\"%s\"%s>%s</script>\n</head>\n", id, buf.String(), content)),
			1)
	}

	body = bytes.Replace(body, []byte("<div id=\"root\"></div>"), render, 1)

	return body, nil
}
