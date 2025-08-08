package trivy

import (
	"fmt"
	"os/exec"
)

// RefreshTrivyDB refreshes the Trivy database cache on disk
func RefreshTrivyDB() error {
	svcLog.Info("running Trivy database refresh")

	cmd := exec.Command(
		"/usr/local/bin/trivy",
		"image",
		"--quiet",
		"--format", "json",
		"--download-db-only",
		"--download-java-db-only",
	)

	// Execute the command and capture its output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run Trivy database download: %w; %s", err, string(output))
	}

	svcLog.Info("completed Trivy database refresh")

	return nil
}
