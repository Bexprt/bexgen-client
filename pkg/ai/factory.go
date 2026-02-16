package ai

import (
	"context"
	"fmt"

	cohereembedding "github.com/bexprt/bexgen-client/internal/ai/cohere-embedding"
	novapro "github.com/bexprt/bexgen-client/internal/ai/nova-pro"
	"github.com/bexprt/bexgen-client/pkg/ai/types"
	"github.com/bexprt/bexgen-client/pkg/config"
)

func NewEmbedder(ctx context.Context, cfg *config.RootYAML) (types.Embedder, error) {
	if cfg.Embedding == nil {
		return nil, fmt.Errorf("embedding config not found")
	}
	if cfg.Embedding.Driver == "" {
		return nil, fmt.Errorf("embedding.driver is required")
	}
	modelID, ok := cfg.Embedding.Options["modelId"].(string)
	if !ok {
		return nil, fmt.Errorf("modelId is required")
	}

	switch cfg.Embedding.Driver {
	case "cohere":
		return cohereembedding.NewBedrockCohereEmbedder(ctx, modelID, cfg.Embedding)
	default:
		return nil, fmt.Errorf("unsupported driver: %s", cfg.Embedding.Driver)
	}
}

func NewModelClient(ctx context.Context, cfg *config.RootYAML) (types.Model, error) {
	if cfg.Model == nil {
		return nil, fmt.Errorf("embedding config not found")
	}
	if cfg.Model.Driver == "" {
		return nil, fmt.Errorf("embedding.driver is required")
	}
	modelID, ok := cfg.Model.Options["modelId"].(string)
	if !ok {
		return nil, fmt.Errorf("modelId is required")
	}
	maxTokens, ok := cfg.Model.Options["maxTokens"].(int)
	if !ok {
		return nil, fmt.Errorf("maxTokens is required")
	}
	temperature, ok := cfg.Model.Options["temperature"].(float32)
	if !ok {
		return nil, fmt.Errorf("temperature is required")
	}

	switch cfg.Model.Driver {
	case "nova-pro":
		return novapro.New(ctx, modelID, maxTokens, temperature)
	default:
		return nil, fmt.Errorf("unsupported driver: %s", cfg.Embedding.Driver)
	}
}
