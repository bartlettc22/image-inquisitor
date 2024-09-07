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
	"github.com/bartlettc22/image-inquisitor/internal/sources/export"
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

	masterSummaryReportList := reports.NewSummaryReportList(start)
	masterImageReportList := reports.NewImageReportList(start)

	excludeRegistriesMap := make(map[string]struct{})
	for _, reg := range cfg.ExcludeImageRegistries {
		excludeRegistriesMap[reg] = struct{}{}
	}

	sourceImages, err := sources.FetchImages(ctx, &sources.ImageSourcesConfig{
		ImageSourceTypes: cfg.ImageSources,
		KubernetesSourceConfig: &sources.KubernetesSourceConfig{
			IncludeNamespaces: cfg.IncludeKubernetesNamespaces,
			ExcludeNamespaces: cfg.ExcludeKubernetesNamespaces,
			ExcludeRegistries: excludeRegistriesMap,
		},
		FileSourceConfig: &sources.FileSourceConfig{
			SourceFilePath:    cfg.ImageSourceFilePath,
			ExcludeRegistries: excludeRegistriesMap,
		},
	})
	if err != nil {
		log.Fatalf("%v", err)
	}
	if len(cfg.ExportDestinations) > 0 {
		err := sourceImages.Export(ctx, &export.ExporterConfig{
			ExternalID:   cfg.ExportExternalID,
			Destinations: cfg.ExportDestinations,
			FilePath:     cfg.ExportFilePath,
			GCSBucket:    cfg.ExportGCSBucket,
		})
		if err != nil {
			log.Fatalf("%v", err)
		}
	}

	if cfg.ReportOutputs.Contains(reports.ReportTypeImageKubernetes) {
		kubernetesSourceReport, err := sourceImages.GetKubernetesSourceReports(ctx)
		if err != nil {
			log.Fatalf("%v", err)
		}
		for image, kubeReport := range kubernetesSourceReport.KubeReports() {
			masterImageReportList.AddImageReport(reports.ReportTypeImageKubernetes, image, kubeReport)
		}
	}

	wg := &sync.WaitGroup{}
	mu := &sync.Mutex{}

	if cfg.ReportOutputs.Contains(reports.ReportTypeImageRegistry) ||
		cfg.ReportOutputs.Contains(reports.ReportTypeSummaryRegistry) {
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			registryQueries := querier.NewRegistryQuerier()
			for imageFullName, image := range sourceImages.List() {
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
			trivyReport, err := GetTrivyReport(sourceImages.List().AsSlice())
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

	masterSummaryReportList.GenerateSummaryReports(sourceImages.List(), masterImageReportList)
	masterSummaryReportList.Output()
	masterImageReportList.Output()
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
