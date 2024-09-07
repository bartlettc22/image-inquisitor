package sources

import (
	"context"
	"fmt"

	"github.com/bartlettc22/image-inquisitor/internal/imageUtils"
	"github.com/bartlettc22/image-inquisitor/internal/sources/export"
	exporttypes "github.com/bartlettc22/image-inquisitor/internal/sources/export/types"
	"github.com/bartlettc22/image-inquisitor/internal/sources/types"
)

type SourceImages struct {
	imageList     imageUtils.ImagesList
	sourceReports map[types.ImageSourceType]interface{}
}

type ImageSourcesConfig struct {
	ImageSourceTypes       []types.ImageSourceType
	KubernetesSourceConfig *KubernetesSourceConfig
	FileSourceConfig       *FileSourceConfig
}

func FetchImages(ctx context.Context, config *ImageSourcesConfig) (*SourceImages, error) {

	sourceImages := &SourceImages{
		imageList:     make(imageUtils.ImagesList),
		sourceReports: make(map[types.ImageSourceType]interface{}),
	}

	for _, sourceType := range config.ImageSourceTypes {
		switch sourceType {
		case types.ImageSourceTypeKubernetes:
			kubernetesSource := NewKubernetesSource(config.KubernetesSourceConfig)
			kubernetesSourceReport, err := kubernetesSource.GetReport(ctx)
			if err != nil {
				return nil, err
			}

			for parsedImageName, parsedImage := range kubernetesSourceReport.Images() {
				sourceImages.imageList[parsedImageName] = parsedImage
			}
			sourceImages.sourceReports[sourceType] = kubernetesSourceReport
		case types.ImageSourceTypeFile:
			fileSource := NewFileSource(config.FileSourceConfig)
			fileSourceReport, err := fileSource.GetReport(ctx)
			if err != nil {
				return nil, err
			}

			for parsedImageName, parsedImage := range fileSourceReport.Images() {
				sourceImages.imageList[parsedImageName] = parsedImage
			}
			sourceImages.sourceReports[sourceType] = fileSourceReport
		default:
			return nil, fmt.Errorf("image source unknown")
		}
	}

	return sourceImages, nil
}

func (s *SourceImages) List() imageUtils.ImagesList {
	return s.imageList
}

func (s *SourceImages) GetKubernetesSourceReports(ctx context.Context) (*KubernetesSourceReport, error) {
	if sourceReport, ok := s.sourceReports[types.ImageSourceTypeKubernetes]; ok {
		kubernetesSource, ok := sourceReport.(*KubernetesSourceReport)
		if !ok {
			return nil, fmt.Errorf("report type is not of the right type: %s", types.ImageSourceTypeKubernetes.String())
		}
		return kubernetesSource, nil
	}
	return nil, fmt.Errorf("source report not found: %s", types.ImageSourceTypeKubernetes.String())
}

func (s *SourceImages) Export(ctx context.Context, exporterConfig *export.ExporterConfig) error {
	exporter := export.NewExporter(exporterConfig)
	for sourceType, sourceReport := range s.sourceReports {
		exportableReport, ok := sourceReport.(exporttypes.ExportableReport)
		if !ok {
			return fmt.Errorf("report type is not exportable: %s", sourceType.String())
		}
		exporter.AddReport(sourceType, exportableReport)
	}

	return exporter.Export(ctx)
}
