package blob

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Store represents a blob storage interface
type Store interface {
	PutObject(ctx context.Context, key string, data []byte, contentType string) (int64, error)
	GetObject(ctx context.Context, key string) ([]byte, error)
	PresignGet(ctx context.Context, key string, ttlSeconds int) (string, error)
	DeleteObject(ctx context.Context, key string) error
}

// S3Store implements Store using AWS S3 SDK v2 (compatible with Yandex Object Storage)
type S3Store struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	bucket        string
	endpoint      string
	accessKeyID   string
	secretKey     string
}

// NewS3Store creates a new S3Store for Yandex Object Storage
func NewS3Store(endpoint, region, bucket, accessKeyID, secretKey string) (*S3Store, error) {
	if endpoint == "" || bucket == "" || accessKeyID == "" || secretKey == "" {
		return nil, fmt.Errorf("S3 configuration incomplete: endpoint, bucket, accessKeyID, and secretKey are required")
	}
	if strings.TrimSpace(region) == "" {
		region = "ru-central1"
	}

	// Create custom AWS config with static credentials and custom endpoint
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               endpoint,
			SigningRegion:     region,
			HostnameImmutable: true,
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load S3 config: %w", err)
	}

	client := s3.NewFromConfig(cfg)
	presignClient := s3.NewPresignClient(client)

	return &S3Store{
		client:        client,
		presignClient: presignClient,
		bucket:        bucket,
		endpoint:      endpoint,
		accessKeyID:   accessKeyID,
		secretKey:     secretKey,
	}, nil
}

// PutObject uploads data to S3
func (s *S3Store) PutObject(ctx context.Context, key string, data []byte, contentType string) (int64, error) {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to put object: %w", err)
	}

	return int64(len(data)), nil
}

// PresignGet generates a presigned GET URL
func (s *S3Store) PresignGet(ctx context.Context, key string, ttlSeconds int) (string, error) {
	presignResult, err := s.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(time.Duration(ttlSeconds)*time.Second))

	if err != nil {
		return "", fmt.Errorf("failed to presign GET: %w", err)
	}

	return presignResult.URL, nil
}

// DeleteObject deletes an object from S3
func (s *S3Store) DeleteObject(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

// GetObject downloads an object from S3
func (s *S3Store) GetObject(ctx context.Context, key string) ([]byte, error) {
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read object body: %w", err)
	}

	return data, nil
}
