package registries

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseFullyQualifiedImageReference(t *testing.T) {
	fqReference := "nginx:1.29.0@sha256:84ec966e61a8c7846f509da7eb081c55c1d56817448728924a87ab32f12a72fb"
	fqImageReference, err := ParseFullyQualifiedImageReference(fqReference)
	require.NoError(t, err)
	require.Equal(t, "index.docker.io/library/nginx", fqImageReference.ReferencePrefix)
	require.Equal(t, "1.29.0", fqImageReference.Tag)
	require.Equal(t, "sha256:84ec966e61a8c7846f509da7eb081c55c1d56817448728924a87ab32f12a72fb", fqImageReference.Digest)
	require.Equal(t, "index.docker.io/library/nginx:1.29.0", fqImageReference.TagReference())
	require.Equal(t, "index.docker.io/library/nginx@sha256:84ec966e61a8c7846f509da7eb081c55c1d56817448728924a87ab32f12a72fb", fqImageReference.DigestReference())
}

func TestMakeFullyQualifiedImageReference(t *testing.T) {
	tagReference := "nginx:1.29.0"
	digestReference := "nginx@sha256:84ec966e61a8c7846f509da7eb081c55c1d56817448728924a87ab32f12a72fb"

	fqImageReference, err := MakeFullyQualifiedImageReference(tagReference, digestReference)
	require.NoError(t, err)
	require.Equal(t, "index.docker.io/library/nginx", fqImageReference.ReferencePrefix)
	require.Equal(t, "1.29.0", fqImageReference.Tag)
	require.Equal(t, "sha256:84ec966e61a8c7846f509da7eb081c55c1d56817448728924a87ab32f12a72fb", fqImageReference.Digest)
	require.Equal(t, "index.docker.io/library/nginx:1.29.0", fqImageReference.TagReference())
	require.Equal(t, "index.docker.io/library/nginx@sha256:84ec966e61a8c7846f509da7eb081c55c1d56817448728924a87ab32f12a72fb", fqImageReference.DigestReference())
}

func TestMakeFullyQualifiedImageReferenceMismatch(t *testing.T) {
	tagReference := "nginx:1.29.0"
	digestReference := "grafana@sha256:84ec966e61a8c7846f509da7eb081c55c1d56817448728924a87ab32f12a72fb"

	_, err := MakeFullyQualifiedImageReference(tagReference, digestReference)
	require.ErrorContains(t, err, "fully qualified image references must be from the same registry and repository")
}
