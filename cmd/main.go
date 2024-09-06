package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/imageUtils"
	"github.com/bartlettc22/image-inquisitor/internal/kubernetes"
	"github.com/bartlettc22/image-inquisitor/internal/registries/querier"
	"github.com/bartlettc22/image-inquisitor/internal/reports"
	"github.com/bartlettc22/image-inquisitor/internal/trivy"
	log "github.com/sirupsen/logrus"
)

func main() {

	start := time.Now()

	config := loadConfig()

	if config.LogJSON {
		log.SetFormatter(&log.JSONFormatter{})
	}
	logLevel, err := log.ParseLevel(config.LogLevel)
	if err != nil {
		log.Fatalf("failed to parse log level: %v", err)
	}
	log.SetLevel(logLevel)

	masterSummaryReportList := reports.NewSummaryReportList(start)
	masterImageReportList := reports.NewImageReportList(start)

	excludeRegistriesMap := make(map[string]struct{})
	for _, reg := range config.ExcludeImageRegistries {
		excludeRegistriesMap[reg] = struct{}{}
	}

	masterImagesList := make(imageUtils.ImagesList)

	switch config.ImageSource {
	case ImageListSourceKubernetes:
		k, err := kubernetes.NewKubernetes(&kubernetes.KubernetesConfig{
			IncludeNamespaces: config.IncludeKubernetesNamespaces,
			ExcludeNamespaces: config.ExcludeKubernetesNamespaces,
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

			if config.ReportOutputs.Contains(reports.ReportTypeImageKubernetes) {
				masterImageReportList.AddImageReport(reports.ReportTypeImageKubernetes, image, kubeReport)
			}
		}
	case ImageListSourceFile:
		file, err := os.Open(config.ImageSourceFilePath)
		if err != nil {
			log.Fatalf("error opening file '%s': %v", config.ImageSourceFilePath, err)
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			image := scanner.Text()
			if strings.TrimSpace(image) != "" {
				parsedImage, err := imageUtils.ParseImage(image)
				if err != nil {
					log.Errorf("error parsing image %s, skipping: %v", image, err)
					continue
				}

				if _, ok := excludeRegistriesMap[parsedImage.Registry]; ok {
					continue
				}

				masterImagesList[parsedImage.FullName(false)] = parsedImage
			}
		}
		if err := scanner.Err(); err != nil {
			log.Fatalf("error reading file: '%s': %v", config.ImageSourceFilePath, err)
		}
	default:
		log.Fatalf("image source unknown")
	}

	wg := &sync.WaitGroup{}
	mu := &sync.Mutex{}

	if config.ReportOutputs.Contains(reports.ReportTypeImageRegistry) ||
		config.ReportOutputs.Contains(reports.ReportTypeSummaryRegistry) {
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

	if config.ReportOutputs.Contains(reports.ReportTypeImageVulnerabilities) {
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
