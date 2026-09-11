package trivy

import (
	"testing"

	dbTypes "github.com/aquasecurity/trivy-db/pkg/types"
	trivyTypes "github.com/aquasecurity/trivy/pkg/types"
)

// Severity counts must be identical whether or not the per-CVE detail array is
// built. The detail flag controls payload only; it must never change the numbers.
//
// Regression test: Total.AddSeverity originally sat after the detail append, so
// gating the append also stopped the counting. Critical counts
// collapsed to near zero.
func TestDetailGateDoesNotChangeCounts(t *testing.T) {
	mk := func() RunResults {
		return RunResults{
			"img": &ScanResult{
				Image: "img",
				Report: &trivyTypes.Report{
					Results: trivyTypes.Results{{
						Vulnerabilities: []trivyTypes.DetectedVulnerability{
							{VulnerabilityID: "CVE-1", Vulnerability: dbTypes.Vulnerability{Severity: "CRITICAL"}},
							{VulnerabilityID: "CVE-2", Vulnerability: dbTypes.Vulnerability{Severity: "CRITICAL"}},
							{VulnerabilityID: "CVE-3", Vulnerability: dbTypes.Vulnerability{Severity: "HIGH"}},
						},
					}},
				},
			},
		}
	}

	withDetail := formatReport(mk(), true)["img"]
	without := formatReport(mk(), false)["img"]

	if withDetail.ImageIssues.Total.Critical != without.ImageIssues.Total.Critical {
		t.Errorf("Total.Critical differs: detail=%d nodetail=%d",
			withDetail.ImageIssues.Total.Critical, without.ImageIssues.Total.Critical)
	}
	if withDetail.ImageIssues.Total.High != without.ImageIssues.Total.High {
		t.Errorf("Total.High differs: detail=%d nodetail=%d",
			withDetail.ImageIssues.Total.High, without.ImageIssues.Total.High)
	}
	if without.ImageIssues.Total.Critical != 2 {
		t.Errorf("Total.Critical = %d, want 2", without.ImageIssues.Total.Critical)
	}
	// ...and the payload really is dropped when not requested.
	if n := len(without.ImageIssues.Vulnerabilities.Vulnerabilities); n != 0 {
		t.Errorf("detail array should be empty, got %d entries", n)
	}
	if n := len(withDetail.ImageIssues.Vulnerabilities.Vulnerabilities); n != 3 {
		t.Errorf("detail array should hold 3 entries, got %d", n)
	}
}
