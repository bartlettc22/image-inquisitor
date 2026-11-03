package daemon

import (
	"fmt"
	"os"
	"time"

	yaml "github.com/goccy/go-yaml"
)

const (
	defaultListen       = ":8080"
	defaultDataDir      = "/var/lib/imginq"
	defaultSyncInterval = 10 * time.Minute
	defaultBranchPrefix = "renovate/"
	defaultHelmBinary   = "helm"
	defaultTrivyBinary  = "trivy"

	// GitHubTokenEnv is read when the config file doesn't set github.token
	GitHubTokenEnv = "GITHUB_TOKEN"
)

type Config struct {
	// Listen is the address the API server listens on
	Listen string `yaml:"listen"`

	// DataDir holds the SQLite database
	DataDir string `yaml:"dataDir"`

	// SyncInterval is how often repos are synced, e.g. "10m"
	SyncInterval string `yaml:"syncInterval"`

	GitHub GitHubConfig `yaml:"github"`

	Repos []RepoConfig `yaml:"repos"`

	Kubernetes KubernetesConfig `yaml:"kubernetes"`

	Helm HelmConfig `yaml:"helm"`

	Scan ScanConfig `yaml:"scan"`

	// Renovate is unset when the daemon doesn't run Renovate
	Renovate *RenovateConfig `yaml:"renovate"`

	syncInterval time.Duration
}

type KubernetesConfig struct {
	// Kubeconfig is the kubeconfig file; defaults to $KUBECONFIG or ~/.kube/config
	Kubeconfig string `yaml:"kubeconfig"`

	Clusters []ClusterConfig `yaml:"clusters"`
}

type ClusterConfig struct {
	// Name identifies the cluster in the inventory; defaults to Context, or "in-cluster"
	Name string `yaml:"name"`

	// Context is the kubeconfig context to use
	Context string `yaml:"context"`

	// InCluster uses the pod's service account instead of a kubeconfig context
	InCluster bool `yaml:"inCluster"`

	IncludeNamespaces []string `yaml:"includeNamespaces"`
	ExcludeNamespaces []string `yaml:"excludeNamespaces"`
}

type HelmConfig struct {
	// Binary is the helm executable used to render Argo CD apps' charts
	Binary string `yaml:"binary"`
}

type ScanConfig struct {
	// Disabled turns off scanning PR images for vulnerabilities
	Disabled bool `yaml:"disabled"`

	// Trivy is the trivy executable
	Trivy string `yaml:"trivy"`
}

type GitHubConfig struct {
	APIURL string `yaml:"apiURL"`
	Token  string `yaml:"token"`
}

type RepoConfig struct {
	// Name is the repository as "owner/name", or a pattern such as "owner/*" matching the
	// owner's active (not archived or forked) repositories
	Name string `yaml:"name"`

	// Exclude are names or patterns a pattern's matches are filtered by
	Exclude []string `yaml:"exclude"`

	// BranchPrefix identifies dependency update PRs by their head branch
	BranchPrefix string `yaml:"branchPrefix"`
}

// SyncEvery is the parsed SyncInterval, or the default when unset
func (c *Config) SyncEvery() time.Duration {
	return c.syncInterval
}

// LoadConfig reads a config file, applies defaults and validates it
func LoadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}

	c := &Config{}
	if err := yaml.Unmarshal(b, c); err != nil {
		return nil, fmt.Errorf("error parsing config %s: %w", path, err)
	}

	if c.Listen == "" {
		c.Listen = defaultListen
	}
	if c.DataDir == "" {
		c.DataDir = defaultDataDir
	}

	c.syncInterval = defaultSyncInterval
	if c.SyncInterval != "" {
		c.syncInterval, err = time.ParseDuration(c.SyncInterval)
		if err != nil {
			return nil, fmt.Errorf("invalid syncInterval %q: %w", c.SyncInterval, err)
		}
	}

	if c.Helm.Binary == "" {
		c.Helm.Binary = defaultHelmBinary
	}
	if c.Scan.Trivy == "" {
		c.Scan.Trivy = defaultTrivyBinary
	}

	if c.GitHub.Token == "" {
		c.GitHub.Token = os.Getenv(GitHubTokenEnv)
	}

	if len(c.Repos) == 0 && len(c.Kubernetes.Clusters) == 0 {
		return nil, fmt.Errorf("no repos or kubernetes clusters configured")
	}
	for i := range c.Repos {
		if err := validateRepo(c.Repos[i]); err != nil {
			return nil, err
		}
		if c.Repos[i].BranchPrefix == "" {
			c.Repos[i].BranchPrefix = defaultBranchPrefix
		}
	}

	names := map[string]bool{}
	for i := range c.Kubernetes.Clusters {
		cl := &c.Kubernetes.Clusters[i]
		switch {
		case cl.InCluster && cl.Context != "":
			return nil, fmt.Errorf("kubernetes cluster %d: set context or inCluster, not both", i)
		case !cl.InCluster && cl.Context == "":
			return nil, fmt.Errorf("kubernetes cluster %d: context or inCluster is required", i)
		}
		if cl.Name == "" {
			cl.Name = cl.Context
			if cl.InCluster {
				cl.Name = "in-cluster"
			}
		}
		if names[cl.Name] {
			return nil, fmt.Errorf("duplicate kubernetes cluster name %q", cl.Name)
		}
		names[cl.Name] = true
	}

	if c.Renovate != nil {
		if err := c.Renovate.validate(c); err != nil {
			return nil, fmt.Errorf("renovate: %w", err)
		}
	}

	return c, nil
}
