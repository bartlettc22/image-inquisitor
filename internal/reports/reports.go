package reports

import "time"

type ReportWrapper struct {
	ReportGenerated time.Time   `json:"report_generated"`
	ReportType      string      `json:"report_type"`
	Image           string      `json:"image,omitempty"`
	Report          interface{} `json:"report"`
}
