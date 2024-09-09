package sources

import (
	"context"
	"fmt"

	"github.com/bartlettc22/image-inquisitor/internal/imageUtils"
	"github.com/bartlettc22/image-inquisitor/internal/kubernetes"
	"github.com/bartlettc22/image-inquisitor/internal/sources/types"
)

type KubernetesSource struct {
	*KubernetesSourceConfig
	client *kubernetes.Kubernetes
}

type KubernetesSourceConfig struct {
	IncludeNamespaces []string
	ExcludeNamespaces []string
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

func (s *KubernetesSource) GetReport(ctx context.Context) (map[string]*types.ImageSourceDetails, error) {

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

	details := make(map[string]*types.ImageSourceDetails)

	kubeReport, err := s.client.GetReport()
	if err != nil {
		return nil, fmt.Errorf("error listing images from Kubernetes: %s", err.Error())
	}

	for image, kubeImageReport := range kubeReport {

		details[image] = &types.ImageSourceDetails{
			SourcesByType: map[types.ImageSourceType]interface{}{
				types.ImageSourceTypeKubernetes: kubeImageReport,
			},
		}
	}

	return details, nil
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
