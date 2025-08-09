package trivy

import (
	"time"

	trivyTypes "github.com/aquasecurity/trivy/pkg/types"
)

// Severity is a string representation of a Trivy severity
type Severity string

const (
	Critical Severity = "CRITICAL"
	High     Severity = "HIGH"
	Medium   Severity = "MEDIUM"
	Low      Severity = "LOW"
	Unknown  Severity = "UNKNOWN"
)

// ImageIssues is a list of issues found by Trivy
type ImageIssues struct {
	Misconfigurations []*ImageIssueMisconfiguration `json:"misconfigurations,omitempty" yaml:"misconfigurations,omitempty"`
	Vulnerabilities   []*ImageIssueVulnerability    `json:"vulnerabilities,omitempty" yaml:"vulnerabilities,omitempty"`
	Secrets           []*ImageIssueSecret           `json:"secrets,omitempty" yaml:"secrets,omitempty"`
}

// ImageIssueMisconfiguration is a misconfiguration issue found by Trivy
type ImageIssueMisconfiguration struct {
	Title    string   `json:"title" yaml:"title"`
	Severity Severity `json:"severity" yaml:"severity"`
}

// ImageIssueVulnerability is a vulnerability issue found by Trivy
type ImageIssueVulnerability struct {
	VulnerabilityID string     `json:"vulnerabilityID" yaml:"vulnerabilityID"`
	Severity        Severity   `json:"severity" yaml:"severity"`
	PkgID           string     `json:"pkgID" yaml:"pkgID"`
	PrimaryURL      string     `json:"primaryURL" yaml:"primaryURL"`
	Title           string     `json:"title" yaml:"title"`
	Description     string     `json:"description" yaml:"description"`
	NvdV3Score      float64    `json:"nvdV3Score" yaml:"nvdV3Score"`
	PublishedDate   *time.Time `json:"publishedDate" yaml:"publishedDate"`
}

// ImageIssueSecret is a secret issue found by Trivy
type ImageIssueSecret struct {
	Title    string   `json:"title " yaml:"title"`
	Severity Severity `json:"severity" yaml:"severity"`
}

func mustParseSeverity(severity string) Severity {
	switch severity {
	case "LOW":
		return Low
	case "MEDIUM":
		return Medium
	case "HIGH":
		return High
	case "CRITICAL":
		return Critical
	default:
		return Unknown
	}
}

func parseReport(trivyReport *trivyTypes.Report) *ImageIssues {
	issues := &ImageIssues{}
	if trivyReport != nil {
		for _, vulnResults := range trivyReport.Results {
			for _, misconfiguration := range vulnResults.Misconfigurations {
				issues.Misconfigurations = append(issues.Misconfigurations, &ImageIssueMisconfiguration{
					Title:    misconfiguration.Title,
					Severity: mustParseSeverity(misconfiguration.Severity),
				})
			}
			for _, vulnerability := range vulnResults.Vulnerabilities {
				nvdScore := float64(0)
				if nvd, ok := vulnerability.CVSS["nvd"]; ok {
					nvdScore = nvd.V3Score
				}
				issues.Vulnerabilities = append(issues.Vulnerabilities, &ImageIssueVulnerability{
					VulnerabilityID: vulnerability.VulnerabilityID,
					Severity:        mustParseSeverity(vulnerability.Severity),
					PkgID:           vulnerability.PkgID,
					PrimaryURL:      vulnerability.PrimaryURL,
					Title:           vulnerability.Title,
					Description:     vulnerability.Description,
					NvdV3Score:      nvdScore,
					PublishedDate:   vulnerability.PublishedDate,
				})
			}
			for _, secret := range vulnResults.Secrets {
				issues.Secrets = append(issues.Secrets, &ImageIssueSecret{
					Title:    secret.Title,
					Severity: mustParseSeverity(secret.Severity),
				})
			}
		}
	}

	return issues
}
