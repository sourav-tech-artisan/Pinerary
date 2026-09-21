package objectstore

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type ObjectInfo struct {
	Size        int64
	ContentType string
	ETag        string
}

type Store interface {
	Get(context.Context, string) (io.ReadCloser, error)
	Put(context.Context, string, io.Reader, int64, string) error
	PresignPut(context.Context, string, time.Duration) (*url.URL, error)
	PresignGet(context.Context, string, time.Duration) (*url.URL, error)
	Stat(context.Context, string) (ObjectInfo, error)
}

type Deleter interface {
	Delete(context.Context, string) error
}

func (s *MinIOStore) PresignGet(ctx context.Context, key string, expiry time.Duration) (*url.URL, error) {
	result, err := s.client.PresignedGetObject(ctx, s.bucket, key, expiry, nil)
	if err != nil {
		return nil, fmt.Errorf("presign object download: %w", err)
	}
	return result, nil
}

type MinIOStore struct {
	client *minio.Client
	bucket string
}

func NewMinIOStore(endpoint, accessKey, secretKey, bucket string, useTLS bool) (*MinIOStore, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds: credentials.NewStaticV4(accessKey, secretKey, ""), Secure: useTLS,
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	})
	if err != nil {
		return nil, fmt.Errorf("create S3-compatible client: %w", err)
	}
	if bucket == "" {
		return nil, fmt.Errorf("object-storage bucket is required")
	}
	return &MinIOStore{client: client, bucket: bucket}, nil
}

func (s *MinIOStore) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("check object-storage bucket: %w", err)
	}
	if !exists {
		if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("create object-storage bucket: %w", err)
		}
	}
	return nil
}

func (s *MinIOStore) PresignPut(ctx context.Context, key string, expiry time.Duration) (*url.URL, error) {
	result, err := s.client.PresignedPutObject(ctx, s.bucket, key, expiry)
	if err != nil {
		return nil, fmt.Errorf("presign object upload: %w", err)
	}
	return result, nil
}

func (s *MinIOStore) Stat(ctx context.Context, key string) (ObjectInfo, error) {
	result, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("stat object: %w", err)
	}
	return ObjectInfo{Size: result.Size, ContentType: result.ContentType, ETag: result.ETag}, nil
}

func (s *MinIOStore) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	object, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object: %w", err)
	}
	return object, nil
}

func (s *MinIOStore) Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, reader, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("put object: %w", err)
	}
	return nil
}

func (s *MinIOStore) Delete(ctx context.Context, key string) error {
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("delete object: %w", err)
	}
	return nil
}
