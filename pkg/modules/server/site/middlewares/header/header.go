// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package header

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
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
	headerLogger   string          = "middleware[header]"
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

// Check checks the middleware configuration.
func (m *headerMiddleware) Check(config map[string]interface{}) ([]string, error) {
	var report []string

	var c headerMiddlewareConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		report = append(report, "failed to parse configuration")
		return report, err
	}

	for _, rule := range c.Rules {
		if rule.Path == "" {
			report = append(report, fmt.Sprintf("rule option '%s', missing option or value", "Path"))
		} else {
			_, err := regexp.Compile(rule.Path)
			if err != nil {
				report = append(report, fmt.Sprintf("rule option '%s', invalid regular expression '%s'", "Path", rule.Path))
			}
		}
		for key := range rule.Set {
			if key == "" {
				report = append(report, fmt.Sprintf("rule option '%s', invalid key '%s'", "Set", key))
			}
		}
	}

	if len(report) > 0 {
		return report, errors.New("check failure")
	}

	return nil, nil
}

// Load loads the middleware.
func (m *headerMiddleware) Load(config map[string]interface{}) error {
	var c headerMiddlewareConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		return err
	}

	m.config = &c
	m.logger = log.New(os.Stderr, fmt.Sprint(headerLogger, ": "), log.LstdFlags|log.Lmsgprefix)

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
