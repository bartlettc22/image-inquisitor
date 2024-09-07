package types

type ImageSourceType string

const (
	ImageSourceTypeKubernetes ImageSourceType = "kubernetes"
	ImageSourceTypeFile       ImageSourceType = "file"
	ImageSourceTypeGCS        ImageSourceType = "gcs"
)

func (s ImageSourceType) String() string {
	return string(s)
}

func (s ImageSourceType) IsValid() bool {
	switch s {
	case ImageSourceTypeKubernetes, ImageSourceTypeFile, ImageSourceTypeGCS:
		return true
	default:
		return false
	}
}
