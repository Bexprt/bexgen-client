package novapro

import (
	"context"
	"encoding/json"
	"fmt"

	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	bedrockruntime "github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/bexprt/bexgen-client/pkg/ai/types"
)

type NovaClient struct {
	client      *bedrockruntime.Client
	modelID     string
	MaxTokens   int
	Temperature float32
}

func New(ctx context.Context, modelID string, maxTokens int, temperature float32) (types.Model, error) {
	cfg, err := awsCfg.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	return &NovaClient{
		client:      bedrockruntime.NewFromConfig(cfg),
		modelID:     modelID,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}, nil
}

func (s *NovaClient) Invoke(ctx context.Context, text string) (string, error) {
	payload := map[string]any{
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"text": text,
					},
				},
			},
		},
		"inferenceConfig": map[string]any{
			"maxTokens":   s.MaxTokens,
			"temperature": s.Temperature,
		},
	}

	body, _ := json.Marshal(payload)

	contentType := "application/json"
	accept := "application/json"

	out, err := s.client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     &s.modelID,
		ContentType: &contentType,
		Accept:      &accept,
		Body:        body,
	})
	if err != nil {
		return "", err
	}
	var response struct {
		Output struct {
			Message struct {
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
				Role string `json:"role"`
			} `json:"message"`
		} `json:"output"`
	}

	if err := json.Unmarshal(out.Body, &response); err != nil {
		return "", err
	}

	if len(response.Output.Message.Content) == 0 {
		return "", fmt.Errorf("no summary returned")
	}

	return response.Output.Message.Content[0].Text, nil
}
