package servercmd

import (
	"log"

	"github.com/bartlettc22/image-inquisitor/internal/config"
	"github.com/bartlettc22/image-inquisitor/internal/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Cmd() *cobra.Command {
	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Run the tool in server mode",
		Long:  "Run the tool in server mode",
		Run: func(cmd *cobra.Command, args []string) {
			server := server.NewServer(&server.ServerConfig{
				ImageScanning: true,
				CacheDir:      "/tmp",
			})
			err := server.Start()
			if err != nil {
				log.Fatalf("fatal server error: %v", err)
			}
		},
	}

	config.SetRunFlags(serverCmd)
	config.SetSourceFlags(serverCmd)

	err := viper.BindPFlags(serverCmd.PersistentFlags())
	if err != nil {
		log.Fatal(err)
	}

	return serverCmd
}
