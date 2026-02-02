package messaging

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/bexprt/bexgen-client/internal/messaging/kafka"
	"github.com/bexprt/bexgen-client/pkg/messaging/types"

	"google.golang.org/protobuf/proto"
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

	cfg := root.Messaging
	if cfg.Driver == "" {
		return nil, fmt.Errorf("messaging.driver is required")
	}

	normalizeDriver(cfg.Driver)

	if cfg.Options == nil {
		cfg.Options = make(map[string]any)
	}

	return &cfg, nil
}

func NewPublisher[T proto.Message](ctx context.Context, cfg *types.FactoryConfig) (types.Publisher[T], error) {
	switch normalizeDriver(cfg.Driver) {
	case "kafka":
		return kafka.NewPublisher[T](ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported driver: %s", cfg.Driver)
	}
}

func NewConsumer[T proto.Message](ctx context.Context, cfg *types.FactoryConfig) (types.Consumer[T], error) {
	switch normalizeDriver(cfg.Driver) {
	case "kafka":
		return kafka.NewConsumer[T](ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported driver: %s", cfg.Driver)
	}
}
