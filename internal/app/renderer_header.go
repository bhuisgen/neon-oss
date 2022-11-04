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
)

// headerRenderer implements the header renderer
type headerRenderer struct {
	config  *HeaderRendererConfig
	logger  *log.Logger
	regexps []*regexp.Regexp
	next    Renderer
}

// HeaderRendererConfig implements the header renderer configuration
type HeaderRendererConfig struct {
	Rules []HeaderRule
}

// HeaderRule implements a header rule
type HeaderRule struct {
	Path string
	Set  map[string]string
	Last bool
}

const (
	headerLogger string = "server[header]"
)

// CreateHeaderRenderer creates a new header renderer
func CreateHeaderRenderer(config *HeaderRendererConfig) (*headerRenderer, error) {
	r := headerRenderer{
		config:  config,
		logger:  log.New(os.Stderr, fmt.Sprint(headerLogger, ": "), log.LstdFlags|log.Lmsgprefix),
		regexps: []*regexp.Regexp{},
	}

	err := r.initialize()
	if err != nil {
		return nil, err
	}

	return &r, nil
}

// initialize initializes the renderer
func (r *headerRenderer) initialize() error {
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
func (r *headerRenderer) Handle(w http.ResponseWriter, req *http.Request, info *ServerInfo) {
	for index, regexp := range r.regexps {
		if regexp.MatchString(req.URL.Path) {
			for k, v := range r.config.Rules[index].Set {
				w.Header().Set(k, v)
			}
			if r.config.Rules[index].Last {
				break
			}
		}
	}

	r.next.Handle(w, req, info)
}

// Next configures the next renderer
func (r *headerRenderer) Next(renderer Renderer) {
	r.next = renderer
}
