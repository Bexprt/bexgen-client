package search

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	cfg "github.com/bexprt/bexgen-client/pkg/config"
	searchtypes "github.com/bexprt/bexgen-client/pkg/database/search/types"
	"github.com/elastic/go-elasticsearch/v9"
	"github.com/elastic/go-elasticsearch/v9/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v9/typedapi/esdsl"
	"github.com/elastic/go-elasticsearch/v9/typedapi/types"
	"github.com/elastic/go-elasticsearch/v9/typedapi/types/enums/densevectorsimilarity"
)

type OpenSearchClient struct {
	client *elasticsearch.TypedClient
	index  string
}

type OpenSearchConfig struct {
	Endpoint string `yaml:"endpoint" mapstructure:"endpoint"`
	Username string `yaml:"username" mapstructure:"username"`
	Password string `yaml:"password" mapstructure:"password"`
	Index    string `yaml:"index" mapstructure:"index"`
}

func NewClient(ctx context.Context, cfg *cfg.FactoryConfig) (searchtypes.Index, error) {
	osCfg := &OpenSearchConfig{}
	if len(cfg.Options) > 0 {
		if endpoint, ok := cfg.Options["endpoint"].(string); ok {
			osCfg.Endpoint = endpoint
		}
		if username, ok := cfg.Options["username"].(string); ok {
			osCfg.Username = username
		}
		if password, ok := cfg.Options["password"].(string); ok {
			osCfg.Password = password
		}
		if index, ok := cfg.Options["index"].(string); ok {
			osCfg.Index = index
		}
	}

	// TODO: Add AWS SigV4 signer to transport if needed
	esCfg := elasticsearch.Config{
		Addresses: []string{osCfg.Endpoint},
		Username:  osCfg.Username,
		Password:  osCfg.Password,
		Transport: http.DefaultTransport, // TODO: Replace with SigV4 signer transport for AWS
	}
	client, err := elasticsearch.NewTypedClient(esCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Elasticsearch client: %w", err)
	}

	return &OpenSearchClient{
		client: client,
		index:  osCfg.Index,
	}, nil
}

func (c *OpenSearchClient) Create(ctx context.Context, opts *searchtypes.IndexOptions) error {
	if opts == nil {
		return fmt.Errorf("index options cannot be nil")
	}
	if exists, _ := c.client.Indices.Exists(c.index).IsSuccess(ctx); exists {
		return nil
	}

	settings := &types.IndexSettings{
		Search:           &types.SettingsSearch{},
		NumberOfShards:   opts.Shards,
		NumberOfReplicas: opts.Replicas,
	}

	mappings := esdsl.NewTypeMapping().
		AddProperty("id", esdsl.NewKeywordProperty()).
		AddProperty("content", esdsl.NewTextProperty()).
		AddProperty("embedding", esdsl.NewDenseVectorProperty().
			Dims(1024).
			Index(true).
			Similarity(densevectorsimilarity.Cosine))

	resp, err := c.client.Indices.
		Create(c.index).
		Settings(settings).
		Mappings(mappings).
		Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to create index %s: %w", opts.Name, err)
	}

	if !resp.Acknowledged {
		return fmt.Errorf("index creation not acknowledged for %s", opts.Name)
	}

	return nil
}

func (c *OpenSearchClient) Insert(ctx context.Context, docs *searchtypes.Document) error {
	_, err := c.client.Index(c.index).
		Request(docs).
		Do(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (c *OpenSearchClient) Update(ctx context.Context, docs *searchtypes.Document) error {
	_, err := c.client.Index(c.index).
		Request(docs).
		Do(ctx)
	if err != nil {
		return err
	}
	return nil
}

// Delete removes documents from the OpenSearch index by IDs
func (c *OpenSearchClient) Delete(ctx context.Context, ids []string) error {
	// TODO: Implement delete logic using OpenSearch delete API
	return fmt.Errorf("not implemented")
}

// VectorSearch performs a vector similarity search
func (c *OpenSearchClient) VectorSearch(ctx context.Context, query []float32, opts *searchtypes.VectorSearchOptions) ([]searchtypes.Result, error) {
	// TODO: Implement vector search logic using OpenSearch k-NN plugin
	return nil, fmt.Errorf("not implemented")
}

func (c *OpenSearchClient) Search(
	ctx context.Context,
	query string,
	key string,
	opts *searchtypes.SearchOptions,
) ([]searchtypes.Result, error) {
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	res, err := c.client.Search().
		Index(c.index).
		Request(&search.Request{
			Query: &types.Query{
				Match: map[string]types.MatchQuery{
					key: {
						Query: query,
					},
				},
			},
		}).
		Do(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]searchtypes.Result, 0, len(res.Hits.Hits))

	for _, hit := range res.Hits.Hits {

		// Decode _source
		var source map[string]any
		if len(hit.Source_) > 0 {
			if err := json.Unmarshal(hit.Source_, &source); err != nil {
				return nil, fmt.Errorf("failed to unmarshal _source: %w", err)
			}
		}

		// Extract content text
		var text string
		if v, ok := source["content"].(string); ok {
			text = v
		}

		// Copy metadata (everything except content)
		metadata := make(map[string]any)
		for k, v := range source {
			if k != "content" {
				metadata[k] = v
			}
		}

		// Score
		var score float32
		if hit.Score_ != nil {
			score = float32(*hit.Score_)
		}

		// ID
		var id string
		if hit.Id_ != nil {
			id = *hit.Id_
		}

		results = append(results, searchtypes.Result{
			ID:       id,
			Score:    score,
			Text:     text,
			Metadata: metadata,
		})
	}

	return results, nil
}

// Close closes any resources (noop for OpenSearch client)
func (c *OpenSearchClient) Close(ctx context.Context) error {
	if err := c.client.Close(ctx); err != nil {
		return err
	}
	return nil
}
