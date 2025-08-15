package reports

const (
	// ReportsKind is the kind of a report manifest
	ReportSummaryKind ReportKind = "SummaryReport"
)

// ReportSummary is a summary report
type ReportSummary struct {

	// ImageCount is the number of
	RepoCount             int `json:"repo_count" yaml:"repoCount"`
	DigestCount           int `json:"digest_count" yaml:"digestCount"`
	KubernetesSourceCount int `json:"kubernetes_source_count" yaml:"kubernetesSourceCount"`
	FileSourceCount       int `json:"file_source_count" yaml:"fileSourceCount"`
	TotalIssuesCount      int `json:"total_issues_count" yaml:"totalIssuesCount"`
	IssuesCriticalCount   int `json:"issues_critical_count" yaml:"issuesCriticalCount"`
	IssuesHighCount       int `json:"issues_high_count" yaml:"issuesHighCount"`
	IssuesMediumCount     int `json:"issues_medium_count" yaml:"issuesMediumCount"`
	IssuesLowCount        int `json:"issues_low_count" yaml:"issuesLowCount"`
	IssuesUnknownCount    int `json:"issues_unknown_count" yaml:"issuesUnknownCount"`
}
