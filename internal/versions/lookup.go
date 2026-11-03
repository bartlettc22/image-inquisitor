package versions

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"sigs.k8s.io/yaml"
)

// Lister lists the tags of image repositories and the versions of Helm charts.
// It keeps each Helm repository index it downloads, so use one per lookup run.
type Lister struct {
	http *http.Client

	mu      sync.Mutex
	indexes map[string]*indexResult
}

type indexResult struct {
	once  sync.Once
	index *helmIndex
	err   error
}

type helmIndex struct {
	Entries map[string][]struct {
		Version string `json:"version"`
	} `json:"entries"`
}

func NewLister() *Lister {
	return &Lister{http: &http.Client{Timeout: 2 * time.Minute}, indexes: map[string]*indexResult{}}
}

// ImageTags lists the tags of an image repository, e.g. "index.docker.io/library/nginx"
func (l *Lister) ImageTags(ctx context.Context, repository string) ([]string, error) {
	repo, err := name.NewRepository(repository, name.WeakValidation)
	if err != nil {
		return nil, err
	}
	return remote.List(repo, remote.WithContext(ctx), remote.WithAuthFromKeychain(authn.DefaultKeychain))
}

// ChartVersions lists the versions of a chart in a Helm repository or OCI registry.
// repoURL is written as in an Argo CD source: https://..., oci://... or a scheme-less OCI
// registry path; chart is empty when an OCI repoURL already names the chart.
func (l *Lister) ChartVersions(ctx context.Context, repoURL, chart string) ([]string, error) {
	repoURL = strings.TrimSuffix(repoURL, "/")
	if strings.HasPrefix(repoURL, "oci://") || !strings.Contains(repoURL, "://") {
		ref := strings.TrimPrefix(repoURL, "oci://")
		if chart != "" {
			ref += "/" + chart
		}
		tags, err := l.ImageTags(ctx, ref)
		if err != nil {
			return nil, err
		}
		// OCI tags can't contain "+", so Helm pushes build metadata with "_"
		for i, t := range tags {
			tags[i] = strings.ReplaceAll(t, "_", "+")
		}
		return tags, nil
	}

	index, err := l.index(ctx, repoURL)
	if err != nil {
		return nil, err
	}
	entries, ok := index.Entries[chart]
	if !ok {
		return nil, fmt.Errorf("chart %s not found in %s", chart, repoURL)
	}
	versions := make([]string, 0, len(entries))
	for _, e := range entries {
		versions = append(versions, e.Version)
	}
	return versions, nil
}

func (l *Lister) index(ctx context.Context, repoURL string) (*helmIndex, error) {
	l.mu.Lock()
	r, ok := l.indexes[repoURL]
	if !ok {
		r = &indexResult{}
		l.indexes[repoURL] = r
	}
	l.mu.Unlock()

	r.once.Do(func() {
		r.index, r.err = l.fetchIndex(ctx, repoURL)
	})
	return r.index, r.err
}

func (l *Lister) fetchIndex(ctx context.Context, repoURL string) (*helmIndex, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, repoURL+"/index.yaml", nil)
	if err != nil {
		return nil, err
	}
	resp, err := l.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s/index.yaml: %s", repoURL, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// As helm does; large indexes (tens of MB) are slow to parse with other YAML libraries
	index := &helmIndex{}
	if err := yaml.Unmarshal(body, index); err != nil {
		return nil, fmt.Errorf("error parsing %s/index.yaml: %w", repoURL, err)
	}
	return index, nil
}
