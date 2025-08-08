package export

import (
	"context"

	"github.com/bartlettc22/image-inquisitor/pkg/api/metadata"
)

// Exporter is an interface for exporting manifests
type Exporter interface {
	Export(ctx context.Context, name string, manifest metadata.ManifestObject) error
}
