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

func NewVectorIndex(ctx context.Context, cfg *config.RootYAML) (searchtypes.Index, error) {
	if cfg.Search == nil {
		return nil, fmt.Errorf("search config not found")
	}
	if cfg.Search.Driver == "" {
		return nil, fmt.Errorf("search.driver is required")
	}

	switch cfg.Search.Driver {
	case "opensearch":
		return search.NewClient(ctx, cfg.Search)
	case "elasticsearch":
		return search.NewClient(ctx, cfg.Search)
	default:
		return nil, fmt.Errorf("unsupported driver: %s", cfg.Search.Driver)
	}
}
