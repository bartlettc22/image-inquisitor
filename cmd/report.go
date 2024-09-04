package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/imageUtils"
	"github.com/bartlettc22/image-inquisitor/internal/kubernetes"
	"github.com/bartlettc22/image-inquisitor/internal/registries"
	"github.com/bartlettc22/image-inquisitor/internal/reports"
	"github.com/bartlettc22/image-inquisitor/internal/trivy"
	log "github.com/sirupsen/logrus"
)

type FinalReport struct {
	Summary *FinalReportSummary
	Reports map[string]*ImageReport
}

type FinalReportSummary struct {
	ImageCount          int
	IssuesCriticalCount int
	IssuesHighCount     int
	IssuesMediumCount   int
	IssuesLowCount      int
	IssuesUnknownCount  int
	ByRegistryCount     map[string]int
	RunDurationSeconds  float64
}

type ImageReport struct {
	Image            *imageUtils.Image
	Summary          *ImageReportSummary
	KubernetesReport *kubernetes.KubernetesImageReport
	TrivyReport      *trivy.TrivyImageReport
	RegistryReport   *registries.RegistryImageReport
}

type ImageReportSummary struct {
	Image                   string
	CurrentTag              string
	CurrentTagAgeSeconds    float64
	LatestTag               string
	LatestTagAgeSeconds     float64
	TagOutOfDateBySeconds   float64
	KubernetesResourceCount int
	IssuesCriticalCount     int
	IssuesHighCount         int
	IssuesMediumCount       int
	IssuesLowCount          int
	IssuesUnknownCount      int
}

func (r FinalReport) Images() []string {
	images := []string{}
	for image := range r.Reports {
		images = append(images, image)
	}
	return images
}

func printReport(reportType, image string, report interface{}) {
	wrappedReport := reports.ReportWrapper{
		ReportGenerated: time.Now(),
		ReportType:      reportType,
		Image:           image,
		Report:          report,
	}
	reportOut, err := json.Marshal(wrappedReport)
	if err != nil {
		log.Errorf("error converting '%s' report to JSON; err: %v, out: %v", reportType, err, wrappedReport)
	} else {
		fmt.Println(string(reportOut))
	}
}
