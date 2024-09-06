package reports

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/imageUtils"
	"github.com/bartlettc22/image-inquisitor/internal/kubernetes"
	"github.com/bartlettc22/image-inquisitor/internal/registries"
	"github.com/bartlettc22/image-inquisitor/internal/trivy"
	log "github.com/sirupsen/logrus"
)

type summaryReportList struct {
	reportGenerated time.Time
	reports         map[ReportType]*summaryReportWrapper
}

type summaryReportWrapper struct {
	ReportGenerated time.Time   `json:"report_generated"`
	ReportType      ReportType  `json:"report_type"`
	Contents        interface{} `json:"report"`
}

func NewSummaryReportList(reportGenerated time.Time) *summaryReportList {
	return &summaryReportList{
		reportGenerated: reportGenerated,
		reports:         make(map[ReportType]*summaryReportWrapper),
	}
}

func (rl *summaryReportList) Add(reportType ReportType, reportContents interface{}) {
	rl.reports[reportType] = &summaryReportWrapper{
		ReportGenerated: rl.reportGenerated,
		ReportType:      reportType,
		Contents:        reportContents,
	}
}

func (rl *summaryReportList) Output() {
	for _, report := range rl.reports {
		report.output()
	}
}

func (rl *summaryReportList) GenerateSummaryReports(masterImagesList imageUtils.ImagesList, masterImageReportList *imageReportList) {

	summaryReport := &summaryReport{}
	combinedReport := make(map[string]*summaryImageCombinedReport)
	summaryRegistry := make(map[string]*summaryRegistryReport)

	for image, imageReport := range masterImagesList {
		summaryReport.ImageCount++
		combinedReport[image] = &summaryImageCombinedReport{
			Image: image,
		}
		if _, ok := summaryRegistry[imageReport.Registry]; !ok {
			summaryRegistry[imageReport.Registry] = &summaryRegistryReport{
				Registry: imageReport.Registry,
			}
		}
		summaryRegistry[imageReport.Registry].ImageCount++

	}

	for reportType, imageReportSets := range masterImageReportList.reportSets {
		switch reportType {
		case ReportTypeImageVulnerabilities:
			for image, imageReport := range imageReportSets {
				vulnerabilityReport, ok := imageReport.Contents.(*trivy.TrivyImageReport)
				if !ok {
					log.Error("type assertion failed for summary of trivy.TrivyImageReport report")
				}
				if vulnerabilityReport.ImageIssues != nil {
					summaryReport.IssuesCriticalCount += vulnerabilityReport.ImageIssues.Total.Critical
					summaryReport.IssuesHighCount += vulnerabilityReport.ImageIssues.Total.High
					summaryReport.IssuesMediumCount += vulnerabilityReport.ImageIssues.Total.Medium
					summaryReport.IssuesLowCount += vulnerabilityReport.ImageIssues.Total.Low
					summaryReport.IssuesUnknownCount += vulnerabilityReport.ImageIssues.Total.Unknown
					combinedReport[image].IssuesCriticalCount += vulnerabilityReport.ImageIssues.Total.Critical
					combinedReport[image].IssuesHighCount += vulnerabilityReport.ImageIssues.Total.High
					combinedReport[image].IssuesMediumCount += vulnerabilityReport.ImageIssues.Total.Medium
					combinedReport[image].IssuesLowCount += vulnerabilityReport.ImageIssues.Total.Low
					combinedReport[image].IssuesUnknownCount += vulnerabilityReport.ImageIssues.Total.Unknown
				}
			}
		case ReportTypeImageKubernetes:
			for image, imageReport := range imageReportSets {
				kubernetesReport, ok := imageReport.Contents.(*kubernetes.KubernetesImageReport)
				if !ok {
					log.Error("type assertion failed for summary of kubernetes.KubernetesImageReport report")
				}
				summaryReport.KubernetesResourceCount += len(kubernetesReport.Resources)
				combinedReport[image].KubernetesResourceCount += len(kubernetesReport.Resources)
			}
		case ReportTypeImageRegistry:
			for image, imageReport := range imageReportSets {
				registryReport, ok := imageReport.Contents.(*registries.RegistryImageReport)
				if !ok {
					log.Error("type assertion failed for summary of registries.RegistryImageReport report")
				}
				combinedReport[image].Registry = registryReport.Registry
				combinedReport[image].Tag = registryReport.Tag
				combinedReport[image].TagAgeSeconds = time.Since(registryReport.TagTimestamp).Seconds()
				combinedReport[image].LatestTag = registryReport.LatestTag
				combinedReport[image].LatestTagAgeSeconds = time.Since(registryReport.LatestTagTimestamp).Seconds()
				combinedReport[image].TagOutOfDateBySeconds = registryReport.LatestTagTimestamp.Sub(registryReport.TagTimestamp).Seconds()
			}
		}

	}

	rl.Add(ReportTypeSummary, summaryReport)
	rl.Add(ReportTypeSummaryImageCombined, combinedReport)
	rl.Add(ReportTypeSummaryRegistry, summaryRegistry)
}

func (r *summaryReportWrapper) output() {
	reportOut, err := json.Marshal(r)
	if err != nil {
		log.Errorf("error converting '%s' report to JSON; err: %v, out: %v", r.ReportType, err, r)
	} else {
		fmt.Println(string(reportOut))
	}
}

type summaryReport struct {
	ImageCount              int `json:"ImageCount"`
	KubernetesResourceCount int `json:"KubernetesResourceCount"`
	IssuesCriticalCount     int `json:"IssuesCriticalCount,omitempty"`
	IssuesHighCount         int `json:"IssuesHighCount,omitempty"`
	IssuesMediumCount       int `json:"IssuesMediumCount,omitempty"`
	IssuesLowCount          int `json:"IssuesLowCount,omitempty"`
	IssuesUnknownCount      int `json:"IssuesUnknownCount,omitempty"`
}

type summaryImageCombinedReport struct {
	Image                   string  `json:"Image"`
	Registry                string  `json:"Registry"`
	Tag                     string  `json:"Tag"`
	TagAgeSeconds           float64 `json:"TagAgeSeconds"`
	LatestTag               string  `json:"LatestTag"`
	LatestTagAgeSeconds     float64 `json:"LatestTagAgeSeconds"`
	TagOutOfDateBySeconds   float64 `json:"TagOutOfDateBySeconds"`
	KubernetesResourceCount int     `json:"KubernetesResourceCount"`
	IssuesCriticalCount     int     `json:"IssuesCriticalCount,omitempty"`
	IssuesHighCount         int     `json:"IssuesHighCount,omitempty"`
	IssuesMediumCount       int     `json:"IssuesMediumCount,omitempty"`
	IssuesLowCount          int     `json:"IssuesLowCount,omitempty"`
	IssuesUnknownCount      int     `json:"IssuesUnknownCount,omitempty"`
}

type summaryRegistryReport struct {
	Registry   string `json:"registry"`
	ImageCount int    `json:"ImageCount"`
}
