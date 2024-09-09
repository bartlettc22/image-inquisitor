package types

import "time"

type ImageSourceType string

const (
	ImageSourceTypeKubernetes ImageSourceType = "kubernetes"
	ImageSourceTypeFile       ImageSourceType = "file"
)

func (s ImageSourceType) String() string {
	return string(s)
}

func (s ImageSourceType) IsValid() bool {
	switch s {
	case ImageSourceTypeKubernetes, ImageSourceTypeFile:
		return true
	default:
		return false
	}
}

type ImageSourceDetails struct {
	ImportMetadata *ImageSourceImportMetadata      `json:"importMetadata" yaml:"importMetadata"`
	SourcesByType  map[ImageSourceType]interface{} `json:"sources" yaml:"sources"`
}

type ImageSourceImportMetadata struct {
	ImportType string    `json:"type"`
	Path       string    `json:"path"`
	Created    time.Time `json:"created"`
}
