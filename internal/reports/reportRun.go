package reports

import (
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/inventory"
	"github.com/bartlettc22/image-inquisitor/pkg/api/metadata"
	reportsapi "github.com/bartlettc22/image-inquisitor/pkg/api/v1alpha1/reports"
	"github.com/google/uuid"
)

func GenerateRunReport(inventory inventory.Inventory, runID uuid.UUID, started time.Time, finished time.Time) map[string]*metadata.Manifest {

	report := &reportsapi.ReportRun{
		RunID:           runID.String(),
		Started:         started,
		Finished:        finished,
		DurationSeconds: int(finished.Sub(started).Seconds()),
	}

	return map[string]*metadata.Manifest{
		"": reportsapi.NewReportManifest(reportsapi.ReportRunKind, runID.String(), report),
	}
}
