// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	_ "embed"
	"errors"
	"io/fs"
	"os"
	"reflect"
	"testing"
	"time"
)

func bytePtr(b []byte) *[]byte {
	return &b
}

func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}

func floatPtr(f float32) *float32 {
	return &f
}

func timePtr(t time.Time) *time.Time {
	return &t
}

type testConfigFileInfo struct {
	name     string
	size     int64
	fileMode os.FileMode
	modTime  time.Time
	isDir    bool
	sys      any
}

func (fi testConfigFileInfo) Name() string {
	return fi.name
}

func (fi testConfigFileInfo) Size() int64 {
	return fi.size
}

func (fi testConfigFileInfo) Mode() os.FileMode {
	return fi.fileMode
}
func (fi testConfigFileInfo) ModTime() time.Time {
	return fi.modTime
}

func (fi testConfigFileInfo) IsDir() bool {
	return fi.isDir
}

func (fi testConfigFileInfo) Sys() any {
	return fi.sys
}

func TestNewConfig(t *testing.T) {
	type args struct {
		parser configParser
	}
	tests := []struct {
		name    string
		args    args
		wantNil bool
	}{
		{
			name: "default",
			args: args{
				parser: newConfigParserYAML(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newConfig(tt.args.parser)
			if (got == nil) != tt.wantNil {
				t.Errorf("newConfig() got = %v, wantNil %v", got, tt.wantNil)
			}
		})
	}
}

func TestNewConfigParserYAML(t *testing.T) {
	tests := []struct {
		name    string
		wantNil bool
	}{
		{
			name: "default",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newConfigParserYAML()
			if (got == nil) != tt.wantNil {
				t.Errorf("newConfig() got = %v, wantNil %v", got, tt.wantNil)
			}
		})
	}
}

func TestConfigParserYAMLParse(t *testing.T) {
	type fields struct {
		yamlUnmarshal func(in []byte, out interface{}) error
	}
	type args struct {
		data []byte
		c    *config
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				yamlUnmarshal: func(in []byte, out interface{}) error {
					*out.(*yamlConfig) = yamlConfig{
						Server: []yamlConfigServer{
							{
								Rewrite: &struct {
									Rules []struct {
										Path        string  "yaml:\"path\""
										Replacement string  "yaml:\"replacement\""
										Flag        *string "yaml:\"flag,omitempty\""
										Last        *bool   "yaml:\"last,omitempty\""
									} "yaml:\"rules\""
								}{},
								Header: &struct {
									Rules []struct {
										Path   string            "yaml:\"path\""
										Set    map[string]string "yaml:\"set,omitempty\""
										Add    map[string]string "yaml:\"add,omitempty\""
										Remove []string          "yaml:\"remove,omitempty\""
										Last   *bool             "yaml:\"last,omitempty\""
									} "yaml:\"rules\""
								}{},
								Static: &struct {
									Dir   string "yaml:\"dir\""
									Index *bool  "yaml:\"index,omitempty\""
								}{},
								Robots: &struct {
									Path     *string  "yaml:\"path,omitempty\""
									Hosts    []string "yaml:\"hosts,omitempty\""
									Sitemaps []string "yaml:\"sitemaps,omitempty\""
									Cache    *bool    "yaml:\"cache,omitempty\""
									CacheTTL *int     "yaml:\"cache_ttl,omitempty\""
								}{},
								Sitemap: &struct {
									Root     string "yaml:\"root\""
									Cache    *bool  "yaml:\"cache,omitempty\""
									CacheTTL *int   "yaml:\"cache_ttl,omitempty\""
									Routes   []struct {
										Path         string "yaml:\"path\""
										Kind         string "yaml:\"kind\""
										SitemapIndex []struct {
											Name   string "yaml:\"name\""
											Type   string "yaml:\"type\""
											Static struct {
												Loc string "yaml:\"loc\""
											} "yaml:\"static,omitempty\""
										} "yaml:\"sitemap_index,omitempty\""
										Sitemap []struct {
											Name   string "yaml:\"name\""
											Type   string "yaml:\"type\""
											Static struct {
												Loc        string   "yaml:\"loc\""
												Lastmod    *string  "yaml:\"lastmod,omitempty\""
												Changefreq *string  "yaml:\"changefreq,omitempty\""
												Priority   *float32 "yaml:\"priority,omitempty\""
											} "yaml:\"static,omitempty\""
											List struct {
												Resource                   string   "yaml:\"resource\""
												ResourcePayloadItems       string   "yaml:\"resource_payload_items\""
												ResourcePayloadItemLoc     string   "yaml:\"resource_payload_item_loc\""
												ResourcePayloadItemLastmod *string  "yaml:\"resource_payload_item_lastmod,omitempty\""
												Changefreq                 *string  "yaml:\"changefreq,omitempty\""
												Priority                   *float32 "yaml:\"priority,omitempty\""
											} "yaml:\"list,omitempty\""
										} "yaml:\"sitemap,omitempty\""
									} "yaml:\"routes\""
								}{},
								Index: &struct {
									HTML      string  "yaml:\"html\""
									Bundle    *string "yaml:\"bundle,omitempty\""
									Env       *string "yaml:\"env,omitempty\""
									Container *string "yaml:\"container,omitempty\""
									State     *string "yaml:\"state,omitempty\""
									Timeout   *int    "yaml:\"timeout,omitempty\""
									MaxVMs    *int    "yaml:\"max_vms,omitempty\""
									Cache     *bool   "yaml:\"cache,omitempty\""
									CacheTTL  *int    "yaml:\"cache_ttl,omitempty\""
									Rules     []struct {
										Path  string "yaml:\"path\""
										State []struct {
											Key      string "yaml:\"key\""
											Resource string "yaml:\"resource\""
											Export   *bool  "yaml:\"export\""
										} "yaml:\"state,omitempty\""
										Last *bool "yaml:\"last,omitempty\""
									} "yaml:\"rules\""
								}{},
								Default: &struct {
									File       string "yaml:\"file\""
									StatusCode *int   "yaml:\"status_code,omitempty\""
									Cache      *bool  "yaml:\"cache,omitempty\""
									CacheTTL   *int   "yaml:\"cache_ttl,omitempty\""
								}{},
							},
						},
						Fetcher: yamlConfigFetcher{},
						Loader:  &yamlConfigLoader{},
					}

					return nil
				},
			},
			args: args{
				data: []byte{},
				c:    &config{},
			},
		},
		{
			name: "full",
			fields: fields{
				yamlUnmarshal: func(in []byte, out interface{}) error {
					*out.(*yamlConfig) = yamlConfig{
						Server: []yamlConfigServer{
							{
								ListenAddr:    stringPtr("locahost"),
								ListenPort:    intPtr(8080),
								TLS:           boolPtr(true),
								TLSCAFile:     stringPtr("ca.pem"),
								TLSCertFile:   stringPtr("cert.pem"),
								TLSKeyFile:    stringPtr("key.pem"),
								ReadTimeout:   intPtr(60),
								WriteTimeout:  intPtr(60),
								Compress:      intPtr(1),
								AccessLog:     boolPtr(true),
								AccessLogFile: stringPtr("access.log"),
								Rewrite: &struct {
									Rules []struct {
										Path        string  "yaml:\"path\""
										Replacement string  "yaml:\"replacement\""
										Flag        *string "yaml:\"flag,omitempty\""
										Last        *bool   "yaml:\"last,omitempty\""
									} "yaml:\"rules\""
								}{
									Rules: []struct {
										Path        string  "yaml:\"path\""
										Replacement string  "yaml:\"replacement\""
										Flag        *string "yaml:\"flag,omitempty\""
										Last        *bool   "yaml:\"last,omitempty\""
									}{
										{
											Path:        "/^test/?",
											Replacement: "/replacement",
											Flag:        stringPtr("redirect"),
											Last:        boolPtr(true),
										},
									},
								},
								Header: &struct {
									Rules []struct {
										Path   string            "yaml:\"path\""
										Set    map[string]string "yaml:\"set,omitempty\""
										Add    map[string]string "yaml:\"add,omitempty\""
										Remove []string          "yaml:\"remove,omitempty\""
										Last   *bool             "yaml:\"last,omitempty\""
									} "yaml:\"rules\""
								}{
									Rules: []struct {
										Path   string            "yaml:\"path\""
										Set    map[string]string "yaml:\"set,omitempty\""
										Add    map[string]string "yaml:\"add,omitempty\""
										Remove []string          "yaml:\"remove,omitempty\""
										Last   *bool             "yaml:\"last,omitempty\""
									}{
										{
											Path: "/.*",
											Set: map[string]string{
												"header1": "test1",
											},
											Remove: []string{"header3"},
										},
										{
											Path: "/",
											Add: map[string]string{
												"header2": "test2",
											},
										},
										{
											Path:   "/",
											Remove: []string{"header3"},
											Last:   boolPtr(true),
										},
									},
								},
								Static: &struct {
									Dir   string "yaml:\"dir\""
									Index *bool  "yaml:\"index,omitempty\""
								}{
									Dir:   "/data/static",
									Index: boolPtr(false),
								},
								Robots: &struct {
									Path     *string  "yaml:\"path,omitempty\""
									Hosts    []string "yaml:\"hosts,omitempty\""
									Sitemaps []string "yaml:\"sitemaps,omitempty\""
									Cache    *bool    "yaml:\"cache,omitempty\""
									CacheTTL *int     "yaml:\"cache_ttl,omitempty\""
								}{
									Path:     stringPtr("/robots.txt"),
									Hosts:    []string{"localhost"},
									Sitemaps: []string{"http://localhost/sitemap.xml"},
									Cache:    boolPtr(false),
									CacheTTL: intPtr(60),
								},
								Sitemap: &struct {
									Root     string "yaml:\"root\""
									Cache    *bool  "yaml:\"cache,omitempty\""
									CacheTTL *int   "yaml:\"cache_ttl,omitempty\""
									Routes   []struct {
										Path         string "yaml:\"path\""
										Kind         string "yaml:\"kind\""
										SitemapIndex []struct {
											Name   string "yaml:\"name\""
											Type   string "yaml:\"type\""
											Static struct {
												Loc string "yaml:\"loc\""
											} "yaml:\"static,omitempty\""
										} "yaml:\"sitemap_index,omitempty\""
										Sitemap []struct {
											Name   string "yaml:\"name\""
											Type   string "yaml:\"type\""
											Static struct {
												Loc        string   "yaml:\"loc\""
												Lastmod    *string  "yaml:\"lastmod,omitempty\""
												Changefreq *string  "yaml:\"changefreq,omitempty\""
												Priority   *float32 "yaml:\"priority,omitempty\""
											} "yaml:\"static,omitempty\""
											List struct {
												Resource                   string   "yaml:\"resource\""
												ResourcePayloadItems       string   "yaml:\"resource_payload_items\""
												ResourcePayloadItemLoc     string   "yaml:\"resource_payload_item_loc\""
												ResourcePayloadItemLastmod *string  "yaml:\"resource_payload_item_lastmod,omitempty\""
												Changefreq                 *string  "yaml:\"changefreq,omitempty\""
												Priority                   *float32 "yaml:\"priority,omitempty\""
											} "yaml:\"list,omitempty\""
										} "yaml:\"sitemap,omitempty\""
									} "yaml:\"routes\""
								}{
									Root:     "http://localhost",
									Cache:    boolPtr(false),
									CacheTTL: intPtr(60),
									Routes: []struct {
										Path         string "yaml:\"path\""
										Kind         string "yaml:\"kind\""
										SitemapIndex []struct {
											Name   string "yaml:\"name\""
											Type   string "yaml:\"type\""
											Static struct {
												Loc string "yaml:\"loc\""
											} "yaml:\"static,omitempty\""
										} "yaml:\"sitemap_index,omitempty\""
										Sitemap []struct {
											Name   string "yaml:\"name\""
											Type   string "yaml:\"type\""
											Static struct {
												Loc        string   "yaml:\"loc\""
												Lastmod    *string  "yaml:\"lastmod,omitempty\""
												Changefreq *string  "yaml:\"changefreq,omitempty\""
												Priority   *float32 "yaml:\"priority,omitempty\""
											} "yaml:\"static,omitempty\""
											List struct {
												Resource                   string   "yaml:\"resource\""
												ResourcePayloadItems       string   "yaml:\"resource_payload_items\""
												ResourcePayloadItemLoc     string   "yaml:\"resource_payload_item_loc\""
												ResourcePayloadItemLastmod *string  "yaml:\"resource_payload_item_lastmod,omitempty\""
												Changefreq                 *string  "yaml:\"changefreq,omitempty\""
												Priority                   *float32 "yaml:\"priority,omitempty\""
											} "yaml:\"list,omitempty\""
										} "yaml:\"sitemap,omitempty\""
									}{
										{
											Path: "/sitemap.xml",
											Kind: "sitemapindex",
											SitemapIndex: []struct {
												Name   string "yaml:\"name\""
												Type   string "yaml:\"type\""
												Static struct {
													Loc string "yaml:\"loc\""
												} "yaml:\"static,omitempty\""
											}{
												{
													Name: "test",
													Type: "static",
													Static: struct {
														Loc string "yaml:\"loc\""
													}{
														Loc: "/sitemap2.xml",
													},
												},
											},
										},
										{
											Path: "/sitemap2.xml",
											Kind: "sitemap",
											Sitemap: []struct {
												Name   string "yaml:\"name\""
												Type   string "yaml:\"type\""
												Static struct {
													Loc        string   "yaml:\"loc\""
													Lastmod    *string  "yaml:\"lastmod,omitempty\""
													Changefreq *string  "yaml:\"changefreq,omitempty\""
													Priority   *float32 "yaml:\"priority,omitempty\""
												} "yaml:\"static,omitempty\""
												List struct {
													Resource                   string   "yaml:\"resource\""
													ResourcePayloadItems       string   "yaml:\"resource_payload_items\""
													ResourcePayloadItemLoc     string   "yaml:\"resource_payload_item_loc\""
													ResourcePayloadItemLastmod *string  "yaml:\"resource_payload_item_lastmod,omitempty\""
													Changefreq                 *string  "yaml:\"changefreq,omitempty\""
													Priority                   *float32 "yaml:\"priority,omitempty\""
												} "yaml:\"list,omitempty\""
											}{
												{
													Name: "test1",
													Type: "static",
													Static: struct {
														Loc        string   "yaml:\"loc\""
														Lastmod    *string  "yaml:\"lastmod,omitempty\""
														Changefreq *string  "yaml:\"changefreq,omitempty\""
														Priority   *float32 "yaml:\"priority,omitempty\""
													}{
														Loc:        "/",
														Lastmod:    stringPtr("2022-01-01"),
														Changefreq: stringPtr("daily"),
														Priority:   floatPtr(0.5),
													},
												},
												{
													Name: "test2",
													Type: "list",
													List: struct {
														Resource                   string   "yaml:\"resource\""
														ResourcePayloadItems       string   "yaml:\"resource_payload_items\""
														ResourcePayloadItemLoc     string   "yaml:\"resource_payload_item_loc\""
														ResourcePayloadItemLastmod *string  "yaml:\"resource_payload_item_lastmod,omitempty\""
														Changefreq                 *string  "yaml:\"changefreq,omitempty\""
														Priority                   *float32 "yaml:\"priority,omitempty\""
													}{
														Resource:                   "resource",
														ResourcePayloadItems:       "data",
														ResourcePayloadItemLoc:     "loc",
														ResourcePayloadItemLastmod: stringPtr("lastmod"),
														Changefreq:                 stringPtr("daily"),
														Priority:                   floatPtr(0.5),
													},
												},
											},
										},
									},
								},
								Index: &struct {
									HTML      string  "yaml:\"html\""
									Bundle    *string "yaml:\"bundle,omitempty\""
									Env       *string "yaml:\"env,omitempty\""
									Container *string "yaml:\"container,omitempty\""
									State     *string "yaml:\"state,omitempty\""
									Timeout   *int    "yaml:\"timeout,omitempty\""
									MaxVMs    *int    "yaml:\"max_vms,omitempty\""
									Cache     *bool   "yaml:\"cache,omitempty\""
									CacheTTL  *int    "yaml:\"cache_ttl,omitempty\""
									Rules     []struct {
										Path  string "yaml:\"path\""
										State []struct {
											Key      string "yaml:\"key\""
											Resource string "yaml:\"resource\""
											Export   *bool  "yaml:\"export\""
										} "yaml:\"state,omitempty\""
										Last *bool "yaml:\"last,omitempty\""
									} "yaml:\"rules\""
								}{
									HTML:      "data/index.html",
									Bundle:    stringPtr("data/bundle.js"),
									Env:       stringPtr("test"),
									Container: stringPtr("root"),
									State:     stringPtr("state"),
									MaxVMs:    intPtr(1),
									Timeout:   intPtr(0),
									Cache:     boolPtr(false),
									CacheTTL:  intPtr(60),
									Rules: []struct {
										Path  string "yaml:\"path\""
										State []struct {
											Key      string "yaml:\"key\""
											Resource string "yaml:\"resource\""
											Export   *bool  "yaml:\"export\""
										} "yaml:\"state,omitempty\""
										Last *bool "yaml:\"last,omitempty\""
									}{
										{
											Path: "/.*",
											State: []struct {
												Key      string "yaml:\"key\""
												Resource string "yaml:\"resource\""
												Export   *bool  "yaml:\"export\""
											}{
												{
													Key:      "key1",
													Resource: "resource1",
													Export:   boolPtr(false),
												},
											},
										},
										{
											Path: "/",
											State: []struct {
												Key      string "yaml:\"key\""
												Resource string "yaml:\"resource\""
												Export   *bool  "yaml:\"export\""
											}{
												{
													Key:      "key2",
													Resource: "resource2",
													Export:   boolPtr(false),
												},
											},
											Last: boolPtr(true),
										},
									},
								},
								Default: &struct {
									File       string "yaml:\"file\""
									StatusCode *int   "yaml:\"status_code,omitempty\""
									Cache      *bool  "yaml:\"cache,omitempty\""
									CacheTTL   *int   "yaml:\"cache_ttl,omitempty\""
								}{
									File:       "data/default.html",
									StatusCode: intPtr(200),
									Cache:      boolPtr(false),
									CacheTTL:   intPtr(60),
								},
							},
						},
						Fetcher: yamlConfigFetcher{
							RequestTLSCAFile:   stringPtr("ca.pem"),
							RequestTLSCertFile: stringPtr("cert.pem"),
							RequestTLSKeyFile:  stringPtr("key.pem"),
							RequestTimeout:     intPtr(10),
							RequestRetry:       intPtr(3),
							RequestDelay:       intPtr(4),
							Resources: []struct {
								Name    string            "yaml:\"name\""
								Method  string            "yaml:\"method\""
								URL     string            "yaml:\"url\""
								Params  map[string]string "yaml:\"params,omitempty\""
								Headers map[string]string "yaml:\"headers,omitempty\""
							}{
								{
									Name:   "resource",
									Method: "GET",
									URL:    "http://external",
									Params: map[string]string{
										"param": "value",
									},
									Headers: map[string]string{
										"header": "value",
									},
								},
							},
							Templates: []struct {
								Name    string            "yaml:\"name\""
								Method  string            "yaml:\"method\""
								URL     string            "yaml:\"url\""
								Params  map[string]string "yaml:\"params,omitempty\""
								Headers map[string]string "yaml:\"headers,omitempty\""
							}{
								{
									Name:   "template",
									Method: "GET",
									URL:    "http://external",
									Params: map[string]string{
										"param": "value",
									},
									Headers: map[string]string{
										"header": "value",
									},
								},
							},
						},
						Loader: &yamlConfigLoader{
							ExecStartup:  intPtr(15),
							ExecInterval: intPtr(300),
							ExecWorkers:  intPtr(1),
							Rules: []struct {
								Name   string "yaml:\"name\""
								Type   string "yaml:\"type\""
								Static struct {
									Resource string "yaml:\"resource\""
								} "yaml:\"static,omitempty\""
								Single struct {
									Resource                    string            "yaml:\"resource\""
									ResourcePayloadItem         string            "yaml:\"resource_payload_item\""
									ItemTemplate                string            "yaml:\"item_template\""
									ItemTemplateResource        string            "yaml:\"item_template_resource\""
									ItemTemplateResourceParams  map[string]string "yaml:\"item_template_resource_params,omitempty\""
									ItemTemplateResourceHeaders map[string]string "yaml:\"item_template_resource_headers,omitempty\""
								} "yaml:\"single,omitempty\""
								List struct {
									Resource                    string            "yaml:\"resource\""
									ResourcePayloadItems        string            "yaml:\"resource_payload_items\""
									ItemTemplate                string            "yaml:\"item_template\""
									ItemTemplateResource        string            "yaml:\"item_template_resource\""
									ItemTemplateResourceParams  map[string]string "yaml:\"item_template_resource_params,omitempty\""
									ItemTemplateResourceHeaders map[string]string "yaml:\"item_template_resource_headers,omitempty\""
								} "yaml:\"list,omitempty\""
							}{
								{
									Name: "static",
									Type: "static",
									Static: struct {
										Resource string "yaml:\"resource\""
									}{
										Resource: "resource",
									},
								},
								{
									Name: "single",
									Type: "single",
									Single: struct {
										Resource                    string            "yaml:\"resource\""
										ResourcePayloadItem         string            "yaml:\"resource_payload_item\""
										ItemTemplate                string            "yaml:\"item_template\""
										ItemTemplateResource        string            "yaml:\"item_template_resource\""
										ItemTemplateResourceParams  map[string]string "yaml:\"item_template_resource_params,omitempty\""
										ItemTemplateResourceHeaders map[string]string "yaml:\"item_template_resource_headers,omitempty\""
									}{
										Resource:             "resource",
										ResourcePayloadItem:  "data",
										ItemTemplate:         "template",
										ItemTemplateResource: "new-resource",
										ItemTemplateResourceParams: map[string]string{
											"param": "value",
										},
										ItemTemplateResourceHeaders: map[string]string{
											"header": "value",
										},
									},
								},
								{
									Name: "list",
									Type: "list",
									List: struct {
										Resource                    string            "yaml:\"resource\""
										ResourcePayloadItems        string            "yaml:\"resource_payload_items\""
										ItemTemplate                string            "yaml:\"item_template\""
										ItemTemplateResource        string            "yaml:\"item_template_resource\""
										ItemTemplateResourceParams  map[string]string "yaml:\"item_template_resource_params,omitempty\""
										ItemTemplateResourceHeaders map[string]string "yaml:\"item_template_resource_headers,omitempty\""
									}{
										Resource:             "resource",
										ResourcePayloadItems: "data",
										ItemTemplate:         "template",
										ItemTemplateResource: "new-resource",
										ItemTemplateResourceParams: map[string]string{
											"param": "value",
										},
										ItemTemplateResourceHeaders: map[string]string{
											"header": "value",
										},
									},
								},
							},
						},
					}

					return nil
				},
			},
			args: args{
				data: []byte{},
				c:    &config{},
			},
		},
		{
			name: "error yaml unmarshal",
			fields: fields{
				yamlUnmarshal: func(in []byte, out interface{}) error {
					return errors.New("test error")
				},
			},
			args: args{
				data: []byte{},
				c:    &config{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &configParserYAML{
				yamlUnmarshal: tt.fields.yamlUnmarshal,
			}
			if err := p.parse(tt.args.data, tt.args.c); (err != nil) != tt.wantErr {
				t.Errorf("configParserYAML.parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigParserYAMLParse_Env(t *testing.T) {
	defaultListenAddr := LISTEN_ADDR
	defaultListenPort := LISTEN_PORT

	LISTEN_ADDR = "192.168.0.1"
	LISTEN_PORT = 8081

	defer func() {
		LISTEN_ADDR = defaultListenAddr
		LISTEN_PORT = defaultListenPort
	}()

	type fields struct {
		yamlUnmarshal func(in []byte, out interface{}) error
	}
	type args struct {
		data []byte
		c    *config
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				yamlUnmarshal: func(in []byte, out interface{}) error {
					*out.(*yamlConfig) = yamlConfig{
						Server: []yamlConfigServer{
							{
								Rewrite: &struct {
									Rules []struct {
										Path        string  "yaml:\"path\""
										Replacement string  "yaml:\"replacement\""
										Flag        *string "yaml:\"flag,omitempty\""
										Last        *bool   "yaml:\"last,omitempty\""
									} "yaml:\"rules\""
								}{},
								Header: &struct {
									Rules []struct {
										Path   string            "yaml:\"path\""
										Set    map[string]string "yaml:\"set,omitempty\""
										Add    map[string]string "yaml:\"add,omitempty\""
										Remove []string          "yaml:\"remove,omitempty\""
										Last   *bool             "yaml:\"last,omitempty\""
									} "yaml:\"rules\""
								}{},
								Static: &struct {
									Dir   string "yaml:\"dir\""
									Index *bool  "yaml:\"index,omitempty\""
								}{},
								Robots: &struct {
									Path     *string  "yaml:\"path,omitempty\""
									Hosts    []string "yaml:\"hosts,omitempty\""
									Sitemaps []string "yaml:\"sitemaps,omitempty\""
									Cache    *bool    "yaml:\"cache,omitempty\""
									CacheTTL *int     "yaml:\"cache_ttl,omitempty\""
								}{},
								Sitemap: &struct {
									Root     string "yaml:\"root\""
									Cache    *bool  "yaml:\"cache,omitempty\""
									CacheTTL *int   "yaml:\"cache_ttl,omitempty\""
									Routes   []struct {
										Path         string "yaml:\"path\""
										Kind         string "yaml:\"kind\""
										SitemapIndex []struct {
											Name   string "yaml:\"name\""
											Type   string "yaml:\"type\""
											Static struct {
												Loc string "yaml:\"loc\""
											} "yaml:\"static,omitempty\""
										} "yaml:\"sitemap_index,omitempty\""
										Sitemap []struct {
											Name   string "yaml:\"name\""
											Type   string "yaml:\"type\""
											Static struct {
												Loc        string   "yaml:\"loc\""
												Lastmod    *string  "yaml:\"lastmod,omitempty\""
												Changefreq *string  "yaml:\"changefreq,omitempty\""
												Priority   *float32 "yaml:\"priority,omitempty\""
											} "yaml:\"static,omitempty\""
											List struct {
												Resource                   string   "yaml:\"resource\""
												ResourcePayloadItems       string   "yaml:\"resource_payload_items\""
												ResourcePayloadItemLoc     string   "yaml:\"resource_payload_item_loc\""
												ResourcePayloadItemLastmod *string  "yaml:\"resource_payload_item_lastmod,omitempty\""
												Changefreq                 *string  "yaml:\"changefreq,omitempty\""
												Priority                   *float32 "yaml:\"priority,omitempty\""
											} "yaml:\"list,omitempty\""
										} "yaml:\"sitemap,omitempty\""
									} "yaml:\"routes\""
								}{},
								Index: &struct {
									HTML      string  "yaml:\"html\""
									Bundle    *string "yaml:\"bundle,omitempty\""
									Env       *string "yaml:\"env,omitempty\""
									Container *string "yaml:\"container,omitempty\""
									State     *string "yaml:\"state,omitempty\""
									Timeout   *int    "yaml:\"timeout,omitempty\""
									MaxVMs    *int    "yaml:\"max_vms,omitempty\""
									Cache     *bool   "yaml:\"cache,omitempty\""
									CacheTTL  *int    "yaml:\"cache_ttl,omitempty\""
									Rules     []struct {
										Path  string "yaml:\"path\""
										State []struct {
											Key      string "yaml:\"key\""
											Resource string "yaml:\"resource\""
											Export   *bool  "yaml:\"export\""
										} "yaml:\"state,omitempty\""
										Last *bool "yaml:\"last,omitempty\""
									} "yaml:\"rules\""
								}{},
								Default: &struct {
									File       string "yaml:\"file\""
									StatusCode *int   "yaml:\"status_code,omitempty\""
									Cache      *bool  "yaml:\"cache,omitempty\""
									CacheTTL   *int   "yaml:\"cache_ttl,omitempty\""
								}{},
							},
						},
						Fetcher: yamlConfigFetcher{},
						Loader:  &yamlConfigLoader{},
					}

					return nil
				},
			},
			args: args{
				data: []byte{},
				c:    &config{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &configParserYAML{
				yamlUnmarshal: tt.fields.yamlUnmarshal,
			}
			if err := p.parse(tt.args.data, tt.args.c); (err != nil) != tt.wantErr {
				t.Errorf("configParserYAML.parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckConfig(t *testing.T) {
	type args struct {
		c *config
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "minimal",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{},
						},
					},
				},
			},
		},
		{
			name: "error server missing",
			args: args{
				c: &config{},
			},
			want: []string{
				"server: at least one server must be defined",
			},
			wantErr: true,
		},
		{
			name: "error server missing TLS options",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							TLS:      true,
							Renderer: &ServerRendererConfig{},
						},
					},
				},
			},
			want: []string{
				"server: option 'tls_cert_file', missing option",
				"server: option 'tls_key_file', missing option",
			},
			wantErr: true,
		},
		{
			name: "error server TLS invalid values",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							TLS:         true,
							TLSCAFile:   stringPtr(""),
							TLSCertFile: stringPtr(""),
							TLSKeyFile:  stringPtr(""),
							Renderer:    &ServerRendererConfig{},
						},
					},
				},
			},
			want: []string{
				"server: option 'tls_ca_file', invalid/missing value",
				"server: option 'tls_cert_file', invalid/missing value",
				"server: option 'tls_key_file', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "server open TLS files",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							TLS:         true,
							TLSCAFile:   stringPtr("ca.pem"),
							TLSCertFile: stringPtr("cert.pem"),
							TLSKeyFile:  stringPtr("key.pem"),
							Renderer:    &ServerRendererConfig{},
						},
					},
					osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
						return os.Open(os.DevNull)
					},
				},
			},
		},
		{
			name: "error server failed to open TLS files",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							TLS:         true,
							TLSCAFile:   stringPtr("ca.pem"),
							TLSCertFile: stringPtr("cert.pem"),
							TLSKeyFile:  stringPtr("key.pem"),
							Renderer:    &ServerRendererConfig{},
						},
					},
					osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
						return nil, errors.New("test error")
					},
				},
			},
			want: []string{
				"server: option 'tls_ca_file', failed to open file",
				"server: option 'tls_cert_file', failed to open file",
				"server: option 'tls_key_file', failed to open file",
			},
			wantErr: true,
		},
		{
			name: "error server invalid timeout",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							ReadTimeout:  -1,
							WriteTimeout: -1,
							Renderer:     &ServerRendererConfig{},
						},
					},
				},
			},
			want: []string{
				"server: option 'read_timeout', invalid/missing value",
				"server: option 'write_timeout', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "server compress",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Compress: 1,
							Renderer: &ServerRendererConfig{},
						},
					},
				},
			},
		},
		{
			name: "error server compress invalid value",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Compress: 10,
							Renderer: &ServerRendererConfig{},
						},
					},
				},
			},
			want: []string{
				"server: option 'compress', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "error server access log file empty",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							AccessLog:     true,
							AccessLogFile: stringPtr(""),
							Renderer:      &ServerRendererConfig{},
						},
					},
					osStat: func(name string) (fs.FileInfo, error) {
						return testConfigFileInfo{}, nil
					},
				},
			},
			want: []string{
				"server: option 'access_log_file', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "server access log file open",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							AccessLog:     true,
							AccessLogFile: stringPtr("access.log"),
							Renderer:      &ServerRendererConfig{},
						},
					},
					osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
						return os.Open(os.DevNull)
					},
				},
			},
		},
		{
			name: "error server access log file failed to open",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							AccessLog:     true,
							AccessLogFile: stringPtr("access.log"),
							Renderer:      &ServerRendererConfig{},
						},
					},
					osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
						return nil, errors.New("test error")
					},
				},
			},
			want: []string{
				"server: option 'access_log_file', failed to open file",
			},
			wantErr: true,
		},
		{
			name: "rewrite renderer",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Rewrite: &RewriteRendererConfig{},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
				},
			},
		},
		{
			name: "rewrite renderer rule required options",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Rewrite: &RewriteRendererConfig{
									Rules: []RewriteRule{
										{},
									},
								},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
				},
			},
			want: []string{
				"rewrite: rule option 'path', invalid/missing value",
				"rewrite: rule option 'replacement', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "rewrite renderer rule invalid path",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Rewrite: &RewriteRendererConfig{
									Rules: []RewriteRule{
										{
											Path:        "(",
											Replacement: "/test",
										},
									},
								},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
				},
			},
			want: []string{
				"rewrite: rule option 'path', invalid regular expression",
			},
			wantErr: true,
		},
		{
			name: "header renderer",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Header: &HeaderRendererConfig{},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
				},
			},
		},
		{
			name: "rewrite renderer rule required options",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Header: &HeaderRendererConfig{
									Rules: []HeaderRule{
										{},
									},
								},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
				},
			},
			want: []string{
				"header: rule option 'path', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "rewrite renderer rule invalid path",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Header: &HeaderRendererConfig{
									Rules: []HeaderRule{
										{
											Path: "(",
										},
									},
								},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
				},
			},
			want: []string{
				"header: rule option 'path', invalid regular expression",
			},
			wantErr: true,
		},
		{
			name: "static renderer",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Static: &StaticRendererConfig{
									Dir: "data/static",
								},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
					osStat: func(name string) (fs.FileInfo, error) {
						return testConfigFileInfo{isDir: true}, nil
					},
				},
			},
		},
		{
			name: "error static renderer required options",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Static: &StaticRendererConfig{},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
					osStat: func(name string) (fs.FileInfo, error) {
						return nil, nil
					},
				},
			},
			want: []string{
				"static: option 'dir', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "error static renderer invalid directory",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Static: &StaticRendererConfig{
									Dir: "file",
								},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
					osStat: func(name string) (fs.FileInfo, error) {
						return testConfigFileInfo{isDir: false}, nil
					},
				},
			},
			want: []string{
				"static: option 'dir', failed to open directory",
			},
			wantErr: true,
		},
		{
			name: "robots renderer",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Robots: &RobotsRendererConfig{
									Path:     "/robots.txt",
									Cache:    true,
									CacheTTL: 60,
								},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
				},
			},
		},
		{
			name: "error robots renderer required options",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Robots: &RobotsRendererConfig{},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
					osStat: func(name string) (fs.FileInfo, error) {
						return nil, nil
					},
				},
			},
			want: []string{
				"robots: option 'path', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "sitemap renderer",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Sitemap: &SitemapRendererConfig{
									Root:     "/sitemap.xml",
									Cache:    configDefaultServerSitemapCache,
									CacheTTL: configDefaultServerSitemapCacheTTL,
								},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
				},
			},
		},
		{
			name: "error sitemap renderer invalid values",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Sitemap: &SitemapRendererConfig{
									Root:     "",
									Cache:    true,
									CacheTTL: -1,
								},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
					osStat: func(name string) (fs.FileInfo, error) {
						return nil, nil
					},
				},
			},
			want: []string{
				"sitemap: option 'root', invalid/missing value",
				"sitemap: option 'cache_ttl', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "error sitemap renderer route required options",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Sitemap: &SitemapRendererConfig{
									Root: "/sitemap.xml",
									Routes: []SitemapRoute{
										{},
									},
								},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
					osStat: func(name string) (fs.FileInfo, error) {
						return nil, nil
					},
				},
			},
			want: []string{
				"sitemap: route option 'path', invalid/missing value",
				"sitemap: route option 'kind', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "error sitemap renderer route invalid kind",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Sitemap: &SitemapRendererConfig{
									Root: "/sitemap.xml",
									Routes: []SitemapRoute{
										{
											Path: "/sitemap.xml",
											Kind: "invalid",
										},
									},
								},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
					osStat: func(name string) (fs.FileInfo, error) {
						return nil, nil
					},
				},
			},
			want: []string{
				"sitemap: route option 'kind', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "error sitemap renderer sitemap_index entry required options",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Sitemap: &SitemapRendererConfig{
									Root: "/sitemap.xml",
									Routes: []SitemapRoute{
										{
											Path: "/sitemap.xml",
											Kind: "sitemap_index",
											SitemapIndex: []SitemapIndexEntry{
												{},
											},
										},
									},
								},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
					osStat: func(name string) (fs.FileInfo, error) {
						return nil, nil
					},
				},
			},
			want: []string{
				"sitemap: sitemap_index entry option 'name', invalid/missing value",
				"sitemap: sitemap_index entry option 'type', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "error sitemap renderer sitemap_index entry invalid type",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Sitemap: &SitemapRendererConfig{
									Root: "/sitemap.xml",
									Routes: []SitemapRoute{
										{
											Path: "/sitemap.xml",
											Kind: "sitemap_index",
											SitemapIndex: []SitemapIndexEntry{
												{
													Name: "test",
													Type: "invalid",
												},
											},
										},
									},
								},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
					osStat: func(name string) (fs.FileInfo, error) {
						return nil, nil
					},
				},
			},
			want: []string{
				"sitemap: sitemap_index entry option 'type', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "error sitemap renderer sitemap_index invalid static entry values",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Sitemap: &SitemapRendererConfig{
									Root: "/sitemap.xml",
									Routes: []SitemapRoute{
										{
											Path: "/sitemap.xml",
											Kind: "sitemap_index",
											SitemapIndex: []SitemapIndexEntry{
												{
													Name: "test",
													Type: "static",
													Static: SitemapIndexEntryStatic{
														Loc: "",
													},
												},
											},
										},
									},
								},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
					osStat: func(name string) (fs.FileInfo, error) {
						return nil, nil
					},
				},
			},
			want: []string{
				"sitemap: sitemap_index static entry option 'loc', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "error sitemap renderer sitemap entry required options",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Sitemap: &SitemapRendererConfig{
									Root: "/sitemap.xml",
									Routes: []SitemapRoute{
										{
											Path: "/sitemap.xml",
											Kind: "sitemap",
											Sitemap: []SitemapEntry{
												{},
											},
										},
									},
								},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
					osStat: func(name string) (fs.FileInfo, error) {
						return nil, nil
					},
				},
			},
			want: []string{
				"sitemap: sitemap entry option 'name', invalid/missing value",
				"sitemap: sitemap entry option 'type', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "error sitemap renderer sitemap entry invalid type",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Sitemap: &SitemapRendererConfig{
									Root: "/sitemap.xml",
									Routes: []SitemapRoute{
										{
											Path: "/sitemap.xml",
											Kind: "sitemap",
											Sitemap: []SitemapEntry{
												{
													Name: "test",
													Type: "invalid",
												},
											},
										},
									},
								},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
					osStat: func(name string) (fs.FileInfo, error) {
						return nil, nil
					},
				},
			},
			want: []string{
				"sitemap: sitemap entry option 'type', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "sitemap renderer sitemap static entry options",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Sitemap: &SitemapRendererConfig{
									Root: "/sitemap.xml",
									Routes: []SitemapRoute{
										{
											Path: "/sitemap.xml",
											Kind: "sitemap",
											Sitemap: []SitemapEntry{
												{
													Name: "test",
													Type: "static",
													Static: SitemapEntryStatic{
														Loc:        "http://localhost/",
														Changefreq: stringPtr("always"),
														Priority:   floatPtr(0.5),
													},
												},
											},
										},
									},
								},
							},
						},
					},
					Loader: &LoaderConfig{},
					osStat: func(name string) (fs.FileInfo, error) {
						return nil, nil
					},
				},
			},
		},
		{
			name: "error sitemap renderer sitemap invalid static entry values",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Sitemap: &SitemapRendererConfig{
									Root: "/sitemap.xml",
									Routes: []SitemapRoute{
										{
											Path: "/sitemap.xml",
											Kind: "sitemap",
											Sitemap: []SitemapEntry{
												{
													Name: "test",
													Type: "static",
													Static: SitemapEntryStatic{
														Loc:        "",
														Lastmod:    stringPtr(""),
														Changefreq: stringPtr("invalid"),
														Priority:   floatPtr(-1.0),
													},
												},
											},
										},
									},
								},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
					osStat: func(name string) (fs.FileInfo, error) {
						return nil, nil
					},
				},
			},
			want: []string{
				"sitemap: sitemap static entry option 'loc', invalid/missing value",
				"sitemap: sitemap static entry option 'lastmod', invalid/missing value",
				"sitemap: sitemap static entry option 'changefreq', invalid/missing value",
				"sitemap: sitemap static entry option 'priority', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "sitemap renderer sitemap list entry options",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Sitemap: &SitemapRendererConfig{
									Root: "/sitemap.xml",
									Routes: []SitemapRoute{
										{
											Path: "/sitemap.xml",
											Kind: "sitemap",
											Sitemap: []SitemapEntry{
												{
													Name: "test",
													Type: "list",
													List: SitemapEntryList{
														Resource:               "test",
														ResourcePayloadItems:   "data",
														ResourcePayloadItemLoc: "loc",
														Changefreq:             stringPtr("always"),
														Priority:               floatPtr(0.5),
													},
												},
											},
										},
									},
								},
							},
						},
					},
					Fetcher: &FetcherConfig{
						Resources: []FetcherResource{
							{
								Name:   "test",
								Method: "GET",
								URL:    "http://localhost",
							},
						},
					},
					Loader: &LoaderConfig{},
					osStat: func(name string) (fs.FileInfo, error) {
						return nil, nil
					},
				},
			},
		},
		{
			name: "error sitemap renderer sitemap invalid list entry values",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Sitemap: &SitemapRendererConfig{
									Root: "/sitemap.xml",
									Routes: []SitemapRoute{
										{
											Path: "/sitemap.xml",
											Kind: "sitemap",
											Sitemap: []SitemapEntry{
												{
													Name: "test",
													Type: "list",
													List: SitemapEntryList{
														Resource:                   "",
														ResourcePayloadItems:       "",
														ResourcePayloadItemLoc:     "",
														ResourcePayloadItemLastmod: stringPtr(""),
														Changefreq:                 stringPtr("invalid"),
														Priority:                   floatPtr(-1.0),
													},
												},
											},
										},
									},
								},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
					osStat: func(name string) (fs.FileInfo, error) {
						return nil, nil
					},
				},
			},
			want: []string{
				"sitemap: sitemap list entry option 'resource', invalid/missing value",
				"sitemap: sitemap list entry option 'resource_payload_items', invalid/missing value",
				"sitemap: sitemap list entry option 'resource_payload_item_loc', invalid/missing value",
				"sitemap: sitemap list entry option 'resource_payload_item_lastmod', invalid/missing value",
				"sitemap: sitemap list entry option 'changefreq', invalid/missing value",
				"sitemap: sitemap list entry option 'priority', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "error sitemap renderer sitemap list entry resource found",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Sitemap: &SitemapRendererConfig{
									Root: "/sitemap.xml",
									Routes: []SitemapRoute{
										{
											Path: "/sitemap.xml",
											Kind: "sitemap",
											Sitemap: []SitemapEntry{
												{
													Name: "test",
													Type: "list",
													List: SitemapEntryList{
														Resource:               "test",
														ResourcePayloadItems:   "data",
														ResourcePayloadItemLoc: "loc",
													},
												},
											},
										},
									},
								},
							},
						},
					},
					Fetcher: &FetcherConfig{
						Resources: []FetcherResource{
							{
								Name:   "test",
								Method: "GET",
								URL:    "http://localhost",
							},
						},
					},
					Loader: &LoaderConfig{},
					osStat: func(name string) (fs.FileInfo, error) {
						return nil, nil
					},
				},
			},
		},
		{
			name: "error sitemap renderer sitemap list entry resource not found",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Sitemap: &SitemapRendererConfig{
									Root: "/sitemap.xml",
									Routes: []SitemapRoute{
										{
											Path: "/sitemap.xml",
											Kind: "sitemap",
											Sitemap: []SitemapEntry{
												{
													Name: "test",
													Type: "list",
													List: SitemapEntryList{
														Resource:               "test",
														ResourcePayloadItems:   "data",
														ResourcePayloadItemLoc: "loc",
													},
												},
											},
										},
									},
								},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
					osStat: func(name string) (fs.FileInfo, error) {
						return nil, nil
					},
				},
			},
			want: []string{
				"sitemap: sitemap list entry option 'resource', resource not found",
			},
			wantErr: true,
		},
		{
			name: "index renderer",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Index: &IndexRendererConfig{
									HTML:      "/data/index.html",
									Env:       configDefaultServerIndexEnv,
									Container: configDefaultServerIndexContainer,
									State:     configDefaultServerIndexState,
									Timeout:   configDefaultServerIndexTimeout,
									MaxVMs:    1,
									Cache:     configDefaultServerIndexCache,
									CacheTTL:  configDefaultServerIndexCacheTTL,
								},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
					osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
						return os.Open(os.DevNull)
					},
				},
			},
		},
		{
			name: "error index renderer required options",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Index: &IndexRendererConfig{},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
					osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
						return os.Open(os.DevNull)
					},
				},
			},
			want: []string{
				"index: option 'html', invalid/missing value",
				"index: option 'env', invalid/missing value",
				"index: option 'container', invalid/missing value",
				"index: option 'state', invalid/missing value",
				"index: option 'max_vms', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "error index renderer invalid values",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Index: &IndexRendererConfig{
									HTML:      "",
									Bundle:    stringPtr(""),
									Env:       "",
									Container: "",
									State:     "",
									Timeout:   -1,
									MaxVMs:    0,
									Cache:     true,
									CacheTTL:  -1,
								},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
					osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
						return os.Open(os.DevNull)
					},
				},
			},
			want: []string{
				"index: option 'html', invalid/missing value",
				"index: option 'bundle', invalid/missing value",
				"index: option 'env', invalid/missing value",
				"index: option 'container', invalid/missing value",
				"index: option 'state', invalid/missing value",
				"index: option 'timeout', invalid/missing value",
				"index: option 'max_vms', invalid/missing value",
				"index: option 'cache_ttl', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "error index renderer open html and bundle files",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Index: &IndexRendererConfig{
									HTML:      "data/html/",
									Bundle:    stringPtr("data/bundle/"),
									Env:       configDefaultServerIndexEnv,
									Container: configDefaultServerIndexContainer,
									State:     configDefaultServerIndexState,
									Timeout:   configDefaultServerIndexTimeout,
									MaxVMs:    1,
								},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
					osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
						return nil, errors.New("test error")
					},
				},
			},
			want: []string{
				"index: option 'html', failed to open file",
				"index: option 'bundle', failed to open file",
			},
			wantErr: true,
		},
		{
			name: "error index renderer rule required options",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Index: &IndexRendererConfig{
									HTML:      "/data/index.html",
									Bundle:    stringPtr("/data/bundle.js"),
									Env:       configDefaultServerIndexEnv,
									Container: configDefaultServerIndexContainer,
									State:     configDefaultServerIndexState,
									Timeout:   configDefaultServerIndexTimeout,
									MaxVMs:    1,
									Rules: []IndexRule{
										{
											State: []IndexRuleStateEntry{
												{},
											},
										},
									},
								},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
					osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
						return os.Open(os.DevNull)
					},
				},
			},
			want: []string{
				"index: rule option 'path', invalid/missing value",
				"index: rule state option 'key', invalid/missing value",
				"index: rule state option 'resource', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "index renderer rule invalid path",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Index: &IndexRendererConfig{
									HTML:      "/data/index.html",
									Bundle:    stringPtr("/data/bundle.js"),
									Env:       configDefaultServerIndexEnv,
									Container: configDefaultServerIndexContainer,
									State:     configDefaultServerIndexState,
									Timeout:   configDefaultServerIndexTimeout,
									MaxVMs:    1,
									Rules: []IndexRule{
										{
											Path: "(",
											State: []IndexRuleStateEntry{
												{
													Key:      "test",
													Resource: "test",
												},
											},
										},
									},
								},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
					osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
						return os.Open(os.DevNull)
					},
				},
			},
			want: []string{
				"index: rule option 'path', invalid regular expression",
			},
			wantErr: true,
		},
		{
			name: "default renderer",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Default: &DefaultRendererConfig{
									File:       "/data/default.html",
									StatusCode: configDefaultServerDefaultStatusCode,
								},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
					osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
						return os.Open(os.DevNull)
					},
				},
			},
		},
		{
			name: "error default renderer required options",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Default: &DefaultRendererConfig{},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
					osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
						return os.Open(os.DevNull)
					},
				},
			},
			want: []string{
				"default: option 'file', invalid/missing value",
				"default: option 'status_code', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "error default renderer invalid values",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Default: &DefaultRendererConfig{
									File:       "",
									StatusCode: 600,
									Cache:      false,
									CacheTTL:   -1,
								},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
					osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
						return os.Open(os.DevNull)
					},
				},
			},
			want: []string{
				"default: option 'file', invalid/missing value",
				"default: option 'status_code', invalid/missing value",
				"default: option 'cache_ttl', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "error default renderer open file",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{
								Default: &DefaultRendererConfig{
									File:       "default.html",
									StatusCode: 200,
								},
							},
						},
					},
					Fetcher: &FetcherConfig{},
					Loader:  &LoaderConfig{},
					osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
						return nil, errors.New("test error")
					},
				},
			},
			want: []string{
				"default: option 'file', failed to open file",
			},
			wantErr: true,
		},
		{
			name: "fetcher",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{},
						},
					},
					Fetcher: &FetcherConfig{},
				},
			},
		},
		{
			name: "fetcher TLS options",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{},
						},
					},
					Fetcher: &FetcherConfig{},
				},
			},
		},
		{
			name: "fetcher invalid TLS values",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{},
						},
					},
					Fetcher: &FetcherConfig{
						RequestTLSCAFile:   stringPtr(""),
						RequestTLSCertFile: stringPtr(""),
						RequestTLSKeyFile:  stringPtr(""),
					},
				},
			},
			want: []string{
				"fetcher: option 'request_tls_ca_file', invalid/missing value",
				"fetcher: option 'request_tls_cert_file', invalid/missing value",
				"fetcher: option 'request_tls_key_file', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "fetcher open TLS files",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{},
						},
					},
					Fetcher: &FetcherConfig{
						RequestTLSCAFile:   stringPtr("ca.pem"),
						RequestTLSCertFile: stringPtr("cert.pem"),
						RequestTLSKeyFile:  stringPtr("key.pem"),
					},
					osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
						return os.Open(os.DevNull)
					},
				},
			},
		},
		{
			name: "fetcher failed to open TLS files",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{},
						},
					},
					Fetcher: &FetcherConfig{
						RequestTLSCAFile:   stringPtr("ca.pem"),
						RequestTLSCertFile: stringPtr("cert.pem"),
						RequestTLSKeyFile:  stringPtr("key.pem"),
					},
					osOpenFile: func(name string, flag int, perm fs.FileMode) (*os.File, error) {
						return nil, errors.New("test error")
					},
				},
			},
			want: []string{
				"fetcher: option 'request_tls_ca_file', failed to open file",
				"fetcher: option 'request_tls_cert_file', failed to open file",
				"fetcher: option 'request_tls_key_file', failed to open file",
			},
			wantErr: true,
		},
		{
			name: "error fetcher invalid values",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{},
						},
					},
					Fetcher: &FetcherConfig{
						RequestTimeout: -1,
						RequestRetry:   -1,
						RequestDelay:   -1,
					},
					osStat: func(name string) (fs.FileInfo, error) {
						return testConfigFileInfo{isDir: true}, nil
					},
				},
			},
			want: []string{
				"fetcher: option 'request_timeout', invalid/missing value",
				"fetcher: option 'request_retry', invalid/missing value",
				"fetcher: option 'request_delay', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "error fetcher resource required options",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{},
						},
					},
					Fetcher: &FetcherConfig{
						Resources: []FetcherResource{
							{},
						},
					},
					osStat: func(name string) (fs.FileInfo, error) {
						return testConfigFileInfo{isDir: true}, nil
					},
				},
			},
			want: []string{
				"fetcher: resource option 'name', invalid/missing value",
				"fetcher: resource option 'method', invalid/missing value",
				"fetcher: resource option 'url', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "error fetcher resource invalid method",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{},
						},
					},
					Fetcher: &FetcherConfig{
						Resources: []FetcherResource{
							{
								Name:   "test",
								Method: "get",
								URL:    "http://localhost",
							},
						},
					},
					osStat: func(name string) (fs.FileInfo, error) {
						return testConfigFileInfo{isDir: true}, nil
					},
				},
			},
			want: []string{
				"fetcher: resource option 'method', invalid method",
			},
			wantErr: true,
		},
		{
			name: "error fetcher resource invalid URL",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{},
						},
					},
					Fetcher: &FetcherConfig{
						Resources: []FetcherResource{
							{
								Name:   "test",
								Method: "GET",
								URL:    "localhost/\n",
							},
						},
					},
					osStat: func(name string) (fs.FileInfo, error) {
						return testConfigFileInfo{isDir: true}, nil
					},
				},
			},
			want: []string{
				"fetcher: resource option 'url', invalid URL",
			},
			wantErr: true,
		},
		{
			name: "error fetcher template required options",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{},
						},
					},
					Fetcher: &FetcherConfig{
						Templates: []FetcherTemplate{
							{},
						},
					},
					osStat: func(name string) (fs.FileInfo, error) {
						return testConfigFileInfo{isDir: true}, nil
					},
				},
			},
			want: []string{
				"fetcher: template option 'name', invalid/missing value",
				"fetcher: template option 'method', invalid/missing value",
				"fetcher: template option 'url', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "error fetcher template invalid method",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{},
						},
					},
					Fetcher: &FetcherConfig{
						Templates: []FetcherTemplate{
							{
								Name:   "test",
								Method: "get",
								URL:    "http://localhost",
							},
						},
					},
					osStat: func(name string) (fs.FileInfo, error) {
						return testConfigFileInfo{isDir: true}, nil
					},
				},
			},
			want: []string{
				"fetcher: template option 'method', invalid method",
			},
			wantErr: true,
		},
		{
			name: "error fetcher template invalid URL",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{},
						},
					},
					Fetcher: &FetcherConfig{
						Templates: []FetcherTemplate{
							{
								Name:   "test",
								Method: "GET",
								URL:    "localhost/\n",
							},
						},
					},
					osStat: func(name string) (fs.FileInfo, error) {
						return testConfigFileInfo{isDir: true}, nil
					},
				},
			},
			want: []string{
				"fetcher: template option 'url', invalid URL",
			},
			wantErr: true,
		},
		{
			name: "loader",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{},
						},
					},
					Loader: &LoaderConfig{},
				},
			},
		},
		{
			name: "error loader invalid values",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{},
						},
					},
					Loader: &LoaderConfig{
						ExecStartup:  -1,
						ExecInterval: -1,
						ExecWorkers:  -1,
					},
				},
			},
			want: []string{
				"loader: option 'exec_startup', invalid/missing value",
				"loader: option 'exec_interval', invalid/missing value",
				"loader: option 'exec_workers', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "error loader rule required options",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{},
						},
					},
					Loader: &LoaderConfig{
						Rules: []LoaderRule{
							{},
						},
					},
				},
			},
			want: []string{
				"loader: rule option 'name', invalid/missing value",
				"loader: rule option 'type', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "error loader rule invalid type",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{},
						},
					},
					Loader: &LoaderConfig{
						Rules: []LoaderRule{
							{
								Name: "test",
								Type: "invalid",
							},
						},
					},
				},
			},
			want: []string{
				"loader: rule option 'type', invalid type",
			},
			wantErr: true,
		},
		{
			name: "error loader rule invalid static entry values",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{},
						},
					},
					Loader: &LoaderConfig{
						Rules: []LoaderRule{
							{
								Name: "test",
								Type: "static",
								Static: LoaderRuleStatic{
									Resource: "",
								},
							},
						},
					},
				},
			},
			want: []string{
				"loader: static rule option 'resource', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "error loader rule invalid single entry values",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{},
						},
					},
					Loader: &LoaderConfig{
						Rules: []LoaderRule{
							{
								Name: "test",
								Type: "single",
								Single: LoaderRuleSingle{
									Resource:             "",
									ResourcePayloadItem:  "",
									ItemTemplate:         "",
									ItemTemplateResource: "",
								},
							},
						},
					},
				},
			},
			want: []string{
				"loader: single rule option 'resource', invalid/missing value",
				"loader: single rule option 'resource_payload_item', invalid/missing value",
				"loader: single rule option 'item_template', invalid/missing value",
				"loader: single rule option 'item_template_resource', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "loader rule single entry resource found",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{},
						},
					},
					Fetcher: &FetcherConfig{
						Templates: []FetcherTemplate{
							{
								Name:   "template",
								Method: "GET",
								URL:    "http://localhost",
							},
						},
					},
					Loader: &LoaderConfig{
						Rules: []LoaderRule{
							{
								Name: "test",
								Type: "single",
								Single: LoaderRuleSingle{
									Resource:             "resource",
									ResourcePayloadItem:  "data",
									ItemTemplate:         "template",
									ItemTemplateResource: "new-resource",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "error loader rule single entry resource not found",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{},
						},
					},
					Fetcher: &FetcherConfig{
						Templates: []FetcherTemplate{
							{
								Name:   "other",
								Method: "GET",
								URL:    "http://localhost",
							},
						},
					},
					Loader: &LoaderConfig{
						Rules: []LoaderRule{
							{
								Name: "test",
								Type: "single",
								Single: LoaderRuleSingle{
									Resource:             "resource",
									ResourcePayloadItem:  "data",
									ItemTemplate:         "template",
									ItemTemplateResource: "new-resource",
								},
							},
						},
					},
				},
			},
			want: []string{
				"loader: single rule option 'item_template', template not found",
			},
			wantErr: true,
		},
		{
			name: "error loader rule invalid list entry values",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{},
						},
					},
					Loader: &LoaderConfig{
						Rules: []LoaderRule{
							{
								Name: "test",
								Type: "list",
								List: LoaderRuleList{
									Resource:             "",
									ResourcePayloadItems: "",
									ItemTemplate:         "",
									ItemTemplateResource: "",
								},
							},
						},
					},
				},
			},
			want: []string{
				"loader: list rule option 'resource', invalid/missing value",
				"loader: list rule option 'resource_payload_items', invalid/missing value",
				"loader: list rule option 'item_template', invalid/missing value",
				"loader: list rule option 'item_template_resource', invalid/missing value",
			},
			wantErr: true,
		},
		{
			name: "loader rule list entry resource found",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{},
						},
					},
					Fetcher: &FetcherConfig{
						Templates: []FetcherTemplate{
							{
								Name:   "template",
								Method: "GET",
								URL:    "http://localhost",
							},
						},
					},
					Loader: &LoaderConfig{
						Rules: []LoaderRule{
							{
								Name: "test",
								Type: "list",
								List: LoaderRuleList{
									Resource:             "resource",
									ResourcePayloadItems: "data",
									ItemTemplate:         "template",
									ItemTemplateResource: "new-resource",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "error loader rule list entry resource not found",
			args: args{
				c: &config{
					Server: []*ServerConfig{
						{
							Renderer: &ServerRendererConfig{},
						},
					},
					Fetcher: &FetcherConfig{
						Templates: []FetcherTemplate{
							{
								Name:   "other",
								Method: "GET",
								URL:    "http://localhost",
							},
						},
					},
					Loader: &LoaderConfig{
						Rules: []LoaderRule{
							{
								Name: "test",
								Type: "list",
								List: LoaderRuleList{
									Resource:             "resource",
									ResourcePayloadItems: "data",
									ItemTemplate:         "template",
									ItemTemplateResource: "new-resource",
								},
							},
						},
					},
				},
			},
			want: []string{
				"loader: list rule option 'item_template', template not found",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CheckConfig(tt.args.c)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CheckConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
