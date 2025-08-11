package reports

import "time"

const (
	// ReportRunKind is the kind of a report manifest
	ReportRunKind ReportKind = "RunReport"
)

// ReportRun is a report of a scanning run
type ReportRun struct {
	RunID           string    `json:"runID" yaml:"runID"`
	Started         time.Time `json:"started" yaml:"started"`
	Finished        time.Time `json:"finished" yaml:"finished"`
	DurationSeconds int       `json:"durationSeconds" yaml:"durationSeconds"`
}
