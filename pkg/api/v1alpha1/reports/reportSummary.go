package reports

const (
	// ReportsKind is the kind of a report manifest
	ReportSummaryKind ReportKind = "SummaryReport"
)

// ReportSummary is a summary report
type ReportSummary struct {

	// ImageCount is the number of
	RepoCount             int `json:"repoCount,omitempty" yaml:"repoCount,omitempty"`
	DigestCount           int `json:"digestCount,omitempty" yaml:"digestCount,omitempty"`
	KubernetesSourceCount int `json:"kubernetesSourceCount,omitempty" yaml:"kubernetesSourceCount,omitempty"`
	FileSourceCount       int `json:"fileSourceCount,omitempty" yaml:"fileSourceCount,omitempty"`
	IssuesCriticalCount   int `json:"issuesCriticalCount,omitempty" yaml:"issuesCriticalCount,omitempty"`
	IssuesHighCount       int `json:"issuesHighCount,omitempty" yaml:"issuesHighCount,omitempty"`
	IssuesMediumCount     int `json:"issuesMediumCount,omitempty" yaml:"issuesMediumCount,omitempty"`
	IssuesLowCount        int `json:"issuesLowCount,omitempty" yaml:"issuesLowCount,omitempty"`
	IssuesUnknownCount    int `json:"issuesUnknownCount,omitempty" yaml:"issuesUnknownCount,omitempty"`
}
