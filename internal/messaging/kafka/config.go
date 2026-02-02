package kafka

import (
	"fmt"
	"strings"

	"github.com/bexprt/bexgen-client/pkg/messaging/types"

	kfk "github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type KafkaClientType int

const (
	ClientProducer KafkaClientType = iota
	ClientConsumer
)

type OffsetReset string

const (
	OffsetEarliest OffsetReset = "earliest"
	OffsetLatest   OffsetReset = "latest"
	OffsetNone     OffsetReset = "none"
)

type Config struct {
	BootstrapServers string
	SecurityProtocol string
	SASLMechanism    string
	SASLUsername     string
	SASLPassword     string

	Producer struct {
		LingerMs  int
		BatchSize int
		Ack       string
		Buffer    int
		Retries   int
	}

	Consumer struct {
		GroupID         string
		AutoOffsetReset string
		Buffer          int
	}
}

func LoadConfig(cfg *types.FactoryConfig) (*Config, error) {
	options := cfg.Options

	kCfg := &Config{}

	if b, ok := options["brokers"].([]any); ok {
		strs := make([]string, len(b))
		for i, v := range b {
			strs[i] = fmt.Sprintf("%v", v)
		}
		kCfg.BootstrapServers = strings.Join(strs, ",")
	} else {
		return nil, fmt.Errorf("kafka: brokers must be set in options")
	}

	if sec, ok := options["security"].(map[string]any); ok {
		if p, ok := sec["protocol"].(string); ok {
			kCfg.SecurityProtocol = p
		}
		if m, ok := sec["mechanism"].(string); ok {
			kCfg.SASLMechanism = m
		}
		if u, ok := sec["username"].(string); ok {
			kCfg.SASLUsername = u
		}
		if pw, ok := sec["password"].(string); ok {
			kCfg.SASLPassword = pw
		}
	}

	// producer
	if prod, ok := options["producer"].(map[string]any); ok {
		if l, ok := prod["linger_ms"].(int); ok {
			kCfg.Producer.LingerMs = l
		} else if l, ok := prod["linger_ms"].(float64); ok {
			kCfg.Producer.LingerMs = int(l)
		}
		if b, ok := prod["batch_size"].(int); ok {
			kCfg.Producer.BatchSize = b
		} else if b, ok := prod["batch_size"].(float64); ok {
			kCfg.Producer.BatchSize = int(b)
		}
		if r, ok := prod["retries"].(int); ok {
			kCfg.Producer.Retries = r
		} else if b, ok := prod["retries"].(float64); ok {
			kCfg.Producer.Retries = int(b)
		}
		if buf, ok := prod["buffer_size"].(int); ok {
			kCfg.Producer.Buffer = buf
		} else if buf, ok := prod["buffer_size"].(float64); ok {
			kCfg.Producer.Buffer = int(buf)
		}

		if a, ok := prod["ack"].(string); ok {
			kCfg.Producer.Ack = a
		}
	}

	// consumer
	if cons, ok := options["consumer"].(map[string]any); ok {
		if g, ok := cons["group_id"].(string); ok {
			kCfg.Consumer.GroupID = g
		}
		if a, ok := cons["auto_offset_reset"].(string); ok {
			kCfg.Consumer.AutoOffsetReset = a
		}
		if b, ok := cons["buffer_size"].(int); ok {
			kCfg.Consumer.Buffer = b
		}
	}

	return kCfg, nil
}

func buildKafkaConfigMap(cfg *Config, clientType KafkaClientType) *kfk.ConfigMap {
	cm := &kfk.ConfigMap{
		"bootstrap.servers": cfg.BootstrapServers,
	}

	set := func(key string, value any) {
		if err := cm.SetKey(key, value); err != nil {
			fmt.Printf("warning: failed to set Kafka config %s=%v: %v\n", key, value, err)
		}
	}

	if cfg.SecurityProtocol != "" {
		set("security.protocol", cfg.SecurityProtocol)
	}
	if cfg.SASLMechanism != "" {
		set("sasl.mechanism", cfg.SASLMechanism)
		set("sasl.username", cfg.SASLUsername)
		set("sasl.password", cfg.SASLPassword)
	}

	switch clientType {
	case ClientProducer:
		if cfg.Producer.Ack != "" {
			set("acks", cfg.Producer.Ack)
		}
		if cfg.Producer.Retries > 0 {
			set("retries", cfg.Producer.Retries)
		}
		if cfg.Producer.LingerMs > 0 {
			set("linger.ms", cfg.Producer.LingerMs)
		}
		if cfg.Producer.BatchSize > 0 {
			set("batch.num.messages", cfg.Producer.BatchSize)
		}

	case ClientConsumer:
		if cfg.Consumer.GroupID != "" {
			set("group.id", cfg.Consumer.GroupID)
		}
		if cfg.Consumer.AutoOffsetReset != "" {
			set("auto.offset.reset", cfg.Consumer.AutoOffsetReset)
		}
	}

	return cm
}
