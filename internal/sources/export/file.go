package export

import (
	"context"
	"fmt"
	"os"
	"path"

	exportapiv1alpha1 "github.com/bartlettc22/image-inquisitor/internal/sources/export/api/v1alpha1"
	"gopkg.in/yaml.v2"
)

func (e *Exporter) ExportFile(ctx context.Context, report *exportapiv1alpha1.ExportReport) error {

	out, err := yaml.Marshal(report)
	if err != nil {
		return fmt.Errorf("error marshalling export report: %v", err)
	}

	filePath := path.Join(e.FilePath, e.exportfileName())
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(out)
	if err != nil {
		return err
	}

	return nil
}
