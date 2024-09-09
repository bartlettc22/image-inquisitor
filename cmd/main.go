package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/config"
	"github.com/bartlettc22/image-inquisitor/internal/registries/querier"
	"github.com/bartlettc22/image-inquisitor/internal/reports"
	"github.com/bartlettc22/image-inquisitor/internal/sources"
	exportsources "github.com/bartlettc22/image-inquisitor/internal/sources/export"
	importsources "github.com/bartlettc22/image-inquisitor/internal/sources/import"
	"github.com/bartlettc22/image-inquisitor/internal/trivy"
	log "github.com/sirupsen/logrus"
)

func main() {

	start := time.Now()
	ctx := context.Background()

	cfg := config.LoadConfig()

	if cfg.LogJSON {
		log.SetFormatter(&log.JSONFormatter{})
	}
	logLevel, err := log.ParseLevel(cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to parse log level: %v", err)
	}
	log.SetLevel(logLevel)

	inventory, err := sources.GetInventoryFromSources(ctx, &sources.ImageSourcesConfig{
		SourceID:               cfg.SourceID,
		ImageSourceTypes:       cfg.ImageSources,
		ExcludeImageRegistries: cfg.ExcludeImageRegistries,
		KubernetesSourceConfig: &sources.KubernetesSourceConfig{
			IncludeNamespaces: cfg.KubernetesSourceIncludeNamespaces,
			ExcludeNamespaces: cfg.KubernetesSourceExcludeNamespaces,
		},
		FileSourceConfig: &sources.FileSourceConfig{
			SourceFilePath: cfg.ImageSourcesFilePath,
		},
		ImportSourcesConfig: &importsources.ImportSourcesConfig{
			ImportSourcesFrom:             cfg.ImportSourcesFrom,
			ImportSourcesFilePath:         cfg.ImportSourcesFilePath,
			ImportSourcesGCSBucket:        cfg.ImportSourcesGCSBucket,
			ImportSourcesGCSDirectoryPath: cfg.ImportSourcesGCSDirectoryPath,
		},
	})
	if err != nil {
		log.Fatalf("%v", err)
	}
	if len(cfg.ExportSourcesDestinations) > 0 {
		log.Infof("exporting primary sources to: %s", cfg.ExportSourcesDestinations.String())
		err := inventory.Export(ctx, &exportsources.ExporterConfig{
			SourceID:         cfg.SourceID,
			Destinations:     cfg.ExportSourcesDestinations,
			FilePath:         cfg.ExportSourcesFilePath,
			GCSBucket:        cfg.ExportSourcesGCSBucket,
			GCSDirectoryPath: cfg.ExportSourcesGCSDirectoryPath,
		})
		if err != nil {
			log.Fatalf("%v", err)
		}
	}

	// if cfg.ReportOutputs.Contains(reports.ReportTypeImageKubernetes) {
	// 	kubernetesSourceReport, err := inventory.GetKubernetesSourceReports(ctx)
	// 	if err != nil {
	// 		log.Fatalf("%v", err)
	// 	}
	// 	for image, kubeReport := range kubernetesSourceReport.KubeReports() {
	// 		masterImageReportList.AddImageReport(reports.ReportTypeImageKubernetes, image, kubeReport)
	// 	}
	// }

	wg := &sync.WaitGroup{}
	mu := &sync.Mutex{}

	masterSummaryReportList := reports.NewSummaryReportList(start)
	masterImageReportList := reports.NewImageReportList(start)

	if cfg.ReportOutputs.Contains(reports.ReportTypeImageRegistry) ||
		cfg.ReportOutputs.Contains(reports.ReportTypeSummaryRegistry) {
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			registryQueries := querier.NewRegistryQuerier()
			for imageFullName, image := range inventory.ImageComponents() {
				registryReport, err := registryQueries.FetchReport(image)
				if err != nil {
					log.Error(err)
					continue
				}
				mu.Lock()
				masterImageReportList.AddImageReport(reports.ReportTypeImageRegistry, imageFullName, registryReport)
				mu.Unlock()
			}
		}(wg)
	}

	if cfg.ReportOutputs.Contains(reports.ReportTypeImageVulnerabilities) {
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			trivyReport, err := GetTrivyReport(inventory.ImagesAsSlice())
			if err != nil {
				log.Error(err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			for imageFullName, vulnReport := range trivyReport {
				masterImageReportList.AddImageReport(reports.ReportTypeImageVulnerabilities, imageFullName, vulnReport)
			}

		}(wg)
	}
	wg.Wait()

	if cfg.ReportOutputs.Contains(reports.ReportTypeImageSummary) {
		masterSummaryReportList.GenerateSummaryReports(inventory.ImageComponents(), masterImageReportList)
		masterSummaryReportList.Output()
		masterImageReportList.Output()
	}

	log.Infof("done")
}

func GetTrivyReport(images []string) (trivy.TrivyReport, error) {

	trivyOutputDir, err := os.MkdirTemp("/tmp", "trivy_*")
	if err != nil {
		return nil, fmt.Errorf("error creating Trivy tmp directory: %v", err)
	}
	defer func() {
		err := os.RemoveAll(trivyOutputDir)
		if err != nil {
			log.Errorf("Error removing directory: %v\n", err)
		}
	}()

	trivyRunner := trivy.NewTrivyRunner(trivy.TrivyRunnerConfig{
		NumWorkers: 5,
		Images:     images,
		OutputDir:  trivyOutputDir,
	})

	return trivyRunner.Run(), nil
}
