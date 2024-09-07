package types

type ImageSource string

const (
	ImageSourceKubernetes ImageSource = "kubernetes"
	ImageSourceFile       ImageSource = "file"
	ImageSourceGCS        ImageSource = "gcs"
)

func (s ImageSource) String() string {
	return string(s)
}
