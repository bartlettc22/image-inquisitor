package main

import (
	"fmt"

	servecmd "github.com/bartlettc22/image-inquisitor/cmd/serve"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// version is set at build time from the release tag
var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "imginq",
	Version: version,
	Short:   "image-inquisitor shows which container image updates matter",
	Long:    "image-inquisitor tracks Renovate PRs and running images, and shows the CVEs each update fixes or introduces",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initLogging(cmd)
	},
}

func main() {
	rootCmd.PersistentFlags().String("log-level", "info", "The desired logging level.  One of error, warn, info, debug.")
	rootCmd.PersistentFlags().String("log-format", "logfmt", "The desired logging format.  One of logfmt, json.")

	rootCmd.AddCommand(servecmd.Cmd())

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func initLogging(cmd *cobra.Command) error {
	logLevelStr, _ := cmd.Flags().GetString("log-level")
	logLevel, err := log.ParseLevel(logLevelStr)
	if err != nil {
		return err
	}
	log.SetLevel(logLevel)

	switch logFormat, _ := cmd.Flags().GetString("log-format"); logFormat {
	case "logfmt":
		log.SetFormatter(&log.TextFormatter{ForceColors: true})
	case "json":
		log.SetFormatter(&log.JSONFormatter{})
	default:
		return fmt.Errorf("invalid --log-format: %s", logFormat)
	}
	return nil
}
