package main

import (
	"bufio"
	"encoding/json"
	"os"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/imageUtils"
	"github.com/bartlettc22/image-inquisitor/internal/kubernetes"
	"github.com/bartlettc22/image-inquisitor/internal/registries/querier"
	"github.com/bartlettc22/image-inquisitor/internal/trivy"
	log "github.com/sirupsen/logrus"
)

func main() {

	config := loadConfig()

	if config.LogJSON {
		log.SetFormatter(&log.JSONFormatter{})
	}
	logLevel, err := log.ParseLevel(config.LogLevel)
	if err != nil {
		log.Fatalf("failed to parse log level: %v", err)
	}
	log.SetLevel(logLevel)

	finalReport := &FinalReport{
		Summary: &FinalReportSummary{},
		Reports: make(map[string]*ImageReport),
	}

	switch config.ImageSource {
	case ImageListSourceKubernetes:
		k, err := kubernetes.NewKubernetes(&kubernetes.KubernetesConfig{
			IncludeNamespaces: config.IncludeKubernetesNamespaces,
		})
		if err != nil {
			log.Fatal(err)
		}
		kubeReport, err := k.GetReport()
		if err != nil {
			log.Fatalf("error listing images from Kubernetes: %s", err.Error())
		}

		for image, imageReport := range kubeReport {

			parsedImage, err := imageUtils.ParseImage(image)
			if err != nil {
				log.Errorf("error parsing image %s, skipping: %v", image, err)
				continue
			}

			finalReport.Reports[image] = &ImageReport{
				Image:            parsedImage,
				KubernetesReport: imageReport,
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

			parsedImage, err := imageUtils.ParseImage(image)
			if err != nil {
				log.Errorf("error parsing image %s, skipping: %v", image, err)
				continue
			}

			finalReport.Reports[image] = &ImageReport{
				Image: parsedImage,
			}
		}
		if err := scanner.Err(); err != nil {
			log.Fatalf("error reading file: '%s': %v", config.ImageSourceFilePath, err)
		}
	default:
		log.Fatalf("image source unknown")
	}

	if config.RunRegistry {
		ApplyRegistryReport(finalReport)
	}

	if config.RunTrivy {
		ApplyTrivyReport(finalReport)
	}

	ApplySummary(finalReport)

	jsonOut, err := json.MarshalIndent(finalReport, "", "    ")
	if err != nil {
		log.Fatalf("could not format output to JSON")
	}
	log.Info(string(jsonOut))
}

func ApplyRegistryReport(finalReport *FinalReport) {
	registryQueries := querier.NewRegistryQuerier()

	for image, imageReport := range finalReport.Reports {
		report, err := registryQueries.FetchReport(imageReport.Image)
		if err != nil {
			log.Error(err)
			continue
		}

		finalReport.Reports[image].RegistryReport = report

	}
}

func ApplyTrivyReport(finalReport *FinalReport) {

	trivyOutputDir, err := os.MkdirTemp("/tmp", "trivy_*")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err := os.RemoveAll(trivyOutputDir)
		if err != nil {
			log.Errorf("Error removing directory: %v\n", err)
		}
	}()

	trivyRunner := trivy.NewTrivyRunner(trivy.TrivyRunnerConfig{
		NumWorkers: 5,
		Images:     finalReport.Images(),
		OutputDir:  trivyOutputDir,
	})

	trivyReport := trivyRunner.Run()
	for image, report := range trivyReport {
		finalReport.Reports[image].TrivyReport = report
	}

}

func ApplySummary(finalReport *FinalReport) {
	finalReportSummary := &FinalReportSummary{
		ByRegistryCount: make(map[string]int),
	}
	for image, imageReport := range finalReport.Reports {
		finalReport.Reports[image].Summary = &ImageReportSummary{
			CurrentTag:              imageReport.RegistryReport.CurrentTag,
			CurrentTagAgeSeconds:    time.Since(imageReport.RegistryReport.CurrentTagTimestamp).Seconds(),
			LatestTag:               imageReport.RegistryReport.LatestTag,
			LatestTagAgeSeconds:     time.Since(imageReport.RegistryReport.LatestTagTimestamp).Seconds(),
			TagOutOfDateBySeconds:   imageReport.RegistryReport.LatestTagTimestamp.Sub(imageReport.RegistryReport.CurrentTagTimestamp).Seconds(),
			KubernetesResourceCount: len(imageReport.KubernetesReport.Resources),
			IssuesCriticalCount:     imageReport.TrivyReport.ImageIssues.Total.Critical,
			IssuesHighCount:         imageReport.TrivyReport.ImageIssues.Total.High,
			IssuesMediumCount:       imageReport.TrivyReport.ImageIssues.Total.Medium,
			IssuesLowCount:          imageReport.TrivyReport.ImageIssues.Total.Low,
			IssuesUnknownCount:      imageReport.TrivyReport.ImageIssues.Total.Unknown,
		}
		finalReportSummary.ImageCount++
		finalReportSummary.IssuesCriticalCount += imageReport.TrivyReport.ImageIssues.Total.Critical
		finalReportSummary.IssuesHighCount += imageReport.TrivyReport.ImageIssues.Total.High
		finalReportSummary.IssuesMediumCount += imageReport.TrivyReport.ImageIssues.Total.Medium
		finalReportSummary.IssuesLowCount += imageReport.TrivyReport.ImageIssues.Total.Low
		finalReportSummary.IssuesUnknownCount += imageReport.TrivyReport.ImageIssues.Total.Unknown
		if _, ok := finalReportSummary.ByRegistryCount[imageReport.Image.Registry]; !ok {
			finalReportSummary.ByRegistryCount[imageReport.Image.Registry] = 0
		}
		finalReportSummary.ByRegistryCount[imageReport.Image.Registry]++
	}
	finalReport.Summary = finalReportSummary
}
