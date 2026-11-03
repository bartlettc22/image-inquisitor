package daemon

import (
	"testing"

	"github.com/bartlettc22/image-inquisitor/internal/argocd"
	"github.com/bartlettc22/image-inquisitor/internal/store"
	"github.com/bartlettc22/image-inquisitor/internal/workloads"
	"github.com/stretchr/testify/assert"
)

func TestVersionTargets(t *testing.T) {
	containers := []workloads.Container{
		{Image: "nginx:1.27"},
		{Image: "docker.io/library/nginx:1.28"},
		{Image: "ghcr.io/org/app:v2.0.0@sha256:abc"},
		{Image: "ghcr.io/org/pinned@sha256:abc"},
		{Image: "busybox"},
	}
	apps := []argocd.Application{{
		Sources: []argocd.Source{
			{RepoURL: "https://charts.example.com", Chart: "web", TargetRevision: "1.x"},
			{RepoURL: "oci://ghcr.io/org/charts/dash", Path: ".", TargetRevision: "12.0.0"},
			{RepoURL: "https://github.com/org/charts.git", Path: "charts/common"},
			{RepoURL: "https://github.com/org/deploy.git", Ref: "values"},
		},
		Revisions:   []string{"1.2.0", "sha256:def", "abc123", "def456"},
		SourceTypes: []string{"Helm", "Helm", "Helm", ""},
	}}

	assert.Equal(t, map[versionTarget]map[string]bool{
		{kind: store.VersionKindImage, repository: "index.docker.io/library/nginx"}:                                                       {"1.27": true, "1.28": true},
		{kind: store.VersionKindImage, repository: "ghcr.io/org/app"}:                                                                     {"v2.0.0": true},
		{kind: store.VersionKindChart, repository: "https://charts.example.com/web", repoURL: "https://charts.example.com", chart: "web"}: {"1.2.0": true},
		{kind: store.VersionKindChart, repository: "oci://ghcr.io/org/charts/dash", repoURL: "oci://ghcr.io/org/charts/dash"}:             {"12.0.0": true},
	}, versionTargets(containers, apps))
}
