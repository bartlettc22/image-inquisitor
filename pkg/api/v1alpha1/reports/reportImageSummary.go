package reports

import "time"

const (
	// ReportImageSummaryKind is the kind of a report manifest
	ReportImageSummaryKind ReportKind = "ImageSummaryReport"
)

// ReportSuReportImageSummary is a summary report
type ReportImageSummary []*ImageSummary

// ImageSummary is a summary of an image
type ImageSummary struct {
	Ref                                string    `json:"ref" yaml:"ref"`
	Tag                                string    `json:"tag" yaml:"tag"`
	Digest                             string    `json:"digest" yaml:"digest"`
	Created                            time.Time `json:"created" yaml:"created"`
	AgeSeconds                         int       `json:"ageSeconds" yaml:"ageSeconds"`
	SourceCount                        int       `json:"sourceCount" yaml:"sourceCount"`
	IsLatestSemver                     bool      `json:"isLatestSemver" yaml:"isLatestSemver"`
	LatestSemverTag                    string    `json:"latestSemverTag" yaml:"latestSemverTag"`
	LatestSemverDigest                 string    `json:"latestSemverDigest" yaml:"latestSemverDigest"`
	LatestSemverCreated                time.Time `json:"latestSemverCreated" yaml:"latestSemverCreated"`
	LatestSemverAgeSeconds             int       `json:"latestSemverAgeSeconds" yaml:"latestSemverAgeSeconds"`
	OutOfDateBySeconds                 int       `json:"outOfDateBySeconds" yaml:"outOfDateBySeconds"`
	IssuesCriticalCount                int       `json:"issuesCriticalCount" yaml:"issuesCriticalCount"`
	IssuesHighCount                    int       `json:"issuesHighCount" yaml:"issuesHighCount"`
	IssuesMediumCount                  int       `json:"issuesMediumCount" yaml:"issuesMediumCount"`
	IssuesLowCount                     int       `json:"issuesLowCount" yaml:"issuesLowCount"`
	IssuesUnknownCount                 int       `json:"issuesUnknownCount" yaml:"issuesUnknownCount"`
	LatestIssuesCriticalCount          int       `json:"latestIssuesCriticalCount" yaml:"latestIssuesCriticalCount"`
	LatestIssuesHighCount              int       `json:"latestIssuesHighCount" yaml:"latestIssuesHighCount"`
	LatestIssuesMediumCount            int       `json:"latestIssuesMediumCount" yaml:"latestIssuesMediumCount"`
	LatestIssuesLowCount               int       `json:"latestIssuesLowCount" yaml:"latestIssuesLowCount"`
	LatestIssuesUnknownCount           int       `json:"latestIssuesUnknownCount" yaml:"latestIssuesUnknownCount"`
	IssuesCriticalChangeByLatestSemver int       `json:"issuesCriticalChangeByLatestSemver" yaml:"issuesCriticalChangeByLatestSemver"`
	IssuesHighChangeByLatestSemver     int       `json:"issuesHighChangeByLatestSemver" yaml:"issuesHighChangeByLatestSemver"`
	IssuesMediumChangeByLatestSemver   int       `json:"issuesMediumChangeByLatestSemver" yaml:"issuesMediumChangeByLatestSemver"`
	IssuesLowChangeByLatestSemver      int       `json:"issuesLowChangeByLatestSemver" yaml:"issuesLowChangeByLatestSemver"`
	IssuesUnknownChangeByLatestSemver  int       `json:"issuesUnknownChangeByLatestSemver" yaml:"issuesUnknownChangeByLatestSemver"`
}
