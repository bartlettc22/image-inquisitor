package config

import (
	"github.com/bartlettc22/image-inquisitor/internal/inventory/inventory"
	"github.com/bartlettc22/image-inquisitor/internal/trivy"
	"github.com/spf13/viper"
)

func InventoryGeneratorFromConfig() (*inventory.InventoryGenerator, error) {
	inventoryConfig := &inventory.InventoryConfig{
		LatestSemverScanningEnabled: viper.GetBool("latest-semver-scan"),
		SecurityScanningEnabled:     viper.GetBool("security-scan"),
	}

	if inventoryConfig.SecurityScanningEnabled {
		scanner, err := trivy.NewTrivyScanner(&trivy.TrivyScannerConfig{})
		if err != nil {
			return nil, err
		}
		inventoryConfig.SecurityScanner = scanner
	}

	inventoryGenerator, err := inventory.NewInventoryGenerator(inventoryConfig)
	if err != nil {
		return nil, err
	}

	return inventoryGenerator, nil
}
