package daemon

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/scan"
	"github.com/bartlettc22/image-inquisitor/internal/store"
	"github.com/bartlettc22/image-inquisitor/internal/versions"
	"github.com/google/go-containerregistry/pkg/name"
	"golang.org/x/sync/errgroup"
)

const (
	// releaseDateRetryAfter is how long a failed build date lookup waits before it is retried
	releaseDateRetryAfter = time.Hour

	releaseDateConcurrency = 4
	releaseDateTimeout     = 30 * time.Second

	// severityNone turns off the fast lane or the risky soak
	severityNone = "NONE"
)

// Soak lanes: how long an update to a running image's newest version waits before Renovate
// may propose it
const (
	SoakLaneDefault = "default"
	SoakLaneMajor   = "major"

	// SoakLaneFast soaks briefly: a minor or patch update that fixes more CVEs at least
	// fastLaneSeverity in the running version than it introduces
	SoakLaneFast = "fast"

	// SoakLaneRisky soaks longer: the update introduces more CVEs at least riskySeverity than it fixes
	SoakLaneRisky = "risky"
)

// SoakConfig sets Renovate's soak (minimumReleaseAge). The daemon is the soak clock for the
// newest version of each running image: it dates each from the registry and tells Renovate
// which may be proposed, as Renovate only knows release dates for Docker Hub images. Ages use
// Renovate's format, e.g. "14 days".
type SoakConfig struct {
	// Default is the soak of every update
	Default string `json:"default"`

	// Major is the soak of major updates; defaults to Default
	Major string `json:"major"`

	// FastLaneSeverity: minor and patch updates fixing more CVEs this severe or worse in the
	// running version than they introduce soak for FastLane instead (default CRITICAL; NONE
	// turns it off). Majors keep the major soak.
	FastLaneSeverity string `json:"fastLaneSeverity"`

	// FastLane is the soak of fast-lane updates, so broken releases still get caught (default 3 days)
	FastLane string `json:"fastLane"`

	// RiskySeverity: updates introducing more CVEs this severe or worse than they fix soak for
	// Risky instead (default CRITICAL; NONE turns it off)
	RiskySeverity string `json:"riskySeverity"`

	// Risky is the soak of risky updates; defaults to twice the longer of Default and Major
	Risky string `json:"risky"`

	defaultAge, majorAge, fastAge, riskyAge time.Duration
}

func (s *SoakConfig) validate() error {
	var err error
	if s.Default == "" {
		return errors.New("soak.default is required")
	}
	if s.defaultAge, err = parseAge(s.Default); err != nil {
		return fmt.Errorf("soak.default: %w", err)
	}
	s.majorAge = s.defaultAge
	if s.Major != "" {
		if s.majorAge, err = parseAge(s.Major); err != nil {
			return fmt.Errorf("soak.major: %w", err)
		}
	}
	s.fastAge = 3 * 24 * time.Hour
	if s.FastLane != "" {
		if s.fastAge, err = parseAge(s.FastLane); err != nil {
			return fmt.Errorf("soak.fastLane: %w", err)
		}
	}
	s.riskyAge = 2 * max(s.defaultAge, s.majorAge)
	if s.Risky != "" {
		if s.riskyAge, err = parseAge(s.Risky); err != nil {
			return fmt.Errorf("soak.risky: %w", err)
		}
	}
	// Shown as configured, so fill in the defaults
	s.Major, s.FastLane, s.Risky = renovateAge(s.majorAge), renovateAge(s.fastAge), renovateAge(s.riskyAge)

	for _, sev := range []*string{&s.FastLaneSeverity, &s.RiskySeverity} {
		*sev = strings.ToUpper(*sev)
		if *sev == "" {
			*sev = scan.SeverityCritical
		}
		if *sev != severityNone && severityRank(*sev) < 0 {
			return fmt.Errorf("invalid severity %q: one of CRITICAL, HIGH, MEDIUM, LOW or NONE", *sev)
		}
	}
	return nil
}

var ageRe = regexp.MustCompile(`^(\d+)\s*(minute|hour|day|week|month)s?$`)

// parseAge parses a Renovate age such as "14 days", "3 weeks" or "12 hours", or a Go duration
func parseAge(s string) (time.Duration, error) {
	if m := ageRe.FindStringSubmatch(strings.TrimSpace(strings.ToLower(s))); m != nil {
		n, _ := strconv.Atoi(m[1])
		unit := map[string]time.Duration{
			"minute": time.Minute, "hour": time.Hour, "day": 24 * time.Hour,
			"week": 7 * 24 * time.Hour, "month": 30 * 24 * time.Hour,
		}[m[2]]
		return time.Duration(n) * unit, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid age %q: e.g. \"14 days\", \"12 hours\"", s)
	}
	return d, nil
}

// renovateAge formats a duration as a Renovate age, in days when whole
func renovateAge(d time.Duration) string {
	if d%(24*time.Hour) == 0 {
		return plural(int(d/(24*time.Hour)), "day")
	}
	return plural(int(d.Round(time.Hour)/time.Hour), "hour")
}

func severityRank(sev string) int {
	for i, s := range scan.Severities {
		if s == sev {
			return i
		}
	}
	return -1
}

// atLeast returns how many CVEs of counts are sev or worse
func atLeast(counts scan.Counts, sev string) int {
	if sev == severityNone {
		return 0
	}
	n := 0
	for _, s := range scan.Severities[:severityRank(sev)+1] {
		n += counts[s]
	}
	return n
}

// SoakDecision is how long the update of a running image tag to its newest version soaks
type SoakDecision struct {
	Repository string `json:"repository"`
	Current    string `json:"current"`
	Newest     string `json:"newest"`

	Lane   string `json:"lane"`
	Reason string `json:"reason"`

	// Soak is the soak as a Renovate age; empty in the fast lane
	Soak string `json:"soak,omitempty"`

	// ReleasedAt is the newest version's build date, or when the daemon first saw it
	// (ReleaseSource "build date" or "first seen")
	ReleasedAt    *time.Time `json:"releasedAt,omitempty"`
	ReleaseSource string     `json:"releaseSource,omitempty"`

	// EligibleAt is when the soak ends
	EligibleAt *time.Time `json:"eligibleAt,omitempty"`

	// Released is whether Renovate is told it may propose the update now
	Released bool `json:"released"`

	// ScanPending is set until both versions are scanned; the fast lane needs their CVEs
	ScanPending bool `json:"scanPending,omitempty"`
}

// cveDelta is what an update fixes and introduces; known is false until both sides are scanned
type cveDelta struct {
	fixed, introduced scan.Counts
	known             bool
}

// decideSoak decides the soak of an update from current to newest
func (s *SoakConfig) decideSoak(repository, current, newest string, delta cveDelta, release *store.ReleaseDate, now time.Time) SoakDecision {
	d := SoakDecision{Repository: repository, Current: current, Newest: newest, Lane: SoakLaneDefault}
	soak := s.defaultAge
	if versions.Bump(current, newest) == versions.BumpMajor {
		d.Lane, soak = SoakLaneMajor, s.majorAge
	}
	d.Reason = d.Lane + " soak"

	// Fixed and introduced CVEs are weighed against each other at each lane's severity
	switch {
	case !delta.known:
		d.ScanPending = true
	case atLeast(delta.introduced, s.RiskySeverity) > atLeast(delta.fixed, s.RiskySeverity):
		d.Lane, soak = SoakLaneRisky, max(soak, s.riskyAge)
		d.Reason = cveReason(delta, s.RiskySeverity, false)
	case d.Lane != SoakLaneMajor && atLeast(delta.fixed, s.FastLaneSeverity) > atLeast(delta.introduced, s.FastLaneSeverity):
		d.Lane, soak = SoakLaneFast, s.fastAge
		d.Reason = cveReason(delta, s.FastLaneSeverity, true)
	}
	d.Soak = renovateAge(soak)

	if release != nil {
		released := release.Released()
		eligible := released.Add(soak)
		d.ReleasedAt, d.EligibleAt = &released, &eligible
		d.ReleaseSource = "build date"
		if release.Created.IsZero() {
			d.ReleaseSource = "first seen"
		}
	}
	d.Released = d.EligibleAt != nil && !now.Before(*d.EligibleAt)
	return d
}

// cveReason describes what an update fixes and introduces at sev or worse, leading with the
// side that decided its lane, e.g. "fixes 25 critical CVEs, introduces 1"
func cveReason(delta cveDelta, sev string, fixesFirst bool) string {
	fixed, introduced := atLeast(delta.fixed, sev), atLeast(delta.introduced, sev)
	if fixesFirst {
		reason := "fixes " + cveSummary(delta.fixed, sev) + " in the running version"
		if introduced > 0 {
			reason += fmt.Sprintf(", introduces %d", introduced)
		}
		return reason
	}
	reason := "introduces " + cveSummary(delta.introduced, sev)
	if fixed > 0 {
		reason += fmt.Sprintf(", fixes %d", fixed)
	}
	return reason
}

// cveSummary describes the CVEs of counts that are sev or worse, e.g. "2 critical, 5 high CVEs"
func cveSummary(counts scan.Counts, sev string) string {
	parts := []string{}
	for _, s := range scan.Severities[:severityRank(sev)+1] {
		if counts[s] > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", counts[s], strings.ToLower(s)))
		}
	}
	noun := " CVEs"
	if atLeast(counts, sev) == 1 {
		noun = " CVE"
	}
	return strings.Join(parts, ", ") + noun
}

// renovateRules are the packageRules that carry the soak decisions to Renovate, matching the
// exact newest version: released updates get no soak, risky ones the risky soak (so Renovate's
// own Docker Hub dates don't release them sooner). Others keep Renovate's soak, which holds
// images from registries Renovate has no release dates for.
func (s *SoakConfig) renovateRules(decisions []SoakDecision) []map[string]any {
	rules := []map[string]any{}
	if s.majorAge != s.defaultAge {
		rules = append(rules, map[string]any{"matchUpdateTypes": []string{"major"}, "minimumReleaseAge": s.Major})
	}
	for _, d := range decisions {
		rule := map[string]any{
			"matchDatasources":  []string{"docker"},
			"matchPackageNames": []string{renovatePackageName(d.Repository)},
			"matchNewValue":     "/^" + regexp.QuoteMeta(d.Newest) + "$/",
		}
		switch {
		case d.Released:
			rule["minimumReleaseAge"] = nil
		case d.Lane == SoakLaneRisky:
			rule["minimumReleaseAge"] = d.Soak
		default:
			continue
		}
		rules = append(rules, rule)
	}
	return rules
}

// renovatePackageName is the name Renovate's docker datasource gives a repository:
// Docker Hub images are docker.io/library/nginx rather than index.docker.io/library/nginx
func renovatePackageName(repository string) string {
	if path, ok := strings.CutPrefix(repository, name.DefaultRegistry+"/"); ok {
		return "docker.io/" + path
	}
	return repository
}

// soakState holds the last soak decisions for the API and Renovate runs
type soakState struct {
	mu        sync.Mutex
	decisions []SoakDecision
}

// SoakDecisions returns the last soak decisions; nil when no soak is configured
func (d *Daemon) SoakDecisions() []SoakDecision {
	d.soak.mu.Lock()
	defer d.soak.mu.Unlock()
	return d.soak.decisions
}

// decideSoaks dates the newest version of each running image tag that has a newer one,
// decides each update's soak from the CVEs it fixes and introduces, and keeps the decisions
func (d *Daemon) decideSoaks(ctx context.Context) (string, error) {
	cfg := d.config.Renovate.Soak
	vers, err := d.store.ListVersions(ctx)
	if err != nil {
		return "", err
	}
	containers, err := d.store.ListContainers(ctx)
	if err != nil {
		return "", err
	}

	// The running ref of each repository and tag, as the scan loop scans it
	running := map[string]string{}
	for _, c := range containers {
		if c.Digest == "" {
			continue
		}
		base, _, _ := strings.Cut(c.Image, "@")
		r, err := name.ParseReference(base, name.WeakValidation)
		if err != nil {
			continue
		}
		if t, ok := r.(name.Tag); ok {
			key := r.Context().Name() + ":" + t.TagStr()
			if _, seen := running[key]; !seen {
				running[key] = scan.RunningRef(c.Image, c.Digest)
			}
		}
	}

	updates := []store.Version{}
	for _, v := range vers {
		if v.Kind == store.VersionKindImage && v.Error == "" && v.Latest != "" && v.Latest != v.Current {
			updates = append(updates, v)
		}
	}

	dates, err := d.releaseDates(ctx, updates)
	if err != nil {
		return "", err
	}

	now := time.Now().UTC()
	decisions := []SoakDecision{}
	lanes := map[string]int{}
	released := 0
	for _, v := range updates {
		newest := scan.TagRef(v.Repository, v.Latest)
		delta := cveDelta{}
		if from, ok := running[v.Repository+":"+v.Current]; ok {
			if delta, err = d.cveDelta(ctx, from, newest); err != nil {
				return "", err
			}
		}
		dec := cfg.decideSoak(v.Repository, v.Current, v.Latest, delta, dates[newest], now)
		decisions = append(decisions, dec)
		lanes[dec.Lane]++
		if dec.Released {
			released++
		}
	}

	d.soak.mu.Lock()
	d.soak.decisions = decisions
	d.soak.mu.Unlock()

	return fmt.Sprintf("%s: %d released, %d fast lane, %d risky", plural(len(decisions), "update"), released, lanes[SoakLaneFast], lanes[SoakLaneRisky]), nil
}

// releaseDates returns the release dates of the updates' newest versions by ref, looking up
// the build dates not yet known (or failed more than releaseDateRetryAfter ago)
func (d *Daemon) releaseDates(ctx context.Context, updates []store.Version) (map[string]*store.ReleaseDate, error) {
	dates, err := d.store.ListReleaseDates(ctx)
	if err != nil {
		return nil, err
	}

	mu := sync.Mutex{}
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(releaseDateConcurrency)
	for _, v := range updates {
		ref := scan.TagRef(v.Repository, v.Latest)
		known := dates[ref]
		if known != nil && (known.Error == "" || time.Since(known.CheckedAt) < releaseDateRetryAfter) {
			continue
		}
		g.Go(func() error {
			now := time.Now().UTC()
			rd := &store.ReleaseDate{Ref: ref, FirstSeen: now, CheckedAt: now}
			if known != nil {
				rd.FirstSeen = known.FirstSeen
			}
			lookupCtx, cancel := context.WithTimeout(gctx, releaseDateTimeout)
			defer cancel()
			created, lookupErr := d.imageCreated(lookupCtx, ref)
			if lookupErr != nil {
				svcLog.WithField("ref", ref).WithError(lookupErr).Warn("error reading image build date")
				rd.Error = lookupErr.Error()
			}
			rd.Created = created
			if err := d.store.SaveReleaseDate(gctx, rd); err != nil {
				return err
			}
			mu.Lock()
			dates[ref] = rd
			mu.Unlock()
			return nil
		})
	}
	return dates, g.Wait()
}

// cveDelta compares the stored scans of two references
func (d *Daemon) cveDelta(ctx context.Context, from, to string) (cveDelta, error) {
	vulns := func(ref string) ([]scan.Vulnerability, bool, error) {
		res, err := d.store.GetResolution(ctx, ref)
		if errors.Is(err, store.ErrNotFound) || (err == nil && res.Error != "") {
			return nil, false, nil
		}
		if err != nil {
			return nil, false, err
		}
		sc, err := d.store.GetScan(ctx, res.Digest)
		if errors.Is(err, store.ErrNotFound) || (err == nil && (sc.Error != "" || sc.Skipped != "" || sc.ScannedAt.IsZero())) {
			return nil, false, nil
		}
		if err != nil {
			return nil, false, err
		}
		v, err := d.store.GetScanVulnerabilities(ctx, res.Digest)
		return v, err == nil, err
	}

	fromVulns, ok, err := vulns(from)
	if err != nil || !ok {
		return cveDelta{}, err
	}
	toVulns, ok, err := vulns(to)
	if err != nil || !ok {
		return cveDelta{}, err
	}
	fixed, introduced := scan.Diff(fromVulns, toVulns)
	return cveDelta{fixed: scan.CountBySeverity(fixed), introduced: scan.CountBySeverity(introduced), known: true}, nil
}
