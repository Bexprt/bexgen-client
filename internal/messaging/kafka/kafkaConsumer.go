package kafka

import (
	"context"
	"fmt"
	"time"

	kfk "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"google.golang.org/protobuf/proto"

	"github.com/bexprt/bexgen-client/pkg/config"
	"github.com/bexprt/bexgen-client/pkg/messaging/types"
	"github.com/bexprt/bexgen-client/pkg/topics"
)

type Consumer[T proto.Message] struct {
	consumer *kfk.Consumer
	config   *Config
	buffer   int
	topic    topics.Topic[T]
	Msg      proto.Message
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewConsumer[T proto.Message](ctx context.Context, cfg *config.FactoryConfig, topic topics.Topic[T]) (*Consumer[T], error) {
	kCfg, err := LoadConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("kafka consumer: %w", err)
	}

	cm := buildKafkaConfigMap(kCfg, ClientConsumer)

	cons, err := kfk.NewConsumer(cm)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka consumer: %w", err)
	}

	cctx, cancel := context.WithCancel(ctx)
	return &Consumer[T]{
		consumer: cons,
		config:   kCfg,
		buffer:   kCfg.Consumer.Buffer,
		topic:    topic,
		ctx:      cctx,
		cancel:   cancel,
	}, nil
}

func (c *Consumer[T]) Open() (<-chan *types.Message[T], error) {
	if err := c.consumer.SubscribeTopics([]string{c.topic.Name}, nil); err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	msgChan := make(chan *types.Message[T], c.buffer)

	go func() {
		defer close(msgChan)
		for {
			select {
			case <-c.ctx.Done():
				return
			default:
				m, err := c.consumer.ReadMessage(100 * time.Millisecond)
				if err != nil {
					if err.(kfk.Error).Code() == kfk.ErrTimedOut {
						continue
					}
					fmt.Printf("Consumer error: %v\n", err)
					return
				}

				headers := map[string]string{}
				for _, h := range m.Headers {
					headers[h.Key] = string(h.Value)
				}

				key := string(m.Key)
				value := c.topic.New()
				if err := proto.Unmarshal(m.Value, value); err != nil {
					fmt.Printf("unmarshal error: %v\n", err)
					continue
				}

				msgChan <- &types.Message[T]{
					Key:     &key,
					Value:   value,
					Headers: headers,
					Ack: func() error {
						_, err := c.consumer.CommitMessage(m)
						return err
					},
				}
			}
		}
	}()

	return msgChan, nil
}

func (c *Consumer[T]) Close() error {
	c.cancel()
	if c.consumer != nil {
		return c.consumer.Close()
	}
	return nil
}
