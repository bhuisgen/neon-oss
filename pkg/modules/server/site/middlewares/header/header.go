// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package header

import (
	"errors"
	"log"
	"net/http"
	"regexp"

	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
)

// headerMiddleware implements the header middleware.
type headerMiddleware struct {
	config  *headerMiddlewareConfig
	logger  *log.Logger
	regexps []*regexp.Regexp
}

// headerMiddlewareConfig implements the header middleware configuration.
type headerMiddlewareConfig struct {
	Rules []HeaderRule
}

// HeaderRule implements a header rule.
type HeaderRule struct {
	Path string
	Set  map[string]string
	Last bool
}

const (
	headerModuleID module.ModuleID = "server.site.middleware.header"
)

// init initializes the module.
func init() {
	module.Register(headerMiddleware{})
}

// ModuleInfo returns the module information.
func (m headerMiddleware) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: headerModuleID,
		NewInstance: func() module.Module {
			return &headerMiddleware{}
		},
	}
}

// Init initializes the middleware.
func (m *headerMiddleware) Init(config map[string]interface{}, logger *log.Logger) error {
	m.logger = logger

	if err := mapstructure.Decode(config, &m.config); err != nil {
		m.logger.Print("failed to parse configuration")
		return err
	}

	var errInit bool

	for _, rule := range m.config.Rules {
		if rule.Path == "" {
			m.logger.Printf("rule option '%s', missing option or value", "Path")
			errInit = true
			continue
		}
		_, err := regexp.Compile(rule.Path)
		if err != nil {
			m.logger.Printf("rule option '%s', invalid regular expression '%s'", "Path", rule.Path)
			errInit = true
			continue
		}
		for key := range rule.Set {
			if key == "" {
				m.logger.Printf("rule option '%s', invalid key '%s'", "Set", key)
				errInit = true
				continue
			}
		}
	}

	if errInit {
		return errors.New("init error")
	}

	return nil
}

// Register registers the middleware.
func (m *headerMiddleware) Register(site core.ServerSite) error {
	err := site.RegisterMiddleware(m.Handler)
	if err != nil {
		return err
	}

	return nil
}

// Start starts the middleware.
func (m *headerMiddleware) Start() error {
	for _, rule := range m.config.Rules {
		re, err := regexp.Compile(rule.Path)
		if err != nil {
			return err
		}
		m.regexps = append(m.regexps, re)
	}

	return nil
}

// Stop stops the middleware.
func (m *headerMiddleware) Stop() {
	m.regexps = []*regexp.Regexp{}
}

// Handler implements the middleware handler.
func (m *headerMiddleware) Handler(next http.Handler) http.Handler {
	f := func(w http.ResponseWriter, r *http.Request) {
		for index, regexp := range m.regexps {
			if regexp.MatchString(r.URL.Path) {
				for k, v := range m.config.Rules[index].Set {
					w.Header().Set(k, v)
				}
				if m.config.Rules[index].Last {
					break
				}
			}
		}

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(f)
}

var _ core.ServerSiteMiddlewareModule = (*headerMiddleware)(nil)
