package search

import (
	"context"
	"fmt"

	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	bedrockruntime "github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/bexprt/bexgen-client/pkg/config"
)

type BedrockCohereEmbedder struct {
	client    *bedrockruntime.Client
	model     string
	dimension int
}

func NewBedrockCohereEmbedder(ctx context.Context, cfg *config.FactoryConfig) (*BedrockCohereEmbedder, error) {
	acfg, err := awsCfg.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("config not found")
	}
	embedder := &BedrockCohereEmbedder{}
	if len(cfg.Options) > 0 {
		if dimension, ok := cfg.Options["dimension"].(int); ok {
			embedder.dimension = dimension
		}
		if model, ok := cfg.Options["model"].(string); ok {
			embedder.model = model
		}
	}

	client := bedrockruntime.NewFromConfig(acfg)
	embedder.client = client
	return embedder, nil
}

func (e *BedrockCohereEmbedder) Embed(ctx context.Context, text *string) (*[]float32, error) {
	// TODO: Implement call to AWS Bedrock Cohere embedding endpoint
	return nil, fmt.Errorf("not implemented")
}

func (e *BedrockCohereEmbedder) EmbedBatch(ctx context.Context, texts *[]string) (*[][]float32, error) {
	// TODO: Implement batch call to AWS Bedrock Cohere embedding endpoint
	return nil, fmt.Errorf("not implemented")
}

func (e *BedrockCohereEmbedder) Dimension() int {
	return e.dimension
}

func (e *BedrockCohereEmbedder) Model() string {
	return e.model
}
