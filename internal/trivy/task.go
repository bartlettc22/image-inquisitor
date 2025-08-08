package trivy

import (
	"fmt"

	"github.com/bartlettc22/image-inquisitor/internal/worker"
)

type TrivyScanImageTask struct {
	OutputDir string
	Ref       string
	result    *TrivyScanImageResult
	errors    []error
}

type TrivyScanImageResult struct {
	Ref    string
	Issues *ImageIssues
}

func (t *TrivyScanImageTask) Run() worker.Result {
	issues, err := Scan(t.Ref, t.OutputDir)
	if err != nil {
		t.errors = append(t.errors, fmt.Errorf("failed to run Trivy for image %s: %w", t.Ref, err))
		return t
	}

	t.result = &TrivyScanImageResult{
		Ref:    t.Ref,
		Issues: issues,
	}

	return t
}

func (t *TrivyScanImageTask) Result() interface{} {
	return t.result
}

func (t *TrivyScanImageTask) Errors() []error {
	return t.errors
}
