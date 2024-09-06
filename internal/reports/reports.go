package reports

type ReportType string

const (
	ReportTypeSummary              ReportType = "summary"
	ReportTypeSummaryImageCombined ReportType = "summaryImageCombined"
	ReportTypeSummaryRegistry      ReportType = "summaryRegistry"
	ReportTypeImageSummary         ReportType = "imageSummary"
	ReportTypeImageRegistry        ReportType = "imageRegistry"
	ReportTypeImageVulnerabilities ReportType = "imageVulnerabilities"
	ReportTypeImageKubernetes      ReportType = "imageKubernetes"
)

func (rt ReportType) IsSummaryReportType() bool {
	switch rt {
	case ReportTypeSummary:
	case ReportTypeSummaryImageCombined:
	case ReportTypeSummaryRegistry:
	default:
		return false
	}

	return true
}

func (rt ReportType) IsImageReportType() bool {
	switch rt {
	case ReportTypeImageSummary:
	case ReportTypeImageRegistry:
	case ReportTypeImageVulnerabilities:
	case ReportTypeImageKubernetes:
	default:
		return false
	}

	return true
}

func (rt ReportType) String() string {
	return string(rt)
}

func IsValidReportType(reportType string) bool {
	switch reportType {
	case ReportTypeSummary.String():
	case ReportTypeSummaryImageCombined.String():
	case ReportTypeSummaryRegistry.String():
	case ReportTypeImageSummary.String():
	case ReportTypeImageRegistry.String():
	case ReportTypeImageVulnerabilities.String():
	case ReportTypeImageKubernetes.String():
	default:
		return false
	}

	return true
}
