package trivy

import (
	"time"

	trivyTypes "github.com/aquasecurity/trivy/pkg/types"
)

// Severity is a string representation of a Trivy severity
type Severity string

// IssueType is a string representation of a Trivy issue type
type IssueType string

const (
	SeverityCritical Severity = "CRITICAL"
	SeverityHigh     Severity = "HIGH"
	SeverityMedium   Severity = "MEDIUM"
	SeverityLow      Severity = "LOW"
	SeverityUnknown  Severity = "UNKNOWN"

	IssueTypeVulnerability    IssueType = "VULNERABILITY"
	IssueTypeSecret           IssueType = "SECRET"
	IssueTypeMisconfiguration IssueType = "MISCONFIGURATION"
)

type ImageIssues []*ImageIssue

// ImageIssue is a issue found by Trivy
type ImageIssue struct {
	Type            IssueType `json:"type" yaml:"type"`
	Title           string    `json:"title" yaml:"title"`
	Severity        Severity  `json:"severity" yaml:"severity"`
	VulnerabilityID string    `json:"vulnerability_id,omitempty" yaml:"vulnerabilityID,omitempty"`
	PkgID           string    `json:"pkg_id,omitempty" yaml:"pkgID,omitempty"`
	PrimaryURL      string    `json:"primary_url,omitempty" yaml:"primaryURL,omitempty"`
	// Description     string     `json:"description,omitempty" yaml:"description,omitempty"`
	NvdV3Score    float64    `json:"nvd_v3_score,omitempty" yaml:"nvdV3Score,omitempty"`
	PublishedDate *time.Time `json:"published_date,omitempty" yaml:"publishedDate,omitempty"`
}

func mustParseSeverity(severity string) Severity {
	switch severity {
	case "LOW":
		return SeverityLow
	case "MEDIUM":
		return SeverityMedium
	case "HIGH":
		return SeverityHigh
	case "CRITICAL":
		return SeverityCritical
	default:
		return SeverityUnknown
	}
}

func parseReport(trivyReport *trivyTypes.Report) ImageIssues {
	var issues ImageIssues
	if trivyReport != nil {
		for _, vulnResults := range trivyReport.Results {
			for _, misconfiguration := range vulnResults.Misconfigurations {
				issues = append(issues, &ImageIssue{
					Type:     IssueTypeMisconfiguration,
					Title:    misconfiguration.Title,
					Severity: mustParseSeverity(misconfiguration.Severity),
				})
			}
			for _, vulnerability := range vulnResults.Vulnerabilities {
				nvdScore := float64(0)
				if nvd, ok := vulnerability.CVSS["nvd"]; ok {
					nvdScore = nvd.V3Score
				}
				issues = append(issues, &ImageIssue{
					Type:            IssueTypeVulnerability,
					VulnerabilityID: vulnerability.VulnerabilityID,
					Severity:        mustParseSeverity(vulnerability.Severity),
					PkgID:           vulnerability.PkgID,
					PrimaryURL:      vulnerability.PrimaryURL,
					Title:           vulnerability.Title,
					// Description:     vulnerability.Description,
					NvdV3Score:    nvdScore,
					PublishedDate: vulnerability.PublishedDate,
				})
			}
			for _, secret := range vulnResults.Secrets {
				issues = append(issues, &ImageIssue{
					Type:     IssueTypeSecret,
					Title:    secret.Title,
					Severity: mustParseSeverity(secret.Severity),
				})
			}
		}
	}

	return issues
}
