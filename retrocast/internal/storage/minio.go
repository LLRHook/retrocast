package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOClient wraps a MinIO client with bucket-scoped operations.
type MinIOClient struct {
	client   *minio.Client
	bucket   string
	endpoint string
}

// NewMinIOClient creates a MinIO client and ensures the bucket exists.
func NewMinIOClient(endpoint, accessKey, secretKey, bucket string) (*MinIOClient, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
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
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("minio make bucket: %w", err)
		}
	}

	return &MinIOClient{
		client:   client,
		bucket:   bucket,
		endpoint: endpoint,
	}, nil
}

// Upload stores an object in the bucket.
func (m *MinIOClient) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	_, err := m.client.PutObject(ctx, m.bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

// GetURL returns the public URL for an object.
func (m *MinIOClient) GetURL(key string) string {
	return fmt.Sprintf("http://%s/%s/%s", m.endpoint, m.bucket, key)
}

// Delete removes an object from the bucket.
func (m *MinIOClient) Delete(ctx context.Context, key string) error {
	return m.client.RemoveObject(ctx, m.bucket, key, minio.RemoveObjectOptions{})
}
