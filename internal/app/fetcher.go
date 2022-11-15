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

// fetcher implements a fetcher
type fetcher struct {
	config                         *FetcherConfig
	logger                         *log.Logger
	requester                      FetchRequester
	resources                      map[string]Resource
	resourcesLock                  sync.RWMutex
	data                           Cache
	osReadFile                     func(name string) ([]byte, error)
	x509CertPoolAppendCertsFromPEM func(pool *x509.CertPool, pemCerts []byte) bool
	tlsLoadX509KeyPair             func(certFile, keyFile string) (tls.Certificate, error)
}

// FetcherConfig implements a fetcher configuration
type FetcherConfig struct {
	RequestTLSCAFile   *string
	RequestTLSCertFile *string
	RequestTLSKeyFile  *string
	RequestHeaders     map[string]string
	RequestTimeout     int
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
	fetcherLogger string = "fetcher"
)

// fetcherOsReadFile redirects to os.ReadFile
func fetcherOsReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// fetcherTlsLoadX509KeyPair redirects to tls.LoadX509KeyPair
func fetcherTlsLoadX509KeyPair(certFile string, keyFile string) (tls.Certificate, error) {
	return tls.LoadX509KeyPair(certFile, keyFile)
}

// fetcherX509CertPoolAppendCertsFromPEM redirects to x509.CertPool.AppendCertsFromPEM
func fetcherX509CertPoolAppendCertsFromPEM(pool *x509.CertPool, pemCerts []byte) bool {
	return pool.AppendCertsFromPEM(pemCerts)
}

// CreateFetcher creates a new fetcher instance
func CreateFetcher(config *FetcherConfig) (*fetcher, error) {
	return createFetcherWithRequester(config, nil)
}

// createFetcherWithRequester creates a new fetcher instance with the given requester
func createFetcherWithRequester(config *FetcherConfig, fetchRequester FetchRequester) (*fetcher, error) {
	f := &fetcher{
		config:                         config,
		logger:                         log.New(os.Stderr, fmt.Sprint(fetcherLogger, ": "), log.LstdFlags|log.Lmsgprefix),
		requester:                      fetchRequester,
		resources:                      make(map[string]Resource),
		data:                           newCache(),
		osReadFile:                     fetcherOsReadFile,
		tlsLoadX509KeyPair:             fetcherTlsLoadX509KeyPair,
		x509CertPoolAppendCertsFromPEM: fetcherX509CertPoolAppendCertsFromPEM,
	}

	err := f.initialize(fetchRequester)
	if err != nil {
		return nil, err
	}

	return f, nil
}

// initialize initializes the fetcher
func (f *fetcher) initialize(fetchRequester FetchRequester) error {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if f.config.RequestTLSCAFile != nil {
		ca, err := f.osReadFile(*f.config.RequestTLSCAFile)
		if err != nil {
			return err
		}

		caCertPool := x509.NewCertPool()
		if !f.x509CertPoolAppendCertsFromPEM(caCertPool, ca) {
			return errors.New("CA certificate not added into pool")
		}

		tlsConfig.RootCAs = caCertPool

		if f.config.RequestTLSCertFile != nil && f.config.RequestTLSKeyFile != nil {
			clientCert, err := f.tlsLoadX509KeyPair(*f.config.RequestTLSCertFile, *f.config.RequestTLSKeyFile)
			if err != nil {
				return err
			}

			tlsConfig.Certificates = []tls.Certificate{clientCert}
		}
	}

	transport := http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout: time.Duration(f.config.RequestTimeout) * time.Second,
		}).Dial,
		TLSClientConfig:       tlsConfig,
		TLSHandshakeTimeout:   time.Duration(f.config.RequestTimeout) * time.Second,
		ResponseHeaderTimeout: time.Duration(f.config.RequestTimeout) * time.Second,
		ExpectContinueTimeout: time.Duration(f.config.RequestTimeout) * time.Second,
		ForceAttemptHTTP2:     true,
	}

	client := http.Client{
		Transport: &transport,
		Timeout:   time.Duration(f.config.RequestTimeout) * time.Second,
	}

	if fetchRequester == nil {
		f.requester = newFetchRequester(f.logger, &client, f.config.RequestHeaders)
	}

	for _, r := range f.config.Resources {
		f.Register(Resource{
			Name:    r.Name,
			Method:  r.Method,
			URL:     r.URL,
			Params:  r.Params,
			Headers: r.Headers,
		})
	}

	return nil
}

// Fetch fetches a registered resource
func (f *fetcher) Fetch(ctx context.Context, name string) error {
	f.resourcesLock.RLock()
	defer f.resourcesLock.RUnlock()

	if r, ok := f.resources[name]; ok {
		data, err := f.requester.Fetch(ctx, r.Method, r.URL, r.Params, r.Headers)
		if err != nil {
			f.logger.Printf("Failed to fetch resource '%s': %s", r.Name, err)

			return err
		}

		f.data.Set(r.Name, data, time.Duration(r.TTL)*time.Second)

		return nil
	}

	return fmt.Errorf("no resource found")
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
	f.resourcesLock.Lock()
	defer f.resourcesLock.Unlock()

	obj := f.data.Get(name)
	if obj == nil {
		return nil, fmt.Errorf("no data found")
	}

	return obj.([]byte), nil
}

// Register registers a resource
func (f *fetcher) Register(r Resource) {
	f.resourcesLock.Lock()
	defer f.resourcesLock.Unlock()

	f.resources[r.Name] = r
}

// Unregister unregisters a resource
func (f *fetcher) Unregister(name string) {
	f.resourcesLock.Lock()
	defer f.resourcesLock.Unlock()

	delete(f.resources, name)
}

// CreateResourceFromTemplate creates a resource from a template
func (f *fetcher) CreateResourceFromTemplate(template string, resource string, params map[string]string,
	headers map[string]string) (*Resource, error) {
	for _, t := range f.config.Templates {
		if t.Name != template {
			continue
		}

		var rParams map[string]string
		if t.Params != nil {
			rParams = make(map[string]string)
			for k, v := range t.Params {
				rParams[k] = v
			}
			for k, v := range params {
				rParams[k] = v
			}
		}

		var rHeaders map[string]string
		if t.Headers != nil {
			rHeaders = make(map[string]string)
			for k, v := range t.Headers {
				rHeaders[k] = v
			}
			for k, v := range headers {
				rHeaders[k] = v
			}
		}

		return &Resource{
			Name:    resource,
			Method:  t.Method,
			URL:     t.URL,
			Params:  rParams,
			Headers: rHeaders,
		}, nil
	}

	return nil, errors.New("template not found")
}

// FetchRequester
type FetchRequester interface {
	Fetch(ctx context.Context, method string, url string, params map[string]string,
		headers map[string]string) ([]byte, error)
}

// fetchRequester implements the default fetch requester
type fetchRequester struct {
	logger                    *log.Logger
	client                    *http.Client
	headers                   map[string]string
	retry                     int
	delay                     time.Duration
	httpNewRequestWithContext func(ctx context.Context, method string, url string, body io.Reader) (*http.Request, error)
	httpClientDo              func(client *http.Client, req *http.Request) (*http.Response, error)
	ioReadAll                 func(r io.Reader) ([]byte, error)
}

// fetchRequesterHttpNewRequestWithContext redirects to http.NewRequestWithContext
func fetchRequesterHttpNewRequestWithContext(ctx context.Context, method string, url string,
	body io.Reader) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, method, url, body)
}

// fetchRequesterHttpClientDo redirects to http.Client.Do
func fetchRequesterHttpClientDo(client *http.Client, req *http.Request) (*http.Response, error) {
	return client.Do(req)
}

// fetchRequesterIoReadAll redirects to io.ReadAll
func fetchRequesterIoReadAll(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}

// newFetchRequester creates a new fetch requester instance
func newFetchRequester(logger *log.Logger, client *http.Client, headers map[string]string) *fetchRequester {
	return &fetchRequester{
		logger:                    logger,
		client:                    client,
		headers:                   headers,
		httpNewRequestWithContext: fetchRequesterHttpNewRequestWithContext,
		httpClientDo:              fetchRequesterHttpClientDo,
		ioReadAll:                 fetchRequesterIoReadAll,
	}
}

// Fetch fetches an API with the given parameters
func (r *fetchRequester) Fetch(ctx context.Context, method string, url string, params map[string]string,
	headers map[string]string) ([]byte, error) {
	req, err := r.httpNewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		r.logger.Printf("Failed to create request: %s", err)

		return nil, err
	}

	query := req.URL.Query()
	for key, value := range params {
		query.Add(key, value)
	}
	req.URL.RawQuery = query.Encode()

	for key, value := range r.headers {
		req.Header.Set(key, value)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	response, err := r.httpClientDo(r.client, req)
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		r.logger.Printf("Failed to send request: %s %s %s", method, url, err)

		return nil, err
	}

	responseBody, err := r.ioReadAll(response.Body)
	if err != nil {
		r.logger.Printf("Failed to read response: %s", err)

		return nil, err
	}

	if DEBUG {
		r.logger.Printf("Fetch request: method=%s, url=%s, code=%d\n", method, req.URL.String(), response.StatusCode)
	}

	switch response.StatusCode {
	case 429, 500, 502, 503, 504:
		return nil, fmt.Errorf("request error %d", response.StatusCode)

	default:
		if response.StatusCode < 200 || response.StatusCode > 299 {
			return nil, fmt.Errorf("request error %d", response.StatusCode)
		}
	}

	return responseBody, nil
}
