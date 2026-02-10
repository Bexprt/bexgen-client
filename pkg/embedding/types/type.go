package types

import "context"

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
