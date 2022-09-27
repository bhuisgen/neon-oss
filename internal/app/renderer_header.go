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
	Renderer
	next Renderer

	config  *HeaderRendererConfig
	logger  *log.Logger
	regexps []*regexp.Regexp
}

// HeaderRendererConfig implements the header renderer configuration
type HeaderRendererConfig struct {
	Enable bool
	Rules  []HeaderRule
}

// HeaderRule implements a header rule
type HeaderRule struct {
	Path string
	Add  map[string]string
	Last bool
}

const (
	HEADER_LOGGER string = "renderer[header]"
)

// CreateHeaderRenderer creates a new header renderer
func CreateHeaderRenderer(config *HeaderRendererConfig) (*headerRenderer, error) {
	logger := log.New(os.Stdout, fmt.Sprint(HEADER_LOGGER, ": "), log.LstdFlags|log.Lmsgprefix)

	regexps := []*regexp.Regexp{}
	for _, rule := range config.Rules {
		r, err := regexp.Compile(rule.Path)
		if err != nil {
			return nil, err
		}

		regexps = append(regexps, r)
	}

	return &headerRenderer{
		config:  config,
		logger:  logger,
		regexps: regexps,
	}, nil
}

// handle implements the header handler
func (r *headerRenderer) handle(w http.ResponseWriter, req *http.Request) {
	for index, regexp := range r.regexps {
		if regexp.MatchString(req.URL.Path) {
			for k, v := range r.config.Rules[index].Add {
				if w.Header().Get(k) == "" {
					w.Header().Add(k, v)
				}
			}

			if r.config.Rules[index].Last {
				break
			}
		}
	}

	r.next.handle(w, req)
}

// setNext configures the next renderer
func (r *headerRenderer) setNext(renderer Renderer) {
	r.next = renderer
}
