package types

import (
	"context"
)

type Model interface {
	Invoke(ctx context.Context, propmt string) (string, error)
}

type Rerank interface {
	Rank(ctx context.Context, texts []string) ([]string, error)
}

type EmbeddingInputType string

const (
	InputSearchDocument EmbeddingInputType = "search_document"
	InputSearchQuery    EmbeddingInputType = "search_query"
	InputClassification EmbeddingInputType = "classification"
	InputClustering     EmbeddingInputType = "clustering"
)

type Embedder interface {
	Embed(ctx context.Context, texts []string, embedType EmbeddingInputType) ([][]float32, error)

	Dimension() int
}
