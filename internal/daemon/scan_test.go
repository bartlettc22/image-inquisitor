package daemon

import (
	"testing"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/prdiff"
	"github.com/bartlettc22/image-inquisitor/internal/render"
	"github.com/bartlettc22/image-inquisitor/internal/store"
	"github.com/bartlettc22/image-inquisitor/internal/workloads"
	"github.com/stretchr/testify/assert"
)

func TestScanTargets(t *testing.T) {
	prs := []*store.PullRequest{
		{Changes: []prdiff.Change{
			{Kind: prdiff.KindImage, From: "nginx:1.27", To: "nginx:1.28"},
			{Kind: prdiff.KindHelmChart, Name: "web", From: "1.0.0", To: "1.1.0"},
		}, ChartRenders: []store.ChartRender{{Images: []render.ImageDiff{
			{From: "example.com/web:1.0.0", To: "example.com/web:1.1.0"},
			{From: "", To: "example.com/sidecar:2.0.0"},
		}}}},
		{Changes: []prdiff.Change{{Kind: prdiff.KindImage, From: "nginx:1.27", To: "nginx:1.29"}}},
	}
	containers := []workloads.Container{
		{Image: "nginx:1.27", Digest: "sha256:aaa"},
		{Image: "example.com/db@sha256:old", Digest: "sha256:bbb"},
		{Image: "example.com/backup:1.0"},
		{Image: "nginx:1.27", Digest: "sha256:aaa"},
	}
	versions := []store.Version{
		{Kind: store.VersionKindImage, Repository: "index.docker.io/library/nginx", Current: "1.27", Latest: "1.29.3"},
		{Kind: store.VersionKindImage, Repository: "example.com/backup", Current: "1.0", Latest: "1.0"},
		{Kind: store.VersionKindImage, Repository: "example.com/db", Current: "2", Latest: "3", Error: "denied"},
		{Kind: store.VersionKindChart, Repository: "https://charts.example.com/web", Current: "1.0.0", Latest: "1.1.0"},
	}
	refs, kinds := []string{}, []string{}
	for _, target := range scanTargets(prs, containers, versions) {
		refs = append(refs, target.ref)
		kinds = append(kinds, target.kind)
	}
	assert.Equal(t, []string{
		"nginx:1.27", "nginx:1.28", "example.com/web:1.0.0", "example.com/web:1.1.0", "example.com/sidecar:2.0.0", "nginx:1.29",
		"nginx:1.27@sha256:aaa", "example.com/db@sha256:bbb",
		"index.docker.io/library/nginx:1.29.3",
	}, refs, "only image tags with a newer version, without a listing error")
	assert.Equal(t, []string{"pr", "pr", "pr", "pr", "pr", "pr", "running", "running", "newest"}, kinds)
}

func TestResolutionStale(t *testing.T) {
	now := time.Now()
	assert.False(t, resolutionStale(&store.Resolution{Ref: "nginx:1.27", ResolvedAt: now.Add(-time.Hour)}))
	assert.True(t, resolutionStale(&store.Resolution{Ref: "nginx:1.27", ResolvedAt: now.Add(-7 * time.Hour)}), "tags move")
	assert.False(t, resolutionStale(&store.Resolution{Ref: "nginx@sha256:abc", ResolvedAt: now.Add(-100 * time.Hour)}), "digests don't")
	assert.True(t, resolutionStale(&store.Resolution{Ref: "nginx:1.27", Error: "boom", ResolvedAt: now.Add(-2 * time.Hour)}))
	assert.False(t, resolutionStale(&store.Resolution{Ref: "nginx:1.27", Error: "boom", ResolvedAt: now.Add(-time.Minute)}))
}
