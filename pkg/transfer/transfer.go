package transfer

import (
	"context"

	"github.com/bartlettc22/image-inquisitor/pkg/api/metadata"
)

// Transferer is an interface for importing and exporting manifests
type Transferer interface {
	Export(ctx context.Context, name string, manifest *metadata.Manifest) error
	Import(ctx context.Context) ([]*metadata.Manifest, error)
}
