package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/config"
	"github.com/bartlettc22/image-inquisitor/internal/imageUtils"
	"github.com/bartlettc22/image-inquisitor/internal/kubernetes"
	"github.com/bartlettc22/image-inquisitor/internal/registries/querier"
	"github.com/bartlettc22/image-inquisitor/internal/reports"
	"github.com/bartlettc22/image-inquisitor/internal/sources"
	"github.com/bartlettc22/image-inquisitor/internal/sources/export"
	sourcetypes "github.com/bartlettc22/image-inquisitor/internal/sources/types"
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
	exporter := export.NewExporter(&export.ExporterConfig{
		ExternalID:   cfg.ExportExternalID,
		Destinations: cfg.ExportDestinations,
		FilePath:     cfg.ExportFilePath,
		GCSBucket:    cfg.ExportGCSBucket,
	})

	excludeRegistriesMap := make(map[string]struct{})
	for _, reg := range cfg.ExcludeImageRegistries {
		excludeRegistriesMap[reg] = struct{}{}
	}

	masterImagesList := make(imageUtils.ImagesList)

	for _, source := range cfg.ImageSources {
		switch source {
		case config.ImageListSourceKubernetes:
			k, err := kubernetes.NewKubernetes(&kubernetes.KubernetesConfig{
				IncludeNamespaces: cfg.IncludeKubernetesNamespaces,
				ExcludeNamespaces: cfg.ExcludeKubernetesNamespaces,
			})
			if err != nil {
				log.Fatal(err)
			}
			kubeReport, err := k.GetReport()
			if err != nil {
				log.Fatalf("error listing images from Kubernetes: %s", err.Error())
			}

			for image, kubeReport := range kubeReport {

				parsedImage, err := imageUtils.ParseImage(image)
				if err != nil {
					log.Errorf("error parsing image %s, skipping: %v", image, err)
					continue
				}

				// if excluded, ignore
				if _, ok := excludeRegistriesMap[parsedImage.Registry]; ok {
					continue
				}

				masterImagesList[parsedImage.FullName(false)] = parsedImage

				if cfg.ReportOutputs.Contains(reports.ReportTypeImageKubernetes) {
					masterImageReportList.AddImageReport(reports.ReportTypeImageKubernetes, image, kubeReport)
				}
			}
			exporter.AddReport(sourcetypes.ImageSourceKubernetes, &kubeReport)
		case config.ImageListSourceFile:
			fileSource := sources.NewFileSource(&sources.FileSourceConfig{
				SourceFilePath:    cfg.ImageSourceFilePath,
				ExcludeRegistries: excludeRegistriesMap,
			})
			fileSourceReport, err := fileSource.GetReport(ctx)
			if err != nil {
				log.Fatalf("%v", err)
			}
			for parsedImageName, parsedImage := range fileSourceReport.Images() {
				masterImagesList[parsedImageName] = parsedImage
			}
			exporter.AddReport(sourcetypes.ImageSourceFile, fileSourceReport)
		default:
			log.Fatalf("image source unknown")
		}
	}

	err = exporter.Export(ctx)
	if err != nil {
		log.Errorf("%v", err)
	}

	wg := &sync.WaitGroup{}
	mu := &sync.Mutex{}

	if cfg.ReportOutputs.Contains(reports.ReportTypeImageRegistry) ||
		cfg.ReportOutputs.Contains(reports.ReportTypeSummaryRegistry) {
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			registryQueries := querier.NewRegistryQuerier()
			for imageFullName, image := range masterImagesList {
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
			trivyReport, err := GetTrivyReport(masterImagesList.List())
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

	masterSummaryReportList.GenerateSummaryReports(masterImagesList, masterImageReportList)
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
