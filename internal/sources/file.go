package sources

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/bartlettc22/image-inquisitor/internal/imageUtils"
	log "github.com/sirupsen/logrus"
)

type FileSourceReport struct {
	sourceFilePath string
	imagesList     imageUtils.ImagesList
}

type FileSourceExportReport struct {
	FilePath string `yaml:"filePath"`
}

type FileSource struct {
	*FileSourceConfig
}

type FileSourceConfig struct {
	SourceFilePath    string
	ExcludeRegistries map[string]struct{}
}

func NewFileSource(config *FileSourceConfig) *FileSource {
	return &FileSource{
		FileSourceConfig: config,
	}
}

func (s *FileSource) GetReport(ctx context.Context) (*FileSourceReport, error) {

	fileSourceReport := &FileSourceReport{
		sourceFilePath: s.SourceFilePath,
		imagesList:     make(imageUtils.ImagesList),
	}

	file, err := os.Open(s.SourceFilePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file '%s': %v", s.SourceFilePath, err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		image := scanner.Text()
		if strings.TrimSpace(image) != "" {
			parsedImage, err := imageUtils.ParseImage(image)
			if err != nil {
				log.Errorf("error parsing image %s, skipping: %v", image, err)
				continue
			}

			if s.ExcludeRegistries != nil {
				if _, ok := s.ExcludeRegistries[parsedImage.Registry]; ok {
					continue
				}
			}
			fileSourceReport.imagesList[parsedImage.FullName(false)] = parsedImage
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: '%s': %v", s.SourceFilePath, err)
	}

	return fileSourceReport, nil
}

func (s *FileSourceReport) Images() imageUtils.ImagesList {
	return s.imagesList
}

func (s *FileSourceReport) Export() map[string]interface{} {
	exportReport := make(map[string]interface{})
	for parsedImageName := range s.imagesList {
		exportReport[parsedImageName] = &FileSourceExportReport{
			FilePath: s.sourceFilePath,
		}
	}
	return exportReport
}
