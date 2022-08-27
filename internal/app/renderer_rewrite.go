// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"log"
	"net/http"
	"regexp"
	"strings"
)

const (
	REWRITE_RULE_FLAG_REDIRECT  = "redirect"
	REWRITE_RULE_FLAG_PERMANENT = "permanent"
)

// rewriteRenderer implements the rewrite renderer
type rewriteRenderer struct {
	Renderer
	next Renderer

	config   *RewriteRendererConfig
	logger   *log.Logger
	rewrites []*regexp.Regexp
}

// RewriteRendererConfig implements the rewrite renderer configuration
type RewriteRendererConfig struct {
	Enable bool
	Rules  []RewriteRule
}

// RewriteRule implements a rewrite rule
type RewriteRule struct {
	Regex       string
	Replacement string
	Flag        string
}

// CreateRewriteRenderer creates a new rewrite renderer
func CreateRewriteRenderer(config *RewriteRendererConfig) (*rewriteRenderer, error) {
	rewrites := []*regexp.Regexp{}

	for _, rule := range config.Rules {
		rewrite, err := regexp.Compile(rule.Regex)
		if err != nil {
			return nil, err
		}

		rewrites = append(rewrites, rewrite)
	}

	return &rewriteRenderer{
		config:   config,
		logger:   log.Default(),
		rewrites: rewrites,
	}, nil
}

// handle implements the rewrite handler
func (r *rewriteRenderer) handle(w http.ResponseWriter, req *http.Request) {
	if !r.config.Enable {
		r.next.handle(w, req)

		return
	}

	for rewriteIndex, rewrite := range r.rewrites {
		if rewrite.MatchString(req.URL.Path) {
			stop := false
			status := http.StatusFound

			switch r.config.Rules[rewriteIndex].Flag {
			case REWRITE_RULE_FLAG_REDIRECT:
				stop = true
				status = http.StatusFound
			case REWRITE_RULE_FLAG_PERMANENT:
				stop = true
				status = http.StatusMovedPermanently
			}
			if strings.HasPrefix(r.config.Rules[rewriteIndex].Replacement, "http://") ||
				strings.HasPrefix(r.config.Rules[rewriteIndex].Replacement, "https://") {
				stop = true
			}

			if stop {
				http.Redirect(w, req, r.config.Rules[rewriteIndex].Replacement, status)

				r.logger.Printf("Rewrite processed (url=%s, status=%d, target=%s)", req.URL.Path, status,
					r.config.Rules[rewriteIndex].Replacement)

				return
			}

			url := req.URL.Path
			req.URL.Path = r.config.Rules[rewriteIndex].Replacement

			r.logger.Printf("Rewrite processed (url=%s, status=%d, target=%s)", url, status, req.URL.Path)

			break
		}
	}

	r.next.handle(w, req)
}

// setNext configures the next renderer
func (r *rewriteRenderer) setNext(renderer Renderer) {
	r.next = renderer
}
