package metadata

import (
	"time"
)

// ManifestObject is an interface for manifest objects
type ManifestObject interface {
	ObjectMetadata() *Metadata
}

// Manifest is a generic manifest object
type Manifest struct {
	Version  string    `json:"version" yaml:"version"`
	Kind     string    `json:"kind" yaml:"kind"`
	Metadata *Metadata `json:"metadata" yaml:"metadata"`
	Spec     any       `json:"spec" yaml:"spec"`
}

// Metadata is a generic metadata object
type Metadata struct {
	Created time.Time `json:"created" yaml:"created"`
}

// Ensure Metadata implements Resource
var _ ManifestObject = &Manifest{}

// ObjectMetadata returns the metadata object
func (m *Manifest) ObjectMetadata() *Metadata {
	return m.Metadata
}

// NewManifest creates a new manifest object
func NewManifest(APIVersion, kind string, spec any) *Manifest {
	return &Manifest{
		Version: APIVersion,
		Kind:    kind,
		Metadata: &Metadata{
			Created: time.Now().UTC(),
		},
		Spec: spec,
	}
}
