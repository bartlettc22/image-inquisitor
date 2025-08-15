package mock

import (
	"bytes"
	"context"
	"io"

	"cloud.google.com/go/storage"
	gcsclient "github.com/bartlettc22/image-inquisitor/pkg/transfer/gcs/client"
	"github.com/stretchr/testify/mock"
	"google.golang.org/api/iterator"
)

// Mock implementations using testify/mock
type MockGCSClient struct {
	mock.Mock
}

func (m *MockGCSClient) Bucket(name string) gcsclient.BucketHandle {
	args := m.Called(name)
	return args.Get(0).(gcsclient.BucketHandle)
}

func (m *MockGCSClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

type MockBucketHandle struct {
	mock.Mock
}

func (m *MockBucketHandle) Object(name string) gcsclient.ObjectHandle {
	args := m.Called(name)
	return args.Get(0).(gcsclient.ObjectHandle)
}

func (m *MockBucketHandle) Objects(ctx context.Context, query *storage.Query) gcsclient.ObjectIterator {
	args := m.Called(ctx, query)
	return args.Get(0).(gcsclient.ObjectIterator)
}

func (m *MockBucketHandle) Create(ctx context.Context, projectID string, attrs *storage.BucketAttrs) error {
	args := m.Called(ctx, projectID, attrs)
	return args.Error(0)
}

type MockObjectIterator struct {
	mock.Mock
	objects []storage.ObjectAttrs
	index   int
}

func NewMockObjectIterator(objects []storage.ObjectAttrs) *MockObjectIterator {
	return &MockObjectIterator{
		objects: objects,
		index:   -1,
	}
}

func (m *MockObjectIterator) Next() (*storage.ObjectAttrs, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.ObjectAttrs), args.Error(1)
}

func (m *MockObjectIterator) PageInfo() *iterator.PageInfo {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*iterator.PageInfo)
}

// Convenience method to create a mock iterator that behaves like a real one
func (m *MockObjectIterator) SetupSequentialCalls() {
	for _, obj := range m.objects {
		objCopy := obj // Important: capture the value, not the reference
		m.On("Next").Return(&objCopy, nil).Once()
	}
	// Final call returns iterator.Done
	m.On("Next").Return((*storage.ObjectAttrs)(nil), iterator.Done).Once()
}

type MockObjectHandle struct {
	mock.Mock
}

func (m *MockObjectHandle) NewWriter(ctx context.Context) io.WriteCloser {
	args := m.Called(ctx)
	return args.Get(0).(io.WriteCloser)
}

func (m *MockObjectHandle) NewReader(ctx context.Context) (io.ReadCloser, error) {
	args := m.Called(ctx)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockObjectHandle) Delete(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockObjectHandle) Attrs(ctx context.Context) (*storage.ObjectAttrs, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.ObjectAttrs), args.Error(1)
}

// Mock writer for testing
type MockWriter struct {
	data []byte
	mock.Mock
}

func (m *MockWriter) Write(p []byte) (n int, err error) {
	m.data = append(m.data, p...)
	args := m.Called(p)
	return args.Int(0), args.Error(1)
}

func (m *MockWriter) Close() error {
	args := m.Called()
	return args.Error(0)
}

// Mock reader for testing
type MockReader struct {
	data   io.Reader
	closed bool
	mock.Mock
}

func NewMockReader(data []byte) *MockReader {
	return &MockReader{
		data: bytes.NewReader(data),
	}
}

func (m *MockReader) Read(p []byte) (n int, err error) {
	return m.data.Read(p)
}

func (m *MockReader) Close() error {
	m.closed = true
	args := m.Called()
	return args.Error(0)
}
