package config

import (
	"fmt"
	"strings"

	transferapi "github.com/bartlettc22/image-inquisitor/pkg/transfer"
	transferfileapi "github.com/bartlettc22/image-inquisitor/pkg/transfer/file"
	transfergcsapi "github.com/bartlettc22/image-inquisitor/pkg/transfer/gcs"
	"github.com/spf13/viper"
)

// ImporterFromConfig returns an importer from the configuration
// If the configuration does not specify an importer, returns nil
func ImporterFromConfig() (transferapi.Transferer, error) {
	destination := viper.GetString("import-from")
	if destination == "" {
		return nil, nil
	}
	return TransfererFromConfig(destination)
}

func ExporterFromConfig() (transferapi.Transferer, error) {
	destination := viper.GetString("export-destination")
	if destination == "" {
		return nil, fmt.Errorf("--export-destination not specified")
	}
	return TransfererFromConfig(destination)
}

func TransfererFromConfig(destination string) (transferapi.Transferer, error) {

	protocol, path, err := ParseDestination(destination)
	if err != nil {
		return nil, err
	}

	switch protocol {
	case DestinationFile:
		return transferfileapi.NewFileTransferer(&transferfileapi.FileTransfererConfig{
			Path: path,
		})
	case DestinationGCS:
		pathParts := strings.Split(path, "/")
		transferconfig := &transfergcsapi.GCSTransfererConfig{
			Bucket: pathParts[0],
		}
		if len(pathParts) > 1 {
			transferconfig.Path = strings.Join(pathParts[1:], "/")
		}
		return transfergcsapi.NewGCSTransferer(transferconfig)
	default:
		return nil, fmt.Errorf("invalid protocol '%s' in transfer location: %s", protocol, destination)
	}
}
