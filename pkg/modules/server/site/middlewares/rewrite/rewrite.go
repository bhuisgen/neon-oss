// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rewrite

import (
	"errors"
	"log"
	"net/http"
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
	rewriteModuleID module.ModuleID = "server.site.middleware.rewrite"

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

// Init initializes the middleware.
func (m *rewriteMiddleware) Init(config map[string]interface{}, logger *log.Logger) error {
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
		} else {
			re, err := regexp.Compile(rule.Path)
			if err != nil {
				m.logger.Printf("rule option '%s', invalid regular expression '%s'", "Path", rule.Path)
				errInit = true
			} else {
				m.regexps = append(m.regexps, re)
			}
		}
		if rule.Replacement == "" {
			m.logger.Printf("rule option '%s', missing option or value", "Replacement")
			errInit = true
		}
		if rule.Flag != nil {
			switch *rule.Flag {
			case rewriteRuleFlagPermanent:
			case rewriteRuleFlagRedirect:
			default:
				m.logger.Printf("rule option '%s', invalid value '%s'", "Flag", *rule.Flag)
				errInit = true
			}
		}
	}

	if errInit {
		return errors.New("init error")
	}

	return nil
}

// Register registers the middleware.
func (m *rewriteMiddleware) Register(site core.ServerSite) error {
	err := site.RegisterMiddleware(m.Handler)
	if err != nil {
		return err
	}

	return nil
}

// Start starts the middleware.
func (m *rewriteMiddleware) Start() error {
	return nil
}

// Stop stops the middleware.
func (m *rewriteMiddleware) Stop() {
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

var _ core.ServerSiteMiddlewareModule = (*rewriteMiddleware)(nil)
