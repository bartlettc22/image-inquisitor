package exportapiv1alpha1

import (
	exportapimetadata "github.com/bartlettc22/image-inquisitor/internal/sources/export/api/metadata"
	importtypes "github.com/bartlettc22/image-inquisitor/internal/sources/import/types"
	sourcetypes "github.com/bartlettc22/image-inquisitor/internal/sources/types"
)

const (
	APIVersion = "v1alpha1"
)

type ExportReport struct {
	ExportMetadata exportapimetadata.ExportMetadata `yaml:",inline"`
	Spec           ExportImageList                  `yaml:"spec"`
}

type ExportImageList struct {
	Images map[string]*ExportImage
}

type ExportImage struct {
	Sources map[sourcetypes.ImageSourceType]interface{}
}

func (e *ExportReport) SourceID() string {
	return e.ExportMetadata.SourceID
}

func (e *ExportReport) SourceDetails(from importtypes.ImportFrom, path string) map[string]*sourcetypes.ImageSourceDetails {
	imageSourceDetails := make(map[string]*sourcetypes.ImageSourceDetails)
	for imageName, exportImageList := range e.Spec.Images {
		imageSourceDetails[imageName] = &sourcetypes.ImageSourceDetails{
			ImportMetadata: &sourcetypes.ImageSourceImportMetadata{
				ImportType: from.String(),
				Path:       path,
				Created:    e.ExportMetadata.Created,
			},
			SourcesByType: exportImageList.Sources,
		}
	}

	return imageSourceDetails
}
