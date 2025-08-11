package transfergcs

import (
	"context"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/bartlettc22/image-inquisitor/pkg/api/metadata"
	clientmock "github.com/bartlettc22/image-inquisitor/pkg/transfer/gcs/client/mock"
	yaml "github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGCSTransfererRequireBucket(t *testing.T) {
	_, err := NewGCSTransferer(&GCSTransfererConfig{})
	require.ErrorContains(t, err, "bucket not specified")
}

func TestGCSTransfererExport(t *testing.T) {
	ctx := context.Background()

	testManifests := metadata.NewManifest("1", "testKind", "", "abc123")
	testManifestBytes, err := yaml.Marshal(testManifests)
	require.NoError(t, err)

	// Setup mocks
	mockClient := new(clientmock.MockGCSClient)
	mockBucket := new(clientmock.MockBucketHandle)
	mockObject := new(clientmock.MockObjectHandle)
	mockWriter := new(clientmock.MockWriter)

	// Configure mock expectations
	mockClient.On("Bucket", "test-bucket").Return(mockBucket)
	mockBucket.On("Object", "test-path/test-file.yaml").Return(mockObject)
	mockObject.On("NewWriter", ctx).Return(mockWriter)
	mockWriter.On("Write", testManifestBytes).Return(len(testManifestBytes), nil)
	mockWriter.On("Close").Return(nil)

	transferer, err := NewGCSTransferer(&GCSTransfererConfig{
		StorageClient: mockClient,
		Bucket:        "test-bucket",
		Path:          "test-path",
	})
	require.NoError(t, err)

	err = transferer.Export(ctx, "test-file", testManifests)

	// Assertions
	require.NoError(t, err)
	mockClient.AssertExpectations(t)
	mockBucket.AssertExpectations(t)
	mockObject.AssertExpectations(t)
	mockWriter.AssertExpectations(t)
}

func TestGCSTransfererImportFile(t *testing.T) {
	ctx := context.Background()

	// Setup imported data
	testManifest1 := metadata.NewManifest("1", "testKind", "", "abc123")
	testManifest1Bytes, err := yaml.Marshal(testManifest1)
	require.NoError(t, err)
	testManifest2 := metadata.NewManifest("2", "testKind", "", "abc123")
	testManifest2Bytes, err := yaml.Marshal(testManifest2)
	require.NoError(t, err)
	testManifest3 := metadata.NewManifest("3", "testKind", "", "abc123")
	testManifest3Bytes, err := yaml.Marshal(testManifest3)
	require.NoError(t, err)

	// Setup mocks
	mockClient := new(clientmock.MockGCSClient)
	mockBucket := new(clientmock.MockBucketHandle)
	mockObject1 := new(clientmock.MockObjectHandle)
	mockObject2 := new(clientmock.MockObjectHandle)
	mockObject3 := new(clientmock.MockObjectHandle)
	mockReader1 := clientmock.NewMockReader(testManifest1Bytes)
	mockReader2 := clientmock.NewMockReader(testManifest2Bytes)
	mockReader3 := clientmock.NewMockReader(testManifest3Bytes)

	// Create test objects
	testObjects := []storage.ObjectAttrs{
		{Name: "test-path/file1.txt"},
		{Name: "test-path/file2.txt"},
		{Name: "test-path/subdir/file3.txt"},
	}

	mockIterator := clientmock.NewMockObjectIterator(testObjects)
	mockIterator.SetupSequentialCalls()

	// Configure mock expectations
	mockClient.On("Bucket", "test-bucket").Return(mockBucket)
	mockBucket.On("Objects", ctx, &storage.Query{Prefix: "test-path/"}).Return(mockIterator)
	mockBucket.On("Object", "test-path/file1.txt").Return(mockObject1)
	mockBucket.On("Object", "test-path/file2.txt").Return(mockObject2)
	mockBucket.On("Object", "test-path/subdir/file3.txt").Return(mockObject3)
	mockObject1.On("NewReader", ctx).Return(mockReader1, nil)
	mockObject2.On("NewReader", ctx).Return(mockReader2, nil)
	mockObject3.On("NewReader", ctx).Return(mockReader3, nil)
	mockReader1.On("Close").Return(nil)
	mockReader2.On("Close").Return(nil)
	mockReader3.On("Close").Return(nil)

	transferer, err := NewGCSTransferer(&GCSTransfererConfig{
		StorageClient: mockClient,
		Bucket:        "test-bucket",
		Path:          "test-path/",
	})
	require.NoError(t, err)
	results, err := transferer.Import(ctx)

	// Assertions
	assert.NoError(t, err)
	require.Len(t, results, 3)
	assert.Equal(t, "1", results[0].Version)
	assert.Equal(t, "2", results[1].Version)
	assert.Equal(t, "3", results[2].Version)
	mockClient.AssertExpectations(t)
	mockBucket.AssertExpectations(t)
	mockIterator.AssertExpectations(t)
}
