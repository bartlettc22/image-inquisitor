package importtypes

import (
	"fmt"

	sourcetypes "github.com/bartlettc22/image-inquisitor/internal/sources/types"
)

type ImportFrom string

const (
	ImportFromFile ImportFrom = "file"
	ImportFromGCS  ImportFrom = "gcs"
)

func (e ImportFrom) String() string {
	return string(e)
}

type ImportFromList map[string]ImportFrom

func (im ImportFromList) Add(from string) error {
	switch from {
	case string(ImportFromFile):
	case string(ImportFromGCS):
	default:
		return fmt.Errorf("invalid import from: %s", from)
	}
	im[from] = ImportFrom(from)
	return nil
}

func (im ImportFromList) Contains(find ImportFrom) bool {
	_, ok := im[string(find)]
	return ok
}

type Importable interface {
	SourceID() string
	SourceDetails(from ImportFrom, filePath string) map[string]*sourcetypes.ImageSourceDetails
}
