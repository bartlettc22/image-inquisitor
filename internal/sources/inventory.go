package sources

import (
	"context"

	"github.com/bartlettc22/image-inquisitor/internal/imageUtils"
	exportsources "github.com/bartlettc22/image-inquisitor/internal/sources/export"
	"github.com/bartlettc22/image-inquisitor/internal/sources/types"
	log "github.com/sirupsen/logrus"
)

// ImageInventory represents a full list of images to be evaulated along
// with the full details for the images
type ImageInventory struct {
	*ImageInventoryConfig

	// ImageDetails contains all of the details we have on the image
	// Map key is the image's fully qualified name
	ImageDetails map[string]*ImageDetails `json:"images" yaml:"images"`
}

type ImageInventoryConfig struct {
	ExcludeRegistries map[string]struct{}
}

// ImageDetails contains all of the details we have on a particular image
type ImageDetails struct {

	// ImageComponents are the components of the image (registry, tag, etc.)
	ImageComponents *imageUtils.Image `json:"components" yaml:"components"`

	// ImageSourceDetailsByID contains the source(s) details
	// Key is the sourceID
	ImageSourceDetailsByID map[string]*types.ImageSourceDetails `json:"sourceDetails" yaml:"sourceDetails"`

	// RegistryDetails {
	// 	CurrentTagTimestamp
	// 	LatestTag
	// 	LatestTagTimestamp
	// }

	// ImageIssues {
	// 	"Vulnerabilities" {
	// 		Source: "Trivy"
	// 		Vulns: []{
	// 			x
	// 		}
	// 	}
	// }
}

func NewImageInventory(config *ImageInventoryConfig) *ImageInventory {
	return &ImageInventory{
		ImageInventoryConfig: config,
		ImageDetails:         make(map[string]*ImageDetails),
	}
}

func (inv *ImageInventory) qualifyImage(imageName string) (string, error) {
	// To ensure all image names are fully qualified, we'll parse each one
	// to the standard format
	parsedImage, err := imageUtils.ParseImage(imageName)
	if err != nil {
		return "", err
	}

	fullyQualifiedName := parsedImage.FullyQualifiedName(false)

	// Skip if part of the excluded registries list
	if _, ok := inv.ExcludeRegistries[parsedImage.Registry]; ok {
		return "", nil
	}

	if _, ok := inv.ImageDetails[fullyQualifiedName]; !ok {
		inv.ImageDetails[fullyQualifiedName] = &ImageDetails{
			ImageComponents:        parsedImage,
			ImageSourceDetailsByID: make(map[string]*types.ImageSourceDetails),
		}
	}

	return fullyQualifiedName, nil
}

func (inv ImageInventory) AddImageSourceDetails(sourceID string, imageSourceDetails map[string]*types.ImageSourceDetails) error {
	for imageName, imageSourceDetails := range imageSourceDetails {

		fullyQualifiedName, err := inv.qualifyImage(imageName)
		if err != nil {
			return err
		}

		if currentImageSourceDetails, ok := inv.ImageDetails[fullyQualifiedName].ImageSourceDetailsByID[sourceID]; ok {
			for desiredSourceType := range imageSourceDetails.SourcesByType {
				for currentSourceType := range currentImageSourceDetails.SourcesByType {
					if currentSourceType == desiredSourceType {
						log.Warnf("attempting to merge two inventory lists with the same sourceID and type; image: %s, sourceID: %s, sourceType: %s", fullyQualifiedName, sourceID, desiredSourceType)
						return nil
					}
				}
			}

		}
		inv.ImageDetails[fullyQualifiedName].ImageSourceDetailsByID[sourceID] = imageSourceDetails
		// }
	}

	return nil
}

func (inv ImageInventory) ImageComponents() imageUtils.ImagesList {
	results := make(imageUtils.ImagesList)
	for image, details := range inv.ImageDetails {
		results[image] = details.ImageComponents
	}
	return results
}

func (inv ImageInventory) Export(ctx context.Context, exporterConfig *exportsources.ExporterConfig) error {
	exporter := exportsources.NewExporter(exporterConfig)
	for fullyQualifiedImageName, imageDetails := range inv.ImageDetails {
		for sourceID, imageSourceDetails := range imageDetails.ImageSourceDetailsByID {
			// Only export if we match our primary source ID
			if sourceID == exporterConfig.SourceID {
				exporter.AddImageSourceDetails(fullyQualifiedImageName, imageSourceDetails.SourcesByType)
			}
		}
	}

	return exporter.Export(ctx)
}

// ImagesAsSlice returns a slice of fully qualified image names
func (inv ImageInventory) ImagesAsSlice() []string {
	result := []string{}
	for imageName := range inv.ImageDetails {
		result = append(result, imageName)
	}
	return result
}
