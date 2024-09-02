package main

import (
	"github.com/bartlettc22/image-inquisitor/internal/imageUtils"
	"github.com/bartlettc22/image-inquisitor/internal/kubernetes"
	"github.com/bartlettc22/image-inquisitor/internal/registries"
	"github.com/bartlettc22/image-inquisitor/internal/trivy"
)

type FinalReport map[string]*ImageReport

type ImageReport struct {
	Image            *imageUtils.Image
	KubernetesReport *kubernetes.KubernetesImageReport
	TrivyReport      *trivy.TrivyImageReport
	RegistryReport   *registries.ImageReport
}

func (r FinalReport) Images() []string {
	images := []string{}
	for image, _ := range r {
		images = append(images, image)
	}
	return images
}
