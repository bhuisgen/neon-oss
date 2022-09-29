// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

// fetcher implements the resources fetcher
type fetcher struct {
	config        *FetcherConfig
	logger        *log.Logger
	client        *http.Client
	resources     map[string]*Resource
	resourcesLock sync.RWMutex
	data          *cache
}

// FetcherConfig implements the resources fetcher configuration
type FetcherConfig struct {
	RequestTLSCAFile   *string
	RequestTLSCertFile *string
	RequestTLSKeyFile  *string
	RequestHeaders     map[string]string
	RequestTimeout     int
	RequestRetry       int
	RequestDelay       int
	Resources          []FetcherResource
	Templates          []FetcherTemplate
}

// FetcherResource implements a fetcher resource
type FetcherResource struct {
	Name    string
	Method  string
	URL     string
	Params  map[string]string
	Headers map[string]string
}

// FetcherTemplate implements a fetcher template
type FetcherTemplate struct {
	Name    string
	Method  string
	URL     string
	Params  map[string]string
	Headers map[string]string
}

const (
	FETCHER_LOGGER string = "fetcher"
)

// NewFetcher creates a new instance
func NewFetcher(config *FetcherConfig) *fetcher {
	logger := log.New(os.Stdout, fmt.Sprint(FETCHER_LOGGER, ": "), log.LstdFlags|log.Lmsgprefix)

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if config.RequestTLSCAFile != nil {
		ca, err := os.ReadFile(*config.RequestTLSCAFile)
		if err != nil {
			log.Fatal(err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(ca) {
			log.Fatal(err)
		}

		tlsConfig.RootCAs = caCertPool

		if config.RequestTLSCertFile != nil && config.RequestTLSKeyFile != nil {
			clientCert, err := tls.LoadX509KeyPair(*config.RequestTLSCertFile, *config.RequestTLSKeyFile)
			if err != nil {
				log.Fatal(err)
			}

			tlsConfig.Certificates = []tls.Certificate{clientCert}
		}
	}

	transport := http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout: time.Duration(config.RequestTimeout) * time.Second,
		}).Dial,
		TLSClientConfig:       tlsConfig,
		TLSHandshakeTimeout:   time.Duration(config.RequestTimeout) * time.Second,
		ResponseHeaderTimeout: time.Duration(config.RequestTimeout) * time.Second,
		ExpectContinueTimeout: time.Duration(config.RequestTimeout) * time.Second,
		ForceAttemptHTTP2:     true,
	}

	client := http.Client{
		Transport: &transport,
		Timeout:   time.Duration(config.RequestTimeout) * time.Second,
	}

	fetcher := &fetcher{
		config:        config,
		logger:        logger,
		client:        &client,
		resources:     make(map[string]*Resource),
		resourcesLock: sync.RWMutex{},
		data:          NewCache(),
	}

	for _, r := range config.Resources {
		fetcher.Register(&Resource{
			Name:    r.Name,
			Method:  r.Method,
			URL:     r.URL,
			Params:  r.Params,
			Headers: r.Headers,
		})
	}

	return fetcher
}

// request fetches an API with the given parameters
func (f *fetcher) request(ctx context.Context, method string, url string, params map[string]string,
	headers map[string]string) ([]byte, error) {
	r, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		f.logger.Printf("Failed to create request: %s", err)

		return nil, err
	}

	query := r.URL.Query()
	for key, value := range params {
		query.Add(key, value)
	}
	r.URL.RawQuery = query.Encode()

	for key, value := range f.config.RequestHeaders {
		r.Header.Set(key, value)
	}
	for key, value := range headers {
		r.Header.Set(key, value)
	}

	var attempt int
	for {
		attempt += 1

		response, err := f.client.Do(r)
		if err != nil {
			f.logger.Printf("Failed to send request: %s %s %s", method, url, err)

			return nil, err
		}
		defer response.Body.Close()

		responseBody, err := io.ReadAll(response.Body)
		if err != nil {
			f.logger.Printf("Failed to read request response: %s", err)

			return nil, err
		}

		if _, ok := os.LookupEnv("DEBUG"); ok {
			f.logger.Printf("Fetch request: method=%s, url=%s, code=%d\n", method, r.URL.String(), response.StatusCode)
		}

		switch response.StatusCode {
		case 429, 500, 502, 503, 504:
			if attempt >= f.config.RequestRetry {
				return nil, fmt.Errorf("request error %d", response.StatusCode)
			}

			if f.config.RequestDelay > 0 {
				f.logger.Printf("Retrying request attempt %d/%d, delaying for %d seconds", attempt, f.config.RequestRetry,
					f.config.RequestDelay)

				time.Sleep(time.Duration(f.config.RequestDelay) * time.Second)
			} else {
				f.logger.Printf("Retrying request attempt %d/%d", attempt, f.config.RequestRetry)
			}

			continue

		default:
			if response.StatusCode < 200 || response.StatusCode > 299 {
				return nil, fmt.Errorf("request error %d", response.StatusCode)
			}

			break
		}

		return responseBody, nil
	}
}

// Fetch fetches a registered resource
func (f *fetcher) Fetch(ctx context.Context, name string) error {
	f.resourcesLock.RLock()
	defer f.resourcesLock.RUnlock()

	if r, ok := f.resources[name]; ok {
		data, err := f.request(ctx, r.Method, r.URL, r.Params, r.Headers)
		if err != nil {
			f.logger.Printf("Failed to fetch resource '%s': %s", r.Name, err)

			return err
		}

		f.data.Set(r.Name, data, time.Duration(r.TTL)*time.Second)

		return nil
	}

	return fmt.Errorf("no resource found")
}

// FetchAll fetches all registered resources
func (f *fetcher) FetchAll(ctx context.Context, skip bool) error {
	f.resourcesLock.RLock()
	defer f.resourcesLock.RUnlock()

	var err error
	for _, r := range f.resources {
		data, err := f.request(ctx, r.Method, r.URL, r.Params, r.Headers)
		if err != nil {
			f.logger.Printf("Failed to fetch resource '%s': %s", r.Name, err)

			if skip {
				continue
			}

			return err
		}

		f.data.Set(r.Name, data, time.Duration(r.TTL)*time.Second)
	}

	return err
}

// Exists checks if a resource exists
func (f *fetcher) Exists(name string) bool {
	f.resourcesLock.Lock()
	defer f.resourcesLock.Unlock()

	if _, ok := f.resources[name]; !ok {
		return false
	}

	return true
}

// Get returns the last fetched data of a resource
func (f *fetcher) Get(name string) ([]byte, error) {
	obj := f.data.Get(name)
	if obj == nil {
		return nil, fmt.Errorf("no data found")
	}

	return obj.([]byte), nil
}

// Register registers a resource
func (f *fetcher) Register(r *Resource) {
	f.resourcesLock.Lock()
	defer f.resourcesLock.Unlock()

	f.resources[r.Name] = r
}

// Unregister unregisters a resource
func (f *fetcher) Unregister(name string) {
	f.resourcesLock.Lock()
	defer f.resourcesLock.Unlock()

	if _, ok := f.resources[name]; ok {
		delete(f.resources, name)
	}
}

// CreateResourceFromTemplate creates a resource from a template
func (f *fetcher) CreateResourceFromTemplate(template string, resource string, params map[string]string,
	headers map[string]string) (*Resource, error) {
	for _, t := range f.config.Templates {
		if t.Name != template {
			continue
		}

		rParams := make(map[string]string)
		for k, v := range t.Params {
			rParams[k] = v
		}
		for k, v := range params {
			rParams[k] = v
		}

		rHeaders := make(map[string]string)
		for k, v := range t.Headers {
			rHeaders[k] = v
		}
		for k, v := range headers {
			rHeaders[k] = v
		}

		return &Resource{
			Name:    resource,
			Method:  t.Method,
			URL:     t.URL,
			Params:  rParams,
			Headers: rHeaders,
		}, nil
	}

	return nil, errors.New("failed to find template")
}
