package storage

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/bexprt/bexgen-client/internal/storage/s3"
	"github.com/bexprt/bexgen-client/pkg/storage/types"

	"gopkg.in/yaml.v3"
)

const defaultConfigPath = "/etc/bextract/config.yaml"

var driverCache = &sync.Map{}

func normalizeDriver(d string) string {
	if v, ok := driverCache.Load(d); ok {
		return v.(string)
	}
	nd := d
	driverCache.Store(d, nd)
	return nd
}

func LoadConfig(path string) (*types.FactoryConfig, error) {
	if path == "" {
		path = defaultConfigPath
	}

	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var root types.RootYAML
	if err := yaml.Unmarshal(file, &root); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	cfg := root.Storage
	if cfg.Driver == "" {
		return nil, fmt.Errorf("storage.driver is required")
	}

	normalizeDriver(cfg.Driver)

	if cfg.Options == nil {
		cfg.Options = make(map[string]any)
	}

	return &cfg, nil
}

func NewObjectStorage(ctx context.Context, cfg *types.FactoryConfig) (types.ObjectStorage, error) {
	switch normalizeDriver(cfg.Driver) {
	case "s3":
		return s3.NewClient(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported driver: %s", cfg.Driver)
	}
}
