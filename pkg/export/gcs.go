package export

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"

	"cloud.google.com/go/storage"
	"github.com/bartlettc22/image-inquisitor/pkg/api/metadata"
	yaml "github.com/goccy/go-yaml"
)

// GCSExporterConfig is the configuration for exporting to GCS
type GCSExporterConfig struct {
	StorageClient *storage.Client
	Bucket        string
	Path          string
}

// GCSExporter is an exporter for exporting to GCS
type GCSExporter struct {
	*GCSExporterConfig
}

// Ensure GCSExporter implements Exporter
var _ Exporter = &GCSExporter{}

// NewGCSExporter creates a new GCS exporter
func NewGCSExporter(config *GCSExporterConfig) (*GCSExporter, error) {

	if config.Bucket == "" {
		return nil, fmt.Errorf("export to GCS, bucket not specified")
	}

	client := config.StorageClient
	if client == nil {
		client, err := storage.NewClient(context.Background())
		if err != nil {
			return nil, fmt.Errorf("export to GCS, failed to create storage client: %w", err)
		}
		defer client.Close()
	}

	return &GCSExporter{
		config,
	}, nil
}

// Export exports a manifest to GCS with the given name as a component of the file name
func (e *GCSExporter) Export(ctx context.Context, name string, manifest metadata.ManifestObject) error {

	fileName := filepath.Join(e.Path, name, ".yaml")

	bucket := e.StorageClient.Bucket(e.Bucket)
	wc := bucket.Object(fileName).NewWriter(ctx)

	yamlBytes, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("export to GCS, error marshalling export: %w", err)
	}
	yamlReader := bytes.NewReader(yamlBytes)

	if _, err := io.Copy(wc, yamlReader); err != nil {
		return fmt.Errorf("export to GCS, io.Copy: %w", err)
	}

	// Closing the writer completes the upload
	if err := wc.Close(); err != nil {
		return fmt.Errorf("export to GCS, failed to upload: %w", err)
	}

	return nil
}
