package file

import (
	"testing"
	"testing/fstest"

	sourcesapi "github.com/bartlettc22/image-inquisitor/pkg/api/v1alpha1/sources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileSource(t *testing.T) {

	imageList := `python:3
		hashicorp/vault:1.14.8
		quay.io/thanos/thanos:v0.36.0
	`

	mockFS := fstest.MapFS{
		"images.txt": &fstest.MapFile{
			Data: []byte(imageList),
			Mode: 0644,
		},
	}

	fileName := "images.txt"
	fileSourceGenerator, err := NewFileSourceGenerator("x", fileName)
	require.NoError(t, err)
	fileSourceGenerator.fs = mockFS

	sources, err := fileSourceGenerator.Generate()
	require.NoError(t, err)

	require.Len(t, sources, 3)

	assert.Equal(t, "file", sources[0].Type)
	assert.Equal(t, "python:3", sources[0].ImageReference)
	assert.Equal(t, fileName, sources[0].SourceDetails.(*sourcesapi.FileSource).File)
	assert.Equal(t, 1, sources[0].SourceDetails.(*sourcesapi.FileSource).Line)

	assert.Equal(t, "file", sources[1].Type)
	assert.Equal(t, "hashicorp/vault:1.14.8", sources[1].ImageReference)
	assert.Equal(t, fileName, sources[1].SourceDetails.(*sourcesapi.FileSource).File)
	assert.Equal(t, 2, sources[1].SourceDetails.(*sourcesapi.FileSource).Line)

	assert.Equal(t, "file", sources[2].Type)
	assert.Equal(t, "quay.io/thanos/thanos:v0.36.0", sources[2].ImageReference)
	assert.Equal(t, fileName, sources[2].SourceDetails.(*sourcesapi.FileSource).File)
	assert.Equal(t, 3, sources[2].SourceDetails.(*sourcesapi.FileSource).Line)

}
