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
	RequestTLSCAFile   string
	RequestTLSCertFile string
	RequestTLSKeyFile  string
	RequestHeaders     map[string]string
	RequestTimeout     int
	RequestRetry       int
	RequestDelay       int
	Resources          []FetcherResource
	Templates          []FetcherTemplate
}

// FetcherResource implements a fetcher resource
type FetcherResource struct {
	Key     string
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

// NewFetcher creates a new instance
func NewFetcher(config *FetcherConfig) *fetcher {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if config.RequestTLSCAFile != "" {
		ca, err := os.ReadFile(config.RequestTLSCAFile)
		if err != nil {
			log.Fatal(err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(ca) {
			log.Fatal(err)
		}

		tlsConfig.RootCAs = caCertPool

		if config.RequestTLSCertFile != "" && config.RequestTLSKeyFile != "" {
			clientCert, err := tls.LoadX509KeyPair(config.RequestTLSCertFile, config.RequestTLSKeyFile)
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
		logger:        log.Default(),
		client:        &client,
		resources:     make(map[string]*Resource),
		resourcesLock: sync.RWMutex{},
		data:          NewCache(),
	}

	for _, resource := range config.Resources {
		fetcher.Register(&Resource{
			Key:     resource.Key,
			Method:  resource.Method,
			URL:     resource.URL,
			Params:  resource.Params,
			Headers: resource.Headers,
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
			f.logger.Printf("Fetch request: %s %s %s %s\n", method, r.URL.Path, r.URL.RawQuery, response.Status)
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
func (f *fetcher) Fetch(ctx context.Context, key string) error {
	f.resourcesLock.RLock()
	defer f.resourcesLock.RUnlock()

	if resource, ok := f.resources[key]; ok {
		data, err := f.request(ctx, resource.Method, resource.URL, resource.Params, resource.Headers)
		if err != nil {
			f.logger.Printf("Failed to fetch resource '%s': %s", resource.Key, err)

			return err
		}

		f.data.Set(resource.Key, data, time.Duration(resource.TTL)*time.Second)

		return nil
	}

	return fmt.Errorf("no resource found")
}

// FetchAll fetches all registered resources
func (f *fetcher) FetchAll(ctx context.Context, skip bool) error {
	f.resourcesLock.RLock()
	defer f.resourcesLock.RUnlock()

	var err error
	for _, resource := range f.resources {
		data, err := f.request(ctx, resource.Method, resource.URL, resource.Params, resource.Headers)
		if err != nil {
			f.logger.Printf("Failed to fetch resource '%s': %s", resource.Key, err)

			if skip {
				continue
			}

			return err
		}

		f.data.Set(resource.Key, data, time.Duration(resource.TTL)*time.Second)
	}

	return err
}

// Exists checks if a resource already exists
func (f *fetcher) Exists(key string) bool {
	f.resourcesLock.Lock()
	defer f.resourcesLock.Unlock()

	if _, ok := f.resources[key]; !ok {
		return false
	}

	return true
}

// Get returns the last fetched data of a resource
func (f *fetcher) Get(key string) ([]byte, error) {
	obj := f.data.Get(key)
	if obj == nil {
		return nil, fmt.Errorf("no data found")
	}

	return obj.([]byte), nil
}

// Register registers a resource
func (f *fetcher) Register(resource *Resource) {
	f.resourcesLock.Lock()
	defer f.resourcesLock.Unlock()

	f.resources[resource.Key] = resource
}

// Unregister unregisters a resource
func (f *fetcher) Unregister(key string) {
	f.resourcesLock.Lock()
	defer f.resourcesLock.Unlock()

	if _, ok := f.resources[key]; ok {
		delete(f.resources, key)
	}
}

// CreateResourceFromTemplate creates a resource from a template
func (f *fetcher) CreateResourceFromTemplate(name string, key string, params map[string]string,
	headers map[string]string) (*Resource, error) {
	for _, template := range f.config.Templates {
		if template.Name != name {
			continue
		}

		resourceParams := make(map[string]string)
		for k, v := range template.Params {
			resourceParams[k] = v
		}
		for k, v := range params {
			resourceParams[k] = v
		}

		resourceHeaders := make(map[string]string)
		for k, v := range template.Headers {
			resourceHeaders[k] = v
		}
		for k, v := range headers {
			resourceHeaders[k] = v
		}

		return &Resource{
			Key:     key,
			Method:  template.Method,
			URL:     template.URL,
			Params:  resourceParams,
			Headers: resourceHeaders,
		}, nil
	}

	return nil, errors.New("failed to find template")
}
