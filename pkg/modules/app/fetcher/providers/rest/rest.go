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
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PaesslerAG/jsonpath"
	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/log"
	"github.com/bhuisgen/neon/pkg/module"
)

// restProvider implements the rest provider.
type restProvider struct {
	config                         *restProviderConfig
	logger                         *slog.Logger
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
	TLSCAFiles          *[]string         `mapstructure:"tlsCAFiles"`
	TLSCertFiles        *[]string         `mapstructure:"tlsCertFiles"`
	TLSKeyFiles         *[]string         `mapstructure:"tlsKeyFiles"`
	Timeout             *int              `mapstructure:"timeout"`
	ConnectTimeout      *int              `mapstructure:"connectTimeout"`
	MaxIdleConns        *int              `mapstructure:"maxIdleConns"`
	MaxIdleConnsPerHost *int              `mapstructure:"maxIdleConnsPerHost"`
	IdleConnTimeout     *int              `mapstructure:"idleConnTimeout"`
	MaxConnsPerHost     *int              `mapstructure:"maxConnsPerHost"`
	Retry               *int              `mapstructure:"retry"`
	RetryDelay          *int              `mapstructure:"retryDelay"`
	Headers             map[string]string `mapstructure:"headers"`
	Params              map[string]string `mapstructure:"params"`
}

// restResourceConfig implements the rest resource configuration.
type restResourceConfig struct {
	Method     *string           `mapstructure:"method"`
	URL        string            `mapstructure:"url"`
	Params     map[string]string `mapstructure:"params"`
	Headers    map[string]string `mapstructure:"headers"`
	Next       *bool             `mapstructure:"next"`
	NextParser *string           `mapstructure:"nextParser"`
	NextFilter *string           `mapstructure:"nextFilter"`
}

const (
	restModuleID module.ModuleID = "app.fetcher.provider.rest"

	restConfigDefaultTimeout             int = 15
	restConfigDefaultConnectTimeout      int = 5
	restConfigDefaultMaxIdleConns        int = 100
	restConfigDefaultMaxIdleConnsPerHost int = 100
	restConfigDefaultIdleConnTimeout     int = 60
	restConfigDefaultMaxConnsPerHost     int = 100
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

// init initializes the package.
func init() {
	module.Register(restProvider{})
}

// ModuleInfo returns the module information.
func (p restProvider) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID:           restModuleID,
		LoadModule:   func() {},
		UnloadModule: func() {},
		NewInstance: func() module.Module {
			return &restProvider{
				logger:                         slog.New(log.NewHandler(os.Stderr, string(restModuleID), nil)),
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
func (p *restProvider) Init(config map[string]interface{}) error {
	if err := mapstructure.Decode(config, &p.config); err != nil {
		p.logger.Error("Failed to parse configuration", "err", err)
		return fmt.Errorf("parse config: %v", err)
	}

	var errConfig bool

	if p.config.TLSCAFiles != nil {
		for _, item := range *p.config.TLSCAFiles {
			if item == "" {
				p.logger.Error("Invalid value", "option", "TLSCAFiles", "value", item)
				errConfig = true
				continue
			}
			file, err := p.osOpenFile(item, os.O_RDONLY, 0)
			if err != nil {
				p.logger.Error("Failed to open file", "option", "TLSCAFiles", "value", item)
				errConfig = true
				continue
			}
			_ = p.osClose(file)
			fi, err := p.osStat(item)
			if err != nil {
				p.logger.Error("Failed to stat file", "option", "TLSCAFiles", "value", item)
				errConfig = true
				continue
			}
			if fi.IsDir() {
				p.logger.Error("File is a directory", "option", "TLSCAFiles", "value", item)
				errConfig = true
				continue
			}
		}
	}
	if p.config.TLSCertFiles != nil {
		for _, item := range *p.config.TLSCertFiles {
			if item == "" {
				p.logger.Error("Invalid value", "option", "TLSCertFiles", "value", item)
				errConfig = true
				continue
			}
			file, err := p.osOpenFile(item, os.O_RDONLY, 0)
			if err != nil {
				p.logger.Error("Failed to open file", "option", "TLSCertFiles", "value", item)
				errConfig = true
				continue
			}
			_ = p.osClose(file)
			fi, err := p.osStat(item)
			if err != nil {
				p.logger.Error("Failed to stat file '%s'", "option", "TLSCertFiles", "value", item)
				errConfig = true
				continue
			}
			if fi.IsDir() {
				p.logger.Error("File is a directory", "option", "TLSCertFiles", "value", item)
				errConfig = true
				continue
			}
		}
	}
	if p.config.TLSKeyFiles != nil {
		if p.config.TLSCertFiles == nil || len(*p.config.TLSKeyFiles) != len(*p.config.TLSCertFiles) {
			p.logger.Error("Missing value(s)", "option", "TLSKeyFiles")
			errConfig = true
		}
		for _, item := range *p.config.TLSKeyFiles {
			if item == "" {
				p.logger.Error("Invalid value", "option", "TLSKeyFiles", "value", item)
				errConfig = true
				continue
			}
			file, err := p.osOpenFile(item, os.O_RDONLY, 0)
			if err != nil {
				p.logger.Error("Failed to open file", "option", "TLSKeyFiles", "value", item)
				errConfig = true
				continue
			}
			_ = p.osClose(file)
			fi, err := p.osStat(item)
			if err != nil {
				p.logger.Error("Failed to stat file", "option", "TLSKeyFiles", "value", item)
				errConfig = true
				continue
			}
			if fi.IsDir() {
				p.logger.Error("File is a directory", "option", "TLSKeyFiles", "value", item)
				errConfig = true
				continue
			}
		}
	}
	if p.config.Timeout == nil {
		defaultValue := restConfigDefaultTimeout
		p.config.Timeout = &defaultValue
	}
	if *p.config.Timeout < 0 {
		p.logger.Error("Invalid value", "option", "Timeout", "value", *p.config.Timeout)
		errConfig = true
	}
	if p.config.ConnectTimeout == nil {
		defaultValue := restConfigDefaultConnectTimeout
		p.config.ConnectTimeout = &defaultValue
	}
	if *p.config.ConnectTimeout < 0 {
		p.logger.Error("Invalid value", "option", "ConnectTimeout", "value", *p.config.Timeout)
		errConfig = true
	}
	if p.config.MaxIdleConns == nil {
		defaultValue := restConfigDefaultMaxIdleConns
		p.config.MaxIdleConns = &defaultValue
	}
	if *p.config.MaxIdleConns < 0 {
		p.logger.Error("Invalid value", "option", "MaxIdleConns", "value", *p.config.MaxIdleConns)
		errConfig = true
	}
	if p.config.MaxIdleConnsPerHost == nil {
		defaultValue := restConfigDefaultMaxIdleConnsPerHost
		p.config.MaxIdleConnsPerHost = &defaultValue
	}
	if *p.config.MaxIdleConnsPerHost < 0 {
		p.logger.Error("Invalid value", "option", "MaxIdleConnsPerHost", "value", *p.config.MaxIdleConnsPerHost)
		errConfig = true
	}
	if p.config.IdleConnTimeout == nil {
		defaultValue := restConfigDefaultIdleConnTimeout
		p.config.IdleConnTimeout = &defaultValue
	}
	if *p.config.IdleConnTimeout < 0 {
		p.logger.Error("Invalid value", "option", "IdleConnTimeout", "value", *p.config.IdleConnTimeout)
		errConfig = true
	}
	if p.config.MaxConnsPerHost == nil {
		defaultValue := restConfigDefaultMaxConnsPerHost
		p.config.MaxConnsPerHost = &defaultValue
	}
	if *p.config.MaxConnsPerHost < 0 {
		p.logger.Error("Invalid value", "option", "MaxConnsPerHost", "value", *p.config.MaxConnsPerHost)
		errConfig = true
	}
	if p.config.Retry == nil {
		defaultValue := restConfigDefaultRetry
		p.config.Retry = &defaultValue
	}
	if *p.config.Retry < 0 {
		p.logger.Error("Invalid value", "option", "Retry", "value", *p.config.Retry)
		errConfig = true
	}
	if p.config.RetryDelay == nil {
		defaultValue := restConfigDefaultRetryDelay
		p.config.RetryDelay = &defaultValue
	}
	if *p.config.RetryDelay < 0 {
		p.logger.Error("Invalid value", "option", "RetryDelay", "value", *p.config.RetryDelay)
		errConfig = true
	}
	for k := range p.config.Headers {
		if k == "" {
			p.logger.Error("Invalid key", "option", "Headers", "key", k)
			errConfig = true
		}
	}
	for k := range p.config.Params {
		if k == "" {
			p.logger.Error("Invalid key", "option", "Params", "key", k)
			errConfig = true
		}
	}

	if errConfig {
		return errors.New("config")
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if p.config.TLSCAFiles != nil {
		caCertPool := x509.NewCertPool()
		for _, tlsCAFile := range *p.config.TLSCAFiles {
			ca, err := p.osReadFile(tlsCAFile)
			if err != nil {
				return fmt.Errorf("read file %s: %v", tlsCAFile, err)
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
				return fmt.Errorf("load keypair %s/%s: %v", (*p.config.TLSCertFiles)[i], (*p.config.TLSKeyFiles)[i],
					err)
			}
		}
	}

	p.client = http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout: time.Duration(*p.config.ConnectTimeout) * time.Second,
			}).Dial,
			TLSClientConfig:       tlsConfig,
			TLSHandshakeTimeout:   time.Duration(*p.config.Timeout) * time.Second,
			ResponseHeaderTimeout: time.Duration(*p.config.Timeout) * time.Second,
			ExpectContinueTimeout: time.Duration(*p.config.Timeout) * time.Second,
			MaxIdleConns:          *p.config.MaxIdleConns,
			MaxIdleConnsPerHost:   *p.config.MaxIdleConnsPerHost,
			IdleConnTimeout:       time.Duration(*p.config.IdleConnTimeout) * time.Second,
			MaxConnsPerHost:       *p.config.MaxConnsPerHost,
			ForceAttemptHTTP2:     true,
		},
		Timeout: time.Duration(*p.config.Timeout) * time.Second,
	}

	return nil
}

// Fetch fetches a resource.
func (p *restProvider) Fetch(ctx context.Context, name string, config map[string]interface{}) (*core.Resource, error) {
	var cfg restResourceConfig
	if err := mapstructure.Decode(config, &cfg); err != nil {
		return nil, fmt.Errorf("parse resource %s config: %v", name, err)
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

// parseLinkNextFromHeader parses the next link from the resource headers.
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

// parseLinkNextFromBody parses the next link from the resource body.
func parseLinkNextFromBody(body []byte, filter string) string {
	var jsonData interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		return ""
	}
	result, err := jsonpath.Get(filter, jsonData)
	if err != nil {
		return ""
	}
	if url, ok := result.(string); ok {
		return url
	}
	return ""
}

// fetchResource fetches the resource.
func (p *restProvider) fetchResource(ctx context.Context, config *restResourceConfig) ([]byte, http.Header, error) {
	req, err := p.httpNewRequestWithContext(ctx, *config.Method, config.URL, nil)
	if err != nil {
		p.logger.Error("Failed to create request", "err", err)
		return nil, nil, fmt.Errorf("create request: %v", err)
	}

	query := req.URL.Query()
	for key, value := range p.config.Params {
		query.Set(key, value)
	}
	for key, value := range config.Params {
		query.Set(key, value)
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
		startTime := time.Now()

		response, err := p.httpClientDo(&p.client, req)
		if err != nil {
			p.logger.Error("Failed to send request", "err", err)
			return nil, nil, fmt.Errorf("send requests: %v", err)
		}
		defer response.Body.Close()
		responseBody, err := p.ioReadAll(response.Body)
		if err != nil {
			p.logger.Error("Failed to read response", "err", err)
			return nil, nil, fmt.Errorf("read response: %v", err)
		}

		p.logger.Debug("Request processed", "method", req.Method, "url", req.URL.String(),
			"code", response.StatusCode, "duration", time.Since(startTime).Round(time.Millisecond))

		switch response.StatusCode {
		case 429, 500, 502, 503, 504:
			if attempt >= *p.config.Retry {
				p.logger.Error("Request error", "method", req.Method, "url", req.URL.String(), "code", response.StatusCode)
				return nil, nil, fmt.Errorf("request error %d", response.StatusCode)
			}

			p.logger.Warn("Retrying request", "method", req.Method, "url", req.URL.String(), "code", response.StatusCode,
				"attempt", attempt, "retries", *p.config.Retry, "delay", *p.config.RetryDelay)

			time.Sleep(time.Duration(*p.config.RetryDelay) * time.Second)

			continue

		default:
			if response.StatusCode < 200 || response.StatusCode > 299 {
				p.logger.Error("Request error", "method", req.Method, "url", req.URL.String(), "code", response.StatusCode)
				return nil, nil, fmt.Errorf("request error %d", response.StatusCode)
			}
		}

		return responseBody, response.Header, nil
	}
}

var _ core.FetcherProviderModule = (*restProvider)(nil)
