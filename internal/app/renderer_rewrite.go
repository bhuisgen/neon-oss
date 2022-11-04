// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

// rewriteRenderer implements the rewrite renderer
type rewriteRenderer struct {
	config  *RewriteRendererConfig
	logger  *log.Logger
	regexps []*regexp.Regexp
	next    Renderer
}

// RewriteRendererConfig implements the rewrite renderer configuration
type RewriteRendererConfig struct {
	Rules []RewriteRule
}

// RewriteRule implements a rewrite rule
type RewriteRule struct {
	Path        string
	Replacement string
	Flag        *string
	Last        bool
}

const (
	rewriteLogger            string = "server[rewrite]"
	rewriteRuleFlagRedirect  string = "redirect"
	rewriteRuleFlagPermanent string = "permanent"
)

// CreateRewriteRenderer creates a new rewrite renderer
func CreateRewriteRenderer(config *RewriteRendererConfig) (*rewriteRenderer, error) {
	r := rewriteRenderer{
		config:  config,
		logger:  log.New(os.Stderr, fmt.Sprint(rewriteLogger, ": "), log.LstdFlags|log.Lmsgprefix),
		regexps: []*regexp.Regexp{},
	}

	err := r.initialize()
	if err != nil {
		return nil, err
	}

	return &r, nil
}

// initialize initializes the renderer
func (r *rewriteRenderer) initialize() error {
	for _, rule := range r.config.Rules {
		re, err := regexp.Compile(rule.Path)
		if err != nil {
			return err
		}
		r.regexps = append(r.regexps, re)
	}

	return nil
}

// Handle implements the renderer
func (r *rewriteRenderer) Handle(w http.ResponseWriter, req *http.Request, info *ServerInfo) {
	var rewrite bool
	var path string = req.URL.Path
	var status int = http.StatusFound
	var redirect bool
	for index, regexp := range r.regexps {
		if regexp.MatchString(path) {
			rewrite = true
			path = r.config.Rules[index].Replacement

			if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
				redirect = true
			}

			if r.config.Rules[index].Flag != nil {
				switch *r.config.Rules[index].Flag {
				case rewriteRuleFlagRedirect:
					status = http.StatusFound
					redirect = true
				case rewriteRuleFlagPermanent:
					status = http.StatusMovedPermanently
					redirect = true
				}
			}

			if r.config.Rules[index].Last {
				break
			}
		}
	}

	if rewrite {
		if redirect {
			http.Redirect(w, req, path, status)
			return
		}
		req.URL.Path = path
	}

	r.next.Handle(w, req, info)
}

// Next configures the next renderer
func (r *rewriteRenderer) Next(renderer Renderer) {
	r.next = renderer
}
