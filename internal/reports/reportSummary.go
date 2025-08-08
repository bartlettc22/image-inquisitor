package reports

import (
	"github.com/bartlettc22/image-inquisitor/internal/inventory"
	"github.com/bartlettc22/image-inquisitor/internal/trivy"
	"github.com/bartlettc22/image-inquisitor/pkg/api/metadata"
	reportsapi "github.com/bartlettc22/image-inquisitor/pkg/api/v1alpha1/reports"
	sourcesapi "github.com/bartlettc22/image-inquisitor/pkg/api/v1alpha1/sources"
)

func GenerateSummaryReport(inventory inventory.Inventory) *metadata.Manifest {

	report := &reportsapi.ReportSummary{}
	if inventory != nil {
		report.RepoCount = len(inventory)
		for _, image := range inventory {
			report.DigestCount += len(image.Digests)
			for _, digest := range image.Digests {
				for _, source := range digest.Sources {
					switch source.Type {
					case sourcesapi.FileSourceType:
						report.FileSourceCount++
					case sourcesapi.KubernetesSourceType:
						report.KubernetesSourceCount++
					}
				}

				for _, issue := range digest.Issues.Vulnerabilities {
					switch issue.Severity {
					case trivy.Critical:
						report.IssuesCriticalCount++
					case trivy.High:
						report.IssuesHighCount++
					case trivy.Medium:
						report.IssuesMediumCount++
					case trivy.Low:
						report.IssuesLowCount++
					case trivy.Unknown:
						report.IssuesUnknownCount++
					}
				}
			}
		}
	}

	return reportsapi.NewReportManifest(reportsapi.ReportSummaryKind, report)
}
