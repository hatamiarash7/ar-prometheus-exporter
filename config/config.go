package config

import (
	"io"

	yaml "gopkg.in/yaml.v2"
)

// Config represents the configuration for the exporter
type Config struct {
	Token    string `yaml:"token"`
	Products struct {
		CDN    bool `yaml:"cdn,omitempty"`
		OBJECT bool `yaml:"object_storage,omitempty"`
	} `yaml:"products,omitempty"`
}

// Load reads YAML from reader and unmarshal in Config
func Load(r io.Reader) (*Config, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	c := &Config{}
	err = yaml.Unmarshal(b, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}
