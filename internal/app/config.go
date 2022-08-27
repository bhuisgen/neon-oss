// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"os"
	"strconv"

	"gopkg.in/yaml.v2"
)

const (
	CONFIG_FILE                            string = "config.yaml"
	CONFIG_DEFAULT_SERVER_LISTENADDR       string = "localhost"
	CONFIG_DEFAULT_SERVER_LISTENPORT       int    = 8080
	CONFIG_DEFAULT_SERVER_TLS              bool   = false
	CONFIG_DEFAULT_SERVER_READTIMEOUT      int    = 60
	CONFIG_DEFAULT_SERVER_WRITETIMEOUT     int    = 60
	CONFIG_DEFAULT_SERVER_REWRITE_ENABLE   bool   = false
	CONFIG_DEFAULT_SERVER_STATIC_ENABLE    bool   = false
	CONFIG_DEFAULT_SERVER_INDEX_ENABLE     bool   = false
	CONFIG_DEFAULT_SERVER_INDEX_ENV        string = "production"
	CONFIG_DEFAULT_SERVER_INDEX_TIMEOUT    int    = 4
	CONFIG_DEFAULT_SERVER_INDEX_CACHE      bool   = false
	CONFIG_DEFAULT_SERVER_INDEX_CACHETTL   int    = 60
	CONFIG_DEFAULT_SERVER_ROBOTS_ENABLE    bool   = false
	CONFIG_DEFAULT_SERVER_ROBOTS_PATH      string = "/robots.txt"
	CONFIG_DEFAULT_SERVER_ROBOTS_CACHE     bool   = false
	CONFIG_DEFAULT_SERVER_ROBOTS_CACHETTL  int    = 60
	CONFIG_DEFAULT_SERVER_SITEMAP_ENABLE   bool   = false
	CONFIG_DEFAULT_SERVER_SITEMAP_CACHE    bool   = false
	CONFIG_DEFAULT_SERVER_SITEMAP_CACHETTL int    = 60
	CONFIG_DEFAULT_FETCHER_REQUESTTIMEOUT  int    = 60
	CONFIG_DEFAULT_FETCHER_REQUESTRETRY    int    = 3
	CONFIG_DEFAULT_FETCHER_REQUESTDELAY    int    = 1
	CONFIG_DEFAULT_LOADER_EXECSTARTUP      int    = 15
	CONFIG_DEFAULT_LOADER_EXECINTERVAL     int    = 60
	CONFIG_DEFAULT_LOADER_EXECWORKERS      int    = 1
)

// Config implements the configuration
type Config struct {
	Server  []*ServerConfig
	Fetcher *FetcherConfig
	Loader  *LoaderConfig
}

type yamlConfigServer struct {
	ListenAddr   *string `yaml:"listen_addr"`
	ListenPort   *int    `yaml:"listen_port"`
	TLS          *bool   `yaml:"tls"`
	TLSCAFile    string  `yaml:"tls_ca_file"`
	TLSCertFile  string  `yaml:"tls_cert_file"`
	TLSKeyFile   string  `yaml:"tls_key_file"`
	ReadTimeout  *int    `yaml:"read_timeout"`
	WriteTimeout *int    `yaml:"write_timeout"`

	Rewrite struct {
		Enable *bool `yaml:"enable"`
		Rules  []struct {
			Regex       string `yaml:"regex"`
			Replacement string `yaml:"replacement"`
			Flag        string `yaml:"flag"`
		} `yaml:"rules"`
	}

	Static struct {
		Enable *bool  `yaml:"enable"`
		Path   string `yaml:"path"`
	} `yaml:"static"`

	Index struct {
		Enable   *bool   `yaml:"enable"`
		HTML     string  `yaml:"html"`
		Bundle   string  `yaml:"bundle"`
		Env      *string `yaml:"env"`
		Timeout  *int    `yaml:"timeout"`
		Cache    *bool   `yaml:"cache"`
		CacheTTL *int    `yaml:"cache_ttl"`
		Routes   []struct {
			Path string `yaml:"path"`
			Data []struct {
				Name     string `yaml:"name"`
				Resource string `yaml:"resource"`
			} `yaml:"data"`
		} `yaml:"routes"`
	} `yaml:"index"`

	Robots struct {
		Enable   *bool    `yaml:"enable"`
		Path     *string  `yaml:"path"`
		Hosts    []string `yaml:"hosts"`
		Cache    *bool    `yaml:"cache"`
		CacheTTL *int     `yaml:"cache_ttl"`
	} `yaml:"robots"`

	Sitemap struct {
		Enable   *bool  `yaml:"enable"`
		Root     string `yaml:"root"`
		Cache    *bool  `yaml:"cache"`
		CacheTTL *int   `yaml:"cache_ttl"`
		Routes   []struct {
			Path         string `yaml:"path"`
			Kind         string `yaml:"kind"`
			SitemapIndex []struct {
				Name   string `yaml:"name"`
				Type   string `yaml:"type"`
				Static struct {
					Loc string `yaml:"loc"`
				} `yaml:"static"`
			} `yaml:"sitemapindex"`
			Sitemap []struct {
				Name   string `yaml:"name"`
				Type   string `yaml:"type"`
				Static struct {
					Loc        string `yaml:"loc"`
					Lastmod    string `yaml:"lastmod"`
					Changefreq string `yaml:"changefreq"`
					Priority   string `yaml:"priority"`
				} `yaml:"static"`
				List struct {
					Resource                   string `yaml:"resource"`
					ResourcePayloadItems       string `yaml:"resource_payload_items"`
					ResourcePayloadItemLoc     string `yaml:"resource_payload_item_loc"`
					ResourcePayloadItemLastmod string `yaml:"resource_payload_item_lastmod"`
					Changefreq                 string `yaml:"changefreq"`
					Priority                   string `yaml:"priority"`
				} `yaml:"list"`
			} `yaml:"sitemap"`
		} `yaml:"routes"`
	} `yaml:"sitemap"`
}

type yamlConfigFetcher struct {
	RequestTLSCAFile   string            `yaml:"request_tls_ca_file"`
	RequestTLSCertFile string            `yaml:"request_tls_cert_file"`
	RequestTLSKeyFile  string            `yaml:"request_tls_key_file"`
	RequestHeaders     map[string]string `yaml:"request_headers"`
	RequestTimeout     *int              `yaml:"request_timeout"`
	RequestRetry       *int              `yaml:"request_retry"`
	RequestDelay       *int              `yaml:"request_delay"`
	Resources          []struct {
		Key     string            `yaml:"key"`
		Method  string            `yaml:"method"`
		URL     string            `yaml:"url"`
		Params  map[string]string `yaml:"params"`
		Headers map[string]string `yaml:"headers"`
	} `yaml:"resources"`
	Templates []struct {
		Name    string            `yaml:"name"`
		Method  string            `yaml:"method"`
		URL     string            `yaml:"url"`
		Params  map[string]string `yaml:"params"`
		Headers map[string]string `yaml:"headers"`
	} `yaml:"templates"`
}

type yamlConfigLoader struct {
	ExecStartup  *int `yaml:"exec_startup"`
	ExecInterval *int `yaml:"exec_interval"`
	ExecWorkers  *int `yaml:"exec_workers"`
	Rules        []struct {
		Name   string `yaml:"name"`
		Type   string `yaml:"type"`
		Static struct {
			Resource string `yaml:"resource"`
		} `yaml:"static"`
		Single struct {
			Resource              string            `yaml:"resource"`
			ResourcePayloadItem   string            `yaml:"resource_payload_item"`
			ItemTemplate          string            `yaml:"item_template"`
			ItemTemplateKey       string            `yaml:"item_template_key"`
			ItemTemplateKeyParams map[string]string `yaml:"item_template_key_params"`
		} `yaml:"single"`
		List struct {
			Resource              string            `yaml:"resource"`
			ResourcePayloadItems  string            `yaml:"resource_payload_items"`
			ItemTemplate          string            `yaml:"item_template"`
			ItemTemplateKey       string            `yaml:"item_template_key"`
			ItemTemplateKeyParams map[string]string `yaml:"item_template_key_params"`
		} `yaml:"list"`
	} `yaml:"rules"`
}

type yamlConfig struct {
	Server  []yamlConfigServer `yaml:"server"`
	Fetcher yamlConfigFetcher  `yaml:"fetcher"`
	Loader  yamlConfigLoader   `yaml:"loader"`
}

// LoadConfig loads the configuration settings
func LoadConfig() (*Config, error) {
	file, ok := os.LookupEnv("CONFIG_FILE")
	if !ok {
		file = CONFIG_FILE
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

	c, err := parse(&y)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func parse(y *yamlConfig) (*Config, error) {
	c := Config{}

	for index, yamlConfigServer := range y.Server {
		serverConfig := ServerConfig{}

		if yamlConfigServer.ListenAddr != nil {
			serverConfig.ListenAddr = *yamlConfigServer.ListenAddr
		} else {
			listenAddr := CONFIG_DEFAULT_SERVER_LISTENADDR
			if v, ok := os.LookupEnv("LISTEN_ADDR"); ok && index == 0 {
				listenAddr = v
			}
			serverConfig.ListenAddr = listenAddr
		}
		if yamlConfigServer.ListenPort != nil {
			serverConfig.ListenPort = *yamlConfigServer.ListenPort
		} else {
			listenPort := CONFIG_DEFAULT_SERVER_LISTENPORT
			if v, ok := os.LookupEnv("LISTEN_PORT"); ok && index == 0 {
				if vInt, err := strconv.ParseInt(v, 10, 0); err == nil {
					listenPort = int(vInt)
				}
			}
			serverConfig.ListenPort = listenPort
		}
		if yamlConfigServer.TLS != nil {
			serverConfig.TLS = *yamlConfigServer.TLS
		} else {
			serverConfig.TLS = CONFIG_DEFAULT_SERVER_TLS
		}
		serverConfig.TLSCAFile = yamlConfigServer.TLSCAFile
		serverConfig.TLSCertFile = yamlConfigServer.TLSCertFile
		serverConfig.TLSKeyFile = yamlConfigServer.TLSKeyFile
		if yamlConfigServer.ReadTimeout != nil {
			serverConfig.ReadTimeout = *yamlConfigServer.ReadTimeout
		} else {
			serverConfig.ReadTimeout = CONFIG_DEFAULT_SERVER_READTIMEOUT
		}
		if yamlConfigServer.WriteTimeout != nil {
			serverConfig.WriteTimeout = *yamlConfigServer.WriteTimeout
		} else {
			serverConfig.WriteTimeout = CONFIG_DEFAULT_SERVER_WRITETIMEOUT
		}

		if yamlConfigServer.Rewrite.Enable != nil {
			serverConfig.Rewrite.Enable = *yamlConfigServer.Rewrite.Enable
		} else {
			serverConfig.Rewrite.Enable = CONFIG_DEFAULT_SERVER_REWRITE_ENABLE
		}
		for _, rewriteRule := range yamlConfigServer.Rewrite.Rules {
			serverConfig.Rewrite.Rules = append(serverConfig.Rewrite.Rules, RewriteRule{
				Regex:       rewriteRule.Regex,
				Replacement: rewriteRule.Replacement,
				Flag:        rewriteRule.Flag,
			})
		}

		if yamlConfigServer.Static.Enable != nil {
			serverConfig.Static.Enable = *yamlConfigServer.Static.Enable
		} else {
			serverConfig.Static.Enable = CONFIG_DEFAULT_SERVER_STATIC_ENABLE
		}
		serverConfig.Static.Path = yamlConfigServer.Static.Path

		if yamlConfigServer.Index.Enable != nil {
			serverConfig.Index.Enable = *yamlConfigServer.Index.Enable
		} else {
			serverConfig.Index.Enable = CONFIG_DEFAULT_SERVER_INDEX_ENABLE
		}
		serverConfig.Index.HTML = yamlConfigServer.Index.HTML
		serverConfig.Index.Bundle = yamlConfigServer.Index.Bundle
		if yamlConfigServer.Index.Env != nil {
			serverConfig.Index.Env = *yamlConfigServer.Index.Env
		} else {
			serverConfig.Index.Env = CONFIG_DEFAULT_SERVER_INDEX_ENV
		}
		if yamlConfigServer.Index.Timeout != nil {
			serverConfig.Index.Timeout = *yamlConfigServer.Index.Timeout
		} else {
			serverConfig.Index.Timeout = CONFIG_DEFAULT_SERVER_INDEX_TIMEOUT
		}
		if yamlConfigServer.Index.Cache != nil {
			serverConfig.Index.Cache = *yamlConfigServer.Index.Cache
		} else {
			serverConfig.Index.Cache = CONFIG_DEFAULT_SERVER_INDEX_CACHE
		}
		if yamlConfigServer.Index.CacheTTL != nil {
			serverConfig.Index.CacheTTL = *yamlConfigServer.Index.CacheTTL
		} else {
			serverConfig.Index.CacheTTL = CONFIG_DEFAULT_SERVER_INDEX_CACHETTL
		}
		for _, indexRoute := range yamlConfigServer.Index.Routes {
			route := IndexRoute{
				Path: indexRoute.Path,
			}
			for _, indexRouteData := range indexRoute.Data {
				route.Data = append(route.Data, IndexRouteData{
					Name:     indexRouteData.Name,
					Resource: indexRouteData.Resource,
				})
			}
			serverConfig.Index.Routes = append(serverConfig.Index.Routes, route)
		}

		if yamlConfigServer.Robots.Enable != nil {
			serverConfig.Robots.Enable = *yamlConfigServer.Robots.Enable
		} else {
			serverConfig.Robots.Enable = CONFIG_DEFAULT_SERVER_ROBOTS_ENABLE
		}
		if yamlConfigServer.Robots.Path != nil {
			serverConfig.Robots.Path = *yamlConfigServer.Robots.Path
		} else {
			serverConfig.Robots.Path = CONFIG_DEFAULT_SERVER_ROBOTS_PATH
		}
		serverConfig.Robots.Hosts = yamlConfigServer.Robots.Hosts
		if yamlConfigServer.Robots.Cache != nil {
			serverConfig.Robots.Cache = *yamlConfigServer.Robots.Cache
		} else {
			serverConfig.Robots.Cache = CONFIG_DEFAULT_SERVER_ROBOTS_CACHE
		}
		if yamlConfigServer.Robots.CacheTTL != nil {
			serverConfig.Robots.CacheTTL = *yamlConfigServer.Robots.CacheTTL
		} else {
			serverConfig.Robots.CacheTTL = CONFIG_DEFAULT_SERVER_ROBOTS_CACHETTL
		}

		if yamlConfigServer.Sitemap.Enable != nil {
			serverConfig.Sitemap.Enable = *yamlConfigServer.Sitemap.Enable
		} else {
			serverConfig.Sitemap.Enable = CONFIG_DEFAULT_SERVER_SITEMAP_ENABLE
		}
		serverConfig.Sitemap.Root = yamlConfigServer.Sitemap.Root
		if yamlConfigServer.Sitemap.Cache != nil {
			serverConfig.Sitemap.Cache = *yamlConfigServer.Sitemap.Cache
		} else {
			serverConfig.Sitemap.Cache = CONFIG_DEFAULT_SERVER_SITEMAP_CACHE
		}
		if yamlConfigServer.Sitemap.CacheTTL != nil {
			serverConfig.Sitemap.CacheTTL = *yamlConfigServer.Sitemap.CacheTTL
		} else {
			serverConfig.Sitemap.CacheTTL = CONFIG_DEFAULT_SERVER_SITEMAP_CACHETTL
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
		fetcherConfig.RequestTimeout = CONFIG_DEFAULT_FETCHER_REQUESTTIMEOUT
	}
	if y.Fetcher.RequestRetry != nil {
		fetcherConfig.RequestRetry = *y.Fetcher.RequestRetry
	} else {
		fetcherConfig.RequestRetry = CONFIG_DEFAULT_FETCHER_REQUESTRETRY
	}
	if y.Fetcher.RequestDelay != nil {
		fetcherConfig.RequestDelay = *y.Fetcher.RequestDelay
	} else {
		fetcherConfig.RequestDelay = CONFIG_DEFAULT_FETCHER_REQUESTDELAY
	}
	for _, resource := range y.Fetcher.Resources {
		fetcherConfig.Resources = append(fetcherConfig.Resources, FetcherResource{
			Key:     resource.Key,
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
		loaderConfig.ExecStartup = CONFIG_DEFAULT_LOADER_EXECSTARTUP
	}
	if y.Loader.ExecInterval != nil {
		loaderConfig.ExecInterval = *y.Loader.ExecInterval
	} else {
		loaderConfig.ExecInterval = CONFIG_DEFAULT_LOADER_EXECINTERVAL
	}
	if y.Loader.ExecWorkers != nil {
		loaderConfig.ExecWorkers = *y.Loader.ExecWorkers
	} else {
		loaderConfig.ExecWorkers = CONFIG_DEFAULT_LOADER_EXECWORKERS
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
