package sources

import (
	"encoding/json"
	"fmt"

	"github.com/bartlettc22/image-inquisitor/pkg/api/metadata"
	"github.com/bartlettc22/image-inquisitor/pkg/api/v1alpha1"
)

const (

	// SourceListKind is the kind of a source list manifest
	SourceListKind = "SourceList"

	// RegistryLatestSemverSourceType is a reserved type of a registry latest semver source
	RegistryLatestSemverSourceType = "registryLatestSemver"
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

func ManifestToSourceList(manifest *metadata.Manifest) (SourceList, error) {

	if manifest.Version != v1alpha1.APIVersion {
		return nil, fmt.Errorf("invalid manifest version: %s", manifest.Version)
	}

	if manifest.Kind != SourceListKind {
		return nil, fmt.Errorf("invalid manifest kind: %s", manifest.Kind)
	}

	spec := &SourcesSpec{}

	// Marshal and unmarshal to get the spec
	specBytes, err := json.Marshal(manifest.Spec)
	if err != nil {
		return nil, fmt.Errorf("error marshalling manifest spec: %w", err)
	}
	if err := json.Unmarshal(specBytes, &spec); err != nil {
		return nil, fmt.Errorf("error unmarshalling manifest spec: %w", err)
	}

	return spec.Sources, nil
}
