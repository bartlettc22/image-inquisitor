package querier

import (
	"github.com/bartlettc22/image-inquisitor/internal/imageUtils"
	"github.com/bartlettc22/image-inquisitor/internal/registries"
	docker_io "github.com/bartlettc22/image-inquisitor/internal/registries/docker-io"
	quay_io "github.com/bartlettc22/image-inquisitor/internal/registries/quay-io"
)

type Registry interface {
	IsRegistry(registry string) bool
	FetchReport(image *imageUtils.Image) (*registries.ImageReport, error)
}

type RegistryQuerier struct {
	registries []Registry
}

func NewRegistryQuerier() *RegistryQuerier {
	rq := &RegistryQuerier{}
	rq.addRegistry(quay_io.NewRegistry())
	rq.addRegistry(docker_io.NewRegistry())
	return rq
}

func (rq *RegistryQuerier) addRegistry(registry Registry) {
	rq.registries = append(rq.registries, registry)
}

func (rq *RegistryQuerier) FetchReport(image *imageUtils.Image) (*registries.ImageReport, error) {
	for _, reg := range rq.registries {
		if reg.IsRegistry(image.Registry) {
			report, err := reg.FetchReport(image)
			if err != nil {
				return nil, err
			}
			return report, nil
		}
	}

	// no matching registry found
	return &registries.ImageReport{
		CurrentTag: image.Tag,
	}, nil
}
