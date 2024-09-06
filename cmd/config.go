package main

import (
	"flag"
	"strings"

	"github.com/bartlettc22/image-inquisitor/internal/reports"
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
	ImageSourcesStr                string
	ImageSources                   []string
	GcsBucket                      string
	ReportOutputsStr               string
	ReportOutputs                  ReportOutputs
	ReportOutputDestinationsStr    string
	ReportOutputDestinations       []string
	ImageSourceFilePath            string
	RunTrivy                       bool
	RunRegistry                    bool
	IncludeKubernetesNamespacesStr string
	IncludeKubernetesNamespaces    []string
	ExcludeKubernetesNamespacesStr string
	ExcludeKubernetesNamespaces    []string
	ExcludeImageRegistriesStr      string
	ExcludeImageRegistries         []string
}

type ReportOutputs []reports.ReportType

func (r ReportOutputs) Contains(reportType reports.ReportType) bool {
	for _, report := range r {
		if report == reportType {
			return true
		}
	}
	return false
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

	flag.StringVar(&config.ImageSourceStr,
		"image-source",
		"kubernetes",
		"Source of images to scan.  Can be one of 'kubernetes' or 'file'. If 'file', must specify 'image-source-file-path' parameter.")

	flag.StringVar(&config.ImageSourcesStr,
		"image-sources",
		"kubernetes",
		"Comma-separated list of image sources to scan.  Can be one or more of of [kubernetes, file, gcs]. If 'file', must specify 'image-source-file-path' parameter.  If 'gcs', must specify 'gcs-*' parameters")

	flag.StringVar(&config.GcsBucket,
		"gcs-bucket",
		"",
		"GCS bucket to pull source images from")

	flag.StringVar(&config.ReportOutputsStr,
		"report-outputs",
		"summary,summaryImageCombined,summaryRegistry,imageSummary,imageRegistry,imageVulnerabilities,imageKubernetes",
		"Comma-separated list of reports to output.  Can be one or more of [summary, summaryImageCombined, summaryRegistry, imageSummary, imageRegistry, imageVulnerabilities, imageKubernetes]")

	flag.StringVar(&config.ReportOutputDestinationsStr,
		"report-output-destinations",
		"stdout",
		"Comma-separated list of output sources.  Can be one or more of [gcs, stdout]")

	flag.StringVar(&config.ImageSourceFilePath,
		"image-source-file-path",
		"",
		"Path of file containing list of images to scan")

	flag.StringVar(&config.IncludeKubernetesNamespacesStr,
		"include-kubernetes-namespaces",
		"",
		"Comma-separated list of Kubernetes namespaces to scan if --image-source=kubernetes")
	flag.StringVar(&config.ExcludeKubernetesNamespacesStr,
		"exclude-kubernetes-namespaces",
		"",
		"Comma-separated list of Kubernetes namespaces to exclude if --image-source=kubernetes")
	flag.StringVar(&config.ExcludeImageRegistriesStr,
		"exclude-image-registries",
		"",
		"Comma-separated list of image registries to exclude")

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

	if config.ExcludeKubernetesNamespacesStr != "" {
		config.ExcludeKubernetesNamespaces = strings.Split(config.ExcludeKubernetesNamespacesStr, ",")
	}

	if config.ExcludeImageRegistriesStr != "" {
		config.ExcludeImageRegistries = strings.Split(config.ExcludeImageRegistriesStr, ",")
	}

	config.ImageSources = strings.Split(config.ImageSourcesStr, ",")
	reportOutputsList := strings.Split(config.ReportOutputsStr, ",")
	for _, r := range reportOutputsList {
		if reports.IsValidReportType(r) {
			config.ReportOutputs = append(config.ReportOutputs, reports.ReportType(r))
		} else {
			log.Fatalf("invalid report type: '%s'", r)
		}
	}

	config.ReportOutputDestinations = strings.Split(config.ReportOutputDestinationsStr, ",")

	for _, d := range config.ReportOutputDestinations {
		switch d {
		case "gcs":
			if config.GcsBucket == "" {
				log.Fatal("gcs-bucket must be set")
			}
		}
	}

	return config
}
