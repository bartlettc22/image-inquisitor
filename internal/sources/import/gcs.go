package importsources

import (
	"context"
	"path/filepath"

	"github.com/bartlettc22/image-inquisitor/internal/gcs"
	log "github.com/sirupsen/logrus"
)

func importGCSInventory(ctx context.Context, bucket, prefix string) (map[string][]byte, error) {

	gcs, err := gcs.NewGCS(ctx)
	if err != nil {
		return nil, err
	}

	files, err := gcs.ReadAllFilesInDir(ctx, bucket, prefix)
	if err != nil {
		return nil, err
	}

	results := make(map[string][]byte)
	for filename, data := range files {
		if filepath.Ext(filename) != ".yaml" {
			log.Warnf("skipping non-yaml file in GCS import path: %s", filename)
			continue
		}

		results[filename] = data
	}

	return results, nil
}
