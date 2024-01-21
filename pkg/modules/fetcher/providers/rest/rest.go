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

// Init initializes the provider.
func (p *restProvider) Init(config map[string]interface{}, logger *log.Logger) error {
	p.logger = logger

	if err := mapstructure.Decode(config, &p.config); err != nil {
		p.logger.Print("failed to parse configuration")
		return err
	}

	var errInit bool

	if p.config.TLSCAFiles != nil {
		for _, item := range *p.config.TLSCAFiles {
			if item == "" {
				p.logger.Printf("option '%s', invalid value '%s'", "TLSCAFiles", item)
				errInit = true
				continue
			}
			file, err := p.osOpenFile(item, os.O_RDONLY, 0)
			if err != nil {
				p.logger.Printf("option '%s', failed to open file '%s'", "TLSCAFiles", item)
				errInit = true
				continue
			}
			p.osClose(file)
			fi, err := p.osStat(item)
			if err != nil {
				p.logger.Printf("option '%s', failed to stat file '%s'", "TLSCAFiles", item)
				errInit = true
				continue
			}
			if err == nil && fi.IsDir() {
				p.logger.Printf("option '%s', '%s' is a directory", "TLSCAFiles", item)
				errInit = true
				continue
			}
		}
	}
	if p.config.TLSCertFiles != nil {
		for _, item := range *p.config.TLSCertFiles {
			if item == "" {
				p.logger.Printf("option '%s', invalid value '%s'", "TLSCertFiles", item)
				errInit = true
				continue
			}
			file, err := p.osOpenFile(item, os.O_RDONLY, 0)
			if err != nil {
				p.logger.Printf("option '%s', failed to open file '%s'", "TLSCertFiles", item)
				errInit = true
				continue
			}
			p.osClose(file)
			fi, err := p.osStat(item)
			if err != nil {
				p.logger.Printf("option '%s', failed to stat file '%s'", "TLSCertFiles", item)
				errInit = true
				continue
			}
			if err == nil && fi.IsDir() {
				p.logger.Printf("option '%s', '%s' is a directory", "TLSCertFiles", item)
				errInit = true
				continue
			}
		}
	}
	if p.config.TLSKeyFiles != nil {
		if p.config.TLSCertFiles == nil || len(*p.config.TLSKeyFiles) != len(*p.config.TLSCertFiles) {
			p.logger.Printf("option '%s', missing value(s)", "TLSKeyFiles")
			errInit = true
		}
		for _, item := range *p.config.TLSKeyFiles {
			if item == "" {
				p.logger.Printf("option '%s', invalid value '%s'", "TLSKeyFiles", item)
				errInit = true
				continue
			}
			file, err := p.osOpenFile(item, os.O_RDONLY, 0)
			if err != nil {
				p.logger.Printf("option '%s', failed to open file '%s'", "TLSKeyFiles", item)
				errInit = true
				continue
			}
			p.osClose(file)
			fi, err := p.osStat(item)
			if err != nil {
				p.logger.Printf("option '%s', failed to stat file '%s'", "TLSKeyFiles", item)
				errInit = true
				continue
			}
			if err == nil && fi.IsDir() {
				p.logger.Printf("option '%s', '%s' is a directory", "TLSKeyFiles", item)
				errInit = true
				continue
			}
		}
	}
	if p.config.Timeout == nil {
		defaultValue := restConfigDefaultTimeout
		p.config.Timeout = &defaultValue
	}
	if *p.config.Timeout < 0 {
		p.logger.Printf("option '%s', invalid value '%d'", "Timeout", *p.config.Timeout)
		errInit = true
	}
	if p.config.MaxConnsPerHost == nil {
		defaultValue := restConfigDefaultMaxConnsPerHost
		p.config.MaxConnsPerHost = &defaultValue
	}
	if *p.config.MaxConnsPerHost < 0 {
		p.logger.Printf("option '%s', invalid value '%d'", "MaxConnsPerHost", *p.config.MaxConnsPerHost)
		errInit = true
	}
	if p.config.MaxIdleConns == nil {
		defaultValue := restConfigDefaultMaxIdleConns
		p.config.MaxIdleConns = &defaultValue
	}
	if *p.config.MaxIdleConns < 0 {
		p.logger.Printf("option '%s', invalid value '%d'", "MaxIdleConns", *p.config.MaxIdleConns)
		errInit = true
	}
	if p.config.MaxIdleConnsPerHost == nil {
		defaultValue := restConfigDefaultMaxIdleConnsPerHost
		p.config.MaxIdleConnsPerHost = &defaultValue
	}
	if *p.config.MaxIdleConnsPerHost < 0 {
		p.logger.Printf("option '%s', invalid value '%d'", "MaxIdleConnsPerHost", *p.config.MaxIdleConnsPerHost)
		errInit = true
	}
	if p.config.IdleConnTimeout == nil {
		defaultValue := restConfigDefaultIdleConnTimeout
		p.config.IdleConnTimeout = &defaultValue
	}
	if *p.config.IdleConnTimeout < 0 {
		p.logger.Printf("option '%s', invalid value '%d'", "IdleConnTimeout", *p.config.IdleConnTimeout)
		errInit = true
	}
	if p.config.Retry == nil {
		defaultValue := restConfigDefaultRetry
		p.config.Retry = &defaultValue
	}
	if *p.config.Retry < 0 {
		p.logger.Printf("option '%s', invalid value '%d'", "Retry", *p.config.Retry)
		errInit = true
	}
	if p.config.RetryDelay == nil {
		defaultValue := restConfigDefaultRetryDelay
		p.config.RetryDelay = &defaultValue
	}
	if *p.config.RetryDelay < 0 {
		p.logger.Printf("option '%s', invalid value '%d'", "RetryDelay", *p.config.RetryDelay)
		errInit = true
	}
	for k := range p.config.Headers {
		if k == "" {
			p.logger.Printf("option '%s', invalid key '%s'", "Headers", k)
			errInit = true
		}
	}
	for k := range p.config.Params {
		if k == "" {
			p.logger.Printf("option '%s', invalid key '%s'", "Params", k)
			errInit = true
		}
	}

	if errInit {
		return errors.New("init error")
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
		tlsConfig.Certificates = make([]tls.Certificate, len(*p.config.TLSCertFiles))
		for i := range *p.config.TLSCertFiles {
			var err error
			tlsConfig.Certificates[i], err = p.tlsLoadX509KeyPair((*p.config.TLSCertFiles)[i], (*p.config.TLSKeyFiles)[i])
			if err != nil {
				return err
			}
		}
	}

	p.client = http.Client{
		Transport: &http.Transport{
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
		},
		Timeout: time.Duration(*p.config.Timeout) * time.Second,
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
			p.logger.Printf("Fetch completed (method=%s, url=%s, code=%d)", req.Method, req.URL.String(), response.StatusCode)
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
