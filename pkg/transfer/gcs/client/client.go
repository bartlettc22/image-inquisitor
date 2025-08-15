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

// ObjectHandle interface wraps object operations
type ObjectHandle interface {
	NewWriter(ctx context.Context) io.WriteCloser
	NewReader(ctx context.Context) (io.ReadCloser, error)
}

// ObjectIterator interface wraps object iteration
type ObjectIterator interface {
	Next() (*storage.ObjectAttrs, error)
	PageInfo() *iterator.PageInfo
}

type GCSClientWrapper struct {
	client *storage.Client
}

var _ GCSClient = &GCSClientWrapper{}

type GCSBucketHandleWrapper struct {
	*storage.BucketHandle
}

var _ BucketHandle = &GCSBucketHandleWrapper{}

type GCSObjectHandleWrapper struct {
	*storage.ObjectHandle
}

var _ ObjectHandle = &GCSObjectHandleWrapper{}

type GCSObjectIteratorWrapper struct {
	*storage.ObjectIterator
}

var _ ObjectIterator = &GCSObjectIteratorWrapper{}

func NewGCSClient(ctx context.Context) (*GCSClientWrapper, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &GCSClientWrapper{client}, nil
}

func (c *GCSClientWrapper) Bucket(name string) BucketHandle {
	return &GCSBucketHandleWrapper{c.client.Bucket(name)}
}

func (c *GCSClientWrapper) Close() error {
	return c.client.Close()
}

func (bh *GCSBucketHandleWrapper) Object(name string) ObjectHandle {
	return &GCSObjectHandleWrapper{bh.BucketHandle.Object(name)}
}

func (bh *GCSBucketHandleWrapper) Objects(ctx context.Context, q *storage.Query) ObjectIterator {
	return &GCSObjectIteratorWrapper{bh.BucketHandle.Objects(ctx, q)}
}

func (bh *GCSBucketHandleWrapper) Create(ctx context.Context, projectID string, attrs *storage.BucketAttrs) error {
	return bh.BucketHandle.Create(ctx, projectID, attrs)
}

func (oh *GCSObjectHandleWrapper) NewWriter(ctx context.Context) io.WriteCloser {
	return oh.ObjectHandle.NewWriter(ctx)
}

func (oh *GCSObjectHandleWrapper) NewReader(ctx context.Context) (io.ReadCloser, error) {
	return oh.ObjectHandle.NewReader(ctx)
}
