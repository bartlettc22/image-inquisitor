package exportsources

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"

	"cloud.google.com/go/storage"
	exportapiv1alpha1 "github.com/bartlettc22/image-inquisitor/internal/sources/export/api/v1alpha1"
	"gopkg.in/yaml.v2"
)

func (e *Exporter) ExportGCS(ctx context.Context, report *exportapiv1alpha1.ExportReport) error {

	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("exportGCS storage.NewClient: %v", err)
	}
	defer client.Close()

	bucket := client.Bucket(e.ExporterConfig.GCSBucket)
	wc := bucket.Object(path.Join(e.GCSDirectoryPath, e.exportfileName())).NewWriter(ctx)

	yamlOut, err := yaml.Marshal(report)
	if err != nil {
		return fmt.Errorf("exportGCS error marshalling export report: %v", err)
	}
	yamlReader := bytes.NewReader(yamlOut)

	if _, err := io.Copy(wc, yamlReader); err != nil {
		return fmt.Errorf("exportGCS io.Copy: %v", err)
	}

	// Close the writer to complete the upload
	if err := wc.Close(); err != nil {
		return fmt.Errorf("exportGCS writer.Close: %v", err)
	}

	return nil
}
