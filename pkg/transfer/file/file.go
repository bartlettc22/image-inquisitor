package transferfile

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bartlettc22/image-inquisitor/pkg/api/metadata"
	"github.com/bartlettc22/image-inquisitor/pkg/transfer"
	"github.com/bartlettc22/image-inquisitor/pkg/utils"
	yaml "github.com/goccy/go-yaml"
)

// FileTransfererConfig is the configuration for exporting/exporting to a file/directory
type FileTransfererConfig struct {
	Path string
}

// FileTransferer is used for importing/exporting manifests to/from a file/directory
type FileTransferer struct {
	*FileTransfererConfig
	pathIsFile bool
	pathIsDir  bool
	pathExists bool
}

// Ensure FileTransferer implements Transferer
var _ transfer.Transferer = &FileTransferer{}

// NewFileTransferer creates a new file importer/exporter
func NewFileTransferer(config *FileTransfererConfig) (*FileTransferer, error) {

	if config.Path == "" {
		return nil, fmt.Errorf("transfer to/from file/directory, path not specified")
	}

	isFile, isDir, exists := utils.CheckPath(config.Path)

	return &FileTransferer{
		FileTransfererConfig: config,
		pathIsFile:           isFile,
		pathIsDir:            isDir,
		pathExists:           exists,
	}, nil
}

// Export exports a manifest to a file with the given name as a component of the file name
func (t *FileTransferer) Export(ctx context.Context, name string, manifest *metadata.Manifest) error {

	if !t.pathExists {
		err := os.MkdirAll(t.Path, 0755)
		if err != nil {
			return fmt.Errorf("export error, failed creating directory '%s': %w", t.Path, err)
		}
	} else {
		if !t.pathIsDir {
			return fmt.Errorf("export error, path '%s' is not a directory", t.Path)
		}
	}

	fileName := filepath.Join(t.Path, name+".yaml")

	yamlBytes, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("transfer to/from file/directory, error marshalling resource: %w", err)
	}

	err = os.WriteFile(fileName, yamlBytes, 0644)
	if err != nil {
		return fmt.Errorf("transfer to/from file/directory, error writing file: %w", err)
	}

	return nil
}

// Import imports manifests from a file/directory
func (t *FileTransferer) Import(ctx context.Context) ([]*metadata.Manifest, error) {

	if !t.pathExists {
		return nil, fmt.Errorf("import error, path '%s' does not exist", t.Path)
	}

	fileNames := []string{}

	if t.pathIsDir {
		entries, err := os.ReadDir(t.Path)
		if err != nil {
			return nil, fmt.Errorf("import error, failed reading directory '%s': %w", t.Path, err)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				fileNames = append(fileNames, filepath.Join(t.Path, entry.Name()))
			}
		}
	} else {
		fileNames = append(fileNames, t.Path)
	}

	var manifests []*metadata.Manifest
	for _, fileName := range fileNames {

		// Check if file extension is yaml
		if !strings.HasSuffix(fileName, ".yaml") {
			continue
		}

		yamlBytes, err := os.ReadFile(fileName)
		if err != nil {
			return nil, fmt.Errorf("import error, failed reading file '%s': %w", fileName, err)
		}

		manifest := &metadata.Manifest{}
		err = yaml.Unmarshal(yamlBytes, manifest)
		if err != nil {
			return nil, fmt.Errorf("import error, failed unmarshalling file '%s': %w", fileName, err)
		}

		manifests = append(manifests, manifest)
	}

	return manifests, nil
}
