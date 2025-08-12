package reports

import (
	"github.com/bartlettc22/image-inquisitor/internal/inventory"
	"github.com/bartlettc22/image-inquisitor/internal/trivy"
	"github.com/bartlettc22/image-inquisitor/pkg/api/metadata"
	reportsapi "github.com/bartlettc22/image-inquisitor/pkg/api/v1alpha1/reports"
	sourcesapi "github.com/bartlettc22/image-inquisitor/pkg/api/v1alpha1/sources"
	"github.com/google/uuid"
)

func GenerateSummaryReport(inventory inventory.Inventory, runID uuid.UUID) map[string]*metadata.Manifest {

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

				if digestDetails.Issues != nil {
					for _, issue := range digestDetails.Issues {
						switch issue.Severity {
						case trivy.SeverityCritical:
							report.IssuesCriticalCount++
						case trivy.SeverityHigh:
							report.IssuesHighCount++
						case trivy.SeverityMedium:
							report.IssuesMediumCount++
						case trivy.SeverityLow:
							report.IssuesLowCount++
						case trivy.SeverityUnknown:
							report.IssuesUnknownCount++
						}
					}
				}
			}
		}
	}

	return map[string]*metadata.Manifest{
		"": reportsapi.NewReportManifest(reportsapi.ReportSummaryKind, runID.String(), report),
	}
}
