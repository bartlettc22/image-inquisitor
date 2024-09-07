package exportapiv1alpha1

import (
	exportapimetadata "github.com/bartlettc22/image-inquisitor/internal/sources/export/api/metadata"
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
