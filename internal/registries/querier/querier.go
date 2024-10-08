package querier

import (
	"github.com/bartlettc22/image-inquisitor/internal/imageUtils"
	"github.com/bartlettc22/image-inquisitor/internal/registries"
	"github.com/bartlettc22/image-inquisitor/internal/registries/docker"
	"github.com/bartlettc22/image-inquisitor/internal/registries/ghcr"
	"github.com/bartlettc22/image-inquisitor/internal/registries/quay"
	log "github.com/sirupsen/logrus"
)

type Registry interface {
	IsRegistry(registry string) bool
	FetchReport(image *imageUtils.Image) (*registries.RegistryImageReport, error)
}

type RegistryQuerier struct {
	registries []Registry
}

func NewRegistryQuerier() *RegistryQuerier {
	rq := &RegistryQuerier{}
	rq.addRegistry(quay.NewRegistry())
	rq.addRegistry(docker.NewRegistry())
	rq.addRegistry(ghcr.NewRegistry())
	return rq
}

func (rq *RegistryQuerier) addRegistry(registry Registry) {
	rq.registries = append(rq.registries, registry)
}

func (rq *RegistryQuerier) FetchReport(image *imageUtils.Image) (*registries.RegistryImageReport, error) {
	log.Debugf("fetching image metadata from registry for: %s", image.Image)
	for _, reg := range rq.registries {
		if reg.IsRegistry(image.Registry) {
			report, err := reg.FetchReport(image)
			if err != nil {
				return nil, err
			}
			report.Registry = image.Registry
			report.Owner = image.Owner
			report.Repository = image.Repository
			log.Debugf("DONE fetching image metadata from registry for: %s", image.Image)
			return report, nil
		}
	}

	// no matching registry found
	log.Warnf("registry not able to be queried for latest tag for: %s", image.Image)
	return &registries.RegistryImageReport{
		Tag: image.Tag,
	}, nil
}
