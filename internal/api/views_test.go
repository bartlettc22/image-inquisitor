package api

import (
	"slices"
	"testing"

	"github.com/bartlettc22/image-inquisitor/internal/argocd"
	"github.com/bartlettc22/image-inquisitor/internal/daemon"
	"github.com/bartlettc22/image-inquisitor/internal/prdiff"
	"github.com/bartlettc22/image-inquisitor/internal/render"
	"github.com/bartlettc22/image-inquisitor/internal/store"
	"github.com/bartlettc22/image-inquisitor/internal/workloads"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseImage(t *testing.T) {
	testMatrix := []struct {
		ref      string
		expected imageRef
	}{
		{ref: "mariadb:11.7", expected: imageRef{repository: "index.docker.io/library/mariadb", tag: "11.7"}},
		{ref: "docker.io/library/mariadb:11.7", expected: imageRef{repository: "index.docker.io/library/mariadb", tag: "11.7"}},
		{ref: "nginx", expected: imageRef{repository: "index.docker.io/library/nginx", tag: "latest"}},
		{ref: "ghcr.io/org/app:1.0@sha256:abc", expected: imageRef{repository: "ghcr.io/org/app", tag: "1.0", digest: "sha256:abc"}},
		{ref: "ghcr.io/org/app@sha256:abc", expected: imageRef{repository: "ghcr.io/org/app", digest: "sha256:abc"}},
		{ref: "registry.local:5000/team/app@sha256:abc", expected: imageRef{repository: "registry.local:5000/team/app", digest: "sha256:abc"}},
	}
	for _, tm := range testMatrix {
		parsed, ok := parseImage(tm.ref)
		require.True(t, ok, tm.ref)
		assert.Equal(t, tm.expected, parsed, tm.ref)
	}
}

func TestPullRequestViewAndRunningImages(t *testing.T) {
	containers := parseContainers([]workloads.Container{
		{Cluster: "bernes", Namespace: "romm", WorkloadKind: "Deployment", WorkloadName: "romm", Pod: "romm-1", Container: "romm", Image: "rommapp/romm:3.9.0", Digest: "sha256:aaa"},
		{Cluster: "bernes", Namespace: "romm", WorkloadKind: "Deployment", WorkloadName: "romm", Pod: "romm-2", Container: "romm", Image: "rommapp/romm:3.9.0", Digest: "sha256:aaa"},
		{Cluster: "bernes", Namespace: "romm", WorkloadKind: "StatefulSet", WorkloadName: "mariadb", Pod: "mariadb-0", Container: "mariadb", Image: "docker.io/library/mariadb:11.6"},
		{Cluster: "bernes", Namespace: "tools", WorkloadKind: "Deployment", WorkloadName: "echo", Pod: "echo-1", Container: "echo", Image: "mendhak/http-https-echo:39"},
	}, newAttribution(nil, nil, nil))

	romm := &store.PullRequest{Repo: "org/deploy", Number: 95, Title: "Update romm", Changes: []prdiff.Change{
		{Kind: prdiff.KindImage, Name: "rommapp/romm", From: "rommapp/romm:3.9.0", To: "rommapp/romm:5.4.0"},
		{Kind: prdiff.KindHelmChart, Name: "chart", From: "1.0.0", To: "2.0.0"},
	}}
	mariadb := &store.PullRequest{Repo: "org/deploy", Number: 93, Title: "Update mariadb", Changes: []prdiff.Change{
		{Kind: prdiff.KindImage, Name: "mariadb", From: "mariadb:11.7", To: "mariadb:12.2"},
	}}

	view := pullRequestView(romm, containers, true)
	require.Len(t, view.Changes, 2)
	assert.Equal(t, []Workload{{
		Cluster: "bernes", Namespace: "romm", Kind: "Deployment", Name: "romm",
		Containers: []string{"romm"}, Digests: []string{"sha256:aaa"},
	}}, view.Changes[0].Running)
	assert.Nil(t, view.Changes[1].Running, "chart changes are unknown until rendered")

	// Running 11.6 while the PR updates 11.7: no workloads run the From version
	assert.Equal(t, []Workload{}, pullRequestView(mariadb, containers, true).Changes[0].Running)
	assert.Nil(t, pullRequestView(mariadb, containers, false).Changes[0].Running, "no inventory")

	images := runningImages(containers, []*store.PullRequest{romm, mariadb}, nil)
	require.Len(t, images, 3)

	assert.Equal(t, "index.docker.io/library/mariadb", images[0].Repository)
	assert.Equal(t, []Coverage{{Repo: "org/deploy", Number: 93, Title: "Update mariadb", From: "mariadb:11.7", To: "mariadb:12.2", MatchesRunning: false}}, images[0].CoveredBy)

	assert.Equal(t, "index.docker.io/mendhak/http-https-echo", images[1].Repository)
	assert.Empty(t, images[1].CoveredBy)

	assert.Equal(t, "index.docker.io/rommapp/romm", images[2].Repository)
	assert.Equal(t, []string{"sha256:aaa"}, images[2].Digests)
	require.Len(t, images[2].CoveredBy, 1)
	assert.True(t, images[2].CoveredBy[0].MatchesRunning)
}

func TestChartChanges(t *testing.T) {
	apps := []argocd.Application{{
		Cluster: "bernes", Name: "certs", DestNamespace: "certs",
		Sources: []argocd.Source{
			{RepoURL: "https://charts.example.com", Chart: "certs", TargetRevision: "1.0.0"},
			{RepoURL: "https://github.com/org/charts.git", Path: "charts/common", TargetRevision: "HEAD"},
		},
		Revisions:   []string{"1.0.0", "abc123"},
		SourceTypes: []string{"Helm", "Helm"},
	}}
	renders := []store.Render{
		{Cluster: "bernes", App: "certs", Source: 0, Revision: "1.0.0", Images: []render.Image{
			{Kind: "Deployment", Name: "certs-controller", Namespace: "certs", Container: "controller", Image: "example.com/certs/controller:1.0.0"},
			{Kind: "Deployment", Name: "certs-webhook", Namespace: "certs", Container: "webhook", Image: "example.com/certs/webhook:1.0.0"},
		}},
		{Cluster: "bernes", App: "certs", Source: 1, Revision: "abc123", Images: []render.Image{
			{Kind: "CronJob", Name: "backup", Namespace: "certs", Container: "backup", Image: "busybox:1.36"},
		}},
	}
	containers := parseContainers([]workloads.Container{
		{Cluster: "bernes", Namespace: "certs", WorkloadKind: "Deployment", WorkloadName: "certs-controller", Container: "controller", Image: "example.com/certs/controller:1.0.0", ArgoApp: "certs"},
		{Cluster: "bernes", Namespace: "certs", WorkloadKind: "Deployment", WorkloadName: "certs-webhook", Container: "webhook", Image: "example.com/certs/webhook:1.0.0", ArgoApp: "certs"},
		{Cluster: "bernes", Namespace: "certs", WorkloadKind: "CronJob", WorkloadName: "backup", Container: "backup", Image: "busybox:1.36", ArgoApp: "certs"},
		{Cluster: "bernes", Namespace: "certs", WorkloadKind: "Deployment", WorkloadName: "manual", Container: "manual", Image: "nginx:1.27", ArgoApp: "certs"},
	}, newAttribution(apps, renders, nil))

	require.NotNil(t, containers[0].source)
	assert.Equal(t, Source{Cluster: "bernes", App: "certs", Index: 0, Kind: "chart", Chart: "certs", RepoURL: "https://charts.example.com", Version: "1.0.0"}, *containers[0].source)
	require.NotNil(t, containers[2].source)
	assert.Equal(t, "common", containers[2].source.Chart)
	assert.Nil(t, containers[3].source, "not in any render")

	pr := &store.PullRequest{
		Repo: "org/deploy", Number: 7, Title: "Update certs",
		Changes: []prdiff.Change{{Kind: prdiff.KindHelmChart, Name: "certs", RepoURL: "https://charts.example.com", From: "1.0.0", To: "1.1.0"}},
		Stages:  []store.Stage{{Name: renderStage, Status: store.StageStatusDone}},
		ChartRenders: []store.ChartRender{{Change: 0, Cluster: "bernes", App: "certs", Source: 0, Images: []render.ImageDiff{
			{Kind: "Deployment", Name: "certs-controller", Namespace: "certs", Container: "controller", From: "example.com/certs/controller:1.0.0", To: "example.com/certs/controller:1.1.0"},
		}}},
	}

	view := pullRequestView(pr, containers, true)
	change := view.Changes[0]
	assert.Len(t, change.Running, 2, "both chart workloads, not the git chart's or the unattributed one")
	require.Len(t, change.Renders, 1)
	require.Len(t, change.Renders[0].Images, 1)
	require.Len(t, change.Renders[0].Images[0].Running, 1)
	assert.Equal(t, "certs-controller", change.Renders[0].Images[0].Running[0].Name)

	images := runningImages(containers, []*store.PullRequest{pr}, nil)
	controller := images[slices.IndexFunc(images, func(i RunningImage) bool { return i.Repository == "example.com/certs/controller" })]
	assert.Equal(t, []Coverage{{
		Repo: "org/deploy", Number: 7, Title: "Update certs",
		From: "example.com/certs/controller:1.0.0", To: "example.com/certs/controller:1.1.0",
		MatchesRunning: true, Via: "certs 1.0.0 → 1.1.0",
	}}, controller.CoveredBy)

	apps[0].Health = "Healthy"
	views := appViews(apps, renders, containers, nil, nil)
	require.Len(t, views, 1)
	assert.Equal(t, 4, views[0].Containers)
	assert.Equal(t, 1, views[0].Unattributed)
	assert.Equal(t, "chart", views[0].Sources[0].Kind)
	assert.Equal(t, "git", views[0].Sources[1].Kind)
}

func TestVersionStatus(t *testing.T) {
	apps := []argocd.Application{{
		Cluster: "bernes", Name: "web", DestNamespace: "web",
		Sources: []argocd.Source{
			{RepoURL: "https://charts.example.com", Chart: "web", TargetRevision: "1.0.0"},
			{RepoURL: "https://github.com/org/charts.git", Path: "charts/common"},
		},
		Revisions:   []string{"1.0.0", "abc123"},
		SourceTypes: []string{"Helm", "Helm"},
	}}
	renders := []store.Render{
		{Cluster: "bernes", App: "web", Source: 0, Revision: "1.0.0", Images: []render.Image{
			{Kind: "Deployment", Name: "web", Namespace: "web", Container: "web", Image: "example.com/web:1.0.0"},
		}},
		{Cluster: "bernes", App: "web", Source: 1, Revision: "abc123", Images: []render.Image{
			{Kind: "Deployment", Name: "worker", Namespace: "web", Container: "worker", Image: "example.com/worker:2.0.0"},
		}},
	}
	versions := newVersionIndex([]store.Version{
		{Kind: store.VersionKindImage, Repository: "example.com/web", Current: "1.0.0", Latest: "1.1.0"},
		{Kind: store.VersionKindImage, Repository: "example.com/worker", Current: "2.0.0", Latest: "2.1.0"},
		{Kind: store.VersionKindImage, Repository: "example.com/api", Current: "3.0.0", Latest: "3.0.0"},
		{Kind: store.VersionKindImage, Repository: "example.com/cron", Current: "1.0.0", Latest: "1.2.0"},
		{Kind: store.VersionKindImage, Repository: "example.com/edge", Current: "latest"},
		{Kind: store.VersionKindChart, Repository: "https://charts.example.com/web", Current: "1.0.0", Latest: "1.3.0"},
	})
	containers := parseContainers([]workloads.Container{
		{Cluster: "bernes", Namespace: "web", WorkloadKind: "Deployment", WorkloadName: "web", Container: "web", Image: "example.com/web:1.0.0", ArgoApp: "web"},
		{Cluster: "bernes", Namespace: "web", WorkloadKind: "Deployment", WorkloadName: "worker", Container: "worker", Image: "example.com/worker:2.0.0", ArgoApp: "web"},
		{Cluster: "bernes", Namespace: "api", WorkloadKind: "Deployment", WorkloadName: "api", Container: "api", Image: "example.com/api:3.0.0"},
		{Cluster: "bernes", Namespace: "ops", WorkloadKind: "CronJob", WorkloadName: "cron", Container: "cron", Image: "example.com/cron:1.0.0"},
		{Cluster: "bernes", Namespace: "edge", WorkloadKind: "Deployment", WorkloadName: "edge", Container: "edge", Image: "example.com/edge:latest"},
	}, newAttribution(apps, renders, versions))
	assert.Equal(t, "1.3.0", containers[0].source.Latest)

	cronPR := &store.PullRequest{Repo: "org/deploy", Number: 1, Changes: []prdiff.Change{
		{Kind: prdiff.KindImage, Name: "example.com/cron", From: "example.com/cron:1.0.0", To: "example.com/cron:1.2.0"},
	}}
	statuses := map[string]string{}
	for _, img := range runningImages(containers, []*store.PullRequest{cronPR}, versions) {
		statuses[img.Repository] = img.Status
	}
	assert.Equal(t, map[string]string{
		"example.com/web":    StatusPinned, // set by a chart from a Helm repository
		"example.com/worker": StatusBehind, // set by values of a git chart
		"example.com/api":    StatusCurrent,
		"example.com/cron":   StatusPending,
		"example.com/edge":   StatusUnknown,
	}, statuses)

	views := appViews(apps, renders, containers, versions, nil)
	assert.Equal(t, StatusBehind, views[0].Sources[0].Status)
	assert.Empty(t, views[0].Sources[1].Status, "git charts have no published versions")

	chartPR := &store.PullRequest{Repo: "org/deploy", Number: 2, Changes: []prdiff.Change{
		{Kind: prdiff.KindHelmChart, Name: "web", RepoURL: "https://charts.example.com/", From: "1.0.0", To: "1.3.0"},
	}}
	views = appViews(apps, renders, containers, versions, []*store.PullRequest{chartPR})
	assert.Equal(t, StatusPending, views[0].Sources[0].Status)
	require.Len(t, views[0].Sources[0].UpdatePRs, 1)
	assert.Equal(t, 2, views[0].Sources[0].UpdatePRs[0].Number)
}

func TestAttachSoaks(t *testing.T) {
	images := []RunningImage{
		{Repository: "index.docker.io/library/nginx", Tag: "1.27"},
		{Repository: "index.docker.io/library/nginx", Tag: "1.29"},
	}
	attachSoaks(images, []daemon.SoakDecision{
		{Repository: "index.docker.io/library/nginx", Current: "1.27", Newest: "1.29", Lane: daemon.SoakLaneFast},
	})
	require.NotNil(t, images[0].Soak)
	assert.Equal(t, daemon.SoakLaneFast, images[0].Soak.Lane)
	assert.Nil(t, images[1].Soak, "already the newest")
}
