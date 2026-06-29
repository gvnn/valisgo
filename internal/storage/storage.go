package storage

import (
	"context"
	"io"

	"gocloud.dev/blob"
)

// Storage defines the interface for our blob storage operations.
type Storage interface {
	Put(ctx context.Context, key string, data io.Reader) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

type blobStorage struct {
	bucket *blob.Bucket
}

// NewBlobStorage creates a new Storage instance backed by a gocloud.dev/blob bucket.
func NewBlobStorage(bucket *blob.Bucket) Storage {
	return &blobStorage{
		bucket: bucket,
	}
}

func (s *blobStorage) Put(ctx context.Context, key string, data io.Reader) error {
	w, err := s.bucket.NewWriter(ctx, key, nil)
	if err != nil {
		return err
	}
	defer w.Close()

	_, err = io.Copy(w, data)
	return err
}

func (s *blobStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return s.bucket.NewReader(ctx, key, nil)
}

func (s *blobStorage) Delete(ctx context.Context, key string) error {
	return s.bucket.Delete(ctx, key)
}
