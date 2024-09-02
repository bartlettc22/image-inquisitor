package main

import (
	"github.com/bartlettc22/image-inquisitor/internal/imageUtils"
	"github.com/bartlettc22/image-inquisitor/internal/kubernetes"
	"github.com/bartlettc22/image-inquisitor/internal/registries"
	"github.com/bartlettc22/image-inquisitor/internal/trivy"
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
	RegistryReport   *registries.ImageReport
}

type ImageReportSummary struct {
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
	for image, _ := range r.Reports {
		images = append(images, image)
	}
	return images
}
