package config

import (
	"github.com/bartlettc22/image-inquisitor/internal/reports"
	"github.com/spf13/viper"
)

func NewReportGeneratorFromConfig() (*reports.ReportGenerator, error) {
	return reports.NewReportGenerator(reports.ReportGeneratorConfig{
		ReportTypes:        viper.GetStringSlice("reports"),
		ReportDestinations: viper.GetStringSlice("report-destinations"),
		ReportFormat:       viper.GetString("report-format"),
		ReportFileDir:      viper.GetString("report-file-dir"),
	})
}
