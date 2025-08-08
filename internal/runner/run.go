package runner

import (
	"github.com/bartlettc22/image-inquisitor/internal/config"
	log "github.com/sirupsen/logrus"
)

var svcLog = log.WithField("service", "runner")

func Run() error {

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

	sources, err := sourceGenerator.Generate()
	if err != nil {
		return err
	}

	// TODO - import sources

	inventoryGenerator.AddImagesFromSourceList(sources)
	inventory := inventoryGenerator.Inventory()

	svcLog.Info("generating reports")
	err = reportGenerator.Generate(inventory)
	if err != nil {
		return err
	}

	svcLog.Info("finished run")

	return nil
}
