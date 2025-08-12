package reports

import (
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/trivy"
	sourcesapi "github.com/bartlettc22/image-inquisitor/pkg/api/v1alpha1/sources"
)

const (
	// ReportImageKind is the kind of a report manifest
	ReportImageKind ReportKind = "ImageReport"
)

// ReportImage contains the details of a single image
type ReportImage struct {
	Ref string `json:"ref" yaml:"ref"`

	Registry         string `json:"registry" yaml:"registry"`
	Repository       string `json:"repository" yaml:"repository"`
	RepositoryPrefix string `json:"repository_prefix" yaml:"repositoryPrefix"`
	Digest           string `json:"digest" yaml:"digest"`

	LatestSemverTag        string    `json:"latest_semver_tag" yaml:"latestSemverTag"`
	LatestSemverDigest     string    `json:"latest_semver_digest" yaml:"latestSemverDigest"`
	LatestSemverCreated    time.Time `json:"latest_semver_created" yaml:"latestSemverCreated"`
	LatestSemverAgeSeconds int       `json:"latest_semver_age_seconds" yaml:"latestSemverAgeSeconds"`

	Created     time.Time `json:"created" yaml:"created"`
	AgeSeconds  int       `json:"age_seconds" yaml:"ageSeconds"`
	SourceCount int       `json:"source_count" yaml:"sourceCount"`

	IsLatestSemver     bool `json:"is_latest_semver" yaml:"isLatestSemver"`
	OutOfDateBySeconds int  `json:"out_of_date_by_seconds" yaml:"outOfDateBySeconds"`

	IssuesCriticalCount int `json:"issues_critical_count" yaml:"issuesCriticalCount"`
	IssuesHighCount     int `json:"issues_high_count" yaml:"issuesHighCount"`
	IssuesMediumCount   int `json:"issues_medium_count" yaml:"issuesMediumCount"`
	IssuesLowCount      int `json:"issues_low_count" yaml:"issuesLowCount"`
	IssuesUnknownCount  int `json:"issues_unknown_count" yaml:"issuesUnknownCount"`

	LatestIssuesCriticalCount int `json:"latest_issues_critical_count" yaml:"latestIssuesCriticalCount"`
	LatestIssuesHighCount     int `json:"latest_issues_high_count" yaml:"latestIssuesHighCount"`
	LatestIssuesMediumCount   int `json:"latest_issues_medium_count" yaml:"latestIssuesMediumCount"`
	LatestIssuesLowCount      int `json:"latest_issues_low_count" yaml:"latestIssuesLowCount"`
	LatestIssuesUnknownCount  int `json:"latest_issues_unknown_count" yaml:"latestIssuesUnknownCount"`

	IssuesCriticalChangeByLatestSemver int `json:"issues_critical_change_by_latest_semver" yaml:"issuesCriticalChangeByLatestSemver"`
	IssuesHighChangeByLatestSemver     int `json:"issues_high_change_by_latest_semver" yaml:"issuesHighChangeByLatestSemver"`
	IssuesMediumChangeByLatestSemver   int `json:"issues_medium_change_by_latest_semver" yaml:"issuesMediumChangeByLatestSemver"`
	IssuesLowChangeByLatestSemver      int `json:"issues_low_change_by_latest_semver" yaml:"issuesLowChangeByLatestSemver"`
	IssuesUnknownChangeByLatestSemver  int `json:"issues_unknown_change_by_latest_semver" yaml:"issuesUnknownChangeByLatestSemver"`

	Sources []*sourcesapi.Source `json:"sources" yaml:"sources"`
	Issues  trivy.ImageIssues    `json:"issues" yaml:"issues"`
}
