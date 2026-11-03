package api

import (
	"testing"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/prdiff"
	"github.com/bartlettc22/image-inquisitor/internal/render"
	"github.com/bartlettc22/image-inquisitor/internal/scan"
	"github.com/bartlettc22/image-inquisitor/internal/store"
	"github.com/bartlettc22/image-inquisitor/internal/workloads"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttachScans(t *testing.T) {
	baseCVE := scan.Vulnerability{ID: "CVE-base", Package: "openssl", Severity: scan.SeverityCritical}
	appCVE := scan.Vulnerability{ID: "CVE-app", Package: "app", Severity: scan.SeverityHigh}
	newCVE := scan.Vulnerability{ID: "CVE-new", Package: "curl", Severity: scan.SeverityMedium}

	vulns := map[string][]scan.Vulnerability{
		"sha256:web1":  {baseCVE, appCVE},
		"sha256:web2":  {newCVE},
		"sha256:hook1": {baseCVE},
		"sha256:hook2": {},
	}
	now := time.Now().UTC()
	idx := &scanIndex{
		resolutions: map[string]store.Resolution{
			"web:1":     {Ref: "web:1", Digest: "sha256:web1"},
			"web:2":     {Ref: "web:2", Digest: "sha256:web2"},
			"hook:1":    {Ref: "hook:1", Digest: "sha256:hook1"},
			"hook:2":    {Ref: "hook:2", Digest: "sha256:hook2"},
			"private:1": {Ref: "private:1", Error: "unauthorized"},
		},
		scans: map[string]store.Scan{},
		vulns: func(sc *store.Scan) ([]scan.Vulnerability, error) { return vulns[sc.Digest], nil },
	}
	for digest, v := range vulns {
		idx.scans[digest] = store.Scan{Digest: digest, Counts: scan.CountBySeverity(v), ScannedAt: now}
	}

	pr := &store.PullRequest{
		Changes: []prdiff.Change{
			{Kind: prdiff.KindImage, From: "web:1", To: "web:2"},
			{Kind: prdiff.KindHelmChart, Name: "hooks", From: "1.0.0", To: "2.0.0"},
		},
		Stages: []store.Stage{{Name: renderStage, Status: store.StageStatusDone}},
		ChartRenders: []store.ChartRender{{Change: 1, Images: []render.ImageDiff{
			{Kind: "Job", Name: "hook", From: "hook:1", To: "hook:2"},
			{Kind: "Deployment", Name: "sidecar", To: "sidecar:1"},
		}}},
	}
	view := pullRequestView(pr, nil, true)
	require.NoError(t, attachScans(view, idx, true))

	web := view.Changes[0].Scan
	assert.Equal(t, ScanDone, web.Status)
	assert.Equal(t, scan.Counts{scan.SeverityCritical: 1, scan.SeverityHigh: 1}, web.Fixed)
	assert.Equal(t, scan.Counts{scan.SeverityMedium: 1}, web.Introduced)
	assert.Equal(t, []scan.Vulnerability{baseCVE, appCVE}, web.FixedVulnerabilities)

	images := view.Changes[1].Renders[0].Images
	assert.Equal(t, scan.Counts{scan.SeverityCritical: 1}, images[0].Scan.Fixed)
	assert.Equal(t, ScanPending, images[1].Scan.Status, "sidecar:1 not resolved yet")
	assert.Nil(t, images[1].Scan.From, "added container")

	// The base image CVE fixed by both images counts once
	assert.Equal(t, &PullRequestScan{
		Status: ScanPending, Images: 3, Pending: 1,
		Fixed:      scan.Counts{scan.SeverityCritical: 1, scan.SeverityHigh: 1},
		Introduced: scan.Counts{scan.SeverityMedium: 1},
	}, view.Scan)
	require.Len(t, view.Stages, 2)
	assert.Equal(t, "scan", view.Stages[1].Name)
	assert.Equal(t, now, view.Stages[1].UpdatedAt)
	assert.Len(t, pr.Stages, 1, "the stored PR is unchanged")

	// Lists are left out unless asked for; a failed side fails the delta
	failed := pullRequestView(&store.PullRequest{Changes: []prdiff.Change{{Kind: prdiff.KindImage, From: "private:1", To: "web:2"}}}, nil, true)
	require.NoError(t, attachScans(failed, idx, false))
	assert.Equal(t, ScanError, failed.Changes[0].Scan.Status)
	assert.Equal(t, "unauthorized", failed.Changes[0].Scan.From.Error)
	assert.Equal(t, store.StageStatusError, failed.Stages[0].Status)

	ok := pullRequestView(&store.PullRequest{Changes: []prdiff.Change{{Kind: prdiff.KindImage, From: "web:1", To: "web:2"}}}, nil, true)
	require.NoError(t, attachScans(ok, idx, false))
	assert.Nil(t, ok.Changes[0].Scan.FixedVulnerabilities)
}

func TestSkippedScan(t *testing.T) {
	idx := &scanIndex{
		resolutions: map[string]store.Resolution{
			"win:1": {Ref: "win:1", Digest: "sha256:win1"},
			"win:2": {Ref: "win:2", Digest: "sha256:win2"},
		},
		scans: map[string]store.Scan{
			"sha256:win1": {Digest: "sha256:win1", Skipped: "no linux/amd64 image"},
			"sha256:win2": {Digest: "sha256:win2", Skipped: "no linux/amd64 image"},
		},
		vulns: func(*store.Scan) ([]scan.Vulnerability, error) { return nil, nil },
	}
	view := pullRequestView(&store.PullRequest{Changes: []prdiff.Change{{Kind: prdiff.KindImage, From: "win:1", To: "win:2"}}}, nil, true)
	require.NoError(t, attachScans(view, idx, false))
	assert.Equal(t, ScanSkipped, view.Changes[0].Scan.Status)
	assert.Equal(t, ScanDone, view.Scan.Status, "skipped images don't hold the PR up")
	assert.Equal(t, 1, view.Scan.Skipped)
}

func TestAttachRunningScans(t *testing.T) {
	oldCVE := scan.Vulnerability{ID: "CVE-old", Package: "openssl", Severity: scan.SeverityCritical}
	keptCVE := scan.Vulnerability{ID: "CVE-kept", Package: "zlib", Severity: scan.SeverityLow}
	vulns := map[string][]scan.Vulnerability{
		"sha256:running": {oldCVE, keptCVE},
		"sha256:newest":  {keptCVE},
	}
	idx := &scanIndex{
		resolutions: map[string]store.Resolution{
			"nginx:1.27@sha256:running":            {Digest: "sha256:running"},
			"index.docker.io/library/nginx:1.29.3": {Digest: "sha256:newest"},
		},
		scans: map[string]store.Scan{},
		vulns: func(sc *store.Scan) ([]scan.Vulnerability, error) { return vulns[sc.Digest], nil },
	}
	for digest, v := range vulns {
		idx.scans[digest] = store.Scan{Digest: digest, Counts: scan.CountBySeverity(v), ScannedAt: time.Now()}
	}

	containers := parseContainers([]workloads.Container{
		{Cluster: "home", Namespace: "web", WorkloadKind: "Deployment", WorkloadName: "nginx", Container: "nginx", Image: "nginx:1.27", Digest: "sha256:running"},
		{Cluster: "home", Namespace: "jobs", WorkloadKind: "CronJob", WorkloadName: "backup", Container: "backup", Image: "example.com/backup:1.0"},
		{Cluster: "home", Namespace: "db", WorkloadKind: "StatefulSet", WorkloadName: "db", Container: "db", Image: "example.com/db:2", Digest: "sha256:db"},
	}, &attribution{})
	versions := newVersionIndex([]store.Version{
		{Kind: store.VersionKindImage, Repository: "index.docker.io/library/nginx", Current: "1.27", Latest: "1.29.3"},
		{Kind: store.VersionKindImage, Repository: "example.com/db", Current: "2", Latest: "2"},
	})
	images := runningImages(containers, nil, versions)
	require.NoError(t, attachRunningScans(images, idx))

	byRepo := map[string]RunningImage{}
	for _, img := range images {
		byRepo[img.Repository] = img
	}

	nginx := byRepo["index.docker.io/library/nginx"]
	require.NotNil(t, nginx.Scan)
	assert.Equal(t, ScanDone, nginx.Scan.Status)
	assert.Equal(t, scan.Counts{scan.SeverityCritical: 1, scan.SeverityLow: 1}, nginx.Scan.Counts)
	require.NotNil(t, nginx.Update)
	assert.Equal(t, scan.Counts{scan.SeverityCritical: 1}, nginx.Update.Fixed)
	assert.Equal(t, scan.Counts{}, nginx.Update.Introduced)

	db := byRepo["example.com/db"]
	assert.Equal(t, ScanPending, db.Scan.Status, "not scanned yet")
	assert.Nil(t, db.Update, "already the newest")

	assert.Nil(t, byRepo["example.com/backup"].Scan, "no running digest to scan")
}
