package config

import (
	"github.com/spf13/cobra"
)

func SetSourceFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringP("source-id", "", "", "Identifier for this instance of the tool.  Used as unique identifier of sources")
	cmd.PersistentFlags().StringP("source", "", "", "Source of images.  Can be one of [kubernetes, file]")
	cmd.PersistentFlags().StringSliceP("source-kubernetes-include-namespaces", "", []string{}, "Comma-separated list of Kubernetes namespaces to scan if --source-kubernetes")
	cmd.PersistentFlags().StringSliceP("source-kubernetes-exclude-namespaces", "", []string{}, "Comma-separated list of Kubernetes namespaces to exclude if --source-kubernetes")
	cmd.PersistentFlags().StringP("source-file-path", "", "", "Path of file containing list of images to scan")
}

func SetRunFlags(cmd *cobra.Command) {
	// Importing
	cmd.PersistentFlags().StringP("import-from", "", "", "Location (directory) of sources to import.  Should take the format <protocol>://<destination>. <protocol> can be one of 'gs' (Google Cloud Storage), or 'file'.")

	// Scanning
	cmd.PersistentFlags().StringSliceP("skip-registry", "", []string{}, "List of registries to skip when scanning")
	cmd.PersistentFlags().BoolP("latest-semver-scan", "", true, "Scan image registry for latest semver tag")
	cmd.PersistentFlags().BoolP("security-scan", "", true, "Whether to run security scan against images")

	// Reporting
	cmd.PersistentFlags().StringSliceP("reports", "", []string{"InventoryReport"}, "List of reports to output.  Can be one or more of [inventory, summary, summaryImageCombined, summaryRegistry, imageSummary, imageRegistry, imageVulnerabilities, imageKubernetes]")
	cmd.PersistentFlags().StringP("report-location", "", "", "Location to write reports.  Should take the format <protocol>://<destination>. <protocol> can be one of 'file' (directory), or 'stdout'.")
	cmd.PersistentFlags().StringP("report-format", "", "json", "The desired output format.  One of json, yaml")
}
