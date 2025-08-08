package sources

import (
	"github.com/bartlettc22/image-inquisitor/pkg/api/metadata"
	"github.com/bartlettc22/image-inquisitor/pkg/api/v1alpha1"
)

const (

	// SourceListKind is the kind of a source list manifest
	SourceListKind = "SourceList"
)

// SourceSpec is the spec of a source list manifest
type SourcesSpec struct {
	Sources SourceList `json:"sources" yaml:"sources"`
}

// SourceList is a list of sources
type SourceList []*Source

// Source is a single source
type Source struct {
	Type           string `json:"type" yaml:"type"`
	SourceID       string `json:"sourceID" yaml:"sourceID"`
	ImageReference string `json:"imageReference" yaml:"imageReference"`
	SourceDetails  any    `json:"sourceDetails" yaml:"sourceDetails"`
}

// NewSourceListManifest creates a new source list manifest
func NewSourceListManifest(sources SourceList) *metadata.Manifest {
	return metadata.NewManifest(v1alpha1.APIVersion, SourceListKind, &SourcesSpec{
		Sources: sources,
	})
}
