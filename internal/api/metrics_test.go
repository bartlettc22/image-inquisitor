package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/daemon"
	"github.com/bartlettc22/image-inquisitor/internal/scan"
	"github.com/bartlettc22/image-inquisitor/internal/store"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSyncer struct{}

func (fakeSyncer) Trigger()                                 {}
func (fakeSyncer) RunRenovate(daemon.RenovateRequest) error { return nil }
func (fakeSyncer) Repos() []daemon.Repo                     { return []daemon.Repo{{Name: "org/quiet"}} }
func (fakeSyncer) SoakDecisions() []daemon.SoakDecision     { return nil }
func (fakeSyncer) RepoPatterns() []daemon.RepoPattern       { return []daemon.RepoPattern{} }
func (fakeSyncer) Activity() daemon.Activity                { return daemon.Activity{} }
func (fakeSyncer) ActivityReport(context.Context) (*daemon.ActivityReport, error) {
	return &daemon.ActivityReport{ScanQueue: &daemon.ScanQueue{Resolve: 1, Generate: 2, Check: 3}}, nil
}

func TestMetrics(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	defer s.Close()

	slot := store.Slot{Cluster: "home", Namespace: "apps", WorkloadKind: "Deployment", WorkloadName: "web", Container: "web"}
	vuln := func(id, sev, fixed string) scan.Vulnerability {
		return scan.Vulnerability{ID: id, Package: "openssl", Severity: sev, Fixed: fixed}
	}
	start := time.Now().UTC().Add(-20 * day)
	_, _, err = s.ReconcileFindings(ctx, "home", map[store.Slot]map[string][]scan.Vulnerability{
		slot: {"sha256:a": {vuln("CVE-1", scan.SeverityCritical, "1.1"), vuln("CVE-2", scan.SeverityHigh, ""), vuln("CVE-3", scan.SeverityCritical, "2.0")}},
	}, start)
	require.NoError(t, err)
	// Two days later the image is updated, fixing CVE-1; CVE-3 has been fixable for 20 days, past the 7-day SLA
	_, _, err = s.ReconcileFindings(ctx, "home", map[store.Slot]map[string][]scan.Vulnerability{
		slot: {"sha256:b": {vuln("CVE-2", scan.SeverityHigh, ""), vuln("CVE-3", scan.SeverityCritical, "2.0")}},
	}, start.Add(2*day))
	require.NoError(t, err)

	require.NoError(t, s.UpsertPullRequest(ctx, &store.PullRequest{
		Repo: "org/deploy", Number: 1, Title: "Update", URL: "https://example.com", Branch: "renovate/x", HeadSHA: "abc",
		Author: "bot", State: store.PullRequestStateOpen, CreatedAt: start, UpdatedAt: start, SyncedAt: start,
	}))

	srv := httptest.NewServer(NewHandler(s, fakeSyncer{}, &Config{}))
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	for _, line := range []string{
		`imginq_open_findings{fixable="true",severity="CRITICAL"} 1`,
		`imginq_open_findings{fixable="false",severity="HIGH"} 1`,
		`imginq_open_findings{fixable="false",severity="LOW"} 0`,
		`imginq_findings_past_sla{severity="CRITICAL"} 1`,
		`imginq_findings_past_sla{severity="HIGH"} 0`,
		`imginq_sla_seconds{severity="CRITICAL"} 604800`,
		`imginq_findings_fixed_total{severity="CRITICAL"} 1`,
		`imginq_time_to_fix_seconds_bucket{severity="CRITICAL",le="86400"} 0`,
		`imginq_time_to_fix_seconds_bucket{severity="CRITICAL",le="259200"} 1`,
		`imginq_time_to_fix_seconds_count{severity="CRITICAL"} 1`,
		`imginq_time_to_fix_seconds_sum{severity="CRITICAL"} 172800`,
		`imginq_open_prs{repo="org/deploy"} 1`,
		`imginq_open_prs{repo="org/quiet"} 0`,
		`imginq_scan_queue{stage="generate"} 2`,
		`go_goroutines`,
	} {
		assert.Contains(t, string(body), line)
	}
	assert.NotContains(t, string(body), "imginq_running_images", "no clusters configured")

	problems, err := testutil.CollectAndLint(&collector{h: &handler{store: s, syncer: fakeSyncer{}}, ctx: ctx})
	require.NoError(t, err)
	assert.Empty(t, problems)
	assert.NotContains(t, string(body), "imginq_last_sync_timestamp_seconds", "no sync has finished")
}
