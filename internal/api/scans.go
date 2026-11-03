package api

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/prdiff"
	"github.com/bartlettc22/image-inquisitor/internal/scan"
	"github.com/bartlettc22/image-inquisitor/internal/store"
)

// Scan statuses of an image, an image change or a PR
const (
	ScanPending = "pending"
	ScanDone    = "done"
	ScanError   = "error"

	// ScanSkipped is an image that isn't scanned, e.g. a Windows image in a chart
	ScanSkipped = "skipped"
)

// scanStage is the PR pipeline stage that summarizes the scans of its image changes
const scanStage = "scan"

// ScanSide is the scan of one side of an image change
type ScanSide struct {
	Ref       string      `json:"ref"`
	Digest    string      `json:"digest,omitempty"`
	Status    string      `json:"status"`
	Error     string      `json:"error,omitempty"`
	Counts    scan.Counts `json:"counts,omitempty"`
	ScannedAt *time.Time  `json:"scannedAt,omitempty"`
}

// ScanDelta compares the vulnerabilities of an image change's From and To images.
// A side is nil for a container a chart change adds or removes.
type ScanDelta struct {
	From       *ScanSide   `json:"from"`
	To         *ScanSide   `json:"to"`
	Status     string      `json:"status"`
	Fixed      scan.Counts `json:"fixed"`
	Introduced scan.Counts `json:"introduced"`

	// FixedVulnerabilities and IntroducedVulnerabilities are only included for a single PR
	FixedVulnerabilities      []scan.Vulnerability `json:"fixedVulnerabilities,omitempty"`
	IntroducedVulnerabilities []scan.Vulnerability `json:"introducedVulnerabilities,omitempty"`

	fixed, introduced []scan.Vulnerability
}

// PullRequestScan sums up the scans of a PR's image changes, counting each vulnerability once
type PullRequestScan struct {
	Status     string      `json:"status"`
	Images     int         `json:"images"`
	Pending    int         `json:"pending"`
	Failed     int         `json:"failed"`
	Skipped    int         `json:"skipped"`
	Fixed      scan.Counts `json:"fixed"`
	Introduced scan.Counts `json:"introduced"`
}

// vulnCache keeps scanned digests' vulnerabilities in memory until they are rescanned
type vulnCache struct {
	mu      sync.Mutex
	entries map[string]vulnCacheEntry
}

type vulnCacheEntry struct {
	scannedAt time.Time
	vulns     []scan.Vulnerability
}

func (c *vulnCache) get(ctx context.Context, s *store.Store, sc *store.Scan) ([]scan.Vulnerability, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if e, ok := c.entries[sc.Digest]; ok && e.scannedAt.Equal(sc.ScannedAt) {
		return e.vulns, nil
	}
	vulns, err := s.GetScanVulnerabilities(ctx, sc.Digest)
	if err != nil {
		return nil, err
	}
	if c.entries == nil {
		c.entries = map[string]vulnCacheEntry{}
	}
	c.entries[sc.Digest] = vulnCacheEntry{scannedAt: sc.ScannedAt, vulns: vulns}
	return vulns, nil
}

// scanIndex looks up the scans of image references
type scanIndex struct {
	resolutions map[string]store.Resolution
	scans       map[string]store.Scan
	vulns       func(sc *store.Scan) ([]scan.Vulnerability, error)
}

func (idx *scanIndex) side(ref string) (*ScanSide, []scan.Vulnerability, error) {
	side := &ScanSide{Ref: ref, Status: ScanPending}
	res, ok := idx.resolutions[ref]
	if !ok {
		return side, nil, nil
	}
	if res.Error != "" {
		side.Status, side.Error = ScanError, res.Error
		return side, nil, nil
	}
	side.Digest = res.Digest

	sc, ok := idx.scans[res.Digest]
	if !ok {
		return side, nil, nil
	}
	if sc.Skipped != "" {
		side.Status, side.Error = ScanSkipped, sc.Skipped
		return side, nil, nil
	}
	if sc.Error != "" {
		side.Status, side.Error = ScanError, sc.Error
		return side, nil, nil
	}
	vulns, err := idx.vulns(&sc)
	if err != nil {
		return nil, nil, err
	}
	scannedAt := sc.ScannedAt
	side.Status, side.Counts, side.ScannedAt = ScanDone, sc.Counts, &scannedAt
	return side, vulns, nil
}

// delta compares the scans of two references; either may be empty for an added or removed container
func (idx *scanIndex) delta(from, to string) (*ScanDelta, error) {
	d := &ScanDelta{Status: ScanDone, Fixed: scan.Counts{}, Introduced: scan.Counts{}}
	var fromVulns, toVulns []scan.Vulnerability
	var err error
	if from != "" {
		if d.From, fromVulns, err = idx.side(from); err != nil {
			return nil, err
		}
	}
	if to != "" {
		if d.To, toVulns, err = idx.side(to); err != nil {
			return nil, err
		}
	}

	// Pending wins over failed, failed over skipped
	for _, side := range []*ScanSide{d.From, d.To} {
		switch {
		case side == nil:
		case side.Status == ScanPending:
			d.Status = ScanPending
		case side.Status == ScanError && d.Status != ScanPending:
			d.Status = ScanError
		case side.Status == ScanSkipped && d.Status == ScanDone:
			d.Status = ScanSkipped
		}
	}
	if d.Status != ScanDone {
		return d, nil
	}

	d.fixed, d.introduced = scan.Diff(fromVulns, toVulns)
	d.Fixed, d.Introduced = scan.CountBySeverity(d.fixed), scan.CountBySeverity(d.introduced)
	return d, nil
}

// withVulnerabilities includes the delta's vulnerability lists in its JSON
func (d *ScanDelta) withVulnerabilities() {
	d.FixedVulnerabilities, d.IntroducedVulnerabilities = d.fixed, d.introduced
}

// attachScans adds the scan delta of each image change, including those of chart changes,
// a summary of them and a scan stage to a PR view. Vulnerability lists are only included
// when withVulnerabilities is set.
func attachScans(view *PullRequestView, idx *scanIndex, withVulnerabilities bool) error {
	deltas := []*ScanDelta{}
	add := func(from, to string) (*ScanDelta, error) {
		d, err := idx.delta(from, to)
		if err != nil {
			return nil, err
		}
		if withVulnerabilities {
			d.withVulnerabilities()
		}
		deltas = append(deltas, d)
		return d, nil
	}

	var err error
	for i := range view.Changes {
		cv := &view.Changes[i]
		if cv.Kind == prdiff.KindImage {
			if cv.Scan, err = add(cv.From, cv.To); err != nil {
				return err
			}
		}
		for r := range cv.Renders {
			for j := range cv.Renders[r].Images {
				img := &cv.Renders[r].Images[j]
				if img.Scan, err = add(img.From, img.To); err != nil {
					return err
				}
			}
		}
	}
	if len(deltas) == 0 {
		return nil
	}

	view.Scan = summarize(deltas)
	stage := store.Stage{Name: scanStage, Status: view.Scan.Status, HeadSHA: view.HeadSHA, UpdatedAt: lastScanned(deltas)}
	switch view.Scan.Status {
	case ScanPending:
		stage.Error = fmt.Sprintf("%d of %d images still to scan", view.Scan.Pending, view.Scan.Images)
	case ScanError:
		stage.Status = store.StageStatusError
		stage.Error = fmt.Sprintf("%d of %d images failed to scan", view.Scan.Failed, view.Scan.Images)
	}
	view.Stages = append(slices.Clone(view.Stages), stage)
	return nil
}

// lastScanned is when the most recently scanned image of the deltas was scanned, else now
func lastScanned(deltas []*ScanDelta) time.Time {
	last := time.Time{}
	for _, d := range deltas {
		for _, side := range []*ScanSide{d.From, d.To} {
			if side != nil && side.ScannedAt != nil && side.ScannedAt.After(last) {
				last = *side.ScannedAt
			}
		}
	}
	if last.IsZero() {
		return time.Now().UTC()
	}
	return last
}

// summarize sums up a PR's deltas, counting a vulnerability fixed or introduced by
// several of its images once
func summarize(deltas []*ScanDelta) *PullRequestScan {
	s := &PullRequestScan{Status: ScanDone, Images: len(deltas), Fixed: scan.Counts{}, Introduced: scan.Counts{}}
	fixed, introduced := map[string]scan.Vulnerability{}, map[string]scan.Vulnerability{}
	for _, d := range deltas {
		switch d.Status {
		case ScanPending:
			s.Pending++
		case ScanError:
			s.Failed++
		case ScanSkipped:
			s.Skipped++
		}
		for _, v := range d.fixed {
			fixed[v.Key()] = v
		}
		for _, v := range d.introduced {
			introduced[v.Key()] = v
		}
	}
	for _, v := range fixed {
		s.Fixed[v.Severity]++
	}
	for _, v := range introduced {
		s.Introduced[v.Severity]++
	}

	switch {
	case s.Pending > 0:
		s.Status = ScanPending
	case s.Failed > 0:
		s.Status = ScanError
	}
	return s
}
