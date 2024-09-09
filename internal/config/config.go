package config

import (
	"flag"
	"strings"

	"github.com/bartlettc22/image-inquisitor/internal/reports"
	exporttypes "github.com/bartlettc22/image-inquisitor/internal/sources/export/types"
	importtypes "github.com/bartlettc22/image-inquisitor/internal/sources/import/types"
	sourcetypes "github.com/bartlettc22/image-inquisitor/internal/sources/types"
	log "github.com/sirupsen/logrus"
)

type Config struct {

	// Identifier for this instance.  Required if importing or exporting.
	SourceID string

	// Logging Configurations
	LogLevel string
	LogJSON  bool

	// Sources
	ImageSourcesStr string
	ImageSources    []sourcetypes.ImageSourceType

	// File Source Configuration
	ImageSourcesFilePath string

	// Kubernetes Source Configuration
	KubernetesSourceIncludeNamespacesStr string
	KubernetesSourceIncludeNamespaces    []string
	KubernetesSourceExcludeNamespacesStr string
	KubernetesSourceExcludeNamespaces    []string

	// Export Sources Configurations
	ExportSourcesDestinationsStr  string
	ExportSourcesDestinations     exporttypes.ExportDestinationList
	ExportSourcesFilePath         string
	ExportSourcesGCSBucket        string
	ExportSourcesGCSDirectoryPath string

	// Import Sources Configurations
	ImportSourcesFromStr          string
	ImportSourcesFrom             importtypes.ImportFromList
	ImportSourcesFilePath         string
	ImportSourcesGCSBucket        string
	ImportSourcesGCSDirectoryPath string

	// Registry Configurations
	ExcludeImageRegistriesStr string
	ExcludeImageRegistries    map[string]struct{}

	// Reports Configuration
	ReportOutputsStr string
	ReportOutputs    ReportOutputs
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

	// Required
	flag.StringVar(&config.SourceID,
		"source-id",
		"",
		"[required] Identifier for this instance of the tool.  Used for filenames and unique identifiers on export/import/reports")

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

	flag.StringVar(&config.ImageSourcesFilePath,
		"image-source-file-path",
		"",
		"Path of file containing list of images to scan")

	flag.StringVar(&config.ImportSourcesFromStr,
		"import-sources-from",
		"",
		"Comma-separated list of remote locations for importing sources from.  Can be one or more of [file, gcs]. If 'gcs', must specify 'gcs-*' parameters")

	flag.StringVar(&config.ImportSourcesFilePath,
		"import-sources-file-path",
		"",
		"Path to file containing sources to be imported")

	flag.StringVar(&config.ImportSourcesGCSBucket,
		"import-sources-gcs-bucket",
		"",
		"GCS bucket to import sources from")

	flag.StringVar(&config.ImportSourcesGCSDirectoryPath,
		"import-sources-gcs-directory-path",
		"",
		"GCS directory to import sources from")

	flag.StringVar(&config.ExportSourcesDestinationsStr,
		"export-sources-destinations",
		"",
		"Comma-separated list of output sources.  Can be one or more of [file, gcs]. If 'gcs', must specify 'gcs-*' parameters")

	flag.StringVar(&config.ExportSourcesFilePath,
		"export-sources-file-path",
		"",
		"Path of directory to dump the export")

	flag.StringVar(&config.ExportSourcesGCSBucket,
		"export-sources-gcs-bucket",
		"",
		"GCS bucket to pull source images from")

	flag.StringVar(&config.ExportSourcesGCSDirectoryPath,
		"export-sources-gcs-directory-path",
		"",
		"GCS directory to export sources to")

	flag.StringVar(&config.ReportOutputsStr,
		"report-outputs",
		"",
		"Comma-separated list of reports to output.  Can be one or more of [summary, summaryImageCombined, summaryRegistry, imageSummary, imageRegistry, imageVulnerabilities, imageKubernetes]")

	flag.StringVar(&config.KubernetesSourceIncludeNamespacesStr,
		"include-kubernetes-namespaces",
		"",
		"Comma-separated list of Kubernetes namespaces to scan if --image-source=kubernetes")
	flag.StringVar(&config.KubernetesSourceExcludeNamespacesStr,
		"exclude-kubernetes-namespaces",
		"",
		"Comma-separated list of Kubernetes namespaces to exclude if --image-source=kubernetes")
	flag.StringVar(&config.ExcludeImageRegistriesStr,
		"exclude-image-registries",
		"",
		"Comma-separated list of image registries to exclude")

	flag.Parse()

	if config.KubernetesSourceIncludeNamespacesStr != "" {
		config.KubernetesSourceIncludeNamespaces = strings.Split(config.KubernetesSourceIncludeNamespacesStr, ",")
	}

	if config.KubernetesSourceExcludeNamespacesStr != "" {
		config.KubernetesSourceExcludeNamespaces = strings.Split(config.KubernetesSourceExcludeNamespacesStr, ",")
	}

	if config.ExcludeImageRegistriesStr != "" {
		config.ExcludeImageRegistries = make(map[string]struct{})
		excludeImageRegistriesSlice := strings.Split(config.ExcludeImageRegistriesStr, ",")
		for _, registryName := range excludeImageRegistriesSlice {
			config.ExcludeImageRegistries[registryName] = struct{}{}
		}
	}

	imageSources := strings.Split(config.ImageSourcesStr, ",")
	for _, imgSource := range imageSources {
		imageSource := sourcetypes.ImageSourceType(imgSource)
		if !imageSource.IsValid() {
			log.Fatalf("invalid value in --image-sources: %s", imageSource)
		}
		config.ImageSources = append(config.ImageSources, imageSource)
		switch imageSource {
		case sourcetypes.ImageSourceTypeFile:
			if config.ImageSourcesFilePath == "" {
				log.Fatalf("must specify 'image-source-file-path' parameter when 'image-sources=file'")
			}
		case sourcetypes.ImageSourceTypeKubernetes:
		default:

		}
	}

	reportOutputsList := strings.Split(config.ReportOutputsStr, ",")
	for _, r := range reportOutputsList {
		// Allows for outputting zero reports
		if r != "" {
			if reports.IsValidReportType(r) {
				config.ReportOutputs = append(config.ReportOutputs, reports.ReportType(r))
			} else {
				log.Fatalf("invalid report type: '%s'", r)
			}
		}
	}

	if config.SourceID == "" {
		log.Fatal("--source-id is required")
	}

	if config.ImportSourcesFromStr != "" {
		config.ImportSourcesFrom = make(importtypes.ImportFromList)
		importFrom := strings.Split(config.ImportSourcesFromStr, ",")
		for _, from := range importFrom {
			err := config.ImportSourcesFrom.Add(from)
			if err != nil {
				log.Fatal(err)
			}
		}
		if config.ImportSourcesFrom.Contains(importtypes.ImportFromGCS) {
			if config.ImportSourcesGCSBucket == "" {
				log.Fatal("import gcs bucket must be set")
			}
		}
		if config.ImportSourcesFrom.Contains(importtypes.ImportFromFile) {
			if config.ImportSourcesFilePath == "" {
				log.Fatal("import file path must be set")
			}
		}
	}

	if config.ExportSourcesDestinationsStr != "" {
		config.ExportSourcesDestinations = make(exporttypes.ExportDestinationList)
		exportDestinations := strings.Split(config.ExportSourcesDestinationsStr, ",")
		for _, dest := range exportDestinations {
			err := config.ExportSourcesDestinations.Add(dest)
			if err != nil {
				log.Fatal(err)
			}
		}
		if config.ExportSourcesDestinations.Contains(exporttypes.ExportDestinationGCS) {
			if config.ExportSourcesGCSBucket == "" {
				log.Fatal("gcs-bucket must be set")
			}
		}
		if config.ExportSourcesDestinations.Contains(exporttypes.ExportDestinationFile) {
			if config.ExportSourcesFilePath == "" {
				log.Fatal("export file path must be set")
			}
		}
	}

	return config
}
