package daemon

import (
	"testing"

	"github.com/bartlettc22/image-inquisitor/internal/argocd"
	"github.com/bartlettc22/image-inquisitor/internal/prdiff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChartTargets(t *testing.T) {
	apps := []argocd.Application{
		{Name: "certs", Sources: []argocd.Source{
			{RepoURL: "https://charts.example.com/", Chart: "certs", TargetRevision: "v1.18.2"},
			{RepoURL: "https://github.com/org/deploy.git", Ref: "values"},
		}},
		{Name: "certs-old", Sources: []argocd.Source{{RepoURL: "https://charts.example.com", Chart: "certs", TargetRevision: "v1.17.0"}}},
		{Name: "dash", Sources: []argocd.Source{{RepoURL: "oci://ghcr.io/org/charts/dash", Path: ".", TargetRevision: "12.0.0"}}},
	}

	targets := chartTargets(prdiff.Change{Kind: prdiff.KindHelmChart, Name: "certs", RepoURL: "https://charts.example.com", From: "v1.18.2", To: "v1.19.0"}, apps)
	require.Len(t, targets, 1)
	assert.Equal(t, "certs", targets[0].app.Name)
	assert.Equal(t, 0, targets[0].source)

	targets = chartTargets(prdiff.Change{Kind: prdiff.KindHelmChart, Name: "dash", RepoURL: "oci://ghcr.io/org/charts/dash", From: "12.0.0", To: "12.1.0"}, apps)
	require.Len(t, targets, 1)
	assert.Equal(t, "dash", targets[0].app.Name)

	assert.Empty(t, chartTargets(prdiff.Change{Kind: prdiff.KindHelmChart, Name: "certs", RepoURL: "https://charts.example.com", From: "v1.0.0", To: "v1.1.0"}, apps))
}
