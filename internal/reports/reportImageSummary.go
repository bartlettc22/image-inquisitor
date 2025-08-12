package reports

import (
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/inventory"
	"github.com/bartlettc22/image-inquisitor/internal/registries"
	"github.com/bartlettc22/image-inquisitor/internal/trivy"
	"github.com/bartlettc22/image-inquisitor/pkg/api/metadata"
	reportsapi "github.com/bartlettc22/image-inquisitor/pkg/api/v1alpha1/reports"
	sourcesapi "github.com/bartlettc22/image-inquisitor/pkg/api/v1alpha1/sources"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

func GenerateImageSummaryReport(inventory inventory.Inventory, runID uuid.UUID) map[string]*metadata.Manifest {

	report := reportsapi.ReportImageSummary{}
	for _, imageRefPrefixDetails := range inventory {

		var latestIssuesCritical, latestIssuesHigh, latestIssuesMedium, latestIssuesLow, latestIssuesUnknown, latestSemverAgeSeconds int
		var latestSemverTag, latestSemverDigest string
		var latestSemverCreated time.Time

		// Get the latest semver details, if it exists
		if latestDigestDetails, ok := imageRefPrefixDetails.Digests[imageRefPrefixDetails.LatestSemverDigest]; ok {
			latestIssuesCritical, latestIssuesHigh, latestIssuesMedium, latestIssuesLow, latestIssuesUnknown = IssuesBySeverity(latestDigestDetails.Issues)
			latestSemverTag = imageRefPrefixDetails.LatestSemverTag
			latestSemverDigest = imageRefPrefixDetails.LatestSemverDigest
			latestSemverCreated = imageRefPrefixDetails.LatestSemverCreated
			latestSemverAgeSeconds = int(time.Since(latestSemverCreated).Seconds())
		}

		for digest, digestDetails := range imageRefPrefixDetails.Digests {
			for _, source := range digestDetails.Sources {
				if source.Type != sourcesapi.RegistryLatestSemverSourceType {

					img, err := registries.NewImage(source.ImageReference)
					if err != nil {
						log.WithField("ref", source.ImageReference).Error("error parsing image reference")
						continue
					}
					noramalizedRef := img.NormalizedRef()

					summary := fetchRefSummary(report, noramalizedRef)
					if summary == nil {
						summary = &reportsapi.ImageSummary{
							Ref:                       noramalizedRef,
							SourceCount:               1,
							Tag:                       img.TagRef(),
							Digest:                    digest,
							Created:                   digestDetails.Created,
							AgeSeconds:                int(time.Since(digestDetails.Created).Seconds()),
							LatestSemverTag:           latestSemverTag,
							LatestSemverDigest:        latestSemverDigest,
							LatestSemverCreated:       latestSemverCreated,
							LatestSemverAgeSeconds:    latestSemverAgeSeconds,
							LatestIssuesCriticalCount: latestIssuesCritical,
							LatestIssuesHighCount:     latestIssuesHigh,
							LatestIssuesMediumCount:   latestIssuesMedium,
							LatestIssuesLowCount:      latestIssuesLow,
							LatestIssuesUnknownCount:  latestIssuesUnknown,
						}

						if !latestSemverCreated.IsZero() {
							summary.OutOfDateBySeconds = int(latestSemverCreated.Sub(digestDetails.Created).Seconds())
						}

						issuesCritical, issuesHigh, issuesMedium, issuesLow, issuesUnknown := IssuesBySeverity(digestDetails.Issues)
						summary.IssuesCriticalCount = issuesCritical
						summary.IssuesHighCount = issuesHigh
						summary.IssuesMediumCount = issuesMedium
						summary.IssuesLowCount = issuesLow
						summary.IssuesUnknownCount = issuesUnknown

						summary.IssuesCriticalChangeByLatestSemver = latestIssuesCritical - issuesCritical
						summary.IssuesHighChangeByLatestSemver = latestIssuesHigh - issuesHigh
						summary.IssuesMediumChangeByLatestSemver = latestIssuesMedium - issuesMedium
						summary.IssuesLowChangeByLatestSemver = latestIssuesLow - issuesLow
						summary.IssuesUnknownChangeByLatestSemver = latestIssuesUnknown - issuesUnknown

						if digest == latestSemverDigest {
							summary.IsLatestSemver = true
						}
						report = append(report, summary)
					} else {
						summary.SourceCount++
					}
				}
			}
		}
	}

	return map[string]*metadata.Manifest{
		"": reportsapi.NewReportManifest(reportsapi.ReportImageSummaryKind, runID.String(), report),
	}
}

func fetchRefSummary(report reportsapi.ReportImageSummary, ref string) *reportsapi.ImageSummary {
	for _, summary := range report {
		if summary.Ref == ref {
			return summary
		}
	}
	return nil
}

func IssuesBySeverity(issues trivy.ImageIssues) (int, int, int, int, int) {

	critical := 0
	high := 0
	medium := 0
	low := 0
	unknown := 0

	for _, issue := range issues {
		switch issue.Severity {
		case trivy.SeverityCritical:
			critical++
		case trivy.SeverityHigh:
			high++
		case trivy.SeverityMedium:
			medium++
		case trivy.SeverityLow:
			low++
		case trivy.SeverityUnknown:
			unknown++
		}
	}

	return critical, high, medium, low, unknown
}
