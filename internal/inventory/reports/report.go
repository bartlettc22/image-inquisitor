package reports

import "time"

type Report struct {
	ReportGenerated string `yaml:"reportGenerated" json:"reportGenerated"`
	ReportType      string `yaml:"reportType" json:"reportType"`
	Report          interface{}
}

func wrapReport(reportName string, report interface{}) *Report {
	generated := time.Now().UTC()
	return &Report{
		ReportGenerated: generated.Format(time.RFC3339),
		ReportType:      reportName,
		Report:          report,
	}
}
