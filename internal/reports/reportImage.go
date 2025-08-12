package reports

import (
	"fmt"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/inventory"
	"github.com/bartlettc22/image-inquisitor/pkg/api/metadata"
	reportsapi "github.com/bartlettc22/image-inquisitor/pkg/api/v1alpha1/reports"
	"github.com/google/uuid"
)

func GenerateImageReports(inventory inventory.Inventory, runID uuid.UUID) map[string]*metadata.Manifest {

	reports := map[string]*reportsapi.ReportImage{}
	for imageRefPrefix, imageRefPrefixDetails := range inventory {

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
			ref := fmt.Sprintf("%s@%s", imageRefPrefix, digest)
			issuesCritical, issuesHigh, issuesMedium, issuesLow, issuesUnknown := IssuesBySeverity(digestDetails.Issues)
			report := &reportsapi.ReportImage{
				Ref:              ref,
				Registry:         imageRefPrefixDetails.Registry,
				Repository:       imageRefPrefixDetails.Repository,
				RepositoryPrefix: imageRefPrefix,
				Digest:           digest,

				LatestSemverTag:        latestSemverTag,
				LatestSemverDigest:     latestSemverDigest,
				LatestSemverCreated:    latestSemverCreated,
				LatestSemverAgeSeconds: latestSemverAgeSeconds,

				Created:     digestDetails.Created,
				AgeSeconds:  int(time.Since(digestDetails.Created).Seconds()),
				SourceCount: len(digestDetails.Sources),

				IsLatestSemver:     digest == latestSemverDigest,
				OutOfDateBySeconds: int(latestSemverCreated.Sub(digestDetails.Created).Seconds()),

				IssuesCriticalCount: issuesCritical,
				IssuesHighCount:     issuesHigh,
				IssuesMediumCount:   issuesMedium,
				IssuesLowCount:      issuesLow,
				IssuesUnknownCount:  issuesUnknown,

				Sources: digestDetails.Sources,
				Issues:  digestDetails.Issues,
			}

			if latestSemverTag != "" {
				report.LatestIssuesCriticalCount = latestIssuesCritical
				report.LatestIssuesHighCount = latestIssuesHigh
				report.LatestIssuesMediumCount = latestIssuesMedium
				report.LatestIssuesLowCount = latestIssuesLow
				report.LatestIssuesUnknownCount = latestIssuesUnknown
				report.IssuesCriticalChangeByLatestSemver = latestIssuesCritical - issuesCritical
				report.IssuesHighChangeByLatestSemver = latestIssuesHigh - issuesHigh
				report.IssuesMediumChangeByLatestSemver = latestIssuesMedium - issuesMedium
				report.IssuesLowChangeByLatestSemver = latestIssuesLow - issuesLow
				report.IssuesUnknownChangeByLatestSemver = latestIssuesUnknown - issuesUnknown
			}

			reports[ref] = report
		}
	}

	manifests := make(map[string]*metadata.Manifest)
	for ref, report := range reports {
		manifests[ref] = reportsapi.NewReportManifest(reportsapi.ReportImageKind, runID.String(), report)
	}

	return manifests
}
