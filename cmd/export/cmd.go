package exportcmd

import (
	"context"
	"fmt"

	"github.com/bartlettc22/image-inquisitor/internal/config"
	sourcesapi "github.com/bartlettc22/image-inquisitor/pkg/api/v1alpha1/sources"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Cmd() *cobra.Command {

	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export image sources",
		Long:  "Export image sources",
		Run:   run,
	}

	config.SetSourceFlags(exportCmd)
	exportCmd.PersistentFlags().StringP("export-destination", "", "", "Destination (directory) of export. Should take the format <protocol>://<destination>. <protocol> can be one of 'gs' (Google Cloud Storage), or 'file'.")

	return exportCmd
}

func run(cmd *cobra.Command, args []string) {

	err := viper.BindPFlags(cmd.PersistentFlags())
	if err != nil {
		log.Fatal(err)
	}

	sourceID := viper.GetString("source-id")
	if sourceID == "" {

		log.WithField("error", "--source-id not specified").Fatal("fatal export error")
	}

	sourceGenerator, err := config.SourceGeneratorFromConfig()
	if err != nil {
		log.WithField("error", err).Fatal("fatal export error")
	}

	exporter, err := config.ExporterFromConfig()
	if err != nil {
		log.WithField("error", err).Fatal("fatal export error")
	}

	sourceList, err := sourceGenerator.Generate()
	if err != nil {
		log.WithField("error", err).Fatal("fatal export error")
	}

	manifest := sourcesapi.NewSourceListManifest(sourceList)

	name := fmt.Sprintf("%s.sources", sourceID)

	err = exporter.Export(context.Background(), name, manifest)
	if err != nil {
		log.WithField("error", err).Fatal("fatal export error")
	}
}
