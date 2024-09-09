package importsources

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

func importFileInventory(ctx context.Context, dirPath string) (map[string][]byte, error) {

	// Check if the directory exists and is indeed a directory
	info, err := os.Stat(dirPath)
	if err != nil {
		return nil, fmt.Errorf("could not stat directory for import: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory, cannot import", dirPath)
	}

	results := make(map[string][]byte)

	// Read the directory
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("could not read directory for import: %w", err)
	}

	for _, file := range files {
		// Skip directories
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(dirPath, file.Name())

		if filepath.Ext(filePath) != "yaml" {
			log.Warnf("skipping non-yaml file in file import path: %s", filePath)
			continue
		}

		// Read file content
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("could not read file %s for import: %w", filePath, err)
		}

		// Append content to result slice
		results[filePath] = content
	}

	return results, nil
}
