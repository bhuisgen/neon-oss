// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rewrite

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
)

// rewriteMiddleware implements the rewrite middleware.
type rewriteMiddleware struct {
	config  *rewriteMiddlewareConfig
	logger  *log.Logger
	regexps []*regexp.Regexp
}

// rewriteMiddlewareConfig implements the rewrite middleware configuration.
type rewriteMiddlewareConfig struct {
	Rules []RewriteRule
}

// RewriteRule implements a rewrite rule.
type RewriteRule struct {
	Path        string
	Replacement string
	Flag        *string
	Last        bool
}

const (
	rewriteModuleID module.ModuleID = "server.middleware.rewrite"
	rewriteLogger   string          = "server.middleware.rewrite"

	rewriteRuleFlagRedirect  string = "redirect"
	rewriteRuleFlagPermanent string = "permanent"
)

// init initializes the module.
func init() {
	module.Register(rewriteMiddleware{})
}

// ModuleInfo returns the module information.
func (m rewriteMiddleware) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: rewriteModuleID,
		NewInstance: func() module.Module {
			return &rewriteMiddleware{}
		},
	}
}

// Check checks the middleware configuration.
func (m *rewriteMiddleware) Check(config map[string]interface{}) ([]string, error) {
	var report []string

	var c rewriteMiddlewareConfig
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
		if rule.Replacement == "" {
			report = append(report, fmt.Sprintf("rule option '%s', missing option or value", "Replacement"))
		}
		if rule.Flag != nil {
			switch *rule.Flag {
			case rewriteRuleFlagPermanent:
			case rewriteRuleFlagRedirect:
			default:
				report = append(report, fmt.Sprintf("rule option '%s', invalid value '%s'", "Flag", *rule.Flag))
			}
		}
	}

	if len(report) > 0 {
		return report, errors.New("check failure")
	}

	return nil, nil
}

// Load loads the middleware.
func (m *rewriteMiddleware) Load(config map[string]interface{}) error {
	var c rewriteMiddlewareConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		return err
	}

	m.config = &c
	m.logger = log.New(os.Stderr, fmt.Sprint(rewriteLogger, ": "), log.LstdFlags|log.Lmsgprefix)

	return nil
}

// Register registers the server resources.
func (m *rewriteMiddleware) Register(registry core.ServerRegistry) error {
	err := registry.RegisterMiddleware(m.Handler)
	if err != nil {
		return err
	}

	return nil
}

// Start starts the middleware.
func (m *rewriteMiddleware) Start(store core.Store, fetcher core.Fetcher) error {
	for _, rule := range m.config.Rules {
		re, err := regexp.Compile(rule.Path)
		if err != nil {
			return err
		}
		m.regexps = append(m.regexps, re)
	}

	return nil
}

// Mount mounts the middleware.
func (m *rewriteMiddleware) Mount() error {
	return nil
}

// Unmount unmounts the middleware.
func (m *rewriteMiddleware) Unmount() {
}

// Stop stops the middleware.
func (m *rewriteMiddleware) Stop() {
	m.regexps = []*regexp.Regexp{}
}

// Handler implements the middleware handler.
func (m *rewriteMiddleware) Handler(next http.Handler) http.Handler {
	f := func(w http.ResponseWriter, r *http.Request) {
		var rewrite bool
		var path string = r.URL.Path
		var status int = http.StatusFound
		var redirect bool
		for index, regexp := range m.regexps {
			if regexp.MatchString(path) {
				rewrite = true
				path = m.config.Rules[index].Replacement

				if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
					redirect = true
				}

				if m.config.Rules[index].Flag != nil {
					switch *m.config.Rules[index].Flag {
					case rewriteRuleFlagRedirect:
						status = http.StatusFound
						redirect = true
					case rewriteRuleFlagPermanent:
						status = http.StatusMovedPermanently
						redirect = true
					}
				}

				if m.config.Rules[index].Last {
					break
				}
			}
		}

		if rewrite {
			if redirect {
				http.Redirect(w, r, path, status)
				return
			}
			r.URL.Path = path
		}

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(f)
}

var _ core.ServerMiddlewareModule = (*rewriteMiddleware)(nil)
