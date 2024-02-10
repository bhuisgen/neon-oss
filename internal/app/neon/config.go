package neon

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// config implements the configuration.
type config struct {
	parser     configParser
	data       map[string]interface{}
	osReadFile func(name string) ([]byte, error)
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

// newConfigParserYAML creates a new YAML config parser.
func newConfigParserYAML() *configParserYAML {
	return &configParserYAML{
		yamlUnmarshal: yaml.Unmarshal,
	}
}

// parse parses the YAML data.
func (p *configParserYAML) parse(data []byte, c *config) error {
	var y map[string]interface{}
	if err := p.yamlUnmarshal(data, &y); err != nil {
		return fmt.Errorf("parse yaml: %v", err)
	}

	c.data = y

	return nil
}

var _ configParser = (*configParserYAML)(nil)

// LoadConfig loads the configuration.
func LoadConfig() (*config, error) {
	name := configDefaultFile
	if v, ok := os.LookupEnv("CONFIG_FILE"); ok && v != "" {
		name = v
	}

	if filepath.Ext(name) != ".yaml" {
		return nil, errors.New("invalid file extension")
	}

	c := newConfig(newConfigParserYAML())

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
func GenerateConfig(template string) error {
	src := "neon.yaml"
	var dst string
	if v, ok := os.LookupEnv("CONFIG_FILE"); ok && v != "" {
		dst = v
	} else {
		dst = src
	}

	if _, err := os.Stat(dst); err == nil {
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
