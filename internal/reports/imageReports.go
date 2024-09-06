package reports

import (
	"encoding/json"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

type imageReportList struct {
	reportGenerated time.Time
	reportSets      map[ReportType]imageReportSet
}

type imageReportSet map[string]*imageReport

type imageReport struct {
	ReportGenerated time.Time   `json:"reportGenerated"`
	ReportType      ReportType  `json:"reportType"`
	Image           string      `json:"image"`
	Contents        interface{} `json:"report"`
}

func NewImageReportList(reportGenerated time.Time) *imageReportList {
	return &imageReportList{
		reportGenerated: reportGenerated,
		reportSets:      make(map[ReportType]imageReportSet),
	}
}

func (rl *imageReportList) Output() {
	for _, i := range rl.reportSets {
		for _, report := range i {
			report.output()
		}
	}
}

func (rl *imageReportList) AddImageReport(reportType ReportType, image string, reportContents interface{}) {
	if _, ok := rl.reportSets[reportType]; !ok {
		rl.reportSets[reportType] = make(imageReportSet)
	}
	rl.reportSets[reportType][image] = &imageReport{
		ReportGenerated: rl.reportGenerated,
		ReportType:      reportType,
		Image:           image,
		Contents:        reportContents,
	}
}

func (r *imageReport) output() {
	reportOut, err := json.Marshal(r)
	if err != nil {
		log.Errorf("error converting '%s' report to JSON; err: %v, out: %v", r.ReportType, err, r)
	} else {
		fmt.Println(string(reportOut))
	}
}
