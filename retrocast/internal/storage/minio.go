package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOClient wraps a MinIO client with bucket-scoped operations.
type MinIOClient struct {
	client *minio.Client
	bucket string
	signer *minio.Client
}

// NewMinIOClient creates a MinIO client and ensures the bucket exists.
func NewMinIOClient(endpoint, accessKey, secretKey, bucket, domain, region string) (*MinIOClient, error) {
	if domain == "" || strings.ContainsAny(domain, ":/?#@ ") {
		return nil, fmt.Errorf("DOMAIN must be a host without scheme, path or port")
	}
	signer, err := minio.New(domain, &minio.Options{Creds: credentials.NewStaticV4(accessKey, secretKey, ""), Secure: true, Region: region})
	if err != nil {
		return nil, fmt.Errorf("minio public signer: %w", err)
	}
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
		Region: region,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client: %w", err)
	}

	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("minio bucket check: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: region}); err != nil {
			return nil, fmt.Errorf("minio make bucket: %w", err)
		}
	}

	return &MinIOClient{
		client: client,
		bucket: bucket,
		signer: signer,
	}, nil
}

// Upload stores an object in the bucket.
func (m *MinIOClient) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	_, err := m.client.PutObject(ctx, m.bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

// GetURL mints a fresh 15-minute URL locally: the fixed region avoids bucket lookup.
func (m *MinIOClient) GetURL(key, contentType string) (string, error) {
	params := make(url.Values)
	params.Set("response-content-disposition", "attachment")
	switch contentType {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		params.Set("response-content-disposition", "inline")
	}
	signed, err := m.signer.PresignedGetObject(context.Background(), m.bucket, key, 15*time.Minute, params)
	if err != nil {
		return "", err
	}
	return signed.String(), nil
}

// Delete removes an object from the bucket.
func (m *MinIOClient) Delete(ctx context.Context, key string) error {
	return m.client.RemoveObject(ctx, m.bucket, key, minio.RemoveObjectOptions{})
}
