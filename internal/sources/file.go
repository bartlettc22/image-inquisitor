package sources

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/bartlettc22/image-inquisitor/internal/sources/types"
)

type FileSourceDetails struct {
	FilePath string `yaml:"filePath"`
}

type FileSource struct {
	*FileSourceConfig
}

type FileSourceConfig struct {
	SourceFilePath string
}

func NewFileSource(config *FileSourceConfig) *FileSource {
	return &FileSource{
		FileSourceConfig: config,
	}
}

func (s *FileSource) GetReport(ctx context.Context) (map[string]*types.ImageSourceDetails, error) {

	details := make(map[string]*types.ImageSourceDetails)

	file, err := os.Open(s.SourceFilePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file '%s': %v", s.SourceFilePath, err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		image := scanner.Text()
		if trimmedImage := strings.TrimSpace(image); trimmedImage != "" {

			details[trimmedImage] = &types.ImageSourceDetails{
				SourcesByType: map[types.ImageSourceType]interface{}{
					types.ImageSourceTypeFile: &FileSourceDetails{
						FilePath: s.SourceFilePath,
					},
				},
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: '%s': %v", s.SourceFilePath, err)
	}

	return details, nil
}
