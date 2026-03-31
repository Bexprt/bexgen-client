package search

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	cfg "github.com/bexprt/bexgen-client/pkg/config"
	searchtypes "github.com/bexprt/bexgen-client/pkg/database/search/types"
	"github.com/elastic/go-elasticsearch/v9"
	"github.com/elastic/go-elasticsearch/v9/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v9/typedapi/esdsl"
	"github.com/elastic/go-elasticsearch/v9/typedapi/types"
	"github.com/elastic/go-elasticsearch/v9/typedapi/types/enums/densevectorsimilarity"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
)

// OpenSearchCompat wraps any transport to replace the go-elasticsearch/v9
// vendor content-type with standard application/json for OpenSearch compat.
type OpenSearchCompat struct {
	Inner http.RoundTripper
}

func (t *OpenSearchCompat) RoundTrip(req *http.Request) (*http.Response, error) {
	ct := req.Header.Get("Content-Type")
	if ct != "" && ct != "application/json" {
		req.Header.Set("Content-Type", "application/json")
	}
	accept := req.Header.Get("Accept")
	if accept != "" && accept != "application/json" {
		req.Header.Set("Accept", "application/json")
	}
	resp, err := t.Inner.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	// Spoof the product header so go-elasticsearch/v9 doesn't reject OpenSearch.
	resp.Header.Set("X-Elastic-Product", "Elasticsearch")
	return resp, nil
}

type SigV4Transport struct {
	Transport   http.RoundTripper
	Signer      *v4.Signer
	Credentials aws.CredentialsProvider
	Region      string
	Service     string
}

func NewSigV4Transport(ctx context.Context, region string) (*SigV4Transport, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	return &SigV4Transport{
		Transport:   http.DefaultTransport,
		Signer:      v4.NewSigner(),
		Credentials: cfg.Credentials,
		Region:      region,
		Service:     "es",
	}, nil
}

func hashPayload(body []byte) string {
	hash := sha256.Sum256(body)
	return hex.EncodeToString(hash[:])
}

func (t *SigV4Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	var bodyBytes []byte
	if req.Body != nil {
		bodyBytes, _ = io.ReadAll(req.Body)
	}

	// Restore body for downstream use
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	creds, err := t.Credentials.Retrieve(req.Context())
	if err != nil {
		return nil, err
	}

	payloadHash := hashPayload(bodyBytes)

	err = t.Signer.SignHTTP(
		req.Context(),
		creds,
		req,
		payloadHash,
		t.Service,
		t.Region,
		time.Now(),
	)
	if err != nil {
		return nil, err
	}

	return t.Transport.RoundTrip(req)
}

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

func NewClientElasticSearch(ctx context.Context, cfg *cfg.FactoryConfig) (searchtypes.Index, error) {
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

	esCfg := elasticsearch.Config{
		Addresses: []string{osCfg.Endpoint},
		Username:  osCfg.Username,
		Password:  osCfg.Password,
		Transport: &OpenSearchCompat{Inner: http.DefaultTransport},
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

func NewClientOpenSearch(ctx context.Context, cfg *cfg.FactoryConfig) (searchtypes.Index, error) {
	osCfg := &OpenSearchConfig{}
	if len(cfg.Options) > 0 {
		if endpoint, ok := cfg.Options["endpoint"].(string); ok {
			osCfg.Endpoint = endpoint
		}
		if index, ok := cfg.Options["index"].(string); ok {
			osCfg.Index = index
		}
	}

	// Create SigV4 transport
	sigv4Transport, err := NewSigV4Transport(ctx, "us-east-1") // TODO: make region configurable
	if err != nil {
		return nil, fmt.Errorf("failed to create sigv4 transport: %w", err)
	}

	esCfg := elasticsearch.Config{
		Addresses: []string{osCfg.Endpoint},
		Transport: &OpenSearchCompat{Inner: sigv4Transport},
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
