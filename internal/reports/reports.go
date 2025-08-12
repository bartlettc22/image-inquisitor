package reports

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/inventory"
	"github.com/bartlettc22/image-inquisitor/pkg/api/metadata"
	reportsapi "github.com/bartlettc22/image-inquisitor/pkg/api/v1alpha1/reports"
	yaml "github.com/goccy/go-yaml"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type ReportType string
type ReportFormat string

var svcLog = log.WithField("service", "reports")

const (
	ReportFormatJSON           ReportFormat = "json"
	ReportFormatYAML           ReportFormat = "yaml"
	ReportFormatSimplifiedJSON ReportFormat = "simplified-json"

	defaultReportFormat = ReportFormatJSON
)

func (rt ReportType) String() string {
	return string(rt)
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
	case reportsapi.ReportImageSummaryKind.String():
		return reportsapi.ReportImageSummaryKind, nil
	case reportsapi.ReportRunKind.String():
		return reportsapi.ReportRunKind, nil
	case reportsapi.ReportImageKind.String():
		return reportsapi.ReportImageKind, nil
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
	case ReportFormatSimplifiedJSON.String():
		return ReportFormatSimplifiedJSON, nil
	default:
		return "", fmt.Errorf("invalid report format: %s", reportFormat)
	}
}

type Report struct {
	ReportGenerated string `yaml:"reportGenerated" json:"reportGenerated"`
	ReportType      string `yaml:"reportType" json:"reportType"`
	Report          any    `yaml:"report" json:"report"`
}

type ReportGeneratorConfig struct {
	ReportTypes  []string
	ReportWriter ReportWriter
	ReportFormat string
}

type ReportGenerator struct {
	runID        uuid.UUID
	started      time.Time
	finished     time.Time
	reportTypes  []reportsapi.ReportKind
	reportFormat ReportFormat
	reportWriter ReportWriter
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

	return &ReportGenerator{
		reportTypes:  reportTypes,
		reportFormat: reportFormat,
		reportWriter: c.ReportWriter,
	}, nil
}

func (rg *ReportGenerator) SetRunStats(runID uuid.UUID, started time.Time, finished time.Time) {
	rg.runID = runID
	rg.started = started
	rg.finished = finished
}

func (rg *ReportGenerator) Generate(inventory inventory.Inventory) error {
	for _, reportType := range rg.reportTypes {
		var reports map[string]*metadata.Manifest
		var err error

		svcLog.Infof("generating report: %s", reportType)
		switch reportType {
		case reportsapi.ReportInventoryKind:
			reports = GenerateInventoryReport(inventory, rg.runID)
		case reportsapi.ReportSummaryKind:
			reports = GenerateSummaryReport(inventory, rg.runID)
		case reportsapi.ReportImageSummaryKind:
			reports = GenerateImageSummaryReport(inventory, rg.runID)
		case reportsapi.ReportRunKind:
			reports = GenerateRunReport(inventory, rg.runID, rg.started, rg.finished)
		case reportsapi.ReportImageKind:
			reports = GenerateImageReports(inventory, rg.runID)
		default:
			return fmt.Errorf("invalid report type: %s", reportType)
		}

		for ref, report := range reports {
			reportFileName := fmt.Sprintf("%s.json", reportType)
			if ref != "" {
				reportFileName = fmt.Sprintf("%s.%s.json", ref, reportType)
			}
			var reportBytes []byte
			switch rg.reportFormat {
			case ReportFormatJSON:
				reportBytes, err = json.Marshal(report)
				if err != nil {
					return err
				}
			case ReportFormatYAML:
				reportBytes, err = yaml.Marshal(report)
				if err != nil {
					return err
				}
				reportFileName = fmt.Sprintf("%s.yaml", reportType)
			case ReportFormatSimplifiedJSON:
				reportSimplified := SimplifiedManifest(report)
				reportBytes, err = json.Marshal(reportSimplified)
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("invalid report format: %s", rg.reportFormat)
			}

			svcLog.Infof("outputting report '%s' to: %s", reportFileName, rg.reportWriter.Location())
			err = rg.reportWriter.WriteReport(reportFileName, reportBytes)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func SimplifiedManifest(manifest *metadata.Manifest) any {
	return &SimplifiedReport{
		Metadata: &SimplifiedReportMetadata{
			Version: manifest.Version,
			Kind:    manifest.Kind,
			Created: manifest.Metadata.Created,
			UUID:    manifest.Metadata.UUID,
		},
		Report: manifest.Spec,
	}
}

type SimplifiedReport struct {
	Metadata *SimplifiedReportMetadata `json:"metadata" yaml:"metadata"`
	Report   any                       `json:"report" yaml:"report"`
}

type SimplifiedReportMetadata struct {
	Version string    `json:"version" yaml:"version"`
	Kind    string    `json:"kind" yaml:"kind"`
	Created time.Time `json:"created" yaml:"created"`
	UUID    string    `json:"uuid,omitempty" yaml:"uuid,omitempty"`
}
