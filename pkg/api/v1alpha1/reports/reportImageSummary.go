package reports

import "time"

const (
	// ReportImageSummaryKind is the kind of a report manifest
	ReportImageSummaryKind ReportKind = "ImageSummaryReport"
)

// ReportImageSummary is a summary report
type ReportImageSummary []*ImageSummary

// ImageSummary is a summary of an image
type ImageSummary struct {
	Ref                                string    `json:"ref" yaml:"ref"`
	Tag                                string    `json:"tag" yaml:"tag"`
	Digest                             string    `json:"digest" yaml:"digest"`
	Created                            time.Time `json:"created" yaml:"created"`
	AgeSeconds                         int       `json:"age_seconds" yaml:"ageSeconds"`
	SourceCount                        int       `json:"source_count" yaml:"sourceCount"`
	IsLatestSemver                     bool      `json:"is_latest_semver" yaml:"isLatestSemver"`
	LatestSemverTag                    string    `json:"latest_semver_tag" yaml:"latestSemverTag"`
	LatestSemverDigest                 string    `json:"latest_semver_digest" yaml:"latestSemverDigest"`
	LatestSemverCreated                time.Time `json:"latest_semver_created" yaml:"latestSemverCreated"`
	LatestSemverAgeSeconds             int       `json:"latest_semver_age_seconds" yaml:"latestSemverAgeSeconds"`
	OutOfDateBySeconds                 int       `json:"out_of_date_by_seconds" yaml:"outOfDateBySeconds"`
	IssuesCriticalCount                int       `json:"issues_critical_count" yaml:"issuesCriticalCount"`
	IssuesHighCount                    int       `json:"issues_high_count" yaml:"issuesHighCount"`
	IssuesMediumCount                  int       `json:"issues_medium_count" yaml:"issuesMediumCount"`
	IssuesLowCount                     int       `json:"issues_low_count" yaml:"issuesLowCount"`
	IssuesUnknownCount                 int       `json:"issues_unknown_count" yaml:"issuesUnknownCount"`
	TotalIssuesCount                   int       `json:"total_issues_count" yaml:"totalIssuesCount"`
	LatestIssuesCriticalCount          int       `json:"latest_issues_critical_count" yaml:"latestIssuesCriticalCount"`
	LatestIssuesHighCount              int       `json:"latest_issues_high_count" yaml:"latestIssuesHighCount"`
	LatestIssuesMediumCount            int       `json:"latest_issues_medium_count" yaml:"latestIssuesMediumCount"`
	LatestIssuesLowCount               int       `json:"latest_issues_low_count" yaml:"latestIssuesLowCount"`
	LatestIssuesUnknownCount           int       `json:"latest_issues_unknown_count" yaml:"latestIssuesUnknownCount"`
	IssuesCriticalChangeByLatestSemver int       `json:"issues_critical_change_by_latest_semver" yaml:"issuesCriticalChangeByLatestSemver"`
	IssuesHighChangeByLatestSemver     int       `json:"issues_high_change_by_latest_semver" yaml:"issuesHighChangeByLatestSemver"`
	IssuesMediumChangeByLatestSemver   int       `json:"issues_medium_change_by_latest_semver" yaml:"issuesMediumChangeByLatestSemver"`
	IssuesLowChangeByLatestSemver      int       `json:"issues_low_change_by_latest_semver" yaml:"issuesLowChangeByLatestSemver"`
	IssuesUnknownChangeByLatestSemver  int       `json:"issues_unknown_change_by_latest_semver" yaml:"issuesUnknownChangeByLatestSemver"`
}
