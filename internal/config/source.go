package config

import (
	"fmt"

	"github.com/bartlettc22/image-inquisitor/internal/source"
	"github.com/bartlettc22/image-inquisitor/internal/source/file"
	"github.com/bartlettc22/image-inquisitor/internal/source/kubernetes"
	sourcesapi "github.com/bartlettc22/image-inquisitor/pkg/api/v1alpha1/sources"
	"github.com/spf13/viper"
)

func SourceGeneratorFromConfig() (source.SourceGenerator, error) {
	sourceID := viper.GetString("source-id")
	if sourceID == "" {
		return nil, fmt.Errorf("--source-id not specified")
	}

	source := viper.GetString("source")
	if source == "" {
		return nil, fmt.Errorf("--source not specified")
	}

	switch source {
	case sourcesapi.KubernetesSourceType:
		includeNamespace := viper.GetStringSlice("source-kubernetes-include-namespaces")
		excludeNamespace := viper.GetStringSlice("source-kubernetes-exclude-namespaces")
		return kubernetes.NewKubernetesSourceGenerator(&kubernetes.KubernetesSourceGeneratorConfig{
			SourceID:          sourceID,
			IncludeNamespaces: includeNamespace,
			ExcludeNamespaces: excludeNamespace,
		})
	case sourcesapi.FileSourceType:
		path := viper.GetString("source-file-path")
		if path == "" {
			return nil, fmt.Errorf("--source-file-path must be specified")
		}

		return file.NewFileSourceGenerator(sourceID, path)
	default:
		return nil, fmt.Errorf("invalid source '%s' specified", source)
	}
}
