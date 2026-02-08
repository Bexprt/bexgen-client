package embedding

import (
	"context"
	"fmt"

	"github.com/bexprt/bexgen-client/internal/embedding"
	"github.com/bexprt/bexgen-client/pkg/config"
	"github.com/bexprt/bexgen-client/pkg/embedding/types"
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
		return embedding.NewBedrockCohereEmbedder(ctx, &modelID, cfg.Embedding)
	default:
		return nil, fmt.Errorf("unsupported driver: %s", cfg.Embedding.Driver)
	}
}
