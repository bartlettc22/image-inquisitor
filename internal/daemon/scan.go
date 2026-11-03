package daemon

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/prdiff"
	"github.com/bartlettc22/image-inquisitor/internal/scan"
	"github.com/bartlettc22/image-inquisitor/internal/store"
	"github.com/bartlettc22/image-inquisitor/internal/workloads"
)

const (
	// resolveMaxAge is how long a tag's digest is trusted before it is resolved again
	resolveMaxAge = 6 * time.Hour

	// rescanAfter is how often an SBOM is checked against the vulnerability database again
	rescanAfter = 24 * time.Hour

	// scanRetryAfter is how long a failed resolution or SBOM waits before it is retried
	scanRetryAfter = time.Hour

	// scanIdle is how long the scan loop waits when there is nothing to do
	scanIdle = time.Minute
)

// scanLoop resolves and scans the images of open PRs, one image at a time, until ctx is
// cancelled. It runs apart from sync, as pulling an image to scan it can take minutes.
func (d *Daemon) scanLoop(ctx context.Context) {
	for {
		worked, err := d.scanNext(ctx)
		d.setScanTask("", "")
		if err != nil {
			svcLog.WithError(err).Error("error scanning")
		}
		if worked && err == nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(scanIdle):
		case <-d.scanTrigger:
		}
	}
}

func (d *Daemon) triggerScan() {
	select {
	case d.scanTrigger <- struct{}{}:
	default:
	}
}

// scanNext does the next unit of scanning work, if any: resolving a reference to a digest,
// generating a digest's SBOM, or checking an old scan against the current database.
// It reports whether it did anything; failures are recorded, only storage errors returned.
func (d *Daemon) scanNext(ctx context.Context) (bool, error) {
	targets, err := d.scanTargets(ctx)
	if err != nil {
		return false, err
	}

	for _, t := range targets {
		ref := t.ref
		step, res, _, err := d.nextScanStep(ctx, ref)
		if err != nil {
			return false, err
		}
		switch step {
		case scanResolve:
			d.setScanTask("resolving", ref)
			return true, d.resolve(ctx, ref)
		case scanGenerate:
			d.setScanTask("generating SBOM", res.Repository+"@"+res.Digest)
			return true, d.scanDigest(ctx, res, true)
		case scanCheck:
			d.setScanTask("checking SBOM", res.Repository+"@"+res.Digest)
			return true, d.scanDigest(ctx, res, false)
		}
	}
	return false, nil
}

// scanStep is the next unit of scanning work an image reference needs
type scanStep int

const (
	// scanNone: up to date, skipped, or waiting to retry a failure
	scanNone scanStep = iota
	scanResolve
	scanGenerate
	scanCheck
)

// nextScanStep returns what a reference needs next, with its resolution and scan, if any
func (d *Daemon) nextScanStep(ctx context.Context, ref string) (scanStep, *store.Resolution, *store.Scan, error) {
	res, err := d.store.GetResolution(ctx, ref)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return scanNone, nil, nil, err
	}
	if res == nil || resolutionStale(res) {
		return scanResolve, res, nil, nil
	}
	if res.Error != "" {
		return scanNone, res, nil, nil
	}

	sc, err := d.store.GetScan(ctx, res.Digest)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return scanNone, res, nil, err
	}
	switch {
	case sc != nil && sc.Skipped != "":
		return scanNone, res, sc, nil
	case sc == nil:
		return scanGenerate, res, nil, nil
	case sc.SBOMAt.IsZero():
		if time.Since(sc.ScannedAt) > scanRetryAfter {
			return scanGenerate, res, sc, nil
		}
	case !d.sbomExists(res.Digest):
		return scanGenerate, res, sc, nil
	case time.Since(sc.ScannedAt) > rescanAfter || (sc.Error != "" && time.Since(sc.ScannedAt) > scanRetryAfter):
		return scanCheck, res, sc, nil
	}
	return scanNone, res, sc, nil
}

// setScanTask records what the scan loop is doing; an empty task means idle
func (d *Daemon) setScanTask(task, image string) {
	d.activity.update(func(a *Activity) {
		a.Scan.Task, a.Scan.Image, a.Scan.TaskStartedAt = task, image, nil
		if task != "" {
			a.Scan.TaskStartedAt = now()
		}
	})
}

// resolutionStale reports whether a reference should be resolved again: a failure after
// scanRetryAfter, a tag (which can move) after resolveMaxAge, a digest reference never
func resolutionStale(res *store.Resolution) bool {
	age := time.Since(res.ResolvedAt)
	if res.Error != "" {
		return age > scanRetryAfter
	}
	return !strings.Contains(res.Ref, "@") && age > resolveMaxAge
}

func (d *Daemon) resolve(ctx context.Context, ref string) error {
	res := &store.Resolution{Ref: ref, ResolvedAt: time.Now().UTC()}
	var err error
	res.Repository, res.Digest, err = scan.Resolve(ctx, ref)
	if err != nil {
		svcLog.WithField("ref", ref).WithError(err).Warn("error resolving image")
		res.Error = err.Error()
	}
	return d.store.SaveResolution(ctx, res)
}

func (d *Daemon) sbomExists(digest string) bool {
	_, err := os.Stat(d.scanner.SBOMPath(digest))
	return err == nil
}

// scanDigest checks a digest's SBOM for vulnerabilities, generating the SBOM first if asked
func (d *Daemon) scanDigest(ctx context.Context, res *store.Resolution, generate bool) error {
	logger := svcLog.WithField("image", res.Repository+"@"+res.Digest)
	started := time.Now()

	sc := &store.Scan{Digest: res.Digest, Repository: res.Repository, Counts: scan.Counts{}, ScannedAt: started.UTC()}
	if !generate {
		previous, err := d.store.GetScan(ctx, res.Digest)
		if err != nil {
			return err
		}
		sc.SBOMAt = previous.SBOMAt
	} else {
		logger.Info("generating SBOM")
		if err := d.scanner.GenerateSBOM(ctx, res.Repository, res.Digest); err != nil {
			if errors.Is(err, scan.ErrNoPlatform) {
				logger.Info("skipping image without a linux/amd64 variant")
				sc.Skipped = err.Error()
			} else {
				logger.WithError(err).Warn("error generating SBOM")
				sc.Error = err.Error()
			}
			return d.store.SaveScan(ctx, sc)
		}
		sc.SBOMAt = time.Now().UTC()
	}

	vulns, err := d.scanner.Vulnerabilities(ctx, res.Digest)
	if err != nil {
		logger.WithError(err).Warn("error scanning SBOM")
		sc.Error = err.Error()
		return d.store.SaveScan(ctx, sc)
	}
	sc.Vulnerabilities = vulns
	sc.Counts = scan.CountBySeverity(vulns)

	logger.WithField("vulnerabilities", len(vulns)).WithField("duration", time.Since(started).Round(time.Second).String()).Info("scanned image")
	return d.store.SaveScan(ctx, sc)
}

// Kinds of scan targets, in the order they are scanned
const (
	// scanTargetPR: either side of an open PR's image change, which PRs show
	scanTargetPR = "pr"

	// scanTargetRunning: the digest a container runs
	scanTargetRunning = "running"

	// scanTargetNewest: the newest version of a running image tag, so the CVEs an update
	// fixes or introduces are known before any PR exists
	scanTargetNewest = "newest"
)

type scanTarget struct {
	ref  string
	kind string
}

// scanTargets loads the open PRs, running containers and versions and returns what to scan
func (d *Daemon) scanTargets(ctx context.Context) ([]scanTarget, error) {
	prs, err := d.store.ListPullRequests(ctx, store.PullRequestStateOpen)
	if err != nil {
		return nil, err
	}
	containers, err := d.store.ListContainers(ctx)
	if err != nil {
		return nil, err
	}
	versions, err := d.store.ListVersions(ctx)
	if err != nil {
		return nil, err
	}
	return scanTargets(prs, containers, versions), nil
}

// scanTargets are the image references to scan, without duplicates: both sides of every
// image change of the open PRs (including those of chart changes) in PR order, then the digest
// each container runs, then the newest version of each running image tag that has a newer one.
// PRs come first as they are what scans were first shown on.
func scanTargets(prs []*store.PullRequest, containers []workloads.Container, versions []store.Version) []scanTarget {
	targets := []scanTarget{}
	seen := map[string]bool{}
	add := func(ref, kind string) {
		if ref != "" && !seen[ref] {
			seen[ref] = true
			targets = append(targets, scanTarget{ref: ref, kind: kind})
		}
	}
	for _, pr := range prs {
		for _, c := range pr.Changes {
			if c.Kind == prdiff.KindImage {
				add(c.From, scanTargetPR)
				add(c.To, scanTargetPR)
			}
		}
		for _, cr := range pr.ChartRenders {
			for _, diff := range cr.Images {
				add(diff.From, scanTargetPR)
				add(diff.To, scanTargetPR)
			}
		}
	}
	for _, c := range containers {
		if c.Digest != "" {
			add(scan.RunningRef(c.Image, c.Digest), scanTargetRunning)
		}
	}
	for _, v := range versions {
		if v.Kind == store.VersionKindImage && v.Error == "" && v.Latest != "" && v.Latest != v.Current {
			add(scan.TagRef(v.Repository, v.Latest), scanTargetNewest)
		}
	}
	return targets
}
