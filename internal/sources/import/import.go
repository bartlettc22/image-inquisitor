package importsources

import (
	"context"
	"fmt"

	exportapimetadata "github.com/bartlettc22/image-inquisitor/internal/sources/export/api/metadata"
	exportapiv1alpha1 "github.com/bartlettc22/image-inquisitor/internal/sources/export/api/v1alpha1"
	importtypes "github.com/bartlettc22/image-inquisitor/internal/sources/import/types"
	sourcetypes "github.com/bartlettc22/image-inquisitor/internal/sources/types"
	"gopkg.in/yaml.v2"
)

type ImportSourcesConfig struct {
	ImportSourcesFrom             importtypes.ImportFromList
	ImportSourcesFilePath         string
	ImportSourcesGCSBucket        string
	ImportSourcesGCSDirectoryPath string
}

func ImportSourcesInventory(ctx context.Context, config *ImportSourcesConfig) (map[string]map[string]*sourcetypes.ImageSourceDetails, error) {

	results := make(map[string]map[string]*sourcetypes.ImageSourceDetails)

	for _, from := range config.ImportSourcesFrom {

		var exportReports map[string][]byte
		var err error

		switch from {
		case importtypes.ImportFromFile:
			exportReports, err = importFileInventory(ctx, config.ImportSourcesFilePath)
			if err != nil {
				return nil, err
			}
		case importtypes.ImportFromGCS:
			exportReports, err = importGCSInventory(ctx, config.ImportSourcesGCSBucket, config.ImportSourcesGCSDirectoryPath)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("error importing sources, unknown type: %s", from)
		}

		for filePath, rawReport := range exportReports {
			// First, unmarshal to get metadata
			exportReportMetadata := &exportapimetadata.ExportMetadata{}
			err := yaml.Unmarshal(rawReport, exportReportMetadata)
			if err != nil {
				return nil, fmt.Errorf("error unmashalling import: %w", err)
			}

			if exportReportMetadata.Kind != exportapimetadata.Kind {
				return nil, fmt.Errorf("import contains invalid 'Kind': %s", exportReportMetadata.Kind)
			}

			var importable importtypes.Importable
			switch exportReportMetadata.Version {
			case exportapiv1alpha1.APIVersion:
				importable = &exportapiv1alpha1.ExportReport{}
				err := yaml.Unmarshal(rawReport, importable)
				if err != nil {
					return nil, fmt.Errorf("error unmashalling v1alpha1 import: %w", err)
				}
			default:
				return nil, fmt.Errorf("import contains invalid 'Version': %s", exportReportMetadata.Version)
			}

			results[importable.SourceID()] = importable.SourceDetails(from, filePath)
		}

	}

	return results, nil
}
