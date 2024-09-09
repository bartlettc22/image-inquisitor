package exportsources

import (
	"context"
	"errors"
	"time"

	exportapimetadata "github.com/bartlettc22/image-inquisitor/internal/sources/export/api/metadata"
	exportapiv1alpha1 "github.com/bartlettc22/image-inquisitor/internal/sources/export/api/v1alpha1"
	exporttypes "github.com/bartlettc22/image-inquisitor/internal/sources/export/types"
	sourcetypes "github.com/bartlettc22/image-inquisitor/internal/sources/types"
)

type Exporter struct {
	*ExporterConfig

	images map[string]*exportapiv1alpha1.ExportImage
}

type ExporterConfig struct {
	SourceID         string
	Destinations     exporttypes.ExportDestinationList
	FilePath         string
	GCSBucket        string
	GCSDirectoryPath string
}

func NewExporter(config *ExporterConfig) *Exporter {
	return &Exporter{
		ExporterConfig: config,
		images:         make(map[string]*exportapiv1alpha1.ExportImage),
	}
}

func (e *Exporter) AddImageSourceDetails(fullyQualifiedImageName string, sourcesByType map[sourcetypes.ImageSourceType]interface{}) {
	e.images[fullyQualifiedImageName] = &exportapiv1alpha1.ExportImage{
		Sources: sourcesByType,
	}
}

func (e *Exporter) Export(ctx context.Context) error {
	report := &exportapiv1alpha1.ExportReport{
		ExportMetadata: exportapimetadata.ExportMetadata{
			Version:  exportapiv1alpha1.APIVersion,
			Kind:     exportapimetadata.Kind,
			Created:  time.Now(),
			SourceID: e.SourceID,
		},
		Spec: exportapiv1alpha1.ExportImageList{
			Images: e.images,
		},
	}

	errList := []error{}

	for _, destination := range e.Destinations {
		switch destination {
		case exporttypes.ExportDestinationFile:
			errList = append(errList, e.ExportFile(ctx, report))
		case exporttypes.ExportDestinationGCS:
			errList = append(errList, e.ExportGCS(ctx, report))
		}
	}

	return errors.Join(errList...)
}

func (e *Exporter) exportfileName() string {
	return e.SourceID + ".yaml"
}
