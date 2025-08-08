package registries

import (
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
)

type FullyQualifiedImageReference struct {
	ReferencePrefix string `yaml:"referencePrefix" json:"referencePrefix"`
	Tag             string `yaml:"tag" json:"tag"`
	Digest          string `yaml:"digest" json:"digest"`
}

func (fqr *FullyQualifiedImageReference) TagReference() string {
	return fmt.Sprintf("%s:%s", fqr.ReferencePrefix, fqr.Tag)
}

func (fqr *FullyQualifiedImageReference) DigestReference() string {
	return fmt.Sprintf("%s@%s", fqr.ReferencePrefix, fqr.Digest)
}

func ParseFullyQualifiedImageReference(fqReference string) (*FullyQualifiedImageReference, error) {
	digestParts := strings.Split(fqReference, "@")
	if len(digestParts) != 2 {
		return nil, fmt.Errorf("invalid fully qualified image reference (malformed digest): '%s'", fqReference)
	}
	tagRef, err := name.ParseReference(digestParts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid fully qualified image reference: '%s': %w", fqReference, err)
	}

	return &FullyQualifiedImageReference{
		ReferencePrefix: tagRef.Context().String(),
		Tag:             tagRef.Identifier(),
		Digest:          digestParts[1],
	}, nil
}

func MakeFullyQualifiedImageReference(tagReference string, digestReference string) (*FullyQualifiedImageReference, error) {
	tagRef, err := name.ParseReference(tagReference)
	if err != nil {
		return nil, fmt.Errorf("invalid image reference (tag): '%s': %w", tagReference, err)
	}

	digestRef, err := name.ParseReference(digestReference)
	if err != nil {
		return nil, fmt.Errorf("invalid image reference (digest): '%s': %w", digestReference, err)
	}

	tagReferencePrefix := tagRef.Context().String()
	digestReferencePrefix := digestRef.Context().String()

	if tagReferencePrefix != digestReferencePrefix {
		return nil, fmt.Errorf("fully qualified image references must be from the same registry and repository: %s != %s", tagReferencePrefix, digestReferencePrefix)
	}

	return &FullyQualifiedImageReference{
		ReferencePrefix: tagReferencePrefix,
		Tag:             tagRef.Identifier(),
		Digest:          digestRef.Identifier(),
	}, nil
}
