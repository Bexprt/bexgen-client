package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	cfg "github.com/bexprt/bexgen-client/pkg/config"
	searchtypes "github.com/bexprt/bexgen-client/pkg/database/search/types"
	"github.com/elastic/go-elasticsearch/v9"
	"github.com/elastic/go-elasticsearch/v9/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v9/typedapi/esdsl"
	"github.com/elastic/go-elasticsearch/v9/typedapi/types"
	"github.com/elastic/go-elasticsearch/v9/typedapi/types/enums/densevectorsimilarity"
)

// OpenSearchCompat wraps an http.RoundTripper to fix two incompatibilities
// between the go-elasticsearch/v9 client and OpenSearch:
//  1. Rewrites the vendor Content-Type to standard application/json
//  2. Injects the X-Elastic-Product response header that the client validates
type OpenSearchCompat struct {
	Wrapped http.RoundTripper
}

func (t *OpenSearchCompat) RoundTrip(req *http.Request) (*http.Response, error) {
	ct := req.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/vnd.elasticsearch") {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := t.Wrapped.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	resp.Header.Set("X-Elastic-Product", "Elasticsearch")

	// Also rewrite response content-type if OpenSearch sends something unexpected
	rct := resp.Header.Get("Content-Type")
	if rct != "" && !strings.Contains(rct, "application/json") {
		resp.Header.Set("Content-Type", "application/json")
	}

	// Discard the body and set an empty one for HEAD requests to prevent client errors
	if req.Method == http.MethodHead && resp.Body != nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		resp.Body = http.NoBody
	}

	return resp, nil
}

type ElasticSearchClient struct {
	client *elasticsearch.TypedClient
	index  string
}

type ElasticSearchConfig struct {
	Endpoint string `yaml:"endpoint" mapstructure:"endpoint"`
	Username string `yaml:"username" mapstructure:"username"`
	Password string `yaml:"password" mapstructure:"password"`
	Index    string `yaml:"index" mapstructure:"index"`
}

func NewClientElasticSearch(ctx context.Context, cfg *cfg.FactoryConfig) (searchtypes.Index, error) {
	osCfg := &ElasticSearchConfig{}
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

	esCfg := elasticsearch.Config{
		Addresses: []string{osCfg.Endpoint},
		Username:  osCfg.Username,
		Password:  osCfg.Password,
		Transport: &OpenSearchCompat{Wrapped: http.DefaultTransport},
	}
	client, err := elasticsearch.NewTypedClient(esCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Elasticsearch client: %w", err)
	}

	return &ElasticSearchClient{
		client: client,
		index:  osCfg.Index,
	}, nil
}

func (c *ElasticSearchClient) Create(ctx context.Context, opts *searchtypes.IndexOptions) error {
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

func (c *ElasticSearchClient) Insert(ctx context.Context, docs *searchtypes.Document) error {
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

func (c *ElasticSearchClient) Update(ctx context.Context, docs *searchtypes.Document) error {
	_, err := c.client.
		Update(c.index, docs.ID).
		Doc(docs).
		Do(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (c *ElasticSearchClient) Delete(ctx context.Context, id string) error {
	_, err := c.client.
		Delete(c.index, id).
		Do(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (c *ElasticSearchClient) GetByID(ctx context.Context, id string) (*searchtypes.Document, error) {
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

func (c *ElasticSearchClient) VectorSearch(
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

func (c *ElasticSearchClient) Search(
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

func (c *ElasticSearchClient) Close(ctx context.Context) error {
	if err := c.client.Close(ctx); err != nil {
		return err
	}
	return nil
}
