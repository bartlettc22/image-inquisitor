package inventory

import (
	"fmt"

	"github.com/bartlettc22/image-inquisitor/internal/registries"
	log "github.com/sirupsen/logrus"
)

type getLatestSemverResult struct {
	ReferencePrefix    string
	LatestSemverTag    string
	LatestSemverDigest string
}

func newGetLatestSemverFunc(referencePrefix string) func() (any, error) {
	return func() (interface{}, error) {
		log.WithField("referencePrefix", referencePrefix).Debug("fetching latest semver tag")

		var err error
		result := &getLatestSemverResult{
			ReferencePrefix: referencePrefix,
		}

		result.LatestSemverTag, err = registries.FetchLatestSemverTagByStr(referencePrefix)
		if err != nil {
			return nil, err
		}

		if result.LatestSemverTag != "" {
			ref := fmt.Sprintf("%s:%s", referencePrefix, result.LatestSemverTag)
			image, err := registries.NewImage(ref)
			if err != nil {
				return nil, err
			}

			result.LatestSemverDigest, err = image.Digest()
			if err != nil {
				return nil, err
			}
		}

		return result, nil
	}
}
