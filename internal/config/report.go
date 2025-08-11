package config

import (
	"fmt"

	"github.com/bartlettc22/image-inquisitor/internal/reports"
	"github.com/spf13/viper"
)

func NewReportGeneratorFromConfig() (*reports.ReportGenerator, error) {

	reportLocation := viper.GetString("report-location")
	protocol, path, err := ParseDestination(reportLocation)
	if err != nil {
		return nil, err
	}

	var reportWriter reports.ReportWriter
	switch protocol {
	case DestinationFile:
		reportWriter, err = reports.NewFileReportWriter(path)
		if err != nil {
			return nil, err
		}
	case DestinationStdout:
		reportWriter = reports.NewStdoutReportWriter()
	default:
		return nil, fmt.Errorf("invalid report location: '%s'", reportLocation)
	}

	return reports.NewReportGenerator(reports.ReportGeneratorConfig{
		ReportTypes:  viper.GetStringSlice("reports"),
		ReportWriter: reportWriter,
		ReportFormat: viper.GetString("report-format"),
	})
}
