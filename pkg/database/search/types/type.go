package types

import "context"

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
	ID       string         `json:"id"`
	Text     string         `json:"content"`
	Metadata map[string]any `json:"metadata"`
	Vector   []float32      `json:"embedding"`
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
	Shards       *string
	Replicas     *string
	Metadata     map[string]any
	VectorColumn string
}

type Index interface {
	Create(ctx context.Context, opts *IndexOptions) error

	Insert(ctx context.Context, docs *Document) error

	Update(ctx context.Context, docs *Document) error

	Delete(ctx context.Context, ids *[]string) error

	Search(ctx context.Context, query *string, key string, opts *SearchOptions) (*[]Result, error)

	VectorSearch(ctx context.Context, query *[]float32, opts *VectorSearchOptions) (*[]Result, error)

	Close(ctx context.Context) error
}
