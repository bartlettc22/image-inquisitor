package reports

import (
	"github.com/bartlettc22/image-inquisitor/internal/inventory"
	"github.com/bartlettc22/image-inquisitor/pkg/api/metadata"
	reportsapi "github.com/bartlettc22/image-inquisitor/pkg/api/v1alpha1/reports"
	"github.com/google/uuid"
)

func GenerateInventoryReport(inventory inventory.Inventory, runID uuid.UUID) map[string]*metadata.Manifest {
	return map[string]*metadata.Manifest{
		"": reportsapi.NewReportManifest(reportsapi.ReportInventoryKind, runID.String(), inventory),
	}
}
