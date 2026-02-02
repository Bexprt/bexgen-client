package config

import (
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

type FactoryConfig struct {
	Driver     string         `yaml:"driver"`
	ConfigPath string         `yaml:"-"`
	Options    map[string]any `yaml:"options"`
}

type RootYAML struct {
	Storage   *FactoryConfig `yaml:"storage"`
	Messaging *FactoryConfig `yaml:"messaging"`
}

const defaultConfigPath = "/etc/bexgen/config.yaml"

var driverCache = &sync.Map{}

func normalizeDriver(d string) string {
	if v, ok := driverCache.Load(d); ok {
		return v.(string)
	}
	nd := d
	driverCache.Store(d, nd)
	return nd
}

func LoadConfig(path string) (*RootYAML, error) {
	if path == "" {
		path = defaultConfigPath
	}

	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var root RootYAML
	if err := yaml.Unmarshal(file, &root); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	cfg := root.Storage
	if cfg.Driver == "" {
		return nil, fmt.Errorf("storage.driver is required")
	}

	normalizeDriver(root.Messaging.Driver)
	normalizeDriver(root.Storage.Driver)

	if root.Messaging.Options == nil {
		root.Messaging.Options = make(map[string]any)
	}
	if root.Storage.Options == nil {
		root.Storage.Options = make(map[string]any)
	}

	return &root, nil
}
