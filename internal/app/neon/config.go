package neon

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/BurntSushi/toml"
)

// config implements the configuration.
type config struct {
	Store      *configStore
	Fetcher    *configFetcher
	Loader     *configLoader
	Server     *configServer
	parser     configParser
	osReadFile func(name string) ([]byte, error)
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

// configServer implements the configuration of the server.
type configServer struct {
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
		Store   map[string]interface{}
		Fetcher map[string]interface{}
		Loader  map[string]interface{}
		Server  map[string]interface{}
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
	if err := p.yamlUnmarshal(data, &y); err != nil {
		return fmt.Errorf("parse yaml: %v", err)
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
	c.Server = &configServer{
		Config: y.Server,
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
		Store   map[string]interface{}
		Fetcher map[string]interface{}
		Loader  map[string]interface{}
		Server  map[string]interface{}
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
	if err := p.tomlUnmarshal(data, &t); err != nil {
		return fmt.Errorf("parse toml: %v", err)
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
	c.Server = &configServer{
		Config: t.Server,
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
		Store   map[string]interface{}
		Fetcher map[string]interface{}
		Loader  map[string]interface{}
		Server  map[string]interface{}
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
	if err := p.jsonUnmarshal(data, &j); err != nil {
		return fmt.Errorf("parse json: %v", err)
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
	c.Server = &configServer{
		Config: j.Server,
	}

	return nil
}

var _ configParser = (*configParserJSON)(nil)

// LoadConfig loads the configuration.
func LoadConfig() (*config, error) {
	name := configDefaultFile
	if CONFIG_FILE != "" {
		name = CONFIG_FILE
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
		return nil, fmt.Errorf("read file %s: %v", name, err)
	}

	if err := c.parser.parse(data, c); err != nil {
		return nil, fmt.Errorf("parse config: %v", err)
	}

	return c, nil
}

//go:embed templates/config/*
var configTemplates embed.FS

// GenerateConfig creates a default configuration file.
func GenerateConfig(syntax string, template string) error {
	var src string
	switch syntax {
	case "yaml":
		src = "neon.yaml"
	case "toml":
		src = "neon.toml"
	case "json":
		src = "neon.json"
	}
	var dst string
	if CONFIG_FILE != "" {
		dst = CONFIG_FILE
	} else {
		dst = src
	}

	_, err := os.Stat(dst)
	if err == nil {
		return fmt.Errorf("file %s already exists", dst)
	}

	data, err := configTemplates.ReadFile(path.Join("templates", "config", template, src))
	if err != nil {
		return fmt.Errorf("read file %s: %v", src, err)
	}
	if err := os.WriteFile(dst, data, 0600); err != nil {
		return fmt.Errorf("write file %s: %v", dst, err)
	}

	return nil
}
