package transfergcs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"

	"cloud.google.com/go/storage"
	"github.com/bartlettc22/image-inquisitor/pkg/api/metadata"
	"github.com/bartlettc22/image-inquisitor/pkg/transfer"
	gcsclient "github.com/bartlettc22/image-inquisitor/pkg/transfer/gcs/client"
	yaml "github.com/goccy/go-yaml"
	"google.golang.org/api/iterator"
)

// GCSTransfererConfig is the configuration for importing/exporting to GCS
type GCSTransfererConfig struct {
	StorageClient gcsclient.GCSClient
	Bucket        string
	Path          string
}

// GCSTransferer is used for importing/exporting manifests to/from GCS
type GCSTransferer struct {
	*GCSTransfererConfig
}

// Ensure GCSExporter implements Transferer
var _ transfer.Transferer = &GCSTransferer{}

// NewGCSTransferer creates a new GCS importer/exporter
func NewGCSTransferer(config *GCSTransfererConfig) (*GCSTransferer, error) {

	if config.Bucket == "" {
		return nil, fmt.Errorf("transfer to/from GCS, bucket not specified")
	}

	if config.StorageClient == nil {
		client, err := gcsclient.NewGCSClient(context.Background())
		if err != nil {
			return nil, fmt.Errorf("transfer to/from GCS, failed to create storage client: %w", err)
		}
		config.StorageClient = client
	}

	return &GCSTransferer{
		config,
	}, nil
}

// Export exports a manifest to GCS with the given name as a component of the file name
func (t *GCSTransferer) Export(ctx context.Context, name string, manifest *metadata.Manifest) error {

	fileName := filepath.Join(t.Path, name+".yaml")

	bucket := t.StorageClient.Bucket(t.Bucket)
	wc := bucket.Object(fileName).NewWriter(ctx)

	yamlBytes, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("transfer to/from GCS, error marshalling export: %w", err)
	}
	yamlReader := bytes.NewReader(yamlBytes)

	if _, err := io.Copy(wc, yamlReader); err != nil {
		return fmt.Errorf("transfer to/from GCS, io.Copy: %w", err)
	}

	// Closing the writer completes the upload
	if err := wc.Close(); err != nil {
		return fmt.Errorf("transfer to/from GCS, failed to upload: %w", err)
	}

	return nil
}

// Import imports manifests from GCS
func (t *GCSTransferer) Import(ctx context.Context) ([]*metadata.Manifest, error) {
	fileContents, err := t.ReadAllFilesWithPrefix(ctx)
	if err != nil {
		return nil, fmt.Errorf("transfer to/from GCS, error reading files: %w", err)
	}

	manifests := []*metadata.Manifest{}
	for fileName, fileBytes := range fileContents {
		manifest := &metadata.Manifest{}
		err = yaml.Unmarshal(fileBytes, manifest)
		if err != nil {
			return nil, fmt.Errorf("import error, failed unmarshalling file '%s': %w", fileName, err)
		}
		manifests = append(manifests, manifest)
	}

	return manifests, nil
}

func (t *GCSTransferer) ReadAllFilesWithPrefix(ctx context.Context) (map[string][]byte, error) {

	files, err := t.listFiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("transfer to/from GCS, error listing files: %w", err)
	}

	results := make(map[string][]byte)
	for _, file := range files {
		data, err := t.readFile(ctx, file)
		if err != nil {
			return nil, fmt.Errorf("transfer to/from GCS, error reading file %s/%s: %v", t.Bucket, file, err)
		}
		results[file] = data
	}

	return results, nil
}

func (t *GCSTransferer) listFiles(ctx context.Context) ([]string, error) {
	var files []string
	it := t.StorageClient.Bucket(t.Bucket).Objects(ctx, &storage.Query{Prefix: t.Path})
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

func (t *GCSTransferer) readFile(ctx context.Context, file string) ([]byte, error) {
	obj := t.StorageClient.Bucket(t.Bucket).Object(file)
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("transfer to/from GCS, failed to create GCS reader: %w", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("transfer to/from GCS, failed to read GCS file data: %w", err)
	}

	return data, nil
}
