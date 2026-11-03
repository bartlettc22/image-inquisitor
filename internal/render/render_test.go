package render

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/argocd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const manifests = `
apiVersion: v1
kind: Service
metadata:
  name: web
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  template:
    spec:
      initContainers:
      - name: migrate
        image: example.com/web:1.0.0
      containers:
      - name: web
        image: example.com/web:1.0.0
      - name: proxy
        image: nginx:1.27
---
apiVersion: batch/v1
kind: CronJob
metadata:
  name: backup
  namespace: jobs
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: busybox:1.36
---
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: db
spec:
  imageName: ghcr.io/cloudnative-pg/postgresql:16
---
`

func TestImages(t *testing.T) {
	images, err := Images([]byte(manifests), "apps")
	require.NoError(t, err)
	assert.Equal(t, []Image{
		{Kind: "Cluster", Name: "db", Namespace: "apps", Image: "ghcr.io/cloudnative-pg/postgresql:16"},
		{Kind: "CronJob", Name: "backup", Namespace: "jobs", Container: "backup", Image: "busybox:1.36"},
		{Kind: "Deployment", Name: "web", Namespace: "apps", Container: "migrate", Init: true, Image: "example.com/web:1.0.0"},
		{Kind: "Deployment", Name: "web", Namespace: "apps", Container: "proxy", Image: "nginx:1.27"},
		{Kind: "Deployment", Name: "web", Namespace: "apps", Container: "web", Image: "example.com/web:1.0.0"},
	}, images)
}

func TestDiff(t *testing.T) {
	web := Image{Kind: "Deployment", Name: "web", Namespace: "apps", Container: "web", Image: "example.com/web:1.0.0"}
	proxy := Image{Kind: "Deployment", Name: "web", Namespace: "apps", Container: "proxy", Image: "nginx:1.27"}
	sidecar := Image{Kind: "Deployment", Name: "web", Namespace: "apps", Container: "sidecar", Image: "envoy:1.30"}

	webNew := web
	webNew.Image = "example.com/web:2.0.0"

	assert.Equal(t, []ImageDiff{
		{Kind: "Deployment", Name: "web", Namespace: "apps", Container: "proxy", From: "nginx:1.27"},
		{Kind: "Deployment", Name: "web", Namespace: "apps", Container: "sidecar", To: "envoy:1.30"},
		{Kind: "Deployment", Name: "web", Namespace: "apps", Container: "web", From: "example.com/web:1.0.0", To: "example.com/web:2.0.0"},
	}, Diff([]Image{web, proxy}, []Image{webNew, sidecar}))

	assert.Empty(t, Diff([]Image{web}, []Image{web}))
}

type fakeFetcher struct {
	files   map[string]string
	archive []byte
}

func (f *fakeFetcher) File(_ context.Context, repoURL, revision, path string) ([]byte, error) {
	content, ok := f.files[repoURL+"@"+revision+":"+path]
	if !ok {
		return nil, ErrNotFound
	}
	return []byte(content), nil
}

func (f *fakeFetcher) Archive(context.Context, string, string) ([]byte, error) {
	return f.archive, nil
}

// fakeHelm writes a helm stand-in that records its arguments and the values files' content,
// and prints manifests for template
func fakeHelm(t *testing.T) (helm, argsFile string) {
	dir := t.TempDir()
	argsFile = filepath.Join(dir, "args")
	manifestsFile := filepath.Join(dir, "manifests.yaml")
	require.NoError(t, os.WriteFile(manifestsFile, []byte(manifests), 0o600))

	helm = filepath.Join(dir, "helm")
	script := `#!/bin/sh
for a in "$@"; do
  echo "$a" >> ` + argsFile + `
  case "$a" in *.yaml) [ -f "$a" ] && sed 's/^/  | /' "$a" >> ` + argsFile + ` ;; esac
done
[ "$1" = template ] && cat ` + manifestsFile + `
exit 0
`
	require.NoError(t, os.WriteFile(helm, []byte(script), 0o755))
	return helm, argsFile
}

func app() *argocd.Application {
	return &argocd.Application{
		Name:          "web",
		DestNamespace: "apps",
		Sources: []argocd.Source{
			{
				RepoURL: "https://charts.example.com", Chart: "web", TargetRevision: "1.x",
				Helm: &argocd.Helm{
					ValueFiles:              []string{"$values/apps/web/values.yaml", "$values/apps/web/missing.yaml"},
					IgnoreMissingValueFiles: true,
					ValuesObject:            []byte(`{"replicas":2}`),
					Parameters:              []argocd.Parameter{{Name: "tag", Value: "1", ForceString: true}},
				},
			},
			{RepoURL: "https://github.com/org/deploy.git", TargetRevision: "main", Ref: "values"},
		},
		Revisions:   []string{"1.2.0", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		SourceTypes: []string{"Helm", ""},
	}
}

func TestRenderHelmRepoChart(t *testing.T) {
	helm, argsFile := fakeHelm(t)
	fetcher := &fakeFetcher{files: map[string]string{
		"https://github.com/org/deploy.git@aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa:apps/web/values.yaml": "image: deployed",
		"https://github.com/org/deploy.git@bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb:apps/web/values.yaml": "image: pr-head",
	}}
	r := &Renderer{Helm: helm, Dir: t.TempDir(), Fetcher: fetcher}

	req := &Request{App: app(), Source: 0, Capabilities: Capabilities{KubeVersion: "v1.37.0", APIVersions: []string{"apps/v1"}}}
	images, err := r.Render(context.Background(), req)
	require.NoError(t, err)
	assert.Len(t, images, 5)

	args, err := os.ReadFile(argsFile)
	require.NoError(t, err)
	got := string(args)
	for _, want := range []string{
		"template\nweb\nweb\n--repo\nhttps://charts.example.com\n--version\n1.2.0\n",
		"--namespace\napps\n--skip-tests\n--kube-version\nv1.37.0\n--api-versions\napps/v1\n",
		"  | image: deployed",
		`  | {"replicas":2}`,
		"--set-string\ntag=1\n",
	} {
		assert.Contains(t, got, want)
	}
	assert.Equal(t, 2, strings.Count(got, "--values"), "the missing value file is skipped")

	// A PR render: new chart version, values from the PR head
	require.NoError(t, os.Remove(argsFile))
	prReq := &Request{
		App: app(), Source: 0, Capabilities: req.Capabilities, Revision: "1.3.0",
		RepoRevisions: map[string]string{"https://github.com/org/deploy.git": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
	}
	_, err = r.Render(context.Background(), prReq)
	require.NoError(t, err)
	args, err = os.ReadFile(argsFile)
	require.NoError(t, err)
	assert.Contains(t, string(args), "--version\n1.3.0\n")
	assert.Contains(t, string(args), "  | image: pr-head")

	assert.NotEqual(t, req.Key(), prReq.Key())
	assert.Equal(t, req.Key(), (&Request{App: app(), Source: 0, Capabilities: req.Capabilities}).Key())
}

func TestRenderOCIAndGitCharts(t *testing.T) {
	helm, argsFile := fakeHelm(t)
	r := &Renderer{Helm: helm, Dir: t.TempDir(), Fetcher: &fakeFetcher{archive: tarball(t, map[string]string{
		"repo-abc/charts/common/Chart.yaml":  "name: common\n",
		"repo-abc/charts/common/values.yaml": "x: 1\n",
	})}}

	oci := &argocd.Application{Name: "dash", DestNamespace: "mon", Sources: []argocd.Source{
		{RepoURL: "oci://ghcr.io/org/charts/dash", Path: ".", TargetRevision: "12.0.0"},
	}}
	_, err := r.Render(context.Background(), &Request{App: oci})
	require.NoError(t, err)
	args, _ := os.ReadFile(argsFile)
	assert.Contains(t, string(args), "template\ndash\noci://ghcr.io/org/charts/dash\n--version\n12.0.0\n")

	require.NoError(t, os.Remove(argsFile))
	sha := "cccccccccccccccccccccccccccccccccccccccc"
	git := &argocd.Application{Name: "common", DestNamespace: "x", Revisions: []string{sha}, Sources: []argocd.Source{
		{RepoURL: "https://github.com/org/charts.git", Path: "charts/common", TargetRevision: "HEAD", Helm: &argocd.Helm{ValueFiles: []string{"values.yaml"}}},
	}}
	_, err = r.Render(context.Background(), &Request{App: git})
	require.NoError(t, err)
	args, _ = os.ReadFile(argsFile)
	assert.Contains(t, string(args), filepath.Join("charts", "common")+"\n--namespace")
	assert.Contains(t, string(args), "  | x: 1")
}

func TestPruneCheckouts(t *testing.T) {
	r := &Renderer{Dir: t.TempDir()}
	old := filepath.Join(r.Dir, "git", "repo", "old")
	recent := filepath.Join(r.Dir, "git", "repo", "recent")
	require.NoError(t, os.MkdirAll(old, 0o755))
	require.NoError(t, os.MkdirAll(recent, 0o755))
	weekAgo := time.Now().Add(-8 * 24 * time.Hour)
	require.NoError(t, os.Chtimes(old, weekAgo, weekAgo))

	removed, err := r.PruneCheckouts(7 * 24 * time.Hour)
	require.NoError(t, err)
	assert.Equal(t, 1, removed)
	assert.NoDirExists(t, old)
	assert.DirExists(t, recent)
}

func TestExtractRejectsTraversal(t *testing.T) {
	archive := tarball(t, map[string]string{"repo/../../escape": "x"})
	assert.Error(t, extract(archive, t.TempDir()))
}

func tarball(t *testing.T, files map[string]string) []byte {
	buf := &bytes.Buffer{}
	gz := gzip.NewWriter(buf)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		require.NoError(t, tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(content)), Typeflag: tar.TypeReg}))
		_, err := tw.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf.Bytes()
}
