package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func historyStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	return s
}

func entry(container, image, digest string, started, ended time.Time) HistoryEntry {
	return HistoryEntry{
		Cluster: "home", Namespace: "web", WorkloadKind: "Deployment", WorkloadName: "app", Pod: "app-1", PodUID: "uid-1",
		Container: container, Image: image, Repository: "example.com/app", Digest: digest, StartedAt: started, EndedAt: ended,
	}
}

func list(t *testing.T, s *Store) []HistoryEntry {
	t.Helper()
	entries, err := s.ListHistory(context.Background(), HistoryQuery{})
	require.NoError(t, err)
	return entries
}

func TestObservePodLifecycle(t *testing.T) {
	ctx := context.Background()
	s := historyStore(t)
	base := time.Unix(1_700_000_000, 0).UTC()
	none := time.Time{}

	// Pending: the image is known, the digest and start aren't
	pending := entry("app", "example.com/app:1", "", none, none)
	require.NoError(t, s.ObservePod(ctx, []HistoryEntry{pending}, base))
	got := list(t, s)
	require.Len(t, got, 1)
	assert.False(t, got[0].Ran)
	assert.Equal(t, base, got[0].StartedAt, "first seen until Kubernetes reports a start")

	// Started earlier than the daemon noticed
	realStart := base.Add(-5 * time.Second)
	running := entry("app", "example.com/app:1", "sha256:a", realStart, none)
	running.Node = "node-1"
	require.NoError(t, s.ObservePod(ctx, []HistoryEntry{running}, base.Add(time.Minute)))
	got = list(t, s)
	require.Len(t, got, 1)
	assert.True(t, got[0].Ran)
	assert.Equal(t, "sha256:a", got[0].Digest)
	assert.Equal(t, "node-1", got[0].Node)
	assert.Equal(t, realStart, got[0].StartedAt)
	assert.Equal(t, base, got[0].RecordedAt)

	// A restart of the same digest is the same entry, keeping the first start
	restarted := entry("app", "example.com/app:1", "sha256:a", base.Add(time.Hour), none)
	require.NoError(t, s.ObservePod(ctx, []HistoryEntry{restarted}, base.Add(time.Hour)))
	got = list(t, s)
	require.Len(t, got, 1)
	assert.Equal(t, realStart, got[0].StartedAt)

	// The same tag re-pulled as another digest on restart is a new entry
	swapped := base.Add(2 * time.Hour)
	require.NoError(t, s.ObservePod(ctx, []HistoryEntry{entry("app", "example.com/app:1", "sha256:b", swapped, none)}, swapped))
	got = list(t, s)
	require.Len(t, got, 2)
	assert.Equal(t, "sha256:b", got[0].Digest)
	assert.True(t, got[0].EndedAt.IsZero())
	assert.Equal(t, "sha256:a", got[1].Digest)
	assert.Equal(t, swapped, got[1].EndedAt)
	assert.Equal(t, EndReasonReplaced, got[1].EndReason)

	// Finishing ends it, and later updates of the finished pod change nothing
	finished := base.Add(3 * time.Hour)
	done := entry("app", "example.com/app:1", "sha256:b", base.Add(150*time.Minute), finished)
	require.NoError(t, s.ObservePod(ctx, []HistoryEntry{done}, finished))
	require.NoError(t, s.ObservePod(ctx, []HistoryEntry{done}, finished.Add(time.Minute)))
	got = list(t, s)
	require.Len(t, got, 2)
	assert.Equal(t, finished, got[0].EndedAt)
	assert.Equal(t, EndReasonStopped, got[0].EndReason)
	assert.Equal(t, swapped, got[0].StartedAt)
}

func TestObservePodFirstSeenFinished(t *testing.T) {
	ctx := context.Background()
	s := historyStore(t)
	at := time.Unix(1_700_000_000, 0).UTC()
	started, finished := at.Add(-time.Hour), at.Add(-time.Minute)

	// A pod that came and went before the first look keeps its real times
	done := entry("job", "example.com/job:1", "sha256:j", started, finished)
	require.NoError(t, s.ObservePod(ctx, []HistoryEntry{done}, at))
	require.NoError(t, s.ObservePod(ctx, []HistoryEntry{done}, at.Add(time.Second)))
	got := list(t, s)
	require.Len(t, got, 1)
	assert.Equal(t, started, got[0].StartedAt)
	assert.Equal(t, finished, got[0].EndedAt)
	assert.Equal(t, at, got[0].RecordedAt)
}

func TestInitContainerRetry(t *testing.T) {
	ctx := context.Background()
	s := historyStore(t)
	at := time.Unix(1_700_000_000, 0).UTC()
	first := entry("init", "example.com/init:1", "sha256:i", at, at.Add(time.Second))
	second := entry("init", "example.com/init:1", "sha256:i", at.Add(10*time.Second), time.Time{})

	require.NoError(t, s.ObservePod(ctx, []HistoryEntry{first}, at))
	require.NoError(t, s.ObservePod(ctx, []HistoryEntry{second}, at.Add(10*time.Second)))
	require.NoError(t, s.ObservePod(ctx, []HistoryEntry{second}, at.Add(11*time.Second)))
	assert.Len(t, list(t, s), 2, "each run of a retried init container, each once")
}

func TestEndPod(t *testing.T) {
	ctx := context.Background()
	s := historyStore(t)
	at := time.Unix(1_700_000_000, 0).UTC()

	require.NoError(t, s.ObservePod(ctx, []HistoryEntry{
		entry("ran", "example.com/app:1", "sha256:a", at, time.Time{}),
		entry("stuck", "example.com/bad:1", "", time.Time{}, time.Time{}),
	}, at))
	require.NoError(t, s.EndPod(ctx, "home", "uid-1", at.Add(time.Minute), EndReasonDeleted))

	reasons := map[string]string{}
	for _, e := range list(t, s) {
		reasons[e.Container] = e.EndReason
		assert.Equal(t, at.Add(time.Minute), e.EndedAt)
	}
	assert.Equal(t, map[string]string{"ran": EndReasonDeleted, "stuck": EndReasonNeverStarted}, reasons)

	// The stuck container stays ended however late another update arrives
	require.NoError(t, s.ObservePod(ctx, []HistoryEntry{entry("stuck", "example.com/bad:1", "", time.Time{}, time.Time{})}, at.Add(time.Hour)))
	assert.Len(t, list(t, s), 2)
}

func TestEndMissingPods(t *testing.T) {
	ctx := context.Background()
	s := historyStore(t)
	at := time.Unix(1_700_000_000, 0).UTC()

	gone, alive, fresh := entry("c", "i:1", "sha256:a", at, time.Time{}), entry("c", "i:1", "sha256:a", at, time.Time{}), entry("c", "i:1", "sha256:a", at, time.Time{})
	gone.PodUID, alive.PodUID, fresh.PodUID = "gone", "alive", "fresh"
	require.NoError(t, s.ObservePod(ctx, []HistoryEntry{gone, alive}, at))
	listedAt := at.Add(time.Hour)
	require.NoError(t, s.ObservePod(ctx, []HistoryEntry{fresh}, listedAt.Add(time.Second)))

	n, err := s.EndMissingPods(ctx, "home", map[string]bool{"alive": true}, listedAt, listedAt)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	for _, e := range list(t, s) {
		if e.PodUID == "gone" {
			assert.Equal(t, EndReasonUnobserved, e.EndReason)
		} else {
			assert.True(t, e.EndedAt.IsZero(), e.PodUID)
		}
	}
}

func TestListHistory(t *testing.T) {
	ctx := context.Background()
	s := historyStore(t)
	at := time.Unix(1_700_000_000, 0).UTC()

	old := entry("c", "example.com/old:1", "sha256:0ld", at, at.Add(time.Hour))
	old.PodUID, old.Repository = "old", "example.com/old"
	now := entry("c", "example.com/app:2", "sha256:n3w", at.Add(2*time.Hour), time.Time{})
	now.PodUID, now.Pod = "new", "App-XYZ"
	require.NoError(t, s.ObservePod(ctx, []HistoryEntry{old, now}, at))

	query := func(q HistoryQuery) []string {
		entries, err := s.ListHistory(ctx, q)
		require.NoError(t, err)
		uids := []string{}
		for _, e := range entries {
			uids = append(uids, e.PodUID)
		}
		return uids
	}
	assert.Equal(t, []string{"new", "old"}, query(HistoryQuery{}), "newest first")
	assert.Equal(t, []string{"old"}, query(HistoryQuery{Repository: "example.com/old"}))
	assert.Equal(t, []string{"old"}, query(HistoryQuery{Text: "0ld"}), "by digest")
	assert.Equal(t, []string{"new"}, query(HistoryQuery{Text: "app-xyz"}), "any case")
	assert.Empty(t, query(HistoryQuery{Text: "%"}), "text is literal")
	assert.Equal(t, []string{"old"}, query(HistoryQuery{At: at.Add(30 * time.Minute)}))
	assert.Empty(t, query(HistoryQuery{At: at.Add(90 * time.Minute)}), "between the two")
	assert.Equal(t, []string{"new"}, query(HistoryQuery{At: at.Add(24 * time.Hour)}), "open entries are present until now")
	assert.Equal(t, []string{"new"}, query(HistoryQuery{Limit: 1}))
}

func TestCoverage(t *testing.T) {
	ctx := context.Background()
	s := historyStore(t)
	at := time.Unix(1_700_000_000, 0).UTC()

	id, err := s.StartCoverage(ctx, "home", at)
	require.NoError(t, err)
	require.NoError(t, s.TouchCoverage(ctx, id, at.Add(time.Minute)))

	coverage, err := s.ListCoverage(ctx)
	require.NoError(t, err)
	assert.Equal(t, []HistoryCoverage{{Cluster: "home", StartedAt: at, LastSeen: at.Add(time.Minute)}}, coverage)
}
