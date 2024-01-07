// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rest

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PaesslerAG/jsonpath"
	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
)

// restProvider implements the rest provider.
type restProvider struct {
	config                         *restProviderConfig
	logger                         *log.Logger
	client                         http.Client
	osOpenFile                     func(name string, flag int, perm fs.FileMode) (*os.File, error)
	osReadFile                     func(name string) ([]byte, error)
	osClose                        func(f *os.File) error
	osStat                         func(name string) (fs.FileInfo, error)
	x509CertPoolAppendCertsFromPEM func(pool *x509.CertPool, pemCerts []byte) bool
	tlsLoadX509KeyPair             func(certFile, keyFile string) (tls.Certificate, error)
	httpNewRequestWithContext      func(ctx context.Context, method string, url string, body io.Reader) (*http.Request, error)
	httpClientDo                   func(client *http.Client, req *http.Request) (*http.Response, error)
	ioReadAll                      func(r io.Reader) ([]byte, error)
}

// restProviderConfig implements the rest provider configuration.
type restProviderConfig struct {
	TLSCAFiles          *[]string
	TLSCertFiles        *[]string
	TLSKeyFiles         *[]string
	Timeout             *int
	MaxConnsPerHost     *int
	MaxIdleConns        *int
	MaxIdleConnsPerHost *int
	IdleConnTimeout     *int
	Retry               *int
	RetryDelay          *int
	Headers             map[string]string
	Params              map[string]string
}

// restResourceConfig implements the rest resource configuration.
type restResourceConfig struct {
	Method     *string
	URL        string
	Params     map[string]string
	Headers    map[string]string
	Next       *bool
	NextParser *string
	NextFilter *string
}

const (
	restModuleID module.ModuleID = "fetcher.provider.rest"
	restLogger   string          = "fetcher.provider[rest]"

	restConfigDefaultTimeout             int = 30
	restConfigDefaultMaxConnsPerHost     int = 100
	restConfigDefaultMaxIdleConns        int = 100
	restConfigDefaultMaxIdleConnsPerHost int = 100
	restConfigDefaultIdleConnTimeout     int = 60
	restConfigDefaultRetry               int = 3
	restConfigDefaultRetryDelay          int = 1

	restResourceNextParserHeader  string = "header"
	restResourceNextParserBody    string = "body"
	restResourceDefaultMethod     string = http.MethodGet
	restResourceDefaultNextParser string = restResourceNextParserBody
)

// restOsOpenFile redirects to os.OpenFile.
func restOsOpenFile(name string, flag int, perm fs.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

// restOsReadFile redirects to os.ReadFile.
func restOsReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// restOsClose redirects to os.Close.
func restOsClose(f *os.File) error {
	return f.Close()
}

// restOsStat redirects to os.Stat.
func restOsStat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

// restX509CertPoolAppendCertsFromPEM redirects to x509.CertPool.AppendCertsFromPEM.
func restX509CertPoolAppendCertsFromPEM(pool *x509.CertPool, pemCerts []byte) bool {
	return pool.AppendCertsFromPEM(pemCerts)
}

// restTLSLoadX509KeyPair redirects to tls.LoadX509KeyPair.
func restTLSLoadX509KeyPair(certFile string, keyFile string) (tls.Certificate, error) {
	return tls.LoadX509KeyPair(certFile, keyFile)
}

// restRequesterHttpNewRequestWithContext redirects to http.NewRequestWithContext.
func restHttpNewRequestWithContext(ctx context.Context, method string, url string,
	body io.Reader) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, method, url, body)
}

// restRequesterHttpClientDo redirects to http.Client.Do.
func restHttpClientDo(client *http.Client, req *http.Request) (*http.Response, error) {
	return client.Do(req)
}

// fetchRequesterIoReadAll redirects to io.ReadAll.
func restIoReadAll(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}

// init initializes the module.
func init() {
	module.Register(restProvider{})
}

// ModuleInfo returns the module information.
func (p restProvider) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: restModuleID,
		NewInstance: func() module.Module {
			return &restProvider{
				osOpenFile:                     restOsOpenFile,
				osReadFile:                     restOsReadFile,
				osClose:                        restOsClose,
				osStat:                         restOsStat,
				x509CertPoolAppendCertsFromPEM: restX509CertPoolAppendCertsFromPEM,
				tlsLoadX509KeyPair:             restTLSLoadX509KeyPair,
				httpClientDo:                   restHttpClientDo,
				httpNewRequestWithContext:      restHttpNewRequestWithContext,
				ioReadAll:                      restIoReadAll,
			}
		},
	}
}

// Check checks the provider configuration.
func (p *restProvider) Check(config map[string]interface{}) ([]string, error) {
	var report []string

	var c restProviderConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		report = append(report, "failed to parse configuration")
		return report, err
	}

	if c.TLSCAFiles != nil {
		for _, item := range *c.TLSCAFiles {
			if item == "" {
				report = append(report, fmt.Sprintf("option '%s', invalid value '%s'", "TLSCAFiles", item))
			} else {
				file, err := p.osOpenFile(item, os.O_RDONLY, 0)
				if err != nil {
					report = append(report, fmt.Sprintf("option '%s', failed to open file '%s'", "TLSCAFiles", item))
				} else {
					p.osClose(file)
					fi, err := p.osStat(item)
					if err != nil {
						report = append(report, fmt.Sprintf("option '%s', failed to stat file '%s'", "TLSCAFiles", item))
					}
					if err == nil && fi.IsDir() {
						report = append(report, fmt.Sprintf("option '%s', '%s' is a directory", "TLSCAFiles", item))
					}
				}
			}
		}
	}
	if c.TLSCertFiles != nil {
		for _, item := range *c.TLSCertFiles {
			if item == "" {
				report = append(report, fmt.Sprintf("option '%s', invalid value '%s'", "TLSCertFiles", item))
			} else {
				file, err := p.osOpenFile(item, os.O_RDONLY, 0)
				if err != nil {
					report = append(report, fmt.Sprintf("option '%s', failed to open file '%s'", "TLSCertFiles", item))
				} else {
					p.osClose(file)
					fi, err := p.osStat(item)
					if err != nil {
						report = append(report, fmt.Sprintf("option '%s', failed to stat file '%s'", "TLSCertFiles", item))
					}
					if err == nil && fi.IsDir() {
						report = append(report, fmt.Sprintf("option '%s', '%s' is a directory", "TLSCertFiles", item))
					}
				}
			}
		}
	}
	if c.TLSKeyFiles != nil {
		if c.TLSCertFiles == nil || len(*c.TLSKeyFiles) != len(*c.TLSCertFiles) {
			report = append(report, fmt.Sprintf("option '%s', missing value(s)", "TLSKeyFiles"))
		}
		for _, item := range *c.TLSKeyFiles {
			if item == "" {
				report = append(report, fmt.Sprintf("option '%s', invalid value '%s'", "TLSKeyFiles", item))
			} else {
				file, err := p.osOpenFile(item, os.O_RDONLY, 0)
				if err != nil {
					report = append(report, fmt.Sprintf("option '%s', failed to open file '%s'", "TLSKeyFiles", item))
				} else {
					p.osClose(file)
					fi, err := p.osStat(item)
					if err != nil {
						report = append(report, fmt.Sprintf("option '%s', failed to stat file '%s'", "TLSKeyFiles", item))
					}
					if err == nil && fi.IsDir() {
						report = append(report, fmt.Sprintf("option '%s', '%s' is a directory", "TLSKeyFiles", item))
					}
				}
			}
		}
	}
	if c.Timeout == nil {
		defaultValue := restConfigDefaultTimeout
		c.Timeout = &defaultValue
	}
	if *c.Timeout < 0 {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%d'", "Timeout", *c.Timeout))
	}
	if c.MaxConnsPerHost == nil {
		defaultValue := restConfigDefaultMaxConnsPerHost
		c.MaxConnsPerHost = &defaultValue
	}
	if *c.MaxConnsPerHost < 0 {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%d'", "MaxConnsPerHost", *c.MaxConnsPerHost))
	}
	if c.MaxIdleConns == nil {
		defaultValue := restConfigDefaultMaxIdleConns
		c.MaxIdleConns = &defaultValue
	}
	if *c.MaxIdleConns < 0 {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%d'", "MaxIdleConns", *c.MaxIdleConns))
	}
	if c.MaxIdleConnsPerHost == nil {
		defaultValue := restConfigDefaultMaxIdleConnsPerHost
		c.MaxIdleConnsPerHost = &defaultValue
	}
	if *c.MaxIdleConnsPerHost < 0 {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%d'", "MaxIdleConnsPerHost",
			*c.MaxIdleConnsPerHost))
	}
	if c.IdleConnTimeout == nil {
		defaultValue := restConfigDefaultIdleConnTimeout
		c.IdleConnTimeout = &defaultValue
	}
	if *c.IdleConnTimeout < 0 {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%d'", "IdleConnTimeout", *c.IdleConnTimeout))
	}
	if c.Retry == nil {
		defaultValue := restConfigDefaultRetry
		c.Retry = &defaultValue
	}
	if *c.Retry < 0 {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%d'", "Retry", *c.Retry))
	}
	if c.RetryDelay == nil {
		defaultValue := restConfigDefaultRetryDelay
		c.RetryDelay = &defaultValue
	}
	if *c.RetryDelay < 0 {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%d'", "RetryDelay", *c.RetryDelay))
	}
	for k := range c.Headers {
		if k == "" {
			report = append(report, fmt.Sprintf("option '%s', invalid key '%s'", "Headers", k))
		}
	}
	for k := range c.Params {
		if k == "" {
			report = append(report, fmt.Sprintf("option '%s', invalid key '%s'", "Params", k))
		}
	}

	if len(report) > 0 {
		return report, errors.New("check failure")
	}

	return nil, nil
}

// Load loads the provider.
func (p *restProvider) Load(config map[string]interface{}) error {
	var c restProviderConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		return err
	}

	p.config = &c
	p.logger = log.New(os.Stderr, restLogger+": ", log.LstdFlags|log.Lmsgprefix)

	if p.config.Timeout == nil {
		defaultValue := restConfigDefaultTimeout
		p.config.Timeout = &defaultValue
	}
	if p.config.MaxConnsPerHost == nil {
		defaultValue := restConfigDefaultMaxConnsPerHost
		p.config.MaxConnsPerHost = &defaultValue
	}
	if p.config.MaxIdleConns == nil {
		defaultValue := restConfigDefaultMaxIdleConns
		p.config.MaxIdleConns = &defaultValue
	}
	if p.config.MaxIdleConnsPerHost == nil {
		defaultValue := restConfigDefaultMaxIdleConnsPerHost
		p.config.MaxIdleConnsPerHost = &defaultValue
	}
	if p.config.IdleConnTimeout == nil {
		defaultValue := restConfigDefaultIdleConnTimeout
		p.config.IdleConnTimeout = &defaultValue
	}
	if p.config.Retry == nil {
		defaultValue := restConfigDefaultRetry
		p.config.Retry = &defaultValue
	}
	if p.config.RetryDelay == nil {
		defaultValue := restConfigDefaultRetryDelay
		p.config.RetryDelay = &defaultValue
	}

	tlsConfig := &tls.Config{}

	if p.config.TLSCAFiles != nil {
		caCertPool := x509.NewCertPool()

		for _, tlsCAFile := range *p.config.TLSCAFiles {
			ca, err := p.osReadFile(tlsCAFile)
			if err != nil {
				return err
			}
			p.x509CertPoolAppendCertsFromPEM(caCertPool, ca)
		}
		tlsConfig.RootCAs = caCertPool
	}

	if p.config.TLSCertFiles != nil {
		var err error
		tlsConfig.Certificates = make([]tls.Certificate, len(*p.config.TLSCertFiles))

		for i := range *p.config.TLSCertFiles {
			tlsConfig.Certificates[i], err = p.tlsLoadX509KeyPair((*p.config.TLSCertFiles)[i], (*p.config.TLSKeyFiles)[i])
			if err != nil {
				return err
			}
		}
	}

	transport := http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout: time.Duration(*p.config.Timeout) * time.Second,
		}).Dial,
		TLSClientConfig:       tlsConfig,
		TLSHandshakeTimeout:   time.Duration(*p.config.Timeout) * time.Second,
		ResponseHeaderTimeout: time.Duration(*p.config.Timeout) * time.Second,
		ExpectContinueTimeout: time.Duration(*p.config.Timeout) * time.Second,
		ForceAttemptHTTP2:     true,
		MaxConnsPerHost:       *p.config.MaxConnsPerHost,
		MaxIdleConns:          *p.config.MaxIdleConns,
		MaxIdleConnsPerHost:   *p.config.MaxIdleConnsPerHost,
		IdleConnTimeout:       time.Duration(*p.config.IdleConnTimeout) * time.Second,
	}

	p.client = http.Client{
		Transport: &transport,
		Timeout:   time.Duration(*p.config.Timeout) * time.Second,
	}

	return nil
}

// Fetch fetches a resource.
func (p *restProvider) Fetch(ctx context.Context, name string, config map[string]interface{}) (*core.Resource, error) {
	var cfg restResourceConfig
	err := mapstructure.Decode(config, &cfg)
	if err != nil {
		return nil, err
	}

	if cfg.Method == nil {
		defaultValue := restResourceDefaultMethod
		cfg.Method = &defaultValue
	}
	if cfg.Next != nil {
		defaultValue := restResourceDefaultNextParser
		cfg.NextParser = &defaultValue
	}

	var data [][]byte

fetch:
	body, headers, err := p.fetchResource(ctx, &cfg)
	if err != nil {
		return nil, err
	}

	data = append(data, body)

	if cfg.Next != nil && *cfg.Next {
		var url string
		switch *cfg.NextParser {
		case restResourceNextParserHeader:
			url = parseLinkNextFromHeader(headers)
		case restResourceNextParserBody:
			url = parseLinkNextFromBody(body, *cfg.NextFilter)
		}
		if url != "" {
			cfg.URL = url
			cfg.Params = map[string]string{}
			goto fetch
		}
	}

	return &core.Resource{
		Data: data,
		TTL:  0,
	}, nil
}

// parseLinkNextFromHeader parses the next link from the resource headers
func parseLinkNextFromHeader(headers http.Header) string {
	for _, header := range headers["Link"] {
		links := strings.Split(header, ",")
		for _, link := range links {
			params := strings.Split(link, ";")
			if len(params) < 2 {
				continue
			}
			for i := 1; i < len(params); i++ {
				if strings.TrimSpace(params[i]) == "rel=\"next\"" {
					return strings.Trim(strings.TrimSpace(params[0]), "<>")
				}
			}
		}
	}
	return ""
}

// parseLinkNextFromBody parses the next link from the resource body
func parseLinkNextFromBody(body []byte, filter string) string {
	var jsonData interface{}
	err := json.Unmarshal(body, &jsonData)
	if err != nil {
		return ""
	}
	result, err := jsonpath.Get(filter, jsonData)
	if err != nil {
		return ""
	}
	url, ok := result.(string)
	if !ok {
		return ""
	}
	return url
}

// fetchResource fetches the resource
func (p *restProvider) fetchResource(ctx context.Context, config *restResourceConfig) ([]byte, http.Header, error) {
	req, err := p.httpNewRequestWithContext(ctx, *config.Method, config.URL, nil)
	if err != nil {
		p.logger.Printf("Failed to create request: %s", err)

		return nil, nil, err
	}

	query := req.URL.Query()
	for key, value := range p.config.Params {
		query.Add(key, value)
	}
	for key, value := range config.Params {
		query.Add(key, value)
	}
	req.URL.RawQuery = query.Encode()

	for key, value := range p.config.Headers {
		req.Header.Set(key, value)
	}
	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	var attempt int
	for {
		attempt += 1

		response, err := p.httpClientDo(&p.client, req)
		if response != nil {
			defer response.Body.Close()
		}
		if err != nil {
			p.logger.Printf("Failed to send request: %s", err)

			return nil, nil, err
		}
		responseBody, err := p.ioReadAll(response.Body)
		if err != nil {
			p.logger.Printf("Failed to read response: %s", err)

			return nil, nil, err
		}

		if core.DEBUG {
			p.logger.Printf("Fetch request: method=%s, url=%s, code=%d\n", req.Method, req.URL.String(), response.StatusCode)
		}

		switch response.StatusCode {
		case 429, 500, 502, 503, 504:
			if attempt >= *p.config.Retry {
				return nil, nil, fmt.Errorf("request error %d", response.StatusCode)
			}

			if *p.config.RetryDelay > 0 {
				p.logger.Printf("Retrying request attempt %d/%d, delaying for %d seconds", attempt, *p.config.Retry,
					*p.config.RetryDelay)

				time.Sleep(time.Duration(*p.config.RetryDelay) * time.Second)
			} else {
				p.logger.Printf("Retrying request attempt %d/%d", attempt, *p.config.Retry)
			}

			continue

		default:
			if response.StatusCode < 200 || response.StatusCode > 299 {
				return nil, nil, fmt.Errorf("request error %d", response.StatusCode)
			}
		}

		return responseBody, response.Header, nil
	}
}

var _ core.FetcherProviderModule = (*restProvider)(nil)
