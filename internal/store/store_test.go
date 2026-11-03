package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/argocd"
	"github.com/bartlettc22/image-inquisitor/internal/prdiff"
	"github.com/bartlettc22/image-inquisitor/internal/render"
	"github.com/bartlettc22/image-inquisitor/internal/scan"
	"github.com/bartlettc22/image-inquisitor/internal/workloads"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore(t *testing.T) {
	ctx := context.Background()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	defer s.Close()

	now := time.Now().UTC().Truncate(time.Second)
	for _, n := range []int{1, 2} {
		require.NoError(t, s.UpsertPullRequest(ctx, &PullRequest{
			Repo: "org/deploy", Number: n, Title: "Update app", URL: "https://example.com",
			Branch: "renovate/app", HeadSHA: "abc", Author: "bot", State: PullRequestStateOpen,
			CreatedAt: now, UpdatedAt: now, SyncedAt: now,
		}))
	}

	changes := []prdiff.Change{
		{File: "values.yaml", Kind: prdiff.KindImage, Name: "example/app", From: "example/app:1", To: "example/app:2"},
	}
	require.NoError(t, s.ReplaceChanges(ctx, "org/deploy", 1, changes))
	require.NoError(t, s.ReplaceChanges(ctx, "org/deploy", 1, changes))

	_, err = s.GetStage(ctx, "org/deploy", 1, "extract")
	assert.True(t, errors.Is(err, ErrNotFound))
	require.NoError(t, s.SetStage(ctx, "org/deploy", 1, &Stage{Name: "extract", Status: StageStatusDone, HeadSHA: "abc", UpdatedAt: now}))

	pr, err := s.GetPullRequest(ctx, "org/deploy", 1)
	require.NoError(t, err)
	assert.Equal(t, changes, pr.Changes)
	assert.Equal(t, []Stage{{Name: "extract", Status: StageStatusDone, HeadSHA: "abc", UpdatedAt: now}}, pr.Stages)
	assert.Equal(t, now, pr.CreatedAt)

	// PR 2 is no longer open
	require.NoError(t, s.CloseMissingPullRequests(ctx, "org/deploy", []int{1}))
	open, err := s.ListPullRequests(ctx, PullRequestStateOpen)
	require.NoError(t, err)
	require.Len(t, open, 1)
	assert.Equal(t, 1, open[0].Number)

	all, err := s.ListPullRequests(ctx, "")
	require.NoError(t, err)
	assert.Len(t, all, 2)

	containers := []workloads.Container{
		{Cluster: "bernes", Namespace: "romm", WorkloadKind: "Deployment", WorkloadName: "romm", Pod: "romm-1", Container: "romm", Image: "rommapp/romm:3.9.0", Digest: "sha256:aaa", ArgoApp: "romm"},
	}
	require.NoError(t, s.ReplaceContainers(ctx, "bernes", containers, now, nil))
	// A failed inventory keeps the previous containers and records the error
	require.NoError(t, s.ReplaceContainers(ctx, "bernes", nil, now, errors.New("unreachable")))
	stored, err := s.ListContainers(ctx)
	require.NoError(t, err)
	assert.Equal(t, containers, stored)
	clusters, err := s.ListClusters(ctx)
	require.NoError(t, err)
	assert.Equal(t, []Cluster{{Name: "bernes", SyncedAt: now, Containers: 1, Error: "unreachable"}}, clusters)

	_, err = s.LastSyncRun(ctx)
	assert.True(t, errors.Is(err, ErrNotFound))
	id, err := s.StartSyncRun(ctx, now)
	require.NoError(t, err)
	require.NoError(t, s.FinishSyncRun(ctx, id, now, 2, errors.New("boom")))
	run, err := s.LastSyncRun(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, run.PullRequests)
	assert.Equal(t, "boom", run.Error)
	require.NotNil(t, run.FinishedAt)
}

func TestApplicationsAndRenders(t *testing.T) {
	ctx := context.Background()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	defer s.Close()

	apps := []argocd.Application{
		{Cluster: "home", Name: "certs", Sources: []argocd.Source{{RepoURL: "https://charts.example.com", Chart: "certs"}}, Revisions: []string{"1.0.0"}},
		{Cluster: "home", Name: "web", Sources: []argocd.Source{{RepoURL: "https://charts.example.com", Chart: "web"}, {RepoURL: "https://charts.example.com", Chart: "db"}}},
	}
	require.NoError(t, s.ReplaceApplications(ctx, "home", apps))
	stored, err := s.ListApplications(ctx)
	require.NoError(t, err)
	assert.Equal(t, apps, stored)

	_, err = s.GetRender(ctx, "home", "certs", 0)
	assert.True(t, errors.Is(err, ErrNotFound))

	now := time.Now().UTC().Truncate(time.Second)
	certs := &Render{Cluster: "home", App: "certs", Source: 0, Key: "k1", Revision: "1.0.0", RenderedAt: now, Images: []render.Image{
		{Kind: "Deployment", Name: "certs", Namespace: "certs", Container: "controller", Image: "example.com/certs:1.0.0"},
	}}
	require.NoError(t, s.SaveRender(ctx, certs))
	require.NoError(t, s.SaveRender(ctx, &Render{Cluster: "home", App: "web", Source: 0, Key: "k2", Error: "boom", RenderedAt: now}))
	require.NoError(t, s.SaveRender(ctx, &Render{Cluster: "home", App: "web", Source: 1, Key: "k3", RenderedAt: now}))

	got, err := s.GetRender(ctx, "home", "certs", 0)
	require.NoError(t, err)
	assert.Equal(t, certs, got)

	// Sources beyond the app's current ones, and apps that are gone, lose their renders
	require.NoError(t, s.DeleteRenders(ctx, "home", "web", 1))
	require.NoError(t, s.ReplaceApplications(ctx, "home", apps[1:]))
	renders, err := s.ListRenders(ctx)
	require.NoError(t, err)
	require.Len(t, renders, 1)
	assert.Equal(t, "web", renders[0].App)
	assert.Equal(t, "boom", renders[0].Error)
	assert.Equal(t, []render.Image{}, renders[0].Images)

	// Chart renders are attached to their PR
	require.NoError(t, s.UpsertPullRequest(ctx, &PullRequest{Repo: "org/deploy", Number: 3, State: PullRequestStateOpen, CreatedAt: now, UpdatedAt: now, SyncedAt: now}))
	chartRenders := []ChartRender{{Change: 0, Cluster: "home", App: "web", Source: 0, Images: []render.ImageDiff{
		{Kind: "Deployment", Name: "web", Namespace: "web", Container: "web", From: "web:1", To: "web:2"},
	}}}
	require.NoError(t, s.ReplaceChartRenders(ctx, "org/deploy", 3, chartRenders))
	pr, err := s.GetPullRequest(ctx, "org/deploy", 3)
	require.NoError(t, err)
	assert.Equal(t, chartRenders, pr.ChartRenders)

	require.NoError(t, s.ReplaceChartRenders(ctx, "org/deploy", 3, nil))
	pr, err = s.GetPullRequest(ctx, "org/deploy", 3)
	require.NoError(t, err)
	assert.Empty(t, pr.ChartRenders)
}

func TestTagListsAndVersions(t *testing.T) {
	ctx := context.Background()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	defer s.Close()

	_, err = s.GetTagList(ctx, VersionKindImage, "index.docker.io/library/nginx")
	assert.True(t, errors.Is(err, ErrNotFound))

	now := time.Now().UTC().Truncate(time.Second)
	tl := &TagList{Kind: VersionKindImage, Repository: "index.docker.io/library/nginx", Tags: []string{"1.27", "1.28"}, FetchedAt: now}
	require.NoError(t, s.SaveTagList(ctx, tl))
	got, err := s.GetTagList(ctx, VersionKindImage, "index.docker.io/library/nginx")
	require.NoError(t, err)
	assert.Equal(t, tl, got)

	versions := []Version{
		{Kind: VersionKindChart, Repository: "https://charts.example.com/web", Current: "1.0.0", Latest: "1.1.0", CheckedAt: now},
		{Kind: VersionKindImage, Repository: "index.docker.io/library/nginx", Current: "1.27", Latest: "1.28", CheckedAt: now},
	}
	require.NoError(t, s.ReplaceVersions(ctx, versions))
	require.NoError(t, s.ReplaceVersions(ctx, versions[1:]))
	stored, err := s.ListVersions(ctx)
	require.NoError(t, err)
	assert.Equal(t, versions[1:], stored)
}

func TestResolutionsAndScans(t *testing.T) {
	ctx := context.Background()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	defer s.Close()

	now := time.Now().UTC().Truncate(time.Second)
	_, err = s.GetResolution(ctx, "nginx:1.27")
	assert.True(t, errors.Is(err, ErrNotFound))
	res := &Resolution{Ref: "nginx:1.27", Repository: "index.docker.io/library/nginx", Digest: "sha256:abc", ResolvedAt: now}
	require.NoError(t, s.SaveResolution(ctx, res))
	got, err := s.GetResolution(ctx, "nginx:1.27")
	require.NoError(t, err)
	assert.Equal(t, res, got)

	vulns := []scan.Vulnerability{{ID: "CVE-1", Package: "zlib", Severity: scan.SeverityHigh}}
	sc := &Scan{Digest: "sha256:abc", Repository: "index.docker.io/library/nginx", Counts: scan.CountBySeverity(vulns), SBOMAt: now, ScannedAt: now, Vulnerabilities: vulns}
	require.NoError(t, s.SaveScan(ctx, sc))
	require.NoError(t, s.SaveScan(ctx, &Scan{Digest: "sha256:def", Repository: "x", Counts: scan.Counts{}, Error: "pull failed"}))

	scans, err := s.ListScans(ctx)
	require.NoError(t, err)
	require.Len(t, scans, 2)
	assert.Equal(t, scan.Counts{scan.SeverityHigh: 1}, scans[0].Counts)
	assert.Nil(t, scans[0].Vulnerabilities, "not loaded by ListScans")
	assert.True(t, scans[1].SBOMAt.IsZero())

	stored, err := s.GetScanVulnerabilities(ctx, "sha256:abc")
	require.NoError(t, err)
	assert.Equal(t, vulns, stored)
}

func TestSyncRunHistory(t *testing.T) {
	ctx := context.Background()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	defer s.Close()

	now := time.Now().UTC().Truncate(time.Millisecond)
	ids := []int64{}
	for i := 0; i < 4; i++ {
		id, err := s.StartSyncRun(ctx, now)
		require.NoError(t, err)
		ids = append(ids, id)
		require.NoError(t, s.AddSyncStep(ctx, id, &SyncStep{Name: "repo a/b", StartedAt: now, FinishedAt: now.Add(1500 * time.Millisecond), Summary: "3 PRs"}))
		require.NoError(t, s.AddSyncStep(ctx, id, &SyncStep{Name: "cluster home", StartedAt: now, FinishedAt: now, Error: "unreachable"}))
		if i < 3 {
			require.NoError(t, s.FinishSyncRun(ctx, id, now, 3, nil))
		}
	}

	// The last run was left unfinished, as by a restart
	require.NoError(t, s.InterruptSyncRuns(ctx, now))
	require.NoError(t, s.PruneSyncRuns(ctx, 2))

	runs, err := s.ListSyncRuns(ctx, 10)
	require.NoError(t, err)
	require.Len(t, runs, 2)
	assert.Equal(t, ids[3], runs[0].ID, "newest first")
	assert.Equal(t, SyncRunInterrupted, runs[0].Error)
	assert.NotNil(t, runs[0].FinishedAt)
	assert.Equal(t, []SyncStep{
		{Name: "repo a/b", StartedAt: now, FinishedAt: now.Add(1500 * time.Millisecond), Summary: "3 PRs"},
		{Name: "cluster home", StartedAt: now, FinishedAt: now, Error: "unreachable"},
	}, runs[1].Steps)

	var steps int
	require.NoError(t, s.db.QueryRow(`SELECT COUNT(*) FROM sync_steps`).Scan(&steps))
	assert.Equal(t, 4, steps, "pruned runs' steps are deleted")

	stats, err := s.TagListStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, &TagListStats{Failures: []TagListFailure{}}, stats)
	require.NoError(t, s.SaveTagList(ctx, &TagList{Kind: VersionKindImage, Repository: "nginx", FetchedAt: now}))
	require.NoError(t, s.SaveTagList(ctx, &TagList{Kind: VersionKindImage, Repository: "redis", Error: "denied", FetchedAt: now.Add(time.Hour)}))
	stats, err = s.TagListStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, stats.Repositories)
	assert.Equal(t, 1, stats.Failed)
	assert.Equal(t, now.Truncate(time.Second), *stats.Oldest)
	assert.Equal(t, []TagListFailure{{Kind: VersionKindImage, Repository: "redis", Error: "denied"}}, stats.Failures)
}

func TestRenovateRuns(t *testing.T) {
	ctx := context.Background()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	defer s.Close()

	now := time.Now().UTC().Truncate(time.Second)
	for i := 0; i < renovateRunsKept+2; i++ {
		run := &RenovateRun{StartedAt: now, Trigger: RenovateTriggerSchedule, Repos: []string{"a/b"}, Status: RenovateStatusRunning}
		require.NoError(t, s.StartRenovateRun(ctx, run))
		require.NoError(t, s.SetRenovateRunJob(ctx, run.ID, "imginq-renovate-x"))
		run.FinishedAt, run.Status, run.Log = &now, RenovateStatusSucceeded, "done"
		require.NoError(t, s.FinishRenovateRun(ctx, run))
	}
	running := &RenovateRun{StartedAt: now, Trigger: RenovateTriggerManual, Repos: []string{"a/b", "c/d"}, Status: RenovateStatusRunning}
	require.NoError(t, s.StartRenovateRun(ctx, running))

	runs, err := s.ListRenovateRuns(ctx, 100)
	require.NoError(t, err)
	assert.Len(t, runs, renovateRunsKept+1, "pruned when a run finishes")
	assert.Equal(t, RenovateStatusRunning, runs[0].Status)
	assert.Nil(t, runs[0].FinishedAt)
	assert.Equal(t, &RenovateRun{
		ID: runs[1].ID, StartedAt: now, FinishedAt: &now, Trigger: RenovateTriggerSchedule,
		Repos: []string{"a/b"}, Job: "imginq-renovate-x", Status: RenovateStatusSucceeded, Log: "done",
	}, runs[1])

	require.NoError(t, s.InterruptRenovateRuns(ctx, now))
	runs, err = s.ListRenovateRuns(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, RenovateStatusInterrupted, runs[0].Status)
	assert.Equal(t, []string{"a/b", "c/d"}, runs[0].Repos)
}
