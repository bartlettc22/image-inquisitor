package export

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
	reports map[sourcetypes.ImageSourceType]exporttypes.ExportableReport
}

type ExporterConfig struct {
	ExternalID   string
	Destinations exporttypes.ExportDestinationList
	FilePath     string
	GCSBucket    string
}

func NewExporter(config *ExporterConfig) *Exporter {
	return &Exporter{
		ExporterConfig: config,
		reports:        make(map[sourcetypes.ImageSourceType]exporttypes.ExportableReport),
	}
}

func (e *Exporter) AddReport(sourceType sourcetypes.ImageSourceType, report exporttypes.ExportableReport) {
	e.reports[sourceType] = report
}

func (e *Exporter) Export(ctx context.Context) error {

	spec := exportapiv1alpha1.ExportImageList{
		Images: make(map[string]*exportapiv1alpha1.ExportImage),
	}

	for sourceType, report := range e.reports {
		for image, sourceReport := range report.Export() {
			if _, ok := spec.Images[image]; !ok {
				spec.Images[image] = &exportapiv1alpha1.ExportImage{
					Sources: make(map[sourcetypes.ImageSourceType]interface{}),
				}
			}
			spec.Images[image].Sources[sourceType] = sourceReport
		}
	}

	report := &exportapiv1alpha1.ExportReport{
		ExportMetadata: exportapimetadata.ExportMetadata{
			Version:    exportapiv1alpha1.APIVersion,
			Kind:       "ExportReport",
			Created:    time.Now(),
			ExternalID: e.ExternalID,
		},
		Spec: spec,
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
	return e.ExternalID + ".yaml"
}
