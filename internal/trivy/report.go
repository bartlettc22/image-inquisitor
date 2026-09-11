package trivy

import (
	"time"

	log "github.com/sirupsen/logrus"
)

type TrivyReport map[string]*TrivyImageReport

type TrivyImageReport struct {
	ImageCreated time.Time `json:"imageCreated"`
	// Age of the image in whole days at report time. Derived from ImageCreated,
	// but emitted as a number because consumers cannot threshold or sort on a
	// timestamp: Grafana compares thresholds against absolute values, so
	// colouring "older than N days" from a date would need a hardcoded cutoff
	// that goes stale. A plain integer is directly sortable and colourable.
	AgeDays     int          `json:"ageDays"`
	ImageIssues *ImageIssues `json:"imageIssues"`

	// Upgrade context, copied from the registry report for the same image after
	// both reports are collected. It is duplicated here on purpose: consumers
	// need "how bad is this image" and "can an upgrade do anything about it" in
	// the SAME record. Keeping them in separate report types forces a join on
	// image name, and in Grafana that means joinByField across two queries whose
	// keys repeat once per scan -- fragile enough to be worth the duplication.
	//
	// UpgradeState is deliberately three-valued, and "check" rather than
	// "behind", because the underlying LatestTag is the highest semver tag in the
	// repository and is wrong more often than right: it flags pinned tracks
	// (node:24-alpine -> 26.8.2), equivalent tags (busybox:1.38 -> 1.38.0) and
	// unrelated tag series (app:v0.17.3 -> v1.6.5). Equality is
	// trustworthy; inequality is not. So "current" is a real signal and anything
	// else only says "worth a look".
	RunningTag string `json:"runningTag,omitempty"`
	LatestTag  string `json:"latestSemverTag,omitempty"`
	// Always one of: current, check, unknown. Never empty -- an absent label and
	// a label reading "unknown" behave very differently in a label filter, and a
	// missing one silently removes the image from filtered views.
	UpgradeState string `json:"upgradeState"`
}

// UpgradeState values. Kept as constants so the dashboards and the scanner
// cannot drift apart on spelling.
const (
	UpgradeStateCurrent = "current"
	UpgradeStateCheck   = "check"
	UpgradeStateUnknown = "unknown"
)

// SetUpgradeContext records how the running tag compares to the newest tag the
// registry knows about. latestTag empty means the registry could not determine
// one, which is a different thing from being out of date.
func (r *TrivyImageReport) SetUpgradeContext(runningTag, latestTag string) {
	r.RunningTag = runningTag
	r.LatestTag = latestTag
	switch {
	case latestTag == "":
		r.UpgradeState = UpgradeStateUnknown
	case runningTag == latestTag:
		r.UpgradeState = UpgradeStateCurrent
	default:
		r.UpgradeState = UpgradeStateCheck
	}
}

type ImageIssues struct {
	Total             *ImageIssueSeverity          `json:"total"`
	Misconfigurations *ImageIssueMisconfigurations `json:"misconfigurations"`
	Vulnerabilities   *ImageIssueVulnerabilities   `json:"vulnerabilities"`
	Secrets           *ImageIssueSecrets           `json:"secrets"`
}

type ImageIssueMisconfigurations struct {
	ImageIssueSeverity
}

type ImageIssueVulnerabilities struct {
	ImageIssueSeverity
	Vulnerabilities []*ImageIssueVulnerability `json:"vulnerabilities"`
}

type ImageIssueVulnerability struct {
	VulnerabilityID string     `json:"registry"`
	Severity        string     `json:"severity"`
	PkgID           string     `json:"pkgID"`
	PrimaryURL      string     `json:"primaryURL"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	NvdV3Score      float64    `json:"nvdV3Score"`
	PublishedDate   *time.Time `json:"publishedDate"`
}

type ImageIssueSecrets struct {
	ImageIssueSeverity
}

type IssueWithSeverity interface {
	AddSeverityUnknown()
	AddSeverityLow()
	AddSeverityMedium()
	AddSeverityHigh()
	AddSeverityCritical()
}

type ImageIssueSeverity struct {
	Unknown  int `json:"unknown"`
	Low      int `json:"low"`
	Medium   int `json:"medium"`
	High     int `json:"high"`
	Critical int `json:"critical"`
}

func (iis *ImageIssueSeverity) AddSeverity(severity string) {
	switch severity {
	case "LOW":
		iis.Low++
	case "MEDIUM":
		iis.Medium++
	case "HIGH":
		iis.High++
	case "CRITICAL":
		iis.Critical++
	default:
		iis.Unknown++
	}
}

func formatReport(runResults RunResults) TrivyReport {
	trivyReport := make(TrivyReport)
	for _, r := range runResults {
		if r.Err != nil {
			log.Errorf("%v; %s", r.Err, r.Output)
			continue
		}
		if r.Report != nil {
			issues := &ImageIssues{
				Misconfigurations: &ImageIssueMisconfigurations{},
				Vulnerabilities:   &ImageIssueVulnerabilities{},
				Secrets:           &ImageIssueSecrets{},
				Total:             &ImageIssueSeverity{},
			}

			for _, vulnResults := range r.Report.Results {
				for _, misconfiguration := range vulnResults.Misconfigurations {
					issues.Misconfigurations.AddSeverity(misconfiguration.Severity)
					issues.Total.AddSeverity(misconfiguration.Severity)
				}
				for _, vulnerability := range vulnResults.Vulnerabilities {
					issues.Vulnerabilities.AddSeverity(vulnerability.Severity)
					nvdScore := float64(0)
					if nvd, ok := vulnerability.CVSS["nvd"]; ok {
						nvdScore = nvd.V3Score
					}
					issues.Vulnerabilities.Vulnerabilities = append(issues.Vulnerabilities.Vulnerabilities, &ImageIssueVulnerability{
						VulnerabilityID: vulnerability.VulnerabilityID,
						Severity:        vulnerability.Severity,
						PkgID:           vulnerability.PkgID,
						PrimaryURL:      vulnerability.PrimaryURL,
						Title:           vulnerability.Title,
						Description:     vulnerability.Description,
						NvdV3Score:      nvdScore,
						PublishedDate:   vulnerability.PublishedDate,
					})
					issues.Total.AddSeverity(vulnerability.Severity)
				}
				for _, secret := range vulnResults.Secrets {
					issues.Secrets.AddSeverity(secret.Severity)
					issues.Total.AddSeverity(secret.Severity)
				}
			}

			created := r.Report.Metadata.ImageConfig.Created.Time
			ageDays := 0
			// Treat an absent or placeholder created date as unknown (0) rather
			// than as a very old image.
			//
			// Two cases produce a bogus age. Go's zero time (year 1) means the
			// field was missing. The Unix epoch means the build deliberately
			// pinned it: reproducible builds commonly set created to
			// 1970-01-01 so the image digest is stable, and
			// registry.k8s.io/external-dns does exactly this -- it reported an
			// age of 20705 days. Left unguarded such an image sorts to the top
			// of any age ranking and colours red permanently, which is worse
			// than admitting the age is unknown.
			if !created.IsZero() && created.Year() > 2000 {
				ageDays = int(time.Since(created).Hours() / 24)
			}

			trivyReport[r.Image] = &TrivyImageReport{
				ImageCreated: created,
				AgeDays:      ageDays,
				ImageIssues:  issues,
			}
		}
	}

	return trivyReport
}
