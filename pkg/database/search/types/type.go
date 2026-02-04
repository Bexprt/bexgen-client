package types

import "context"

type DefaultEngine struct {
	Embedder Embedder
	Index    VectorIndex
}

type Result struct {
	ID       string
	Score    float32
	Text     string
	Metadata map[string]any
}

type DistanceMetric string

const (
	Cosine    DistanceMetric = "cosine"
	Euclidean DistanceMetric = "l2"
	Dot       DistanceMetric = "dot"
)

type Document struct {
	ID       string
	Text     string
	Metadata map[string]any
	Vector   []float32
}

type VectorSearchOptions struct {
	TopK     int
	MinScore float32

	Filter map[string]any
}

type SearchOptions struct {
	TopK int

	Filter map[string]any
}

type IndexOptions struct {
	Name         string
	Dimension    int
	Metric       DistanceMetric
	Shards       int
	Replicas     int
	Metadata     map[string]any
	VectorColumn string
}

type Embedder interface {
	Embed(ctx context.Context, text *string) (*[]float32, error)

	EmbedBatch(ctx context.Context, texts *[]string) (*[][]float32, error)

	Dimension() int

	Model() string
}

type VectorIndex interface {
	Create(ctx context.Context, opts *IndexOptions) error

	Upsert(ctx context.Context, docs *[]Document) error

	Delete(ctx context.Context, ids *[]string) error

	VectorSearch(ctx context.Context, query *[]float32, opts *VectorSearchOptions) (*[]Result, error)

	Search(ctx context.Context, query *string, opts *SearchOptions) (*[]Result, error)

	Close() error
}
