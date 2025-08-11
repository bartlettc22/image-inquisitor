package reports

import (
	"github.com/bartlettc22/image-inquisitor/internal/inventory"
	"github.com/bartlettc22/image-inquisitor/internal/trivy"
	"github.com/bartlettc22/image-inquisitor/pkg/api/metadata"
	reportsapi "github.com/bartlettc22/image-inquisitor/pkg/api/v1alpha1/reports"
	sourcesapi "github.com/bartlettc22/image-inquisitor/pkg/api/v1alpha1/sources"
	"github.com/google/uuid"
)

func GenerateSummaryReport(inventory inventory.Inventory, runID uuid.UUID) *metadata.Manifest {

	report := &reportsapi.ReportSummary{}
	if inventory != nil {
		report.RepoCount = len(inventory)
		for _, imageRefPrefixDetails := range inventory {
			report.DigestCount += len(imageRefPrefixDetails.Digests)
			for _, digestDetails := range imageRefPrefixDetails.Digests {
				for _, source := range digestDetails.Sources {
					switch source.Type {
					case sourcesapi.FileSourceType:
						report.FileSourceCount++
					case sourcesapi.KubernetesSourceType:
						report.KubernetesSourceCount++
					}
				}

				for _, issue := range digestDetails.Issues.Vulnerabilities {
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

	return reportsapi.NewReportManifest(reportsapi.ReportSummaryKind, runID.String(), report)
}
