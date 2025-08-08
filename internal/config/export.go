package config

import (
	"fmt"
	"strings"

	exportapi "github.com/bartlettc22/image-inquisitor/pkg/export"
	"github.com/spf13/viper"
)

type ExportDestination string

const (
	ExportDestinationFile ExportDestination = "file"
	ExportDestinationGCS  ExportDestination = "gs"
)

func (ed ExportDestination) String() string {
	return string(ed)
}

func ExporterFromConfig() (exportapi.Exporter, error) {

	destination := viper.GetString("export-destination")
	if destination == "" {
		return nil, fmt.Errorf("--export-destination not specified")
	}

	protocol, path, err := ParseExportDestination(destination)
	if err != nil {
		return nil, err
	}

	switch protocol {
	case ExportDestinationFile:
		return exportapi.NewFileExporter(&exportapi.ExportToFileConfig{
			Path: path,
		})
	case ExportDestinationGCS:
		return exportapi.NewGCSExporter(&exportapi.GCSExporterConfig{
			Bucket: viper.GetString("export-destination"),
			Path:   path,
		})
	default:
		return nil, fmt.Errorf("invalid protocol '%s' in destination: %s", protocol, destination)
	}
}

func ParseExportDestination(destination string) (ExportDestination, string, error) {

	parts := strings.Split(destination, "://")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid export destination: %s", destination)
	}

	switch protocol := parts[0]; protocol {
	case ExportDestinationFile.String():
		return ExportDestinationFile, parts[1], nil
	case ExportDestinationGCS.String():
		return ExportDestinationGCS, parts[1], nil
	default:
		return "", "", fmt.Errorf("invalid export protocol '%s' in destination: %s", protocol, destination)
	}
}
