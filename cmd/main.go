package main

import (
	"bufio"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/imageUtils"
	"github.com/bartlettc22/image-inquisitor/internal/kubernetes"
	"github.com/bartlettc22/image-inquisitor/internal/registries/querier"
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

	finalReport := &FinalReport{
		Summary: &FinalReportSummary{},
		Reports: make(map[string]*ImageReport),
	}

	excludeRegistriesMap := make(map[string]struct{})
	for _, reg := range config.ExcludeImageRegistries {
		excludeRegistriesMap[reg] = struct{}{}
	}

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

		for image, imageReport := range kubeReport {

			parsedImage, err := imageUtils.ParseImage(image)
			if err != nil {
				log.Errorf("error parsing image %s, skipping: %v", image, err)
				continue
			}

			// if excluded, ignore
			if _, ok := excludeRegistriesMap[parsedImage.Registry]; ok {
				continue
			}

			finalReport.Reports[parsedImage.FullName(false)] = &ImageReport{
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

			// if excluded, ignore
			if _, ok := excludeRegistriesMap[parsedImage.Registry]; ok {
				continue
			}

			finalReport.Reports[parsedImage.FullName(false)] = &ImageReport{
				Image: parsedImage,
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

	if config.RunRegistry {
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			registryQueries := querier.NewRegistryQuerier()
			for image, imageReport := range finalReport.Reports {
				report, err := registryQueries.FetchReport(imageReport.Image)
				if err != nil {
					log.Error(err)
					continue
				}
				mu.Lock()
				finalReport.Reports[image].RegistryReport = report
				mu.Unlock()
			}
			// ApplyRegistryReport(finalReport)
		}(wg)
	}

	if config.RunTrivy {
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			// GetApplyTrivyReport(finalReport)
			trivyReport, err := GetTrivyReport(finalReport.Images())
			if err != nil {
				log.Error(err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			for image, report := range trivyReport {
				finalReport.Reports[image].TrivyReport = report
			}

		}(wg)
	}
	wg.Wait()

	ApplySummary(finalReport, start)

	LogResults(finalReport)
}

// func ApplyRegistryReport(finalReport *FinalReport) {

// 	for image, imageReport := range finalReport.Reports {
// 		report, err := registryQueries.FetchReport(imageReport.Image)
// 		if err != nil {
// 			log.Error(err)
// 			continue
// 		}

// 		finalReport.Reports[image].RegistryReport = report
// 	}
// }

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

	// trivyReport := trivyRunner.Run()
	// for image, report := range trivyReport {
	// 	finalReport.Reports[image].TrivyReport = report
	// }

	return trivyRunner.Run(), nil
}

func ApplySummary(finalReport *FinalReport, start time.Time) {
	finalReportSummary := &FinalReportSummary{
		ByRegistryCount: make(map[string]int),
	}
	for image, imageReport := range finalReport.Reports {
		finalReport.Reports[image].Summary = &ImageReportSummary{
			Image: image,
		}

		if imageReport.KubernetesReport != nil {
			finalReport.Reports[image].Summary.KubernetesResourceCount = len(imageReport.KubernetesReport.Resources)
		}

		if imageReport.RegistryReport != nil {
			finalReport.Reports[image].Summary.CurrentTag = imageReport.RegistryReport.CurrentTag
			finalReport.Reports[image].Summary.CurrentTagAgeSeconds = time.Since(imageReport.RegistryReport.CurrentTagTimestamp).Seconds()
			finalReport.Reports[image].Summary.LatestTag = imageReport.RegistryReport.LatestTag
			finalReport.Reports[image].Summary.LatestTagAgeSeconds = time.Since(imageReport.RegistryReport.LatestTagTimestamp).Seconds()
			finalReport.Reports[image].Summary.TagOutOfDateBySeconds = imageReport.RegistryReport.LatestTagTimestamp.Sub(imageReport.RegistryReport.CurrentTagTimestamp).Seconds()
		}

		if imageReport.TrivyReport != nil {
			if imageReport.TrivyReport.ImageIssues != nil {
				finalReport.Reports[image].Summary.IssuesCriticalCount = imageReport.TrivyReport.ImageIssues.Total.Critical
				finalReport.Reports[image].Summary.IssuesHighCount = imageReport.TrivyReport.ImageIssues.Total.High
				finalReport.Reports[image].Summary.IssuesMediumCount = imageReport.TrivyReport.ImageIssues.Total.Medium
				finalReport.Reports[image].Summary.IssuesLowCount = imageReport.TrivyReport.ImageIssues.Total.Low
				finalReport.Reports[image].Summary.IssuesUnknownCount = imageReport.TrivyReport.ImageIssues.Total.Unknown
				finalReportSummary.IssuesCriticalCount += imageReport.TrivyReport.ImageIssues.Total.Critical
				finalReportSummary.IssuesHighCount += imageReport.TrivyReport.ImageIssues.Total.High
				finalReportSummary.IssuesMediumCount += imageReport.TrivyReport.ImageIssues.Total.Medium
				finalReportSummary.IssuesLowCount += imageReport.TrivyReport.ImageIssues.Total.Low
				finalReportSummary.IssuesUnknownCount += imageReport.TrivyReport.ImageIssues.Total.Unknown
			}
		}
		finalReportSummary.ImageCount++

		if _, ok := finalReportSummary.ByRegistryCount[imageReport.Image.Registry]; !ok {
			finalReportSummary.ByRegistryCount[imageReport.Image.Registry] = 0
		}
		finalReportSummary.ByRegistryCount[imageReport.Image.Registry]++
	}
	finalReportSummary.RunDurationSeconds = time.Since(start).Seconds()
	finalReport.Summary = finalReportSummary
}

func LogResults(finalReport *FinalReport) {

	imageSummary := make(map[string]*ImageReportSummary)
	// imageSummary := []*ImageReportSummary{}

	// Output each image report
	for image, imageReport := range finalReport.Reports {
		// log.WithFields(log.Fields{
		// 	"image":       image,
		// 	"report_type": "image",
		// 	"report":      imageReport,
		// }).Info()
		imageSummary[image] = imageReport.Summary
		// imageSummary = append(imageSummary, imageReport.Summary)

		log.WithFields(log.Fields{
			"image":       image,
			"report_type": "image_summary",
			"report":      imageReport.Summary,
		}).Info()

		log.WithFields(log.Fields{
			"image":       image,
			"report_type": "image_registry",
			"report":      imageReport.RegistryReport,
		}).Info()

		if imageReport.TrivyReport != nil {
			if imageReport.TrivyReport.ImageIssues != nil {
				if imageReport.TrivyReport.ImageIssues.Vulnerabilities != nil {
					log.WithFields(log.Fields{
						"image":       image,
						"report_type": "image_vulnerabilities",
						"report":      imageReport.TrivyReport.ImageIssues.Vulnerabilities.Vulnerabilities,
					}).Info()
				}
			}
		}

		if imageReport.KubernetesReport != nil {
			log.WithFields(log.Fields{
				"image":       image,
				"report_type": "image_kubernetes_resources",
				"report":      imageReport.KubernetesReport.Resources,
			}).Info()
		}
	}

	log.WithFields(log.Fields{
		"report_type": "combined_image_summary",
		"report":      imageSummary,
	}).Info()

	log.WithFields(log.Fields{
		"report_type": "summary",
		"report":      finalReport.Summary,
	}).Info()

}
