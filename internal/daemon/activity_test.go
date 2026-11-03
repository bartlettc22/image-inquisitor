package daemon

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/prdiff"
	"github.com/bartlettc22/image-inquisitor/internal/scan"
	"github.com/bartlettc22/image-inquisitor/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanQueue(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := store.Open(filepath.Join(dir, "imginq.db"))
	require.NoError(t, err)
	defer s.Close()
	d := &Daemon{config: &Config{DataDir: dir}, store: s, scanner: &scan.Scanner{Dir: filepath.Join(dir, "scan")}}

	now := time.Now().UTC()
	require.NoError(t, s.UpsertPullRequest(ctx, &store.PullRequest{Repo: "a/b", Number: 1, State: store.PullRequestStateOpen}))
	require.NoError(t, s.ReplaceChanges(ctx, "a/b", 1, []prdiff.Change{
		{Kind: prdiff.KindImage, From: "unresolved:1", To: "unscanned:1"},
		{Kind: prdiff.KindImage, From: "stale:1", To: "unresolvable:1"},
		{Kind: prdiff.KindImage, From: "windows:1", To: "current:1"},
	}))

	sbom := func(digest string) {
		require.NoError(t, os.MkdirAll(filepath.Dir(d.scanner.SBOMPath(digest)), 0o755))
		require.NoError(t, os.WriteFile(d.scanner.SBOMPath(digest), []byte("{}"), 0o644))
	}
	resolve := func(ref, digest, errStr string) {
		require.NoError(t, s.SaveResolution(ctx, &store.Resolution{Ref: ref, Repository: ref, Digest: digest, Error: errStr, ResolvedAt: now}))
	}
	resolve("unscanned:1", "sha256:1", "")
	resolve("stale:1", "sha256:2", "")
	resolve("unresolvable:1", "", "denied")
	resolve("windows:1", "sha256:3", "")
	resolve("current:1", "sha256:4", "")
	sbom("sha256:2")
	sbom("sha256:4")
	require.NoError(t, s.SaveScan(ctx, &store.Scan{Digest: "sha256:2", Counts: scan.Counts{}, SBOMAt: now, ScannedAt: now.Add(-25 * time.Hour)}))
	require.NoError(t, s.SaveScan(ctx, &store.Scan{Digest: "sha256:3", Counts: scan.Counts{}, Skipped: "no linux/amd64 image", ScannedAt: now}))
	require.NoError(t, s.SaveScan(ctx, &store.Scan{Digest: "sha256:4", Counts: scan.Counts{}, SBOMAt: now, ScannedAt: now}))

	q, err := d.scanQueue(ctx)
	require.NoError(t, err)
	assert.Equal(t, &ScanQueue{Images: 6, PR: 6, Resolve: 1, Generate: 1, Check: 1, Failed: 1, Skipped: 1}, q)
}

func TestCaches(t *testing.T) {
	dir := t.TempDir()
	write := func(path string, size int) {
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, make([]byte, size), 0o644))
	}
	write(filepath.Join(dir, "imginq.db"), 100)
	write(filepath.Join(dir, "imginq.db-wal"), 10)
	write(filepath.Join(dir, "scan", "trivy", "db", "trivy.db"), 1000)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "scan", "trivy", "db", "metadata.json"),
		[]byte(`{"Version":2,"NextUpdate":"2026-11-03T13:00:00Z","UpdatedAt":"2026-11-02T13:00:00Z","DownloadedAt":"2026-11-02T18:00:00Z"}`), 0o644))
	write(filepath.Join(dir, "scan", "sbom", "sha256-1.cdx.json"), 5)
	write(filepath.Join(dir, "scan", "sbom", "sha256-2.cdx.json"), 5)

	d := &Daemon{config: &Config{DataDir: dir}}
	caches := d.caches()

	names := []string{}
	for _, c := range caches {
		names = append(names, c.Name)
	}
	assert.Equal(t, []string{"Trivy vulnerability database", "SBOMs", "Database", "Data directory total"}, names, "missing caches are left out")

	db := caches[0]
	assert.Equal(t, 2, db.Files)
	require.NotNil(t, db.UpdatedAt)
	assert.Equal(t, time.Date(2026, 11, 2, 13, 0, 0, 0, time.UTC), *db.UpdatedAt)
	assert.Equal(t, time.Date(2026, 11, 3, 13, 0, 0, 0, time.UTC), *db.NextUpdateAt)
	assert.Equal(t, Cache{Name: "SBOMs", Bytes: 10, Files: 2}, caches[1])
	assert.Equal(t, Cache{Name: "Database", Bytes: 110, Files: 2}, caches[2])

	// Reused until cacheSizesMaxAge
	write(filepath.Join(dir, "scan", "sbom", "sha256-3.cdx.json"), 5)
	assert.Equal(t, 2, d.caches()[1].Files)
}

func TestPlural(t *testing.T) {
	assert.Equal(t, "1 PR", plural(1, "PR"))
	assert.Equal(t, "0 PRs", plural(0, "PR"))
	assert.Equal(t, "2 repositories", plural(2, "repository"))
	assert.Equal(t, "14 days", plural(14, "day"))
}
