package runcmd

import (
	"log"

	"github.com/bartlettc22/image-inquisitor/internal/config"
	"github.com/bartlettc22/image-inquisitor/internal/runner"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Cmd() *cobra.Command {
	singleRunCmd := &cobra.Command{
		Use:   "run",
		Short: "Run the tool in single run mode",
		Long:  "Run the tool in single run mode",
		Run: func(cmd *cobra.Command, args []string) {
			err := viper.BindPFlags(cmd.PersistentFlags())
			if err != nil {
				log.Fatal(err)
			}
			err = runner.Run()
			if err != nil {
				log.Fatalf("fatal run error: %v", err)
			}
		},
	}

	config.SetSourceFlags(singleRunCmd)
	config.SetRunFlags(singleRunCmd)

	return singleRunCmd
}
