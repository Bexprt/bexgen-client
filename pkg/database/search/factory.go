package search

import (
	"context"
	"fmt"

	"github.com/bexprt/bexgen-client/internal/database/search"
	"github.com/bexprt/bexgen-client/pkg/config"
	searchtypes "github.com/bexprt/bexgen-client/pkg/database/search/types"
)

func check(cfg *config.RootYAML) error {
	if cfg.Storage == nil {
		return fmt.Errorf("storage config not found")
	}
	if cfg.Storage.Driver == "" {
		return fmt.Errorf("storage.driver is required")
	}
	return nil
}

func NewVectorIndex(ctx context.Context, cfg *config.RootYAML) (searchtypes.VectorIndex, error) {
	if cfg.Search == nil {
		return nil, fmt.Errorf("search config not found")
	}
	if cfg.Search.Driver == "" {
		return nil, fmt.Errorf("search.driver is required")
	}

	switch cfg.Storage.Driver {
	case "opensearch":
		return search.NewClient(ctx, cfg.Search)
	case "elasicsearch":
		return search.NewClient(ctx, cfg.Search)
	default:
		return nil, fmt.Errorf("unsupported driver: %s", cfg.Search.Driver)
	}
}

func NewEmbedder(ctx context.Context, cfg *config.RootYAML) (searchtypes.Embedder, error) {
	if cfg.Embedding == nil {
		return nil, fmt.Errorf("embedding config not found")
	}
	if cfg.Embedding.Driver == "" {
		return nil, fmt.Errorf("embedding.driver is required")
	}

	switch cfg.Embedding.Driver {
	case "cohere":
		return search.NewBedrockCohereEmbedder(ctx, cfg.Embedding)
	default:
		return nil, fmt.Errorf("unsupported driver: %s", cfg.Embedding.Driver)
	}
}
