package gcs

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	log "github.com/sirupsen/logrus"
)

var errPrefix = "gcs"

type GCS struct {
	client *storage.Client
}

func NewGCS(ctx context.Context) (*GCS, error) {

	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}

	return &GCS{
		client: client,
	}, nil
}

func (gcs *GCS) Close() error {
	return gcs.client.Close()
}

func (gcs *GCS) ReadAllFilesInDir(ctx context.Context, bucketName string, prefix string) (map[string][]byte, error) {

	files, err := gcs.listFiles(ctx, bucketName, prefix)
	if err != nil {
		log.Fatalf("Error listing files: %v", err)
	}

	results := make(map[string][]byte)
	for _, file := range files {
		data, err := gcs.readGCSFile(ctx, bucketName, file)
		if err != nil {
			return nil, fmt.Errorf("gcs: error reading file gs://%s/%s: %v", bucketName, file, err)
		}
		results["gs://"+filepath.Join(bucketName, file)] = data
	}

	return results, nil
}

func (gcs *GCS) listFiles(ctx context.Context, bucketName, prefix string) ([]string, error) {
	var files []string
	it := gcs.client.Bucket(bucketName).Objects(ctx, &storage.Query{Prefix: prefix})
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}
		files = append(files, attrs.Name)
	}

	return files, nil
}

func (gcs *GCS) readGCSFile(ctx context.Context, bucketName, fileName string) ([]byte, error) {

	bucket := gcs.client.Bucket(bucketName)
	obj := bucket.Object(fileName)
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS reader: %w", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read GCS file data: %w", err)
	}

	return data, nil
}
