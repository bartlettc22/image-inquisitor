package file

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	sourcesapi "github.com/bartlettc22/image-inquisitor/pkg/api/v1alpha1/sources"
)

type FileSourceGenerator struct {
	sourceID    string
	fs          fs.FS
	filePath    string
	relFilePath string
}

func NewFileSourceGenerator(sourceID, filePath string) (*FileSourceGenerator, error) {

	if sourceID == "" {
		return nil, fmt.Errorf("sourceID must be specified")
	}

	if filePath == "" {
		return nil, fmt.Errorf("filePath must be specified")
	}

	return &FileSourceGenerator{
		sourceID:    sourceID,
		filePath:    filePath,
		fs:          os.DirFS(filepath.Dir(filePath)),
		relFilePath: filepath.Base(filePath),
	}, nil
}

// GetReport retrieves all container images used by resources in all namespaces
func (sg *FileSourceGenerator) Generate() (sourcesapi.SourceList, error) {

	file, err := sg.fs.Open(sg.relFilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 1
	sources := sourcesapi.SourceList{}
	for scanner.Scan() {
		lineText := strings.TrimSpace(scanner.Text())
		if lineText != "" {
			sources = append(sources, &sourcesapi.Source{
				Type:           sourcesapi.FileSourceType,
				SourceID:       sg.sourceID,
				ImageReference: lineText,
				SourceDetails: &sourcesapi.FileSource{
					File: sg.filePath,
					Line: lineNumber,
				},
			})
		}
		lineNumber++
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file %s: %v", sg.filePath, err)
	}

	return sources, nil
}
