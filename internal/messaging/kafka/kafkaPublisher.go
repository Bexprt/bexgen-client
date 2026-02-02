package kafka

import (
	"context"
	"fmt"

	kfk "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"google.golang.org/protobuf/proto"

	"github.com/bexprt/bexgen-client/pkg/messaging/types"
	"github.com/bexprt/bexgen-client/pkg/topics"
)

type Publisher[T proto.Message] struct {
	producer *kfk.Producer
	config   *Config
	topic    string
	ctx      context.Context
	cancel   context.CancelFunc
	buffer   int
}

func NewPublisher[T proto.Message](ctx context.Context, cfg *types.FactoryConfig) (*Publisher[T], error) {
	kCfg, err := LoadConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("kafka publisher: %w", err)
	}

	cm := buildKafkaConfigMap(kCfg, ClientProducer)

	prod, err := kfk.NewProducer(cm)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	cctx, cancel := context.WithCancel(ctx)
	return &Publisher[T]{
		producer: prod,
		config:   kCfg,
		topic:    topics.NameFromType[T](),
		ctx:      cctx,
		cancel:   cancel,
		buffer:   kCfg.Producer.Buffer,
	}, nil
}

func (p *Publisher[T]) Open() (chan<- *types.Message[T], error) {
	msgChan := make(chan *types.Message[T], p.buffer)
	delivery := make(chan kfk.Event, 1000)

	go func() {
		defer close(delivery)
		for e := range p.producer.Events() {
			switch ev := e.(type) {
			case *kfk.Message:
				if ev.TopicPartition.Error != nil {
					fmt.Printf("Delivery failed: %v\n", ev.TopicPartition.Error)
				}
			case kfk.Error:
				fmt.Printf("Producer error: %v\n", ev)
			}
		}
	}()

	go func() {
		defer p.producer.Flush(30_000)
		for {
			select {
			case <-p.ctx.Done():
				return
			case m, ok := <-msgChan:
				if !ok {
					return
				}
				val, err := proto.Marshal(m.Value)
				if err != nil {
					fmt.Printf("error marshaling message: %v\n", err)
					continue
				}
				kmsg := &kfk.Message{
					TopicPartition: kfk.TopicPartition{
						Topic:     &p.topic,
						Partition: kfk.PartitionAny,
					},
					Key:   []byte(*m.Key),
					Value: val,
				}
				if err := p.producer.Produce(kmsg, delivery); err != nil {
					fmt.Printf("produce error: %v\n", err)
				}
			}
		}
	}()

	return msgChan, nil
}

func (p *Publisher[T]) Close() error {
	p.cancel()
	if p.producer != nil {
		p.producer.Close()
	}
	return nil
}
