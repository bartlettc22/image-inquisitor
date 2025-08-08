package trivy

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	trivyTypes "github.com/aquasecurity/trivy/pkg/types"
	log "github.com/sirupsen/logrus"
)

// Scan runs Trivy on an image and returns the results
func Scan(ref string, outputdir string) (*ImageIssues, error) {
	log.WithField("ref", ref).Debug("running Trivy scan")

	// Prepare the output file name
	outputFile := fmt.Sprintf("%s/%s.json", outputdir, strings.ReplaceAll(ref, "/", "_"))

	// Create the command to run Trivy
	cmd := exec.Command(
		"/usr/local/bin/trivy",
		"image",
		"--quiet",
		"--format", "json",
		"--skip-db-update",
		"--skip-java-db-update",
		"--output", outputFile,
		ref,
	)

	// Execute the command and capture its output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run Trivy for image %s: %w; %s", ref, err, string(output))
	}

	scanContent, err := os.ReadFile(outputFile)
	if err != nil {
		return nil, fmt.Errorf("failed reading Trivy output file (%s): %v", outputFile, err)
	}

	report := &trivyTypes.Report{}
	err = json.Unmarshal(scanContent, report)
	if err != nil {
		return nil, fmt.Errorf("failed parsing Trivy output file: %v", err)
	}
	issues := parseReport(report)

	log.WithField("ref", ref).Debug("completed Trivy scan")

	return issues, nil
}
