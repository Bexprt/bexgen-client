package coherererank

import (
	"context"
	"fmt"

	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	bedrockruntime "github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/bexprt/bexgen-client/pkg/ai/types"
	"github.com/bexprt/bexgen-client/pkg/config"
)

type CohereClient struct {
	client      *bedrockruntime.Client
	modelID     string
	MaxTokens   int
	Temperature int
}

func new(ctx context.Context, modelID string, cfg *config.FactoryConfig) (types.Rerank, error) {
	acfg, err := awsCfg.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("aws config load failed: %w", err)
	}

	rerank := &CohereClient{
		client:  bedrockruntime.NewFromConfig(acfg),
		modelID: modelID,
	}

	return rerank, nil
}

// TODO: implement Rank
func (c CohereClient) Rank(ctx context.Context, texts []string) ([]string, error) {
	return []string{}, nil
}
