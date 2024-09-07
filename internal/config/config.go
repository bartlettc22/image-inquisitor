package config

import (
	"flag"
	"strings"

	"github.com/bartlettc22/image-inquisitor/internal/reports"
	exporttypes "github.com/bartlettc22/image-inquisitor/internal/sources/export/types"
	sourcetypes "github.com/bartlettc22/image-inquisitor/internal/sources/types"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	LogLevel                       string
	LogJSON                        bool
	ImageSourcesStr                string
	ImageSources                   []sourcetypes.ImageSourceType
	ExportGCSBucket                string
	ReportOutputsStr               string
	ReportOutputs                  ReportOutputs
	ExportDestinationsStr          string
	ExportDestinations             exporttypes.ExportDestinationList
	ExportExternalID               string
	ExportFilePath                 string
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

func LoadConfig() Config {
	config := Config{}

	flag.StringVar(&config.LogLevel,
		"log-level",
		"info",
		"The desired logging level.  One of panic, fatal, error, warn, info, debug, trace.")

	flag.BoolVar(&config.LogJSON,
		"log-json",
		true,
		"Whether to log JSON output")

	flag.StringVar(&config.ImageSourcesStr,
		"image-sources",
		"kubernetes",
		"Comma-separated list of image sources to scan.  Can be one or more of of [kubernetes, file, gcs]. If 'file', must specify 'image-source-file-path' parameter.  If 'gcs', must specify 'gcs-*' parameters")

	flag.StringVar(&config.ImageSourceFilePath,
		"image-source-file-path",
		"",
		"Path of file containing list of images to scan")

	flag.StringVar(&config.ExportDestinationsStr,
		"export-destinations",
		"",
		"Comma-separated list of output sources.  Can be one or more of [file, gcs]. If 'gcs', must specify 'gcs-*' parameters")

	flag.StringVar(&config.ExportExternalID,
		"export-external-id",
		"",
		"Identifier for the export.  Used for filenames and unique identifier on import")

	flag.StringVar(&config.ExportFilePath,
		"export-file-path",
		"",
		"Path of directory to dump the export")

	flag.StringVar(&config.ExportGCSBucket,
		"export-gcs-bucket",
		"",
		"GCS bucket to pull source images from")

	flag.StringVar(&config.ReportOutputsStr,
		"report-outputs",
		"summary,summaryImageCombined,summaryRegistry,imageSummary,imageRegistry,imageVulnerabilities,imageKubernetes",
		"Comma-separated list of reports to output.  Can be one or more of [summary, summaryImageCombined, summaryRegistry, imageSummary, imageRegistry, imageVulnerabilities, imageKubernetes]")

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

	if config.IncludeKubernetesNamespacesStr != "" {
		config.IncludeKubernetesNamespaces = strings.Split(config.IncludeKubernetesNamespacesStr, ",")
	}

	if config.ExcludeKubernetesNamespacesStr != "" {
		config.ExcludeKubernetesNamespaces = strings.Split(config.ExcludeKubernetesNamespacesStr, ",")
	}

	if config.ExcludeImageRegistriesStr != "" {
		config.ExcludeImageRegistries = strings.Split(config.ExcludeImageRegistriesStr, ",")
	}

	imageSources := strings.Split(config.ImageSourcesStr, ",")
	for _, imgSource := range imageSources {
		imageSource := sourcetypes.ImageSourceType(imgSource)
		if !imageSource.IsValid() {
			log.Fatalf("invalid value in --image-sources: %s", imageSource)
		}
		config.ImageSources = append(config.ImageSources, imageSource)
		switch imageSource {
		case sourcetypes.ImageSourceTypeGCS:
			if config.ExportGCSBucket == "" {
				log.Fatal("gcs-bucket must be set")
			}
		case sourcetypes.ImageSourceTypeFile:
			if config.ImageSourceFilePath == "" {
				log.Fatalf("must specify 'image-source-file-path' parameter when 'image-sources=file'")
			}
		case sourcetypes.ImageSourceTypeKubernetes:
		default:

		}
	}

	reportOutputsList := strings.Split(config.ReportOutputsStr, ",")
	for _, r := range reportOutputsList {
		if reports.IsValidReportType(r) {
			config.ReportOutputs = append(config.ReportOutputs, reports.ReportType(r))
		} else {
			log.Fatalf("invalid report type: '%s'", r)
		}
	}

	config.ExportDestinations = make(exporttypes.ExportDestinationList)
	if config.ExportDestinationsStr != "" {

		if config.ExportExternalID == "" {
			log.Fatal("when exporting, export-external-id must be set")
		}

		exportDestinations := strings.Split(config.ExportDestinationsStr, ",")
		for _, dest := range exportDestinations {
			err := config.ExportDestinations.Add(dest)
			if err != nil {
				log.Fatal(err)
			}
		}
		if config.ExportDestinations.Contains(exporttypes.ExportDestinationGCS) {
			if config.ExportGCSBucket == "" {
				log.Fatal("gcs-bucket must be set")
			}
		}
		if config.ExportDestinations.Contains(exporttypes.ExportDestinationFile) {
			if config.ExportFilePath == "" {
				log.Fatal("export file path must be set")
			}
		}
	}

	return config
}
