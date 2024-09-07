package sources

import (
	"context"
	"fmt"

	"github.com/bartlettc22/image-inquisitor/internal/imageUtils"
	"github.com/bartlettc22/image-inquisitor/internal/kubernetes"
	log "github.com/sirupsen/logrus"
)

type KubernetesSource struct {
	*KubernetesSourceConfig
	client *kubernetes.Kubernetes
}

type KubernetesSourceConfig struct {
	IncludeNamespaces []string
	ExcludeNamespaces []string
	ExcludeRegistries map[string]struct{}
}

type KubernetesSourceReport struct {
	imageReports map[string]*kubernetes.KubernetesImageReport
	imagesList   imageUtils.ImagesList
}

func NewKubernetesSource(config *KubernetesSourceConfig) *KubernetesSource {
	return &KubernetesSource{
		KubernetesSourceConfig: config,
	}
}

func (s *KubernetesSource) GetReport(ctx context.Context) (*KubernetesSourceReport, error) {

	var err error

	if s.client == nil {
		s.client, err = kubernetes.NewKubernetes(&kubernetes.KubernetesConfig{
			IncludeNamespaces: s.IncludeNamespaces,
			ExcludeNamespaces: s.ExcludeNamespaces,
		})
		if err != nil {
			return nil, fmt.Errorf("%v", err)
		}
	}

	kubernetesSourceReport := &KubernetesSourceReport{
		imageReports: make(map[string]*kubernetes.KubernetesImageReport),
		imagesList:   make(imageUtils.ImagesList),
	}

	kubeReport, err := s.client.GetReport()
	if err != nil {
		return nil, fmt.Errorf("error listing images from Kubernetes: %s", err.Error())
	}

	for image, kubeImageReport := range kubeReport {

		parsedImage, err := imageUtils.ParseImage(image)
		if err != nil {
			log.Errorf("error parsing image %s, skipping: %v", image, err)
			continue
		}

		if _, ok := s.ExcludeRegistries[parsedImage.Registry]; ok {
			continue
		}

		kubernetesSourceReport.imageReports[parsedImage.FullName(false)] = kubeImageReport
		kubernetesSourceReport.imagesList[parsedImage.FullName(false)] = parsedImage
	}

	return kubernetesSourceReport, nil
}

func (s *KubernetesSourceReport) KubeReports() map[string]*kubernetes.KubernetesImageReport {
	return s.imageReports
}

func (s *KubernetesSourceReport) Images() imageUtils.ImagesList {
	return s.imagesList
}

func (s *KubernetesSourceReport) Export() map[string]interface{} {
	exportReport := make(map[string]interface{})
	for imageName, imageReport := range s.imageReports {
		exportReport[imageName] = imageReport
	}
	return exportReport
}
