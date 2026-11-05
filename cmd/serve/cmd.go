package servecmd

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/api"
	"github.com/bartlettc22/image-inquisitor/internal/daemon"
	"github.com/bartlettc22/image-inquisitor/internal/github"
	"github.com/bartlettc22/image-inquisitor/internal/store"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func Cmd() *cobra.Command {
	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the daemon and its API server",
		Long:  "Run the daemon, which periodically syncs dependency update PRs, and the API server the UI reads from",
		Run:   run,
	}

	serveCmd.Flags().StringP("config", "c", "config.yaml", "Path to the daemon config file")
	serveCmd.Flags().StringP("data-dir", "", "", "Overrides dataDir from the config file")

	return serveCmd
}

func run(cmd *cobra.Command, args []string) {
	configPath, _ := cmd.Flags().GetString("config")

	c, err := daemon.LoadConfig(configPath)
	if err != nil {
		log.WithError(err).Fatal("fatal serve error")
	}

	if dataDir, _ := cmd.Flags().GetString("data-dir"); dataDir != "" {
		c.DataDir = dataDir
	}

	if err := os.MkdirAll(c.DataDir, 0o755); err != nil {
		log.WithError(err).Fatal("fatal serve error")
	}
	s, err := store.Open(filepath.Join(c.DataDir, "imginq.db"))
	if err != nil {
		log.WithError(err).Fatal("fatal serve error")
	}
	defer s.Close()

	if c.GitHub.Token == "" {
		log.Warnf("no GitHub token set (github.token or %s); only public repos can be read", daemon.GitHubTokenEnv)
	}

	d, err := daemon.New(c, github.NewClient(c.GitHub.APIURL, c.GitHub.Token), s)
	if err != nil {
		log.WithError(err).Fatal("fatal serve error")
	}

	apiConfig := &api.Config{
		SyncInterval: c.SyncEvery().String(),
		GitHubAPIURL: c.GitHub.APIURL,
		GitHubToken:  c.GitHub.Token != "",
		Scanning:     d.Scanning(),
		TrivyVersion: d.TrivyVersion(),
		Clusters:     make([]api.ClusterConfig, 0, len(c.Kubernetes.Clusters)),
	}
	if r := c.Renovate; r != nil {
		apiConfig.Renovate = &api.RenovateConfig{Image: r.Image, Cluster: r.Cluster, Namespace: r.Namespace, Soak: r.Soak}
		if r.RunEvery() > 0 {
			apiConfig.Renovate.Interval = r.RunEvery().String()
		}
	}
	for _, cl := range c.Kubernetes.Clusters {
		apiConfig.Clusters = append(apiConfig.Clusters, api.ClusterConfig{
			Name:              cl.Name,
			Context:           cl.Context,
			InCluster:         cl.InCluster,
			IncludeNamespaces: cl.IncludeNamespaces,
			ExcludeNamespaces: cl.ExcludeNamespaces,
		})
	}

	server := &http.Server{
		Addr:              c.Listen,
		Handler:           api.NewHandler(s, d, apiConfig),
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go d.Run(ctx)

	go func() {
		log.WithField("listen", c.Listen).Info("API server listening")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.WithError(err).Fatal("fatal serve error")
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.WithError(err).Error("error shutting down API server")
	}
}
