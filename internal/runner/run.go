package runner

import (
	"context"

	"github.com/bartlettc22/image-inquisitor/internal/config"
	sourcesapi "github.com/bartlettc22/image-inquisitor/pkg/api/v1alpha1/sources"
	log "github.com/sirupsen/logrus"
)

var svcLog = log.WithField("service", "runner")

func Run() error {

	importer, err := config.ImporterFromConfig()
	if err != nil {
		return err
	}

	reportGenerator, err := config.NewReportGeneratorFromConfig()
	if err != nil {
		return err
	}

	sourceGenerator, err := config.SourceGeneratorFromConfig()
	if err != nil {
		return err
	}

	inventoryGenerator, err := config.InventoryGeneratorFromConfig()
	if err != nil {
		return err
	}

	svcLog.Info("starting run")

	if importer != nil {
		manifests, err := importer.Import(context.Background())
		if err != nil {
			return err
		}
		for _, manifest := range manifests {
			sourceList, err := sourcesapi.ManifestToSourceList(manifest)
			if err != nil {
				return err
			}
			inventoryGenerator.AddSources(sourceList)
		}
	}

	sources, err := sourceGenerator.Generate()
	if err != nil {
		return err
	}

	inventoryGenerator.AddSources(sources)
	inventory := inventoryGenerator.Inventory()

	svcLog.Info("generating reports")
	err = reportGenerator.Generate(inventory)
	if err != nil {
		return err
	}

	svcLog.Info("finished run")

	return nil
}
