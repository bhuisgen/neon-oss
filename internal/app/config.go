// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	_ "embed"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config implements the configuration for the instance
type Config struct {
	Server  []*ServerConfig
	Fetcher *FetcherConfig
	Loader  *LoaderConfig
}

// yamlConfig implements the configuration for the parser
type yamlConfig struct {
	Server  []yamlConfigServer `yaml:"server"`
	Fetcher yamlConfigFetcher  `yaml:"fetcher,omitempty"`
	Loader  yamlConfigLoader   `yaml:"loader,omitempty"`
}

// yamlConfigServer implements the server configuration for the parser
type yamlConfigServer struct {
	ListenAddr    *string `yaml:"listen_addr,omitempty"`
	ListenPort    *int    `yaml:"listen_port,omitempty"`
	TLS           *bool   `yaml:"tls,omitempty"`
	TLSCAFile     *string `yaml:"tls_ca_file,omitempty"`
	TLSCertFile   *string `yaml:"tls_cert_file,omitempty"`
	TLSKeyFile    *string `yaml:"tls_key_file,omitempty"`
	ReadTimeout   *int    `yaml:"read_timeout,omitempty"`
	WriteTimeout  *int    `yaml:"write_timeout,omitempty"`
	AccessLog     *bool   `yaml:"access_log,omitempty"`
	AccessLogFile *string `yaml:"access_log_file,omitempty"`

	Rewrite struct {
		Enable *bool `yaml:"enable"`
		Rules  []struct {
			Path        string  `yaml:"path"`
			Replacement string  `yaml:"replacement"`
			Flag        *string `yaml:"flag,omitempty"`
			Last        *bool   `yaml:"last,omitempty"`
		} `yaml:"rules"`
	} `yaml:"rewrite,omitempty"`

	Header struct {
		Enable *bool `yaml:"enable"`
		Rules  []struct {
			Path   string            `yaml:"path"`
			Set    map[string]string `yaml:"set,omitempty"`
			Add    map[string]string `yaml:"add,omitempty"`
			Remove []string          `yaml:"remove,omitempty"`
			Last   *bool             `yaml:"last,omitempty"`
		} `yaml:"rules"`
	} `yaml:"header,omitempty"`

	Static struct {
		Enable *bool  `yaml:"enable"`
		Dir    string `yaml:"dir"`
		Index  *bool  `yaml:"index,omitempty"`
	} `yaml:"static,omitempty"`

	Robots struct {
		Enable   *bool    `yaml:"enable"`
		Path     *string  `yaml:"path,omitempty"`
		Hosts    []string `yaml:"hosts,omitempty"`
		Sitemaps []string `yaml:"sitemaps,omitempty"`
		Cache    *bool    `yaml:"cache,omitempty"`
		CacheTTL *int     `yaml:"cache_ttl,omitempty"`
	} `yaml:"robots,omitempty"`

	Sitemap struct {
		Enable   *bool  `yaml:"enable"`
		Root     string `yaml:"root"`
		Cache    *bool  `yaml:"cache,omitempty"`
		CacheTTL *int   `yaml:"cache_ttl,omitempty"`
		Routes   []struct {
			Path         string `yaml:"path"`
			Kind         string `yaml:"kind"`
			SitemapIndex []struct {
				Name   string `yaml:"name"`
				Type   string `yaml:"type"`
				Static struct {
					Loc string `yaml:"loc"`
				} `yaml:"static,omitempty"`
			} `yaml:"sitemap_index,omitempty"`
			Sitemap []struct {
				Name   string `yaml:"name"`
				Type   string `yaml:"type"`
				Static struct {
					Loc        string   `yaml:"loc"`
					Lastmod    *string  `yaml:"lastmod,omitempty"`
					Changefreq *string  `yaml:"changefreq,omitempty"`
					Priority   *float32 `yaml:"priority,omitempty"`
				} `yaml:"static,omitempty"`
				List struct {
					Resource                   string   `yaml:"resource"`
					ResourcePayloadItems       string   `yaml:"resource_payload_items"`
					ResourcePayloadItemLoc     string   `yaml:"resource_payload_item_loc"`
					ResourcePayloadItemLastmod *string  `yaml:"resource_payload_item_lastmod,omitempty"`
					Changefreq                 *string  `yaml:"changefreq,omitempty"`
					Priority                   *float32 `yaml:"priority,omitempty"`
				} `yaml:"list,omitempty"`
			} `yaml:"sitemap,omitempty"`
		} `yaml:"routes"`
	} `yaml:"sitemap,omitempty"`

	Index struct {
		Enable    *bool   `yaml:"enable"`
		HTML      string  `yaml:"html"`
		Bundle    *string `yaml:"bundle"`
		Env       *string `yaml:"env,omitempty"`
		Container *string `yaml:"container,omitempty"`
		State     *string `yaml:"state,omitempty"`
		Timeout   *int    `yaml:"timeout,omitempty"`
		Cache     *bool   `yaml:"cache,omitempty"`
		CacheTTL  *int    `yaml:"cache_ttl,omitempty"`
		Rules     []struct {
			Path  string `yaml:"path"`
			State []struct {
				Key      string `yaml:"key"`
				Resource string `yaml:"resource"`
				Export   *bool  `yaml:"export"`
			} `yaml:"state,omitempty"`
			Last *bool `yaml:"last,omitempty"`
		} `yaml:"rules"`
	} `yaml:"index,omitempty"`

	Default struct {
		Enable     *bool  `yaml:"enable"`
		File       string `yaml:"file"`
		StatusCode *int   `yaml:"status_code,omitempty"`
		Cache      *bool  `yaml:"cache,omitempty"`
		CacheTTL   *int   `yaml:"cache_ttl,omitempty"`
	} `yaml:"default,omitempty"`
}

// yamlConfigFetcher implements the fetcher configuration for the parser
type yamlConfigFetcher struct {
	RequestTLSCAFile   *string           `yaml:"request_tls_ca_file,omitempty"`
	RequestTLSCertFile *string           `yaml:"request_tls_cert_file,omitempty"`
	RequestTLSKeyFile  *string           `yaml:"request_tls_key_file,omitempty"`
	RequestHeaders     map[string]string `yaml:"request_headers,omitempty"`
	RequestTimeout     *int              `yaml:"request_timeout,omitempty"`
	RequestRetry       *int              `yaml:"request_retry,omitempty"`
	RequestDelay       *int              `yaml:"request_delay,omitempty"`
	Resources          []struct {
		Name    string            `yaml:"name"`
		Method  string            `yaml:"method"`
		URL     string            `yaml:"url"`
		Params  map[string]string `yaml:"params,omitempty"`
		Headers map[string]string `yaml:"headers,omitempty"`
	} `yaml:"resources"`
	Templates []struct {
		Name    string            `yaml:"name"`
		Method  string            `yaml:"method"`
		URL     string            `yaml:"url"`
		Params  map[string]string `yaml:"params,omitempty"`
		Headers map[string]string `yaml:"headers,omitempty"`
	} `yaml:"templates,omitempty"`
}

// yamlConfigLoader implements the loader configuration for the parser
type yamlConfigLoader struct {
	ExecStartup  *int `yaml:"exec_startup,omitempty"`
	ExecInterval *int `yaml:"exec_interval,omitempty"`
	ExecWorkers  *int `yaml:"exec_workers,omitempty"`
	Rules        []struct {
		Name   string `yaml:"name"`
		Type   string `yaml:"type"`
		Static struct {
			Resource string `yaml:"resource"`
		} `yaml:"static,omitempty"`
		Single struct {
			Resource                    string            `yaml:"resource"`
			ResourcePayloadItem         string            `yaml:"resource_payload_item"`
			ItemTemplate                string            `yaml:"item_template"`
			ItemTemplateResource        string            `yaml:"item_template_resource"`
			ItemTemplateResourceParams  map[string]string `yaml:"item_template_resource_params,omitempty"`
			ItemTemplateResourceHeaders map[string]string `yaml:"item_template_resource_headers,omitempty"`
		} `yaml:"single,omitempty"`
		List struct {
			Resource                    string            `yaml:"resource"`
			ResourcePayloadItems        string            `yaml:"resource_payload_items"`
			ItemTemplate                string            `yaml:"item_template"`
			ItemTemplateResource        string            `yaml:"item_template_resource"`
			ItemTemplateResourceParams  map[string]string `yaml:"item_template_resource_params,omitempty"`
			ItemTemplateResourceHeaders map[string]string `yaml:"item_template_resource_headers,omitempty"`
		} `yaml:"list,omitempty"`
	} `yaml:"rules"`
}

const (
	configFile                           string = "config.yaml"
	configDefaultServerListenAddr        string = "localhost"
	configDefaultServerListenPort        int    = 8080
	configDefaultServerReadTimeout       int    = 60
	configDefaultServerWriteTimeout      int    = 60
	configDefaultServerAccessLog         bool   = false
	configDefaultServerRewriteEnable     bool   = false
	configDefaultServerRewriteRuleLast   bool   = false
	configDefaultServerHeaderEnable      bool   = false
	configDefaultServerHeaderRuleLast    bool   = false
	configDefaultServerStaticEnable      bool   = false
	configDefaultServerStaticIndex       bool   = false
	configDefaultServerRobotsEnable      bool   = false
	configDefaultServerRobotsPath        string = "/robots.txt"
	configDefaultServerRobotsCache       bool   = false
	configDefaultServerRobotsCacheTTL    int    = 60
	configDefaultServerSitemapEnable     bool   = false
	configDefaultServerSitemapCache      bool   = false
	configDefaultServerSitemapCacheTTL   int    = 60
	configDefaultServerIndexEnable       bool   = false
	configDefaultServerIndexEnv          string = "production"
	configDefaultServerIndexContainer    string = "root"
	configDefaultServerIndexState        string = "state"
	configDefaultServerIndexTimeout      int    = 4
	configDefaultServerIndexCache        bool   = false
	configDefaultServerIndexCacheTTL     int    = 60
	configDefaultServerIndexRuleLast     bool   = false
	configDefaultServerDefaultEnable     bool   = false
	configDefaultServerDefaultStatusCode int    = 200
	configDefaultServerDefaultCache      bool   = false
	configDefaultServerDefaultCacheTTL   int    = 60
	configDefaultFetcherRequestTimeout   int    = 60
	configDefaultFetcherRequestRetry     int    = 3
	configDefaultFetcherRequestDelay     int    = 1
	configDefaultLoaderExecStartup       int    = 15
	configDefaultLoaderExecInterval      int    = 900
	configDefaultLoaderExecWorkers       int    = 1
)

// LoadConfig loads the configuration settings
func LoadConfig() (*Config, error) {
	file, ok := os.LookupEnv("CONFIG_FILE")
	if !ok {
		file = configFile
	}

	yamlFile, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var y yamlConfig
	err = yaml.Unmarshal(yamlFile, &y)
	if err != nil {
		return nil, err
	}

	c, err := parseConfig(&y)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// parseConfig parses the configuration
func parseConfig(y *yamlConfig) (*Config, error) {
	c := Config{}

	for index, yamlConfigServer := range y.Server {
		serverConfig := ServerConfig{}

		if yamlConfigServer.ListenAddr != nil {
			serverConfig.ListenAddr = *yamlConfigServer.ListenAddr
		} else {
			listenAddr := configDefaultServerListenAddr
			if v, ok := os.LookupEnv("LISTEN_ADDR"); ok && index == 0 {
				listenAddr = v
			}
			serverConfig.ListenAddr = listenAddr
		}
		if yamlConfigServer.ListenPort != nil {
			serverConfig.ListenPort = *yamlConfigServer.ListenPort
		} else {
			listenPort := configDefaultServerListenPort
			if v, ok := os.LookupEnv("LISTEN_PORT"); ok && index == 0 {
				if vInt, err := strconv.ParseInt(v, 10, 0); err == nil {
					listenPort = int(vInt)
				}
			}
			serverConfig.ListenPort = listenPort
		}
		if yamlConfigServer.TLS != nil {
			serverConfig.TLS = *yamlConfigServer.TLS
		}
		if yamlConfigServer.TLSCAFile != nil {
			serverConfig.TLSCAFile = yamlConfigServer.TLSCAFile
		}
		if yamlConfigServer.TLSCertFile != nil {
			serverConfig.TLSCertFile = yamlConfigServer.TLSCertFile
		}
		if yamlConfigServer.TLSKeyFile != nil {
			serverConfig.TLSKeyFile = yamlConfigServer.TLSKeyFile
		}
		if yamlConfigServer.ReadTimeout != nil {
			serverConfig.ReadTimeout = *yamlConfigServer.ReadTimeout
		} else {
			serverConfig.ReadTimeout = configDefaultServerReadTimeout
		}
		if yamlConfigServer.WriteTimeout != nil {
			serverConfig.WriteTimeout = *yamlConfigServer.WriteTimeout
		} else {
			serverConfig.WriteTimeout = configDefaultServerWriteTimeout
		}
		if yamlConfigServer.AccessLog != nil {
			serverConfig.AccessLog = *yamlConfigServer.AccessLog
		} else {
			serverConfig.AccessLog = configDefaultServerAccessLog
		}
		if yamlConfigServer.AccessLogFile != nil {
			serverConfig.AccessLogFile = yamlConfigServer.AccessLogFile
		}

		if yamlConfigServer.Rewrite.Enable != nil {
			serverConfig.Rewrite.Enable = *yamlConfigServer.Rewrite.Enable
		} else {
			serverConfig.Rewrite.Enable = configDefaultServerRewriteEnable
		}
		for _, rewriteRule := range yamlConfigServer.Rewrite.Rules {
			rule := RewriteRule{
				Path:        rewriteRule.Path,
				Replacement: rewriteRule.Replacement,
			}
			if rewriteRule.Flag != nil {
				rule.Flag = rewriteRule.Flag
			}
			serverConfig.Rewrite.Rules = append(serverConfig.Rewrite.Rules, rule)
		}

		if yamlConfigServer.Header.Enable != nil {
			serverConfig.Header.Enable = *yamlConfigServer.Header.Enable
		} else {
			serverConfig.Header.Enable = configDefaultServerHeaderEnable
		}
		for _, headerRule := range yamlConfigServer.Header.Rules {
			rule := HeaderRule{
				Path:   headerRule.Path,
				Set:    headerRule.Set,
				Add:    headerRule.Add,
				Remove: headerRule.Remove,
			}
			if headerRule.Last != nil {
				rule.Last = *headerRule.Last
			} else {
				rule.Last = configDefaultServerHeaderRuleLast
			}
			serverConfig.Header.Rules = append(serverConfig.Header.Rules, rule)
		}

		if yamlConfigServer.Static.Enable != nil {
			serverConfig.Static.Enable = *yamlConfigServer.Static.Enable
		} else {
			serverConfig.Static.Enable = configDefaultServerStaticEnable
		}
		serverConfig.Static.Dir = yamlConfigServer.Static.Dir
		if yamlConfigServer.Static.Index != nil {
			serverConfig.Static.Index = *yamlConfigServer.Static.Index
		} else {
			serverConfig.Static.Index = configDefaultServerStaticIndex
		}

		if yamlConfigServer.Robots.Enable != nil {
			serverConfig.Robots.Enable = *yamlConfigServer.Robots.Enable
		} else {
			serverConfig.Robots.Enable = configDefaultServerRobotsEnable
		}
		if yamlConfigServer.Robots.Path != nil {
			serverConfig.Robots.Path = *yamlConfigServer.Robots.Path
		} else {
			serverConfig.Robots.Path = configDefaultServerRobotsPath
		}
		serverConfig.Robots.Hosts = yamlConfigServer.Robots.Hosts
		serverConfig.Robots.Sitemaps = yamlConfigServer.Robots.Sitemaps
		if yamlConfigServer.Robots.Cache != nil {
			serverConfig.Robots.Cache = *yamlConfigServer.Robots.Cache
		} else {
			serverConfig.Robots.Cache = configDefaultServerRobotsCache
		}
		if yamlConfigServer.Robots.CacheTTL != nil {
			serverConfig.Robots.CacheTTL = *yamlConfigServer.Robots.CacheTTL
		} else {
			serverConfig.Robots.CacheTTL = configDefaultServerRobotsCacheTTL
		}

		if yamlConfigServer.Sitemap.Enable != nil {
			serverConfig.Sitemap.Enable = *yamlConfigServer.Sitemap.Enable
		} else {
			serverConfig.Sitemap.Enable = configDefaultServerSitemapEnable
		}
		serverConfig.Sitemap.Root = yamlConfigServer.Sitemap.Root
		if yamlConfigServer.Sitemap.Cache != nil {
			serverConfig.Sitemap.Cache = *yamlConfigServer.Sitemap.Cache
		} else {
			serverConfig.Sitemap.Cache = configDefaultServerSitemapCache
		}
		if yamlConfigServer.Sitemap.CacheTTL != nil {
			serverConfig.Sitemap.CacheTTL = *yamlConfigServer.Sitemap.CacheTTL
		} else {
			serverConfig.Sitemap.CacheTTL = configDefaultServerSitemapCacheTTL
		}
		for _, sitemapRoute := range yamlConfigServer.Sitemap.Routes {
			route := SitemapRoute{
				Path: sitemapRoute.Path,
				Kind: sitemapRoute.Kind,
			}
			for _, sitemapIndexEntry := range sitemapRoute.SitemapIndex {
				route.SitemapIndex = append(route.SitemapIndex, SitemapIndexEntry{
					Name:   sitemapIndexEntry.Name,
					Type:   sitemapIndexEntry.Type,
					Static: SitemapIndexEntryStatic(sitemapIndexEntry.Static),
				})
			}
			for _, sitemapEntry := range sitemapRoute.Sitemap {
				route.Sitemap = append(route.Sitemap, SitemapEntry{
					Name:   sitemapEntry.Name,
					Type:   sitemapEntry.Type,
					Static: SitemapEntryStatic(sitemapEntry.Static),
					List:   SitemapEntryList(sitemapEntry.List),
				})
			}
			serverConfig.Sitemap.Routes = append(serverConfig.Sitemap.Routes, route)
		}

		if yamlConfigServer.Index.Enable != nil {
			serverConfig.Index.Enable = *yamlConfigServer.Index.Enable
		} else {
			serverConfig.Index.Enable = configDefaultServerIndexEnable
		}
		serverConfig.Index.HTML = yamlConfigServer.Index.HTML
		serverConfig.Index.Bundle = yamlConfigServer.Index.Bundle
		if yamlConfigServer.Index.Env != nil {
			serverConfig.Index.Env = *yamlConfigServer.Index.Env
		} else {
			serverConfig.Index.Env = configDefaultServerIndexEnv
		}
		if yamlConfigServer.Index.Container != nil {
			serverConfig.Index.Container = *yamlConfigServer.Index.Container
		} else {
			serverConfig.Index.Container = configDefaultServerIndexContainer
		}
		if yamlConfigServer.Index.State != nil {
			serverConfig.Index.State = *yamlConfigServer.Index.State
		} else {
			serverConfig.Index.State = configDefaultServerIndexState
		}
		if yamlConfigServer.Index.Timeout != nil {
			serverConfig.Index.Timeout = *yamlConfigServer.Index.Timeout
		} else {
			serverConfig.Index.Timeout = configDefaultServerIndexTimeout
		}
		if yamlConfigServer.Index.Cache != nil {
			serverConfig.Index.Cache = *yamlConfigServer.Index.Cache
		} else {
			serverConfig.Index.Cache = configDefaultServerIndexCache
		}
		if yamlConfigServer.Index.CacheTTL != nil {
			serverConfig.Index.CacheTTL = *yamlConfigServer.Index.CacheTTL
		} else {
			serverConfig.Index.CacheTTL = configDefaultServerIndexCacheTTL
		}
		for _, indexRule := range yamlConfigServer.Index.Rules {
			rule := IndexRule{
				Path: indexRule.Path,
			}
			if indexRule.Last != nil {
				rule.Last = *indexRule.Last
			} else {
				rule.Last = configDefaultServerIndexRuleLast
			}
			for _, indexRuleStateEntry := range indexRule.State {
				entry := IndexRuleStateEntry{
					Key:      indexRuleStateEntry.Key,
					Resource: indexRuleStateEntry.Resource,
				}
				if indexRuleStateEntry.Export != nil {
					entry.Export = indexRuleStateEntry.Export
				}
				rule.State = append(rule.State, entry)
			}
			serverConfig.Index.Rules = append(serverConfig.Index.Rules, rule)
		}

		if yamlConfigServer.Default.Enable != nil {
			serverConfig.Default.Enable = *yamlConfigServer.Default.Enable
		} else {
			serverConfig.Default.Enable = configDefaultServerDefaultEnable
		}
		serverConfig.Default.File = yamlConfigServer.Default.File
		if yamlConfigServer.Default.StatusCode != nil {
			serverConfig.Default.StatusCode = *yamlConfigServer.Default.StatusCode
		} else {
			serverConfig.Default.StatusCode = configDefaultServerDefaultStatusCode
		}
		if yamlConfigServer.Default.Cache != nil {
			serverConfig.Default.Cache = *yamlConfigServer.Default.Cache
		} else {
			serverConfig.Default.Cache = configDefaultServerDefaultCache
		}
		if yamlConfigServer.Default.CacheTTL != nil {
			serverConfig.Default.CacheTTL = *yamlConfigServer.Default.CacheTTL
		} else {
			serverConfig.Default.CacheTTL = configDefaultServerDefaultCacheTTL
		}

		c.Server = append(c.Server, &serverConfig)
	}

	fetcherConfig := FetcherConfig{}
	fetcherConfig.RequestTLSCAFile = y.Fetcher.RequestTLSCAFile
	fetcherConfig.RequestTLSCertFile = y.Fetcher.RequestTLSCertFile
	fetcherConfig.RequestTLSKeyFile = y.Fetcher.RequestTLSKeyFile
	fetcherConfig.RequestHeaders = y.Fetcher.RequestHeaders
	if y.Fetcher.RequestTimeout != nil {
		fetcherConfig.RequestTimeout = *y.Fetcher.RequestTimeout
	} else {
		fetcherConfig.RequestTimeout = configDefaultFetcherRequestTimeout
	}
	if y.Fetcher.RequestRetry != nil {
		fetcherConfig.RequestRetry = *y.Fetcher.RequestRetry
	} else {
		fetcherConfig.RequestRetry = configDefaultFetcherRequestRetry
	}
	if y.Fetcher.RequestDelay != nil {
		fetcherConfig.RequestDelay = *y.Fetcher.RequestDelay
	} else {
		fetcherConfig.RequestDelay = configDefaultFetcherRequestDelay
	}
	for _, resource := range y.Fetcher.Resources {
		fetcherConfig.Resources = append(fetcherConfig.Resources, FetcherResource{
			Name:    resource.Name,
			Method:  resource.Method,
			URL:     resource.URL,
			Params:  resource.Params,
			Headers: resource.Headers,
		})
	}
	for _, template := range y.Fetcher.Templates {
		fetcherConfig.Templates = append(fetcherConfig.Templates, FetcherTemplate{
			Name:    template.Name,
			Method:  template.Method,
			URL:     template.URL,
			Params:  template.Params,
			Headers: template.Headers,
		})
	}

	c.Fetcher = &fetcherConfig

	loaderConfig := LoaderConfig{}
	if y.Loader.ExecStartup != nil {
		loaderConfig.ExecStartup = *y.Loader.ExecStartup
	} else {
		loaderConfig.ExecStartup = configDefaultLoaderExecStartup
	}
	if y.Loader.ExecInterval != nil {
		loaderConfig.ExecInterval = *y.Loader.ExecInterval
	} else {
		loaderConfig.ExecInterval = configDefaultLoaderExecInterval
	}
	if y.Loader.ExecWorkers != nil {
		loaderConfig.ExecWorkers = *y.Loader.ExecWorkers
	} else {
		loaderConfig.ExecWorkers = configDefaultLoaderExecWorkers
	}
	for _, rule := range y.Loader.Rules {
		loaderConfig.Rules = append(loaderConfig.Rules, LoaderRule{
			Name:   rule.Name,
			Type:   rule.Type,
			Static: LoaderRuleStatic(rule.Static),
			Single: LoaderRuleSingle(rule.Single),
			List:   LoaderRuleList(rule.List),
		})
	}

	c.Loader = &loaderConfig

	return &c, nil
}

// TestConfig validates the configuration
func TestConfig(c *Config) ([]string, error) {
	var report []string

	if c.Server == nil {
		report = append(report, fmt.Sprintf("server: at least one server must be defined"))
	}
	for _, server := range c.Server {
		if server.TLS {
			if server.TLSCAFile != nil {
				if *server.TLSCertFile == "" {
					report = append(report, fmt.Sprintf("server: option '%s', invalid/missing value", "tls_ca_file"))
				}
				if *server.TLSCertFile != "" {
					tlsCAFile, err := os.Open(*server.TLSCAFile)
					if err != nil {
						report = append(report, fmt.Sprintf("server: option '%s', failed to open file", "tls_ca_file"))
					}
					if tlsCAFile != nil {
						tlsCAFile.Close()
					}
				}
			}
			if server.TLSCertFile == nil {
				report = append(report, fmt.Sprintf("server: option '%s', missing option", "tls_cert_file"))
			}
			if server.TLSCertFile != nil && *server.TLSCertFile == "" {
				report = append(report, fmt.Sprintf("server: option '%s', invalid/missing value", "tls_cert_file"))
			}
			if server.TLSCertFile != nil && *server.TLSCertFile != "" {
				tlsCertFile, err := os.Open(*server.TLSCertFile)
				if err != nil {
					report = append(report, fmt.Sprintf("server: option '%s', failed to open file", "tls_cert_file"))
				}
				if tlsCertFile != nil {
					tlsCertFile.Close()
				}
			}
			if server.TLSKeyFile == nil {
				report = append(report, fmt.Sprintf("server: option '%s', missing option", "tls_key_file"))
			}
			if server.TLSKeyFile != nil && *server.TLSKeyFile == "" {
				report = append(report, fmt.Sprintf("server: option '%s', invalid/missing value", "tls_key_file"))
			}
			if server.TLSKeyFile != nil && *server.TLSKeyFile != "" {
				tlsKeyFile, err := os.Open(*server.TLSKeyFile)
				if err != nil {
					report = append(report, fmt.Sprintf("server: option '%s', failed to open file", "tls_key_file"))
				}
				if tlsKeyFile != nil {
					tlsKeyFile.Close()
				}
			}
		}
		if server.ReadTimeout < 0 {
			report = append(report, fmt.Sprintf("server: option '%s', invalid/missing value", "read_timeout"))
		}
		if server.WriteTimeout < 0 {
			report = append(report, fmt.Sprintf("server: option '%s', invalid/missing value", "write_timeout"))
		}
		if server.AccessLogFile != nil {
			if *server.AccessLogFile == "" {
				report = append(report, fmt.Sprintf("server: option '%s', invalid/missing value", "access_log_file"))
			}
			if *server.AccessLogFile != "" {
				accessLogFile, err := os.Open(*server.AccessLogFile)
				if err != nil {
					report = append(report, fmt.Sprintf("server: option '%s', failed to open file", "access_log_file"))
				}
				if accessLogFile != nil {
					accessLogFile.Close()
				}
			}
		}

		if server.Rewrite.Enable {
			for _, rule := range server.Rewrite.Rules {
				if rule.Path == "" {
					report = append(report, fmt.Sprintf("rewrite: rule option '%s', invalid/missing value", "path"))
				}
				if rule.Path != "" {
					_, err := regexp.Compile(rule.Path)
					if err != nil {
						report = append(report, fmt.Sprintf("rewrite: rule option '%s', invalid regular expression", "path"))
					}
				}
			}
		}

		if server.Header.Enable {
			for _, rule := range server.Header.Rules {
				if rule.Path == "" {
					report = append(report, fmt.Sprintf("header: rule option '%s', invalid/missing value", "path"))
				}
				if rule.Path != "" {
					_, err := regexp.Compile(rule.Path)
					if err != nil {
						report = append(report, fmt.Sprintf("header: rule option '%s', invalid regular expression", "path"))
					}
				}
			}
		}

		if server.Static.Enable {
			if server.Static.Dir == "" {
				report = append(report, fmt.Sprintf("static: option '%s', invalid/missing value", "dir"))
			}
			if server.Static.Dir != "" {
				dir, err := os.Stat(server.Static.Dir)
				if err != nil {
					report = append(report, fmt.Sprintf("static: option '%s', failed to stat directory", "dir"))
				}
				if dir != nil && !dir.IsDir() {
					report = append(report, fmt.Sprintf("static: option '%s', failed to open directory", "dir"))
				}
			}
		}

		if server.Robots.Enable {
			if server.Robots.Path == "" {
				report = append(report, fmt.Sprintf("robots: rule option '%s', invalid/missing value", "path"))
			}
			if server.Robots.CacheTTL < 0 {
				report = append(report, fmt.Sprintf("robots: option '%s', invalid/missing value", "cache_ttl"))
			}
		}

		if server.Sitemap.Enable {
			if server.Sitemap.Root == "" {
				report = append(report, fmt.Sprintf("sitemap: rule option '%s', invalid/missing value", "root"))
			}
			if server.Sitemap.CacheTTL < 0 {
				report = append(report, fmt.Sprintf("sitemap: option '%s', invalid/missing value", "cache_ttl"))
			}
			for _, route := range server.Sitemap.Routes {
				if route.Path == "" {
					report = append(report, fmt.Sprintf("sitemap: route option '%s', invalid/missing value", "path"))
				}
				if route.Kind == "" {
					report = append(report, fmt.Sprintf("sitemap: route option '%s', invalid/missing value", "kind"))
				}
				if route.Kind != "" {
					validKind := false
					for _, k := range []string{
						sitemapKindSitemapIndex,
						sitemapKindSitemap,
					} {
						if k == route.Kind {
							validKind = true
						}
					}
					if !validKind {
						report = append(report, fmt.Sprintf("sitemap: route option '%s', invalid kind", "kind"))
					}
				}
				if route.Kind == sitemapKindSitemapIndex {
					for _, entry := range route.SitemapIndex {
						if entry.Name == "" {
							report = append(report,
								fmt.Sprintf("sitemap: sitemap_index entry option '%s', invalid/missing value", "name"))
						}
						validType := false
						for _, t := range []string{
							sitemapEntrySitemapIndexTypeStatic,
						} {
							if t == entry.Type {
								validType = true
							}
						}
						if !validType {
							report = append(report,
								fmt.Sprintf("sitemap: sitemap_index entry option '%s', invalid/missing value", "type"))
						}
						if entry.Type == sitemapEntrySitemapIndexTypeStatic {
							if entry.Static.Loc == "" {
								report = append(report,
									fmt.Sprintf("sitemap: sitemap_index static entry option '%s', invalid/missing value", "loc"))
							}
						}
					}
				}
				if route.Kind == sitemapKindSitemap {
					for _, entry := range route.Sitemap {
						if entry.Name == "" {
							report = append(report,
								fmt.Sprintf("sitemap: sitemap entry option '%s', invalid/missing value", "name"))
						}
						if entry.Type == "" {
							report = append(report,
								fmt.Sprintf("sitemap: sitemap entry option '%s', invalid/missing value", "type"))
						}
						if entry.Type != "" {
							validType := false
							for _, t := range []string{
								sitemapEntrySitemapTypeStatic,
								sitemapEntrySitemapTypeList,
							} {
								if t == entry.Type {
									validType = true
								}
							}
							if !validType {
								report = append(report,
									fmt.Sprintf("sitemap: sitemap entry option '%s', invalid type", "type"))
							}
						}
						if entry.Type == sitemapEntrySitemapTypeStatic {
							if entry.Static.Loc == "" {
								report = append(report,
									fmt.Sprintf("sitemap: sitemap static entry option '%s', invalid/missing value", "loc"))
							}
							if entry.Static.Lastmod != nil && *entry.Static.Lastmod == "" {
								report = append(report,
									fmt.Sprintf("sitemap: sitemap static entry option '%s', invalid/missing value", "lastmod"))
							}
							if entry.Static.Changefreq != nil && *entry.Static.Changefreq == "" {
								report = append(report,
									fmt.Sprintf("sitemap: sitemap static entry option '%s', invalid/missing value", "changefreq"))
							}
							if entry.Static.Changefreq != nil && *entry.Static.Changefreq != "" {
								validChangefreq := false
								for _, c := range []string{
									sitemapChangefreqAlways,
									sitemapChangefreqHourly,
									sitemapChangefreqDaily,
									sitemapChangefreqWeekly,
									sitemapChangefreqMonthly,
									sitemapChangefreqYearly,
									sitemapChangefreqNever,
								} {
									if c == *entry.Static.Changefreq {
										validChangefreq = true
									}
								}
								if !validChangefreq {
									report = append(report,
										fmt.Sprintf("sitemap: sitemap static entry option '%s', invalid changefreq", "changefreq"))
								}
							}
							if entry.Static.Priority != nil && (*entry.Static.Priority < 0.0 || *entry.Static.Priority > 1.0) {
								report = append(report,
									fmt.Sprintf("sitemap: sitemap static entry option '%s', invalid/missing value", "priority"))
							}
						}
						if entry.Type == sitemapEntrySitemapTypeList {
							if entry.List.Resource == "" {
								report = append(report,
									fmt.Sprintf("sitemap: sitemap list entry option '%s', invalid/missing value", "resource"))
							}
							if entry.List.Resource != "" {
								resourceExists := false
								for _, resource := range c.Fetcher.Resources {
									if resource.Name == entry.List.Resource {
										resourceExists = true
									}
								}
								if !resourceExists {
									report = append(report,
										fmt.Sprintf("sitemap: sitemap list entry option '%s', resource not found", "resource"))
								}
							}
							if entry.List.ResourcePayloadItems == "" {
								report = append(report,
									fmt.Sprintf("sitemap: sitemap list entry option '%s', invalid/missing value",
										"resource_payload_items"))
							}
							if entry.List.ResourcePayloadItemLoc == "" {
								report = append(report,
									fmt.Sprintf("sitemap: sitemap list entry option '%s', invalid/missing value",
										"resource_payload_item_loc"))
							}
							if entry.List.ResourcePayloadItemLastmod != nil && *entry.List.ResourcePayloadItemLastmod == "" {
								report = append(report,
									fmt.Sprintf("sitemap: sitemap list entry option '%s', invalid/missing value",
										"resource_payload_item_lastmod"))
							}
							if entry.List.Changefreq != nil && *entry.List.Changefreq == "" {
								report = append(report,
									fmt.Sprintf("sitemap: sitemap list entry option '%s', invalid/missing value", "changefreq"))
							}
							if entry.List.Changefreq != nil && *entry.List.Changefreq != "" {
								validChangefreq := false
								for _, c := range []string{
									sitemapChangefreqAlways,
									sitemapChangefreqHourly,
									sitemapChangefreqDaily,
									sitemapChangefreqWeekly,
									sitemapChangefreqMonthly,
									sitemapChangefreqYearly,
									sitemapChangefreqNever,
								} {
									if c == *entry.List.Changefreq {
										validChangefreq = true
									}
								}
								if !validChangefreq {
									report = append(report,
										fmt.Sprintf("sitemap: sitemap list entry option '%s', invalid changefreq", "changefreq"))
								}
							}
							if entry.List.Priority != nil && (*entry.List.Priority < 0.0 || *entry.List.Priority > 1.0) {
								report = append(report,
									fmt.Sprintf("sitemap: sitemap list entry option '%s', invalid/missing value", "priority"))
							}
						}
					}
				}
			}
		}

		if server.Index.Enable {
			if server.Index.HTML == "" {
				report = append(report, fmt.Sprintf("index: option '%s', invalid/missing value", "html"))
			}
			if server.Index.HTML != "" {
				htmlFile, err := os.Open(server.Index.HTML)
				if err != nil {
					report = append(report, fmt.Sprintf("index: option '%s', failed to open file", "html"))
				}
				if htmlFile != nil {
					htmlFile.Close()
				}
			}
			if server.Index.Bundle != nil {
				if *server.Index.Bundle == "" {
					report = append(report, fmt.Sprintf("index: option '%s', invalid/missing value", "bundle"))
				}
				if *server.Index.Bundle != "" {
					bundleFile, err := os.Open(*server.Index.Bundle)
					if err != nil {
						report = append(report, fmt.Sprintf("index: option '%s', failed to open file", "bundle"))
					}
					if bundleFile != nil {
						bundleFile.Close()
					}
				}
			}
			if server.Index.Env == "" {
				report = append(report, fmt.Sprintf("index: option '%s', invalid/missing value", "env"))
			}
			if server.Index.Container == "" {
				report = append(report, fmt.Sprintf("index: option '%s', invalid/missing value", "container"))
			}
			if server.Index.State == "" {
				report = append(report, fmt.Sprintf("index: option '%s', invalid/missing value", "state"))
			}
			if server.Index.Timeout < 0 {
				report = append(report, fmt.Sprintf("index: option '%s', invalid/missing value", "timeout"))
			}
			if server.Index.CacheTTL < 0 {
				report = append(report, fmt.Sprintf("index: option '%s', invalid/missing value", "cache_ttl"))
			}
			for _, rule := range server.Index.Rules {
				if rule.Path == "" {
					report = append(report, fmt.Sprintf("index: rule option '%s', invalid/missing value", "path"))
				}
				if rule.Path != "" {
					_, err := regexp.Compile(rule.Path)
					if err != nil {
						report = append(report, fmt.Sprintf("index: rule option '%s', invalid regular expression", "path"))
					}
				}
				for _, state := range rule.State {
					if state.Key == "" {
						report = append(report, fmt.Sprintf("index: rule state option '%s', invalid/missing value", "key"))
					}
					if state.Resource == "" {
						report = append(report, fmt.Sprintf("index: rule state option '%s', invalid/missing value", "resource"))
					}
				}
			}
		}

		if server.Default.Enable {
			if server.Default.File == "" {
				report = append(report, fmt.Sprintf("default: option '%s', invalid/missing value", "file"))
			}
			if server.Default.File != "" {
				defaultFile, err := os.Open(server.Default.File)
				if err != nil {
					report = append(report, fmt.Sprintf("default: option '%s', failed to open file", "file"))
				}
				if defaultFile != nil {
					defaultFile.Close()
				}
			}
			if server.Default.StatusCode < 100 || server.Default.StatusCode > 599 {
				report = append(report, fmt.Sprintf("default: option '%s', invalid/missing value", "status_code"))
			}
			if server.Default.CacheTTL < 0 {
				report = append(report, fmt.Sprintf("default: option '%s', invalid/missing value", "cache_ttl"))
			}
		}
	}

	if c.Fetcher != nil {
		if c.Fetcher.RequestTLSCAFile != nil {
			if *c.Fetcher.RequestTLSCAFile == "" {
				report = append(report, fmt.Sprintf("fetcher: option '%s', invalid/missing value", "request_tls_ca_file"))
			}
			if *c.Fetcher.RequestTLSCAFile != "" {
				requestTLSCAFile, err := os.Open(*c.Fetcher.RequestTLSCAFile)
				if err != nil {
					report = append(report, fmt.Sprintf("fetcher: option '%s', failed to open file", "request_tls_ca_file"))
				}
				if requestTLSCAFile != nil {
					requestTLSCAFile.Close()
				}
			}
		}
		if c.Fetcher.RequestTLSCertFile != nil {
			if *c.Fetcher.RequestTLSCertFile == "" {
				report = append(report, fmt.Sprintf("fetcher: option '%s', invalid/missing value", "request_tls_cert_file"))
			}
			if *c.Fetcher.RequestTLSCertFile != "" {
				requestTLSCertFile, err := os.Open(*c.Fetcher.RequestTLSCertFile)
				if err != nil {
					report = append(report, fmt.Sprintf("fetcher: option '%s', failed to open file", "request_tls_cert_file"))
				}
				if requestTLSCertFile != nil {
					requestTLSCertFile.Close()
				}
			}
		}
		if c.Fetcher.RequestTLSKeyFile != nil {
			if *c.Fetcher.RequestTLSKeyFile == "" {
				report = append(report, fmt.Sprintf("fetcher: option '%s', invalid/missing value", "request_tls_key_file"))
			}
			if *c.Fetcher.RequestTLSKeyFile != "" {
				requestTLSKeyFile, err := os.Open(*c.Fetcher.RequestTLSKeyFile)
				if err != nil {
					report = append(report, fmt.Sprintf("fetcher: option '%s', failed to open file", "request_tls_key_file"))
				}
				if requestTLSKeyFile != nil {
					requestTLSKeyFile.Close()
				}
			}
		}
		if c.Fetcher.RequestTimeout < 0 {
			report = append(report, fmt.Sprintf("fetcher: option '%s', invalid/missing value", "request_timeout"))
		}
		if c.Fetcher.RequestRetry < 0 {
			report = append(report, fmt.Sprintf("fetcher: option '%s', invalid/missing value", "request_retry"))
		}
		if c.Fetcher.RequestDelay < 0 {
			report = append(report, fmt.Sprintf("fetcher: option '%s', invalid/missing value", "request_delay"))
		}
		for _, resource := range c.Fetcher.Resources {
			if resource.Name == "" {
				report = append(report, fmt.Sprintf("fetcher: resource option '%s', invalid/missing value", "name"))
			}
			if resource.Method == "" {
				report = append(report, fmt.Sprintf("fetcher: resource option '%s', invalid/missing value", "method"))
			}
			if resource.Method != "" {
				validMethod := false
				for _, m := range []string{
					"GET",
					"POST",
					"PATCH",
					"PUT",
					"DELETE",
					"HEAD",
					"OPTIONS",
				} {
					if m == resource.Method {
						validMethod = true
					}
				}
				if !validMethod {
					report = append(report, fmt.Sprintf("fetcher: resource option '%s', invalid method", "method"))
				}
			}
			if resource.URL == "" {
				report = append(report, fmt.Sprintf("fetcher: resource option '%s', invalid/missing value", "url"))
			}
			if resource.URL != "" {
				_, err := url.Parse(resource.URL)
				if err != nil {
					report = append(report, fmt.Sprintf("fetcher: resource option '%s', invalid URL", "url"))
				}
			}
		}
		for _, template := range c.Fetcher.Resources {
			if template.Name == "" {
				report = append(report, fmt.Sprintf("fetcher: template option '%s', invalid/missing value", "name"))
			}
			if template.Method == "" {
				report = append(report, fmt.Sprintf("fetcher: template option '%s', invalid/missing value", "method"))
			}
			if template.Method != "" {
				validMethod := false
				for _, m := range []string{
					"GET",
					"POST",
					"PATCH",
					"PUT",
					"DELETE",
					"HEAD",
					"OPTIONS",
				} {
					if m == template.Method {
						validMethod = true
					}
				}
				if !validMethod {
					report = append(report, fmt.Sprintf("fetcher: template option '%s', invalid method", "method"))
				}
			}
			if template.URL == "" {
				report = append(report, fmt.Sprintf("fetcher: template option '%s', invalid/missing value", "url"))
			}
			if template.URL != "" {
				_, err := url.Parse(template.URL)
				if err != nil {
					report = append(report, fmt.Sprintf("fetcher: template option '%s', invalid URL", "url"))
				}
			}
		}
	}

	if c.Loader != nil {
		if c.Loader.ExecStartup < 0 {
			report = append(report, fmt.Sprintf("loader: option '%s', invalid/missing value", "exec_startup"))
		}
		if c.Loader.ExecInterval < 0 {
			report = append(report, fmt.Sprintf("loader: option '%s', invalid/missing value", "exec_interval"))
		}
		if c.Loader.ExecWorkers < 0 {
			report = append(report, fmt.Sprintf("loader: option '%s', invalid/missing value", "exec_workers"))
		}
		for _, rule := range c.Loader.Rules {
			if rule.Name == "" {
				report = append(report, fmt.Sprintf("loader: rule option '%s', invalid/missing value", "name"))
			}
			if rule.Type == "" {
				report = append(report, fmt.Sprintf("loader: rule option '%s', invalid/missing value", "type"))
			}
			if rule.Type != "" {
				validType := false
				for _, t := range []string{
					loaderTypeStatic,
					loaderTypeSingle,
					loaderTypeList,
				} {
					if t == rule.Type {
						validType = true
					}
				}
				if !validType {
					report = append(report, fmt.Sprintf("loader: rule option '%s', invalid type", "type"))
				}
				if rule.Type == loaderTypeSingle {
					if rule.Static.Resource == "" {
						report = append(report, fmt.Sprintf("loader: static rule option '%s', invalid/missing value", "resource"))
					}
				}
				if rule.Type == loaderTypeSingle {
					if rule.Single.Resource == "" {
						report = append(report, fmt.Sprintf("loader: single rule option '%s', invalid/missing value", "resource"))
					}
					if rule.Single.ResourcePayloadItem == "" {
						report = append(report,
							fmt.Sprintf("loader: single rule option '%s', invalid/missing value", "resource_payload_item"))
					}
					if rule.Single.ItemTemplate == "" {
						report = append(report,
							fmt.Sprintf("loader: single rule option '%s', invalid/missing value", "item_template"))
					}
					if rule.Single.ItemTemplate != "" {
						templateExists := false
						for _, template := range c.Fetcher.Templates {
							if template.Name == rule.Single.ItemTemplate {
								templateExists = true
							}
						}
						if !templateExists {
							report = append(report,
								fmt.Sprintf("loader: single rule option '%s', template not found", "item_template"))
						}
					}
					if rule.Single.ItemTemplateResource == "" {
						report = append(report,
							fmt.Sprintf("loader: single rule option '%s', invalid/missing value", "item_template_resource"))
					}
				}
				if rule.Type == loaderTypeList {
					if rule.List.Resource == "" {
						report = append(report, fmt.Sprintf("loader: list rule option '%s', invalid/missing value", "resource"))
					}
					if rule.List.ResourcePayloadItems == "" {
						report = append(report,
							fmt.Sprintf("loader: list rule option '%s', invalid/missing value", "resource_payload_items"))
					}
					if rule.List.ItemTemplate == "" {
						report = append(report,
							fmt.Sprintf("loader: list rule option '%s', invalid/missing value", "item_template"))
					}
					if rule.List.ItemTemplate != "" {
						templateExists := false
						for _, template := range c.Fetcher.Templates {
							if template.Name == rule.List.ItemTemplate {
								templateExists = true
							}
						}
						if !templateExists {
							report = append(report,
								fmt.Sprintf("loader: list rule option '%s', template not found", "item_template"))
						}
					}
					if rule.List.ItemTemplateResource == "" {
						report = append(report,
							fmt.Sprintf("loader: list rule option '%s', invalid/missing value", "item_template_resource"))
					}
				}
			}
		}
	}

	if len(report) > 0 {
		return report, errors.New("invalid configuration")
	}

	return nil, nil
}

//go:embed templates/init/config.yaml
var template_config []byte

//go:embed templates/init/data/index.html
var template_data_html []byte

//go:embed templates/init/data/bundle.js
var template_data_bundle []byte

//go:embed templates/init/data/static/styles.css
var template_data_static_styles []byte

//go:embed templates/init/data/static/manifest.json
var template_data_static_manifest []byte

//go:embed templates/init/data/static/favicon.ico
var template_data_static_favicon []byte

//go:embed templates/init/data/static/logo.png
var template_data_static_logo []byte

//go:embed templates/init/data/static/logo192.png
var template_data_static_logo192 []byte

//go:embed templates/init/data/static/logo512.png
var template_data_static_logo512 []byte

// GenerateConfig creates a default configuration file
func GenerateConfig() error {
	file, ok := os.LookupEnv("CONFIG_FILE")
	if !ok {
		file = configFile
	}

	_, err := os.Stat(file)
	if err == nil {
		return errors.New("configuration file already exists")
	}

	_, err = os.Stat("data")
	if err == nil {
		return errors.New("data directory already exists")
	}

	err = os.WriteFile(file, template_config, 0640)
	if err != nil {
		return errors.New("failed to generate files")
	}

	err = os.MkdirAll("data/static", 0755)
	if err != nil {
		return errors.New("failed to generate data directory")
	}

	err = os.WriteFile("data/index.html", template_data_html, 0644)
	if err != nil {
		return errors.New("failed to generate HTML file")
	}

	err = os.WriteFile("data/bundle.js", template_data_bundle, 0644)
	if err != nil {
		return errors.New("failed to generate bundle file")
	}

	files := map[string][]byte{
		"data/index.html":           template_data_html,
		"data/bundle.js":            template_data_bundle,
		"data/static/styles.css":    template_data_static_styles,
		"data/static/manifest.json": template_data_static_manifest,
		"data/static/favicon.ico":   template_data_static_favicon,
		"data/static/logo.png":      template_data_static_logo,
		"data/static/logo192.png":   template_data_static_logo192,
		"data/static/logo512.png":   template_data_static_logo512,
	}

	for name, template := range files {
		err = os.WriteFile(name, template, 0644)
		if err != nil {
			return errors.New("failed to write file")
		}
	}

	return nil
}
