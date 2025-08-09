package gcsclient

import (
	"context"
	"io"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

// GCSClient interface wraps the operations we need from GCS
type GCSClient interface {
	Bucket(name string) BucketHandle
	Close() error
}

// BucketHandle interface wraps bucket operations
type BucketHandle interface {
	Object(name string) ObjectHandle
	Objects(ctx context.Context, q *storage.Query) ObjectIterator
	Create(ctx context.Context, projectID string, attrs *storage.BucketAttrs) error
}

// ObjectIterator interface wraps object iteration
type ObjectIterator interface {
	Next() (*storage.ObjectAttrs, error)
	PageInfo() *iterator.PageInfo
}

// ObjectHandle interface wraps object operations
type ObjectHandle interface {
	NewWriter(ctx context.Context) io.WriteCloser
	NewReader(ctx context.Context) (io.ReadCloser, error)
}
