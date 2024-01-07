// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package neon

import (
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/BurntSushi/toml"
	"github.com/bhuisgen/neon/pkg/core"
)

// config implements the configuration.
type config struct {
	Listeners  []*configListener
	Servers    []*configServer
	Store      *configStore
	Fetcher    *configFetcher
	Loader     *configLoader
	parser     configParser
	osReadFile func(name string) ([]byte, error)
}

// configListener implements the configuration of a listener.
type configListener struct {
	Name   string
	Config map[string]interface{}
}

// configServer implements the configuration of a server.
type configServer struct {
	Name   string
	Config map[string]interface{}
}

// configStore implements the configuration of the store.
type configStore struct {
	Config map[string]interface{}
}

// configFetcher implements the configuration of the fetcher.
type configFetcher struct {
	Config map[string]interface{}
}

// configLoader implements the configuration of the loader.
type configLoader struct {
	Config map[string]interface{}
}

const (
	configDefaultFile string = "neon.yaml"
)

// configOsReadFile redirects to os.ReadFile.
func configOsReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// newConfig creates a new config.
func newConfig(parser configParser) *config {
	return &config{
		parser:     parser,
		osReadFile: configOsReadFile,
	}
}

// configParser
type configParser interface {
	parse([]byte, *config) error
}

// configParserYAML implements the YAML configuration parser.
type configParserYAML struct {
	yamlUnmarshal func(in []byte, out interface{}) error
}

type (
	// yamlConfig implements the main configuration for the parser.
	yamlConfig struct {
		Listeners map[string]map[string]interface{}
		Servers   map[string]map[string]interface{}
		Store     map[string]interface{}
		Fetcher   map[string]interface{}
		Loader    map[string]interface{}
	}
)

// newConfigParserYAML creates a new YAML config parser.
func newConfigParserYAML() *configParserYAML {
	return &configParserYAML{
		yamlUnmarshal: yaml.Unmarshal,
	}
}

// parse parses the YAML data.
func (p *configParserYAML) parse(data []byte, c *config) error {
	var y yamlConfig
	err := p.yamlUnmarshal(data, &y)
	if err != nil {
		return err
	}

	for name, listener := range y.Listeners {
		c.Listeners = append(c.Listeners, &configListener{
			Name:   name,
			Config: listener,
		})
	}
	for name, server := range y.Servers {
		c.Servers = append(c.Servers, &configServer{
			Name:   name,
			Config: server,
		})
	}
	c.Store = &configStore{
		Config: y.Store,
	}
	c.Fetcher = &configFetcher{
		Config: y.Fetcher,
	}
	c.Loader = &configLoader{
		Config: y.Loader,
	}

	return nil
}

var _ configParser = (*configParserYAML)(nil)

// configParserTOML implements the TOML configuration parser.
type configParserTOML struct {
	tomlUnmarshal func(in []byte, out interface{}) error
}

type (
	// tomlConfig implements the main configuration for the parser.
	tomlConfig struct {
		Listeners map[string]map[string]interface{}
		Servers   map[string]map[string]interface{}
		Store     map[string]interface{}
		Fetcher   map[string]interface{}
		Loader    map[string]interface{}
	}
)

// newConfigParserTOML creates a new TOML config parser.
func newConfigParserTOML() *configParserTOML {
	return &configParserTOML{
		tomlUnmarshal: toml.Unmarshal,
	}
}

// parse parses the TOML data.
func (p *configParserTOML) parse(data []byte, c *config) error {
	var t tomlConfig
	err := p.tomlUnmarshal(data, &t)
	if err != nil {
		return err
	}

	for name, listener := range t.Listeners {
		c.Listeners = append(c.Listeners, &configListener{
			Name:   name,
			Config: listener,
		})
	}
	for name, server := range t.Servers {
		c.Servers = append(c.Servers, &configServer{
			Name:   name,
			Config: server,
		})
	}
	c.Store = &configStore{
		Config: t.Store,
	}
	c.Fetcher = &configFetcher{
		Config: t.Fetcher,
	}
	c.Loader = &configLoader{
		Config: t.Loader,
	}

	return nil
}

var _ configParser = (*configParserTOML)(nil)

// configParserJSON implements the JSON configuration parser.
type configParserJSON struct {
	jsonUnmarshal func(in []byte, out interface{}) error
}

type (
	// jsonConfig implements the main configuration for the parser.
	jsonConfig struct {
		Listeners map[string]map[string]interface{}
		Servers   map[string]map[string]interface{}
		Store     map[string]interface{}
		Fetcher   map[string]interface{}
		Loader    map[string]interface{}
	}
)

// newConfigParserJSON creates a new JSON config parser.
func newConfigParserJSON() *configParserJSON {
	return &configParserJSON{
		jsonUnmarshal: json.Unmarshal,
	}
}

// parse parses the JSON data.
func (p *configParserJSON) parse(data []byte, c *config) error {
	var j jsonConfig
	err := p.jsonUnmarshal(data, &j)
	if err != nil {
		return err
	}

	for name, listener := range j.Listeners {
		c.Listeners = append(c.Listeners, &configListener{
			Name:   name,
			Config: listener,
		})
	}
	for name, server := range j.Servers {
		c.Servers = append(c.Servers, &configServer{
			Name:   name,
			Config: server,
		})
	}
	c.Store = &configStore{
		Config: j.Store,
	}
	c.Fetcher = &configFetcher{
		Config: j.Fetcher,
	}
	c.Loader = &configLoader{
		Config: j.Loader,
	}

	return nil
}

var _ configParser = (*configParserJSON)(nil)

// LoadConfig loads the configuration.
func LoadConfig() (*config, error) {
	name := configDefaultFile
	if core.CONFIG_FILE != "" {
		name = core.CONFIG_FILE
	}

	var c *config
	switch filepath.Ext(name) {
	case ".yaml":
		c = newConfig(newConfigParserYAML())
	case ".toml":
		c = newConfig(newConfigParserTOML())
	case ".json":
		c = newConfig(newConfigParserJSON())
	default:
		return nil, errors.New("invalid file extension")
	}
	data, err := c.osReadFile(name)
	if err != nil {
		return nil, err
	}
	err = c.parser.parse(data, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

//go:embed templates/init/*
var configTemplatesInit embed.FS

// GenerateConfig creates a default configuration file.
func GenerateConfig(template string) error {
	name := configDefaultFile
	if core.CONFIG_FILE != "" {
		name = core.CONFIG_FILE
	}

	_, err := os.Stat(name)
	if err == nil {
		return errors.New("configuration file already exists")
	}

	err = fs.WalkDir(configTemplatesInit, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		data, err := fs.ReadFile(configTemplatesInit, path)
		if err != nil {
			return err
		}

		dst, err := filepath.Rel(filepath.Join("templates", "init", template), path)
		if err != nil {
			return nil
		}

		err = os.MkdirAll(filepath.Dir(dst), 0755)
		if err != nil {
			return err
		}

		err = os.WriteFile(dst, data, 0644)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return errors.New("failed to copy templates files")
	}

	return nil
}
