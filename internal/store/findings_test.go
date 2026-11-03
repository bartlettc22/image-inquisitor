package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/scan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReconcileFindings(t *testing.T) {
	ctx := context.Background()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	defer s.Close()

	web := Slot{Cluster: "home", Namespace: "apps", WorkloadKind: "Deployment", WorkloadName: "web", Container: "web"}
	db := Slot{Cluster: "home", Namespace: "apps", WorkloadKind: "StatefulSet", WorkloadName: "db", Container: "db"}
	job := Slot{Cluster: "home", Namespace: "apps", WorkloadKind: "CronJob", WorkloadName: "backup", Container: "backup"}
	vuln := func(id, fixed string) scan.Vulnerability {
		return scan.Vulnerability{ID: id, Package: "openssl", Fixed: fixed, Severity: scan.SeverityHigh}
	}
	t0 := time.Unix(1_700_000_000, 0).UTC()
	day := func(n int) time.Time { return t0.Add(time.Duration(n) * 24 * time.Hour) }

	reconcile := func(at time.Time, running map[Slot]map[string][]scan.Vulnerability) (int, int) {
		opened, closed, err := s.ReconcileFindings(ctx, "home", running, at)
		require.NoError(t, err)
		return opened, closed
	}
	byID := func() map[string][]Finding {
		findings, err := s.ListFindings(ctx)
		require.NoError(t, err)
		m := map[string][]Finding{}
		for _, f := range findings {
			m[f.WorkloadName+"/"+f.VulnID] = append(m[f.WorkloadName+"/"+f.VulnID], f)
		}
		return m
	}

	opened, closed := reconcile(day(0), map[Slot]map[string][]scan.Vulnerability{
		// Two pods mid-rollout: CVE-1 in both digests is one finding
		web: {"sha256:w1": {vuln("CVE-1", ""), vuln("CVE-2", "")}, "sha256:w1b": {vuln("CVE-1", "")}},
		db:  {"sha256:d1": {vuln("CVE-3", ""), vuln("CVE-4", "")}},
		job: {"sha256:j1": {vuln("CVE-5", "")}},
	})
	assert.Equal(t, [2]int{5, 0}, [2]int{opened, closed})

	opened, closed = reconcile(day(2), map[Slot]map[string][]scan.Vulnerability{
		web: {"sha256:w1": {vuln("CVE-1", "1.1"), vuln("CVE-2", "")}},
		// Same digest, CVE-4 dropped by the database
		db: {"sha256:d1": {vuln("CVE-3", "")}},
		// Not scanned yet: left alone
		job: nil,
	})
	assert.Equal(t, [2]int{0, 1}, [2]int{opened, closed})

	opened, closed = reconcile(day(5), map[Slot]map[string][]scan.Vulnerability{
		// Updated to a digest fixing CVE-1 but still with CVE-2
		web: {"sha256:w2": {vuln("CVE-2", "")}},
		db:  {"sha256:d1": {vuln("CVE-3", "")}},
		// job gone
	})
	assert.Equal(t, [2]int{0, 2}, [2]int{opened, closed})

	f := byID()
	cve1 := f["web/CVE-1"][0]
	assert.Equal(t, day(0), cve1.FirstSeen)
	assert.Equal(t, day(2), cve1.FixAvailable)
	assert.Equal(t, day(2), cve1.LastSeen)
	assert.Equal(t, day(5), cve1.ClosedAt)
	assert.Equal(t, CloseReasonUpdated, cve1.CloseReason)

	cve2 := f["web/CVE-2"][0]
	assert.True(t, cve2.ClosedAt.IsZero(), "still running in the new digest")
	assert.True(t, cve2.FixAvailable.IsZero())
	assert.Equal(t, "sha256:w2", cve2.Digest)
	assert.Equal(t, day(5), cve2.LastSeen)

	assert.Equal(t, CloseReasonWithdrawn, f["db/CVE-4"][0].CloseReason)
	assert.Equal(t, day(2), f["db/CVE-4"][0].ClosedAt)
	assert.True(t, f["db/CVE-3"][0].ClosedAt.IsZero())

	assert.Equal(t, CloseReasonRemoved, f["backup/CVE-5"][0].CloseReason)
	assert.Equal(t, day(5), f["backup/CVE-5"][0].ClosedAt)
	assert.Equal(t, day(0), f["backup/CVE-5"][0].LastSeen, "not seen while unknown")

	// A rollback brings CVE-1 back as a new finding
	opened, closed = reconcile(day(6), map[Slot]map[string][]scan.Vulnerability{
		web: {"sha256:w1": {vuln("CVE-1", "1.1"), vuln("CVE-2", "")}},
		db:  {"sha256:d1": {vuln("CVE-3", "")}},
	})
	assert.Equal(t, [2]int{1, 0}, [2]int{opened, closed})
	f = byID()
	require.Len(t, f["web/CVE-1"], 2)
	assert.Equal(t, day(6), f["web/CVE-1"][1].FirstSeen)
	assert.Equal(t, day(6), f["web/CVE-1"][1].FixAvailable)

	n, err := s.CountOpenFindings(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, n)

	// Other clusters' findings are untouched
	_, closed, err = s.ReconcileFindings(ctx, "work", nil, day(7))
	require.NoError(t, err)
	assert.Zero(t, closed)
}
