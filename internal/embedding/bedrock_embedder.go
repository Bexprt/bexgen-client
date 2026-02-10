package embedding

import (
	"context"
	"encoding/json"
	"fmt"

	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	bedrockruntime "github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/bexprt/bexgen-client/pkg/config"
	"github.com/bexprt/bexgen-client/pkg/embedding/types"
)

type EmbeddingType string

const (
	EmbeddingFloat  EmbeddingType = "float"
	EmbeddingInt8   EmbeddingType = "int8"
	EmbeddingUint8  EmbeddingType = "uint8"
	EmbeddingBinary EmbeddingType = "binary"
	EmbeddingUBin   EmbeddingType = "ubinary"
)

type BedrockCohereEmbedder struct {
	client    *bedrockruntime.Client
	dimension int
	modelID   *string
}

type CohereEmbedRequest struct {
	InputType      types.EmbeddingInputType `json:"input_type"`
	Texts          []string                 `json:"texts,omitempty"`
	EmbeddingTypes []EmbeddingType          `json:"embedding_types,omitempty"`
	OutputDim      int                      `json:"output_dimension,omitempty"`
	MaxTokens      int                      `json:"max_tokens,omitempty"`
	Truncate       string                   `json:"truncate,omitempty"`
}

type CohereEmbedResponse struct {
	ID         string `json:"id"`
	Embeddings struct {
		Float   [][]float32 `json:"float,omitempty"`
		Int8    [][]int8    `json:"int8,omitempty"`
		Uint8   [][]uint8   `json:"uint8,omitempty"`
		Binary  []string    `json:"binary,omitempty"`
		UBinary []string    `json:"ubinary,omitempty"`
	} `json:"embeddings"`
	ResponseType string   `json:"response_type"`
	Texts        []string `json:"texts,omitempty"`
}

func NewBedrockCohereEmbedder(
	ctx context.Context,
	modelID *string,
	cfg *config.FactoryConfig,
) (types.Embedder, error) {
	acfg, err := awsCfg.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("aws config load failed: %w", err)
	}

	embedder := &BedrockCohereEmbedder{
		client:  bedrockruntime.NewFromConfig(acfg),
		modelID: modelID,
	}

	if dim, ok := cfg.Options["dimension"].(int); ok {
		embedder.dimension = dim
	} else {
		embedder.dimension = 1024
	}

	return embedder, nil
}

func (e *BedrockCohereEmbedder) BuildRequest(
	ctx context.Context,
	req *CohereEmbedRequest,
) (*bedrockruntime.InvokeModelInput, error) {
	contentType := "application/json"
	accept := "*/*"

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	return &bedrockruntime.InvokeModelInput{
		ModelId:     e.modelID,
		ContentType: &contentType,
		Accept:      &accept,
		Body:        payload,
	}, nil
}

func (e *BedrockCohereEmbedder) Embed(
	ctx context.Context,
	texts []string,
	embedType types.EmbeddingInputType,
) ([][]float32, error) {
	req := &CohereEmbedRequest{
		InputType:      embedType,
		Texts:          texts,
		EmbeddingTypes: []EmbeddingType{EmbeddingFloat},
		OutputDim:      e.dimension,
	}

	params, err := e.BuildRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	result, err := e.client.InvokeModel(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("bedrock invoke failed: %w", err)
	}

	var response CohereEmbedResponse
	if err := json.Unmarshal(result.Body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Embeddings.Float) == 0 {
		return nil, fmt.Errorf("no embeddings returned from model")
	}

	return response.Embeddings.Float, nil
}

func (e *BedrockCohereEmbedder) Dimension() int {
	return e.dimension
}
