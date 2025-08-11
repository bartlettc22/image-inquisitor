package reports

import (
	"fmt"
	"os"
	"path/filepath"
)

type ReportWriter interface {
	WriteReport(name string, reportBytes []byte) error
	Location() string
}

// Ensure writers implements ReportWriter
var _ ReportWriter = &StdoutReportWriter{}
var _ ReportWriter = &FileReportWriter{}

type StdoutReportWriter struct{}

func NewStdoutReportWriter() *StdoutReportWriter {
	return &StdoutReportWriter{}
}

func (w *StdoutReportWriter) WriteReport(name string, reportBytes []byte) error {
	fmt.Println(string(reportBytes))
	return nil
}

func (w *StdoutReportWriter) Location() string {
	return "stdout"
}

type FileReportWriter struct {
	Path string
}

func NewFileReportWriter(path string) (*FileReportWriter, error) {
	return &FileReportWriter{
		Path: path,
	}, nil
}

func (w *FileReportWriter) WriteReport(name string, reportBytes []byte) error {
	reportFileName := fmt.Sprintf("%s.yaml", name)
	err := os.WriteFile(filepath.Join(w.Path, reportFileName), reportBytes, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (w *FileReportWriter) Location() string {
	return fmt.Sprintf("file://%s", w.Path)
}
