package reports

import (
	"github.com/bartlettc22/image-inquisitor/pkg/api/metadata"
	"github.com/bartlettc22/image-inquisitor/pkg/api/v1alpha1"
)

type ReportKind string

func (rk ReportKind) String() string {
	return string(rk)
}

type ReportSpec struct {
	Report any `json:"report" yaml:"report"`
}

// NewReportManifest creates a new report manifest
func NewReportManifest(reportKind ReportKind, uuid string, report any) *metadata.Manifest {
	return metadata.NewManifest(v1alpha1.APIVersion, reportKind.String(), uuid, &ReportSpec{
		Report: report,
	})
}
