package s3

import (
	"context"
	"fmt"
	"io"

	cfg "github.com/bexprt/bexgen-client/pkg/config"
	"github.com/bexprt/bexgen-client/pkg/storage/types"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Client struct {
	client *s3.Client
	bucket string
	ctx    context.Context
}

type S3Config struct {
	Bucket    string `yaml:"bucket" mapstructure:"bucket"`
	Region    string `yaml:"region" mapstructure:"region"`
	Endpoint  string `yaml:"endpoint" mapstructure:"endpoint"` // For MinIO or custom S3-compatible services
	AccessID  string `yaml:"access_id" mapstructure:"access_id"`
	SecretKey string `yaml:"secret_key" mapstructure:"secret_key"`
}

func NewClient(ctx context.Context, cfg *cfg.FactoryConfig) (types.ObjectStorage, error) {
	// Parse S3 configuration from options
	s3Cfg := &S3Config{}

	if len(cfg.Options) > 0 {
		// If bucket is specified in options, use those values
		if bucket, ok := cfg.Options["bucket"].(string); ok {
			s3Cfg.Bucket = bucket
		}
		if region, ok := cfg.Options["region"].(string); ok {
			s3Cfg.Region = region
		}
		if endpoint, ok := cfg.Options["endpoint"].(string); ok {
			s3Cfg.Endpoint = endpoint
		}
		if accessID, ok := cfg.Options["access_id"].(string); ok {
			s3Cfg.AccessID = accessID
		}
		if secretKey, ok := cfg.Options["secret_key"].(string); ok {
			s3Cfg.SecretKey = secretKey
		}
	}

	// If bucket is not set in config, fail
	if s3Cfg.Bucket == "" {
		return nil, fmt.Errorf("bucket name is required in storage configuration")
	}

	// Set default region if not specified
	if s3Cfg.Region == "" {
		s3Cfg.Region = "us-east-1"
	}

	// Build AWS configuration
	var awsCfg aws.Config
	var err error

	if s3Cfg.AccessID != "" && s3Cfg.SecretKey != "" {
		// Use provided credentials (for MinIO or custom S3)
		awsCfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(s3Cfg.Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				s3Cfg.AccessID,
				s3Cfg.SecretKey,
				"",
			)),
		)
	} else {
		// Use AWS default credentials chain (env vars, IAM role, etc.)
		awsCfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(s3Cfg.Region),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with optional custom endpoint
	clientOptions := func(o *s3.Options) {
		if s3Cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(s3Cfg.Endpoint)
			o.UsePathStyle = true // Required for MinIO compatibility
		}
	}

	s3Client := s3.NewFromConfig(awsCfg, clientOptions)

	return &Client{
		client: s3Client,
		bucket: s3Cfg.Bucket,
		ctx:    ctx,
	}, nil
}

func (c Client) Store(path string, r io.Reader) error {
	params := s3.PutObjectInput{
		Bucket: &c.bucket,
		Key:    &path,
		Body:   r,
	}
	_, err := c.client.PutObject(c.ctx, &params)
	if err != nil {
		return err
	}
	return nil
}

func (c Client) Get(path string) (io.ReadCloser, error) {
	params := s3.GetObjectInput{
		Bucket: &c.bucket,
		Key:    &path,
	}

	payload, err := c.client.GetObject(c.ctx, &params)
	if err != nil {
		return nil, err
	}

	return payload.Body, nil
}
