package daemon

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/scan"
	"github.com/bartlettc22/image-inquisitor/internal/store"
	"github.com/bartlettc22/image-inquisitor/internal/workloads"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAge(t *testing.T) {
	for in, want := range map[string]time.Duration{
		"14 days": 14 * 24 * time.Hour, "1 day": 24 * time.Hour, "3 weeks": 21 * 24 * time.Hour,
		"12 hours": 12 * time.Hour, "30 minutes": 30 * time.Minute, "36h": 36 * time.Hour,
	} {
		got, err := parseAge(in)
		require.NoError(t, err, in)
		assert.Equal(t, want, got, in)
	}
	_, err := parseAge("two weeks")
	assert.Error(t, err)

	assert.Equal(t, "14 days", renovateAge(14*24*time.Hour))
	assert.Equal(t, "1 day", renovateAge(24*time.Hour))
	assert.Equal(t, "36 hours", renovateAge(36*time.Hour))
}

func soakConfig(t *testing.T, s SoakConfig) *SoakConfig {
	t.Helper()
	require.NoError(t, s.validate())
	return &s
}

func TestSoakConfigValidate(t *testing.T) {
	s := soakConfig(t, SoakConfig{Default: "14 days", Major: "21 days"})
	assert.Equal(t, scan.SeverityCritical, s.FastLaneSeverity)
	assert.Equal(t, scan.SeverityCritical, s.RiskySeverity)
	assert.Equal(t, 42*24*time.Hour, s.riskyAge, "twice the longer soak")
	assert.Equal(t, 3*24*time.Hour, s.fastAge)
	assert.Equal(t, "42 days", s.Risky, "defaults are filled in to be shown")

	assert.ErrorContains(t, (&SoakConfig{}).validate(), "soak.default is required")
	assert.ErrorContains(t, (&SoakConfig{Default: "14 days", FastLaneSeverity: "severe"}).validate(), "invalid severity")
	assert.NoError(t, (&SoakConfig{Default: "14 days", FastLaneSeverity: "none", RiskySeverity: "high"}).validate())
}

func TestDecideSoak(t *testing.T) {
	s := soakConfig(t, SoakConfig{Default: "14 days", Major: "21 days", Risky: "30 days", FastLaneSeverity: "HIGH"})
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	built := func(daysAgo int) *store.ReleaseDate {
		return &store.ReleaseDate{Created: now.AddDate(0, 0, -daysAgo), FirstSeen: now}
	}
	known := func(fixed, introduced scan.Counts) cveDelta {
		return cveDelta{fixed: fixed, introduced: introduced, known: true}
	}
	none := scan.Counts{}

	tests := []struct {
		name               string
		current, newest    string
		delta              cveDelta
		release            *store.ReleaseDate
		lane, soak, reason string
		released           bool
	}{
		{"soaking", "1.27.0", "1.29.3", known(none, none), built(3), SoakLaneDefault, "14 days", "default soak", false},
		{"soaked", "1.27.0", "1.29.3", known(none, none), built(15), SoakLaneDefault, "14 days", "default soak", true},
		{"major soaking", "1.27.0", "2.0.0", known(none, none), built(15), SoakLaneMajor, "21 days", "major soak", false},
		{"fast lane soaking", "1.27.0", "1.29.3", known(scan.Counts{"HIGH": 2, "LOW": 3}, none), built(1), SoakLaneFast, "3 days", "fixes 2 high CVEs in the running version", false},
		{"fast lane soaked", "1.27.0", "1.29.3", known(scan.Counts{"HIGH": 2}, none), built(4), SoakLaneFast, "3 days", "fixes 2 high CVEs in the running version", true},
		{"major keeps the major soak", "1.27.0", "2.0.0", known(scan.Counts{"CRITICAL": 3}, none), built(4), SoakLaneMajor, "21 days", "major soak", false},
		{"fixes more than it introduces", "1.27.0", "1.29.3", known(scan.Counts{"CRITICAL": 25}, scan.Counts{"CRITICAL": 1}), built(4), SoakLaneFast, "3 days", "fixes 25 critical CVEs in the running version, introduces 1", true},
		{"introduces more than it fixes", "1.27.0", "1.29.3", known(scan.Counts{"CRITICAL": 1}, scan.Counts{"CRITICAL": 14}), built(15), SoakLaneRisky, "30 days", "introduces 14 critical CVEs, fixes 1", false},
		{"risky soaked", "1.27.0", "1.29.3", known(none, scan.Counts{"CRITICAL": 1}), built(31), SoakLaneRisky, "30 days", "introduces 1 critical CVE", true},
		{"risky major", "1.27.0", "2.0.0", known(none, scan.Counts{"CRITICAL": 1}), built(31), SoakLaneRisky, "30 days", "introduces 1 critical CVE", true},
		{"as many fixed as introduced", "1.27.0", "1.29.3", known(scan.Counts{"HIGH": 1}, scan.Counts{"HIGH": 1}), built(3), SoakLaneDefault, "14 days", "default soak", false},
		{"scan pending", "1.27.0", "1.29.3", cveDelta{}, built(3), SoakLaneDefault, "14 days", "default soak", false},
		{"no release date", "1.27.0", "1.29.3", known(none, none), nil, SoakLaneDefault, "14 days", "default soak", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := s.decideSoak("ghcr.io/example/app", tt.current, tt.newest, tt.delta, tt.release, now)
			assert.Equal(t, tt.lane, d.Lane)
			assert.Equal(t, tt.soak, d.Soak)
			assert.Equal(t, tt.reason, d.Reason)
			assert.Equal(t, tt.released, d.Released)
		})
	}

	// Reproducible builds are dated by when the tag was first seen
	d := s.decideSoak("ghcr.io/example/app", "1.0", "1.1", known(none, none), &store.ReleaseDate{FirstSeen: now.AddDate(0, 0, -20)}, now)
	assert.Equal(t, "first seen", d.ReleaseSource)
	assert.True(t, d.Released)
}

func TestSoakRenovateRules(t *testing.T) {
	s := soakConfig(t, SoakConfig{Default: "14 days", Major: "21 days", Risky: "30 days"})
	rules := s.renovateRules([]SoakDecision{
		{Repository: "index.docker.io/library/nginx", Newest: "1.29.3", Lane: SoakLaneFast, Released: true},
		{Repository: "ghcr.io/example/app", Newest: "2.0.0-rc.1", Lane: SoakLaneRisky, Soak: "30 days"},
		{Repository: "ghcr.io/example/other", Newest: "3.1", Lane: SoakLaneDefault, Soak: "14 days"},
	})
	b, err := json.Marshal(rules)
	require.NoError(t, err)
	assert.JSONEq(t, `[
		{"matchUpdateTypes": ["major"], "minimumReleaseAge": "21 days"},
		{"matchDatasources": ["docker"], "matchPackageNames": ["docker.io/library/nginx"], "matchNewValue": "/^1\\.29\\.3$/", "minimumReleaseAge": null},
		{"matchDatasources": ["docker"], "matchPackageNames": ["ghcr.io/example/app"], "matchNewValue": "/^2\\.0\\.0-rc\\.1$/", "minimumReleaseAge": "30 days"}
	]`, string(b), "still soaking in the default lane needs no rule")
}

func TestRenovateGlobalConfigWithSoak(t *testing.T) {
	yaml := strings.Replace(renovateConfigYAML, "    minimumReleaseAge: 14 days\n", "", 1) + "  soak:\n    default: 14 days\n    major: 21 days\n"
	c, err := loadConfig(t, yaml)
	require.NoError(t, err)

	global := c.Renovate.globalConfig([]string{"example/deploy"}, []map[string]any{{"matchNewValue": "/^1$/"}})
	assert.Equal(t, "14 days", global["minimumReleaseAge"])
	rules := global["packageRules"].([]any)
	require.Len(t, rules, 2, "configured rules first, then the soak's")
	assert.Equal(t, "/^1$/", rules[1].(map[string]any)["matchNewValue"])

	_, err = loadConfig(t, renovateConfigYAML+"  soak:\n    default: 14 days\n")
	assert.ErrorContains(t, err, "config.minimumReleaseAge is set by the daemon")
}

func TestDecideSoaks(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(filepath.Join(t.TempDir(), "imginq.db"))
	require.NoError(t, err)
	defer s.Close()

	c := &Config{Renovate: &RenovateConfig{Soak: soakConfig(t, SoakConfig{Default: "14 days"})}}
	created := time.Now().UTC().AddDate(0, 0, -20).Truncate(time.Second)
	lookups := 0
	d := &Daemon{config: c, store: s, imageCreated: func(context.Context, string) (time.Time, error) {
		lookups++
		return created, nil
	}}

	require.NoError(t, s.ReplaceContainers(ctx, "home", []workloads.Container{
		{Cluster: "home", Namespace: "web", WorkloadKind: "Deployment", WorkloadName: "nginx", Container: "nginx", Image: "nginx:1.27", Digest: "sha256:old"},
	}, time.Now(), nil))
	require.NoError(t, s.ReplaceVersions(ctx, []store.Version{
		{Kind: store.VersionKindImage, Repository: "index.docker.io/library/nginx", Current: "1.27", Latest: "1.29", CheckedAt: time.Now()},
	}))
	now := time.Now().UTC()
	require.NoError(t, s.SaveResolution(ctx, &store.Resolution{Ref: "nginx:1.27@sha256:old", Digest: "sha256:old", ResolvedAt: now}))
	require.NoError(t, s.SaveResolution(ctx, &store.Resolution{Ref: "index.docker.io/library/nginx:1.29", Digest: "sha256:new", ResolvedAt: now}))
	critical := scan.Vulnerability{ID: "CVE-1", Package: "openssl", Severity: scan.SeverityCritical}
	require.NoError(t, s.SaveScan(ctx, &store.Scan{Digest: "sha256:old", Vulnerabilities: []scan.Vulnerability{critical}, Counts: scan.Counts{"CRITICAL": 1}, SBOMAt: now, ScannedAt: now}))
	require.NoError(t, s.SaveScan(ctx, &store.Scan{Digest: "sha256:new", Vulnerabilities: []scan.Vulnerability{}, Counts: scan.Counts{}, SBOMAt: now, ScannedAt: now}))

	summary, err := d.decideSoaks(ctx)
	require.NoError(t, err)
	assert.Equal(t, "1 update: 1 released, 1 fast lane, 0 risky", summary)
	dec := d.SoakDecisions()[0]
	assert.Equal(t, SoakLaneFast, dec.Lane)
	assert.Equal(t, "fixes 1 critical CVE in the running version", dec.Reason)
	assert.Equal(t, created, *dec.ReleasedAt)
	assert.Equal(t, "build date", dec.ReleaseSource)

	_, err = d.decideSoaks(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, lookups, "build dates are looked up once")
}
