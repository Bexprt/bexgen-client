package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	cfg "github.com/bexprt/bexgen-client/pkg/config"
	searchtypes "github.com/bexprt/bexgen-client/pkg/database/search/types"

	"github.com/aws/aws-sdk-go-v2/config"
	opensearch "github.com/opensearch-project/opensearch-go/v4"
	opensearchapi "github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	requestsigner "github.com/opensearch-project/opensearch-go/v4/signer/awsv2"
)

type OpenSearchClient struct {
	client *opensearch.Client
	index  string
}

type OpenSearchConfig struct {
	Endpoint string
	Region   string
	Index    string
}

func NewClientOpenSearch(ctx context.Context, cfg *cfg.FactoryConfig) (searchtypes.Index, error) {
	osCfg := &OpenSearchConfig{}

	if len(cfg.Options) > 0 {
		osCfg.Endpoint = cfg.Options["endpoint"].(string)
		osCfg.Region = cfg.Options["region"].(string)
		osCfg.Index = cfg.Options["index"].(string)
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(osCfg.Region))
	if err != nil {
		return nil, err
	}

	signer, err := requestsigner.NewSignerWithService(awsCfg, "es")
	if err != nil {
		return nil, err
	}

	client, err := opensearch.NewClient(opensearch.Config{
		Addresses: []string{osCfg.Endpoint},
		Signer:    signer,
	})
	if err != nil {
		return nil, err
	}

	return &OpenSearchClient{
		client: client,
		index:  osCfg.Index,
	}, nil
}

func (c *OpenSearchClient) Create(ctx context.Context, opts *searchtypes.IndexOptions) error {
	body := map[string]any{
		"settings": map[string]any{
			"number_of_shards":   opts.Shards,
			"number_of_replicas": opts.Replicas,
			"knn":                true,
		},
		"mappings": map[string]any{
			"properties": map[string]any{
				"id":      map[string]any{"type": "keyword"},
				"content": map[string]any{"type": "text"},
				"summary": map[string]any{"type": "text"},
				"embedding": map[string]any{
					"type":      "knn_vector",
					"dimension": 1024,
				},
			},
		},
	}

	b, _ := json.Marshal(body)

	req := opensearchapi.IndicesCreateReq{
		Index: c.index,
		Body:  bytes.NewReader(b),
	}

	var resp map[string]any
	_, err := c.client.Do(ctx, req, &resp)
	return err
}

func (c *OpenSearchClient) Insert(ctx context.Context, doc *searchtypes.Document) error {
	b, _ := json.Marshal(doc)

	req := opensearchapi.IndexReq{
		Index:      c.index,
		DocumentID: doc.ID,
		Body:       bytes.NewReader(b),
	}

	var resp map[string]any
	_, err := c.client.Do(ctx, req, &resp)
	return err
}

func (c *OpenSearchClient) Update(ctx context.Context, doc *searchtypes.Document) error {
	body := map[string]any{"doc": doc}
	b, _ := json.Marshal(body)

	req := opensearchapi.UpdateReq{
		Index:      c.index,
		DocumentID: doc.ID,
		Body:       bytes.NewReader(b),
	}

	var resp map[string]any
	_, err := c.client.Do(ctx, req, &resp)
	return err
}

func (c *OpenSearchClient) Delete(ctx context.Context, id string) error {
	req := opensearchapi.DocumentDeleteReq{
		Index:      c.index,
		DocumentID: id,
	}

	var resp map[string]any
	_, err := c.client.Do(ctx, req, &resp)
	return err
}

func (c *OpenSearchClient) GetByID(ctx context.Context, id string) (*searchtypes.Document, error) {
	req := opensearchapi.DocumentGetReq{
		Index:      c.index,
		DocumentID: id,
	}

	var resp struct {
		Found  bool                 `json:"found"`
		Source searchtypes.Document `json:"_source"`
	}

	_, err := c.client.Do(ctx, req, &resp)
	if err != nil {
		return nil, err
	}

	if !resp.Found {
		return nil, fmt.Errorf("document not found")
	}

	return &resp.Source, nil
}

func (c *OpenSearchClient) Search(
	ctx context.Context,
	query string,
	key searchtypes.SearchFileds,
	opts *searchtypes.SearchOptions,
) ([]searchtypes.Result, error) {
	body := map[string]any{
		"query": map[string]any{
			"match": map[string]any{
				string(key): query,
			},
		},
	}

	b, _ := json.Marshal(body)

	req := opensearchapi.SearchReq{
		Indices: []string{c.index},
		Body:    bytes.NewReader(b),
	}

	var resp map[string]any
	_, err := c.client.Do(ctx, req, &resp)
	if err != nil {
		return nil, err
	}

	return parseResults(resp), nil
}

func (c *OpenSearchClient) VectorSearch(
	ctx context.Context,
	query []float32,
	opts *searchtypes.SearchOptions,
) ([]searchtypes.Result, error) {
	body := map[string]any{
		"query": map[string]any{
			"knn": map[string]any{
				"embedding": map[string]any{
					"vector": query,
					"k":      10,
				},
			},
		},
	}

	b, _ := json.Marshal(body)

	req := opensearchapi.SearchReq{
		Indices: []string{c.index},
		Body:    bytes.NewReader(b),
	}

	var resp map[string]any
	_, err := c.client.Do(ctx, req, &resp)
	if err != nil {
		return nil, err
	}

	return parseResults(resp), nil
}

func parseResults(raw map[string]any) []searchtypes.Result {
	hits := raw["hits"].(map[string]any)["hits"].([]any)

	results := make([]searchtypes.Result, 0, len(hits))

	for _, h := range hits {
		hit := h.(map[string]any)

		src := hit["_source"].(map[string]any)

		text, _ := src["content"].(string)
		score, _ := hit["_score"].(float64)
		id, _ := hit["_id"].(string)

		results = append(results, searchtypes.Result{
			ID:       id,
			Score:    float32(score),
			Text:     text,
			Metadata: src,
		})
	}

	return results
}

func (c *OpenSearchClient) Close(ctx context.Context) error {
	return nil
}
