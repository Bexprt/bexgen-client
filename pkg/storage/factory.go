package storage

import (
	"context"
	"fmt"

	"github.com/bexprt/bexgen-client/internal/storage/s3"
	"github.com/bexprt/bexgen-client/pkg/config"
	"github.com/bexprt/bexgen-client/pkg/storage/types"
)

func NewObjectStorage(ctx context.Context, cfg *config.RootYAML) (types.ObjectStorage, error) {
	if cfg.Storage == nil {
		fmt.Errorf("Storage config not found")
	}
	if cfg.Storage.Driver == "" {
		return nil, fmt.Errorf("storage.driver is required")
	}

	switch cfg.Storage.Driver {
	case "s3":
		return s3.NewClient(ctx, cfg.Storage)
	default:
		return nil, fmt.Errorf("unsupported driver: %s", cfg.Storage.Driver)
	}
}
