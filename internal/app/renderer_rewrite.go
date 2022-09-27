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
	Renderer
	next Renderer

	config  *RewriteRendererConfig
	logger  *log.Logger
	regexps []*regexp.Regexp
}

// RewriteRendererConfig implements the rewrite renderer configuration
type RewriteRendererConfig struct {
	Enable bool
	Rules  []RewriteRule
}

// RewriteRule implements a rewrite rule
type RewriteRule struct {
	Path        string
	Replacement string
	Flag        string
	Last        bool
}

const (
	REWRITE_LOGGER              string = "renderer[rewrite]"
	REWRITE_RULE_FLAG_REDIRECT  string = "redirect"
	REWRITE_RULE_FLAG_PERMANENT string = "permanent"
)

// CreateRewriteRenderer creates a new rewrite renderer
func CreateRewriteRenderer(config *RewriteRendererConfig) (*rewriteRenderer, error) {
	logger := log.New(os.Stdout, fmt.Sprint(REWRITE_LOGGER, ": "), log.LstdFlags|log.Lmsgprefix)

	regexps := []*regexp.Regexp{}
	for _, rule := range config.Rules {
		r, err := regexp.Compile(rule.Path)
		if err != nil {
			return nil, err
		}

		regexps = append(regexps, r)
	}

	return &rewriteRenderer{
		config:  config,
		logger:  logger,
		regexps: regexps,
	}, nil
}

// handle implements the rewrite handler
func (r *rewriteRenderer) handle(w http.ResponseWriter, req *http.Request) {
	for index, regexp := range r.regexps {
		if regexp.MatchString(req.URL.Path) {
			stop := false
			status := http.StatusFound

			switch r.config.Rules[index].Flag {
			case REWRITE_RULE_FLAG_REDIRECT:
				stop = true
				status = http.StatusFound
			case REWRITE_RULE_FLAG_PERMANENT:
				stop = true
				status = http.StatusMovedPermanently
			}
			if strings.HasPrefix(r.config.Rules[index].Replacement, "http://") ||
				strings.HasPrefix(r.config.Rules[index].Replacement, "https://") {
				stop = true
			}

			if stop {
				http.Redirect(w, req, r.config.Rules[index].Replacement, status)

				r.logger.Printf("Rewrite processed (url=%s, status=%d, target=%s)", req.URL.Path, status,
					r.config.Rules[index].Replacement)

				return
			}

			url := req.URL.Path
			req.URL.Path = r.config.Rules[index].Replacement

			r.logger.Printf("Rewrite processed (url=%s, status=%d, target=%s)", url, status, req.URL.Path)

			if r.config.Rules[index].Last {
				break
			}
		}
	}

	r.next.handle(w, req)
}

// setNext configures the next renderer
func (r *rewriteRenderer) setNext(renderer Renderer) {
	r.next = renderer
}
