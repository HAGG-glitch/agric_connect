package storage

import (
	"context"
	"io"
	"time"
)

type SaveObjectInput struct {
	Content     io.Reader
	ContentType string
	Path        string
}

type StoredObject struct {
	Path         string
	ContentType  string
	SizeBytes    int64
}

type ObjectStorage interface {
	Save(ctx context.Context, input SaveObjectInput) (StoredObject, error)
	Delete(ctx context.Context, path string) error
	SignedURL(ctx context.Context, path string, expiry time.Duration) (string, error)
	Download(ctx context.Context, path string) (io.ReadCloser, error)
}
