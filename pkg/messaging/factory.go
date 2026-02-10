package messaging

import (
	"context"
	"fmt"

	"github.com/bexprt/bexgen-client/internal/messaging/kafka"
	"github.com/bexprt/bexgen-client/pkg/config"
	"github.com/bexprt/bexgen-client/pkg/messaging/types"
	"github.com/bexprt/bexgen-client/pkg/topics"

	"google.golang.org/protobuf/proto"
)

func check(cfg *config.RootYAML) error {
	if cfg.Storage == nil {
		return fmt.Errorf("messaging config not found")
	}
	if cfg.Storage.Driver == "" {
		return fmt.Errorf("messaging.driver is required")
	}
	return nil
}

func NewPublisher[T proto.Message](ctx context.Context, cfg *config.RootYAML, topic topics.Topic[T]) (types.Publisher[T], error) {
	err := check(cfg)
	if err != nil {
		return nil, err
	}
	switch cfg.Messaging.Driver {
	case "kafka":
		return kafka.NewPublisher[T](ctx, cfg.Messaging, topic)
	default:
		return nil, fmt.Errorf("unsupported driver: %s", cfg.Messaging.Driver)
	}
}

func NewConsumer[T proto.Message](ctx context.Context, cfg *config.RootYAML, topic topics.Topic[T]) (types.Consumer[T], error) {
	err := check(cfg)
	if err != nil {
		return nil, err
	}
	switch cfg.Messaging.Driver {
	case "kafka":
		return kafka.NewConsumer[T](ctx, cfg.Messaging, topic)
	default:
		return nil, fmt.Errorf("unsupported driver: %s", cfg.Messaging.Driver)
	}
}
