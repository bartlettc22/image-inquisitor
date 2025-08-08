package export

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bartlettc22/image-inquisitor/pkg/api/metadata"
	yaml "github.com/goccy/go-yaml"
)

// ExportToFileConfig is the configuration for exporting to a file
type ExportToFileConfig struct {
	Path string
}

// FileExporter is an exporter for exporting to a file
type FileExporter struct {
	*ExportToFileConfig
}

// Ensure FileExporter implements Exporter
var _ Exporter = &FileExporter{}

// NewFileExporter creates a new file exporter
func NewFileExporter(config *ExportToFileConfig) (*FileExporter, error) {

	if config.Path == "" {
		return nil, fmt.Errorf("export to file, path not specified")
	}

	err := os.MkdirAll(config.Path, 0755)
	if err != nil {
		return nil, fmt.Errorf("export to file, error creating directory '%s': %w", config.Path, err)
	}

	return &FileExporter{
		config,
	}, nil
}

// Export exports a manifest to a file with the given name as a component of the file name
func (e *FileExporter) Export(ctx context.Context, name string, manifest metadata.ManifestObject) error {

	fileName := filepath.Join(e.Path, name+".yaml")

	yamlBytes, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("export to file, error marshalling resource: %w", err)
	}

	err = os.WriteFile(fileName, yamlBytes, 0644)
	if err != nil {
		return fmt.Errorf("export to file, error writing file: %w", err)
	}

	return nil
}
