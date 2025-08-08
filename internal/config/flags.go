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
	cmd.PersistentFlags().BoolP("latest-semver-scan", "", true, "Scan image registry for latest semver tag")
	cmd.PersistentFlags().BoolP("security-scan", "", true, "Whether to run security scan against images")
	cmd.PersistentFlags().StringSliceP("reports", "", []string{"inventory"}, "List of reports to output.  Can be one or more of [inventory, summary, summaryImageCombined, summaryRegistry, imageSummary, imageRegistry, imageVulnerabilities, imageKubernetes]")
	cmd.PersistentFlags().StringSliceP("report-destinations", "", []string{"stdout"}, "Comma-separated list of output destinations.  Can be one or more of [stdout, file]. If 'file', must specify 'report-file-dir' parameter")
	cmd.PersistentFlags().StringP("report-format", "", "json", "The desired output format.  One of json, yaml")
	cmd.PersistentFlags().StringP("report-file-dir", "", "", "Path of directory to output the reports. Must be specified if 'report-destinations' contains 'file'")
}
