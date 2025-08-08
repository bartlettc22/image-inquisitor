package reports

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/bartlettc22/image-inquisitor/internal/inventory"
	"github.com/bartlettc22/image-inquisitor/pkg/api/metadata"
	reportsapi "github.com/bartlettc22/image-inquisitor/pkg/api/v1alpha1/reports"
	yaml "github.com/goccy/go-yaml"
	log "github.com/sirupsen/logrus"
)

type ReportType string
type ReportFormat string
type ReportDestination string

var svcLog = log.WithField("service", "reports")

const (
	ReportFormatJSON ReportFormat = "json"
	ReportFormatYAML ReportFormat = "yaml"

	ReportDestinationStdout ReportDestination = "stdout"
	ReportDestinationFile   ReportDestination = "file"

	defaultReportFormat = ReportFormatJSON
)

func (rt ReportType) String() string {
	return string(rt)
}

func (rd ReportDestination) String() string {
	return string(rd)
}

func (rf ReportFormat) String() string {
	return string(rf)
}

func ParseReportType(reportType string) (reportsapi.ReportKind, error) {
	switch reportType {
	case reportsapi.ReportInventoryKind.String():
		return reportsapi.ReportInventoryKind, nil
	case reportsapi.ReportSummaryKind.String():
		return reportsapi.ReportSummaryKind, nil
	default:
		return "", fmt.Errorf("invalid report type: %s", reportType)
	}
}

func ParseReportFormat(reportFormat string) (ReportFormat, error) {
	switch reportFormat {
	case ReportFormatJSON.String():
		return ReportFormatJSON, nil
	case ReportFormatYAML.String():
		return ReportFormatYAML, nil
	default:
		return "", fmt.Errorf("invalid report format: %s", reportFormat)
	}
}

func ParseReportDestination(reportDestination string) (ReportDestination, error) {
	switch reportDestination {
	case ReportDestinationStdout.String():
		return ReportDestinationStdout, nil
	case ReportDestinationFile.String():
		return ReportDestinationFile, nil
	default:
		return "", fmt.Errorf("invalid report destination: %s", reportDestination)
	}
}

type Report struct {
	ReportGenerated string `yaml:"reportGenerated" json:"reportGenerated"`
	ReportType      string `yaml:"reportType" json:"reportType"`
	Report          any    `yaml:"report" json:"report"`
}

type ReportGeneratorConfig struct {
	ReportTypes        []string
	ReportDestinations []string
	ReportFormat       string
	ReportFileDir      string
}

type ReportGenerator struct {
	reportTypes        []reportsapi.ReportKind
	reportFormat       ReportFormat
	reportDestinations []ReportDestination
	reportFileDir      string
}

func NewReportGenerator(c ReportGeneratorConfig) (*ReportGenerator, error) {

	var err error

	reportTypes := make([]reportsapi.ReportKind, 0)
	for _, reportTypeStr := range c.ReportTypes {
		reportType, err := ParseReportType(reportTypeStr)
		if err != nil {
			return nil, err
		}
		reportTypes = append(reportTypes, reportType)
	}

	reportFormat := defaultReportFormat
	if c.ReportFormat != "" {
		reportFormat, err = ParseReportFormat(c.ReportFormat)
		if err != nil {
			return nil, err
		}
	}

	reportDestinations := make([]ReportDestination, 0)
	for _, reportDestinationStr := range c.ReportDestinations {
		reportDestination, err := ParseReportDestination(reportDestinationStr)
		if err != nil {
			return nil, err
		}
		reportDestinations = append(reportDestinations, reportDestination)
	}

	reportFileDir := ""
	if slices.Contains(c.ReportDestinations, ReportDestinationFile.String()) {
		if c.ReportFileDir == "" {
			return nil, fmt.Errorf("reportFileDir must be set when reportDestination is file")
		}
		reportFileDir = c.ReportFileDir
	}

	err = os.MkdirAll(reportFileDir, 0755)
	if err != nil {
		return nil, err
	}

	return &ReportGenerator{
		reportTypes:        reportTypes,
		reportFormat:       reportFormat,
		reportDestinations: reportDestinations,
		reportFileDir:      reportFileDir,
	}, nil
}

func (rg *ReportGenerator) Generate(inventory inventory.Inventory) error {
	for _, reportType := range rg.reportTypes {
		var report *metadata.Manifest
		var err error

		svcLog.Infof("generating report: %s", reportType)
		switch reportType {
		case reportsapi.ReportInventoryKind:
			report = GenerateInventoryReport(inventory)
		case reportsapi.ReportSummaryKind:
			report = GenerateSummaryReport(inventory)
		default:
			return fmt.Errorf("invalid report type: %s", reportType)
		}

		var reportBytes []byte
		var reportFileName string
		switch rg.reportFormat {
		case ReportFormatJSON:
			reportBytes, err = json.MarshalIndent(report, "", "  ")
			if err != nil {
				return err
			}
			reportFileName = fmt.Sprintf("%s.json", reportType)
		case ReportFormatYAML:
			reportBytes, err = yaml.Marshal(report)
			if err != nil {
				return err
			}
			reportFileName = fmt.Sprintf("%s.yaml", reportType)
		default:
			return fmt.Errorf("invalid report format: %s", rg.reportFormat)
		}

		for _, reportDestination := range rg.reportDestinations {
			svcLog.Infof("outputting report to: %s", reportDestination)
			switch reportDestination {
			case ReportDestinationStdout:
				fmt.Println(string(reportBytes))
			case ReportDestinationFile:
				err = os.WriteFile(filepath.Join(rg.reportFileDir, reportFileName), reportBytes, 0644)
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("invalid report destination: %s", reportDestination)
			}
		}
	}

	return nil
}
