package main

import (
	"flag"
	"strings"

	log "github.com/sirupsen/logrus"
)

type ImageListSource string

const (
	ImageListSourceUnknown    ImageListSource = "unknown"
	ImageListSourceKubernetes ImageListSource = "kubernetes"
	ImageListSourceFile       ImageListSource = "file"
)

func (s ImageListSource) IsValid() bool {
	switch s {
	case ImageListSourceKubernetes, ImageListSourceFile:
		return true
	default:
		return false
	}
}

type Config struct {
	LogLevel                       string
	LogJSON                        bool
	ImageSourceStr                 string
	ImageSource                    ImageListSource
	ImageSourceFilePath            string
	RunTrivy                       bool
	RunRegistry                    bool
	IncludeKubernetesNamespacesStr string
	IncludeKubernetesNamespaces    []string
}

func loadConfig() Config {
	config := Config{}

	flag.StringVar(&config.LogLevel,
		"log-level",
		"info",
		"The desired logging level.  One of panic, fatal, error, warn, info, debug, trace.")

	flag.BoolVar(&config.LogJSON,
		"log-json",
		true,
		"Whether to log JSON output")

	flag.BoolVar(&config.RunTrivy,
		"run-trivy",
		true,
		"Whether to run Trivy issue scanner.")

	flag.BoolVar(&config.RunRegistry,
		"run-registry",
		true,
		"Whether to run Registry metadata.")

	flag.StringVar(&config.ImageSourceStr,
		"image-source",
		"kubernetes",
		"Source of images to scan.  Can be one of 'kubernetes' or 'file'. If 'file', must specify 'image-source-file-path' parameter.")

	flag.StringVar(&config.ImageSourceFilePath,
		"image-source-file-path",
		"",
		"Path of file containing list of images to scan")

	flag.StringVar(&config.IncludeKubernetesNamespacesStr,
		"include-kubernetes-namespaces",
		"",
		"Comma-separated list of Kubernetes namespaces to scan if --image-source=kubernetes")

	flag.Parse()

	config.ImageSource = ImageListSource(config.ImageSourceStr)

	if !config.ImageSource.IsValid() {
		log.Fatalf("'image-source' parameter invalid. run with '--help' to list valid values")
	}

	if config.ImageSource == ImageListSourceFile {
		if config.ImageSourceFilePath == "" {
			log.Fatalf("must specify 'image-source-file-path' parameter when 'image-source=file'")
		}
	}

	if config.IncludeKubernetesNamespacesStr != "" {
		config.IncludeKubernetesNamespaces = strings.Split(config.IncludeKubernetesNamespacesStr, ",")
	}

	return config
}
