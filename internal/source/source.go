package source

import (
	sourcesapi "github.com/bartlettc22/image-inquisitor/pkg/api/v1alpha1/sources"
)

type SourceGenerator interface {
	Generate() (sourcesapi.SourceList, error)
}
