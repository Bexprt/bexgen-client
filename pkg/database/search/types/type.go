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

type SearchFileds string

const (
	Content SearchFileds = "content"
	Summary SearchFileds = "summary"
)

type Document struct {
	ID      string    `json:"id"`
	Text    string    `json:"content"`
	Summary string    `json:"summary"`
	Vector  []float32 `json:"embedding"`
}

type SearchOptions struct {
	Limit    int
	Offset   int
	TopK     int
	MinScore float32

	Filters map[string]any
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

	Delete(ctx context.Context, id string) error

	GetByID(ctx context.Context, id string) (*Document, error)

	Search(ctx context.Context, query string, key SearchFileds, opts *SearchOptions) ([]Result, error)

	VectorSearch(ctx context.Context, query []float32, opts *SearchOptions) ([]Result, error)

	Close(ctx context.Context) error
}
