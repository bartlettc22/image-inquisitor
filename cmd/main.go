package main

import (
	exportcmd "github.com/bartlettc22/image-inquisitor/cmd/export"
	runcmd "github.com/bartlettc22/image-inquisitor/cmd/run"
	"github.com/bartlettc22/image-inquisitor/internal/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "imginq",
	Short: "image-inquisitor is a tool for scanning and reporting on container images",
	Long:  "image-inquisitor is a tool for scanning and reporting on container images",
}

func main() {

	// General Flags
	rootCmd.PersistentFlags().StringP("log-level", "", "info", "The desired logging level.  One of error, warn, info, debug.")
	rootCmd.PersistentFlags().StringP("log-format", "", "logfmt", "The desired logging format.  One of logfmt, json.")

	err := viper.BindPFlags(rootCmd.PersistentFlags())
	if err != nil {
		log.Fatal(err)
	}

	rootCmd.AddCommand(exportcmd.Cmd())
	rootCmd.AddCommand(runcmd.Cmd())
	// rootCmd.AddCommand(servercmd.Cmd())

	cobra.OnInitialize(config.Init)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
