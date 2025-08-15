package sources

const (
	// KubernetesSourceType is the type of Kubernetes source
	KubernetesSourceType = "kubernetes"
)

// KubernetesSource is a source of images from Kubernetes
type KubernetesSource struct {

	// Namespace is the Kubernetes namespace that the image was found in
	Namespace string `yaml:"namespace" json:"namespace"`

	// Kind is the Kubernetes kind that the image was found in
	Kind string `yaml:"kind" json:"kind"`

	// Name is the Kubernetes resource name that the image was found in
	Name string `yaml:"name" json:"name"`

	// Container is the Kubernetes container name that the image was found in
	Container string `yaml:"container" json:"container"`

	// IsInit is whether the container is an init container
	IsInit bool `yaml:"isInit" json:"isInit"`
}
