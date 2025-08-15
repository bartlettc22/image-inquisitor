package inventory

import (
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/registries"
	"github.com/bartlettc22/image-inquisitor/pkg/api/v1alpha1/sources"
	log "github.com/sirupsen/logrus"
)

type AddSourceResult struct {
	ReferencePrefix string
	Registry        string
	Repo            string
	Digest          string
	Created         time.Time
	Source          *sources.Source
}

func newAddSourceFunc(source *sources.Source) func() (any, error) {
	return func() (any, error) {
		log.WithField("ref", source.ImageReference).Debug("adding source")
		image, err := registries.NewImage(source.ImageReference)
		if err != nil {
			return nil, err
		}

		result := &AddSourceResult{
			Source:          source,
			ReferencePrefix: image.RefPrefix(),
			Registry:        image.Registry(),
			Repo:            image.Repository(),
		}

		result.Digest, err = image.Digest()
		if err != nil {
			return nil, err
		}

		result.Created, err = image.Created()
		if err != nil {
			return nil, err
		}

		return result, nil
	}
}
