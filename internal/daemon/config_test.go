package daemon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadConfig(t *testing.T, yaml string) (*Config, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	return LoadConfig(path)
}

func TestLoadConfigClusters(t *testing.T) {
	tests := []struct {
		name     string
		clusters string
		wantName string
		wantErr  string
	}{
		{name: "context", clusters: "- context: home", wantName: "home"},
		{name: "named context", clusters: "- {context: admin@home, name: home}", wantName: "home"},
		{name: "in cluster", clusters: "- inCluster: true", wantName: "in-cluster"},
		{name: "named in cluster", clusters: "- {inCluster: true, name: home}", wantName: "home"},
		{name: "neither", clusters: "- name: home", wantErr: "context or inCluster is required"},
		{name: "both", clusters: "- {context: home, inCluster: true}", wantErr: "not both"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := loadConfig(t, "kubernetes:\n  clusters:\n  "+strings.ReplaceAll(tt.clusters, "\n", "\n  ")+"\n")
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("got error %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got := c.Kubernetes.Clusters[0].Name; got != tt.wantName {
				t.Errorf("got name %q, want %q", got, tt.wantName)
			}
		})
	}
}
