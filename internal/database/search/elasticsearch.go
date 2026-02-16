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
		AddProperty("summary", esdsl.NewTextProperty()).
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
	_, err := c.client.
		Index(c.index).
		Id(docs.ID).
		Request(docs).
		Do(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (c *OpenSearchClient) Update(ctx context.Context, docs *searchtypes.Document) error {
	_, err := c.client.
		Update(c.index, docs.ID).
		Doc(docs).
		Do(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (c *OpenSearchClient) Delete(ctx context.Context, id string) error {
	_, err := c.client.
		Delete(c.index, id).
		Do(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (c *OpenSearchClient) GetByID(ctx context.Context, id string) (*searchtypes.Document, error) {
	res, err := c.client.
		Get(c.index, id).
		Do(ctx)
	if err != nil {
		return nil, err
	}

	if !res.Found {
		return nil, fmt.Errorf("document not found")
	}

	var doc searchtypes.Document
	if err := json.Unmarshal(res.Source_, &doc); err != nil {
		return nil, err
	}

	return &doc, nil
}

func (c *OpenSearchClient) VectorSearch(
	ctx context.Context,
	query []float32,
	opts *searchtypes.SearchOptions,
) ([]searchtypes.Result, error) {
	if len(query) == 0 {
		return nil, fmt.Errorf("vector query cannot be empty")
	}

	limit := 10
	offset := 0
	k := 10
	numCandidates := 100

	if opts != nil {
		if opts.Limit > 0 {
			limit = opts.Limit
		}
		if opts.Offset > 0 {
			offset = opts.Offset
		}
		if opts.TopK > 0 {
			k = opts.TopK
			numCandidates = k * 20
		}
	}

	var filters []types.Query

	if opts != nil && opts.Filters != nil {
		for field, value := range opts.Filters {
			filters = append(filters, types.Query{
				Term: map[string]types.TermQuery{
					field: {Value: value},
				},
			})
		}
	}

	knnQuery := esdsl.NewKnnQuery().
		Field("embedding").
		QueryVector(query...).
		K(k).
		NumCandidates(numCandidates)

	finalQuery := &types.Query{
		Bool: &types.BoolQuery{
			Must: []types.Query{
				*knnQuery.QueryCaster(),
			},
			Filter: filters,
		},
	}

	res, err := c.client.Search().
		Index(c.index).
		Request(&search.Request{
			From:  &offset,
			Size:  &limit,
			Query: finalQuery,
		}).
		Do(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]searchtypes.Result, 0, len(res.Hits.Hits))

	for _, hit := range res.Hits.Hits {

		var source map[string]any
		if len(hit.Source_) > 0 {
			if err := json.Unmarshal(hit.Source_, &source); err != nil {
				return nil, err
			}
		}

		var text string
		if v, ok := source["content"].(string); ok {
			text = v
		}

		metadata := make(map[string]any)
		for k, v := range source {
			if k != "content" {
				metadata[k] = v
			}
		}

		var score float32
		if hit.Score_ != nil {
			score = float32(*hit.Score_)
		}

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

func (c *OpenSearchClient) Search(
	ctx context.Context,
	query string,
	key searchtypes.SearchFileds,
	opts *searchtypes.SearchOptions,
) ([]searchtypes.Result, error) {
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	limit := 10
	offset := 0

	if opts != nil {
		if opts.Limit > 0 {
			limit = opts.Limit
		}
		if opts.Offset > 0 {
			offset = opts.Offset
		}
	}

	boolQuery := &types.BoolQuery{
		Must: []types.Query{
			{
				Match: map[string]types.MatchQuery{
					string(key): {Query: query},
				},
			},
		},
		Filter: []types.Query{},
	}

	if opts != nil && opts.Filters != nil {
		for field, value := range opts.Filters {
			boolQuery.Filter = append(boolQuery.Filter, types.Query{
				Term: map[string]types.TermQuery{
					field: {Value: value},
				},
			})
		}
	}

	res, err := c.client.Search().
		Index(c.index).
		Request(&search.Request{
			From: &offset,
			Size: &limit,
			Query: &types.Query{
				Bool: boolQuery,
			},
		}).
		Do(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]searchtypes.Result, 0, len(res.Hits.Hits))

	for _, hit := range res.Hits.Hits {
		var source map[string]any
		if len(hit.Source_) > 0 {
			if err := json.Unmarshal(hit.Source_, &source); err != nil {
				return nil, err
			}
		}

		var text string
		if v, ok := source["content"].(string); ok {
			text = v
		}

		metadata := make(map[string]any)
		for k, v := range source {
			if k != "content" {
				metadata[k] = v
			}
		}

		var score float32
		if hit.Score_ != nil {
			score = float32(*hit.Score_)
		}

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

func (c *OpenSearchClient) Close(ctx context.Context) error {
	if err := c.client.Close(ctx); err != nil {
		return err
	}
	return nil
}
