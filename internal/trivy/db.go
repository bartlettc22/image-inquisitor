package trivy

import (
	"fmt"
	"os/exec"
)

// RefreshTrivyDB refreshes the Trivy database cache on disk
func RefreshTrivyDB() error {
	svcLog.Info("running Trivy database refresh")

	// TODO: Take out a global lock so scans can't run while this happening

	cmd := exec.Command(
		"/usr/local/bin/trivy",
		"image",
		"--quiet",
		"--format", "json",
		"--download-db-only",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run Trivy database download: %w; %s", err, string(output))
	}

	cmd = exec.Command(
		"/usr/local/bin/trivy",
		"image",
		"--quiet",
		"--format", "json",
		"--download-java-db-only",
	)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run Trivy database download: %w; %s", err, string(output))
	}

	svcLog.Info("completed Trivy database refresh")

	return nil
}
