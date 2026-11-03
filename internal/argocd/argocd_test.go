package argocd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

func parseYAML(t *testing.T, manifest string) Application {
	obj := map[string]any{}
	require.NoError(t, yaml.Unmarshal([]byte(manifest), &obj))
	app, err := parse(&unstructured.Unstructured{Object: obj})
	require.NoError(t, err)
	return app
}

func TestParseMultiSource(t *testing.T) {
	app := parseYAML(t, `
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata: {name: certs, namespace: argocd}
spec:
  project: default
  destination: {namespace: certs}
  sources:
  - repoURL: https://charts.example.com
    chart: certs
    targetRevision: v1.x
    helm:
      valueFiles: [$values/apps/certs/values.yaml]
  - repoURL: https://github.com/org/charts.git
    path: charts/common
    targetRevision: HEAD
  - repoURL: https://github.com/org/deploy.git
    targetRevision: main
    ref: values
status:
  sync: {status: Synced, revisions: [v1.2.0, abc, def]}
  health: {status: Healthy}
  sourceTypes: [Helm, Helm, ""]
`)
	assert.Equal(t, "certs", app.Name)
	assert.Equal(t, "certs", app.DestNamespace)
	assert.Equal(t, "Synced", app.SyncStatus)
	assert.Equal(t, "Healthy", app.Health)
	require.Len(t, app.Sources, 3)
	assert.Equal(t, []string{"$values/apps/certs/values.yaml"}, app.Sources[0].Helm.ValueFiles)

	assert.True(t, app.IsHelm(0))
	assert.True(t, app.IsHelm(1))
	assert.False(t, app.IsHelm(2), "values ref")
	assert.Equal(t, "v1.2.0", app.Revision(0), "resolved from the range")
	assert.Equal(t, "certs", app.ReleaseName(0))

	assert.Equal(t, "https://charts.example.com/certs", app.Sources[0].ChartRef())
	assert.Equal(t, "https://github.com/org/charts.git//charts/common", app.Sources[1].ChartRef())
	assert.Equal(t, "common", app.Sources[1].ChartName())
}

func TestParseSingleSource(t *testing.T) {
	app := parseYAML(t, `
metadata: {name: dash}
spec:
  destination: {namespace: mon}
  source:
    repoURL: oci://ghcr.io/org/charts/dash
    path: .
    targetRevision: 12.0.0
    helm: {releaseName: grafana}
status:
  sync: {revision: "sha256:abc"}
  sourceType: Helm
`)
	require.Len(t, app.Sources, 1)
	assert.Equal(t, []string{"sha256:abc"}, app.Revisions)
	assert.Equal(t, "12.0.0", app.Version(0), "OCI revisions are digests")
	assert.True(t, app.IsHelm(0))
	assert.Equal(t, "grafana", app.ReleaseName(0))
	assert.Equal(t, "oci://ghcr.io/org/charts/dash", app.Sources[0].ChartRef())
	assert.Equal(t, "dash", app.Sources[0].ChartName())
}

func TestChartRefOCIWithoutScheme(t *testing.T) {
	s := Source{RepoURL: "registry.example.com/charts", Chart: "web"}
	assert.Equal(t, "oci://registry.example.com/charts/web", s.ChartRef())
}
