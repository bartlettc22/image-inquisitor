package daemon

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/store"
)

const (
	// syncRunHistory is how many sync runs are kept and reported
	syncRunHistory = 50

	// renovateRunHistory is how many Renovate runs are kept and reported
	renovateRunHistory = 50

	// cacheSizesMaxAge is how long measured cache sizes are reused; measuring walks the data dir
	cacheSizesMaxAge = time.Minute
)

// Activity is what the daemon's loops are doing right now
type Activity struct {
	Sync     SyncActivity     `json:"sync"`
	Scan     ScanActivity     `json:"scan"`
	Renovate RenovateActivity `json:"renovate"`
}

type RenovateActivity struct {
	Enabled bool `json:"enabled"`
	Running bool `json:"running"`

	// RunID, Job, Repos and StartedAt describe the run in progress
	RunID     int64      `json:"runID,omitempty"`
	Job       string     `json:"job,omitempty"`
	Repos     []string   `json:"repos,omitempty"`
	StartedAt *time.Time `json:"startedAt,omitempty"`

	// NextRunAt is when the next scheduled run starts; unset while one runs or without an interval
	NextRunAt *time.Time `json:"nextRunAt,omitempty"`
}

type SyncActivity struct {
	Running bool `json:"running"`

	// Step is the step running, e.g. "cluster home"
	Step          string     `json:"step,omitempty"`
	StepStartedAt *time.Time `json:"stepStartedAt,omitempty"`

	// NextRunAt is when the next scheduled sync starts; unset while one runs
	NextRunAt *time.Time `json:"nextRunAt,omitempty"`
}

type ScanActivity struct {
	Enabled bool `json:"enabled"`

	// Task is what the scan loop is doing, e.g. "generating SBOM"; empty when idle
	Task          string     `json:"task,omitempty"`
	Image         string     `json:"image,omitempty"`
	TaskStartedAt *time.Time `json:"taskStartedAt,omitempty"`
}

// activity holds the daemon's Activity for the loops to update and the API to read
type activity struct {
	mu sync.Mutex
	a  Activity
}

func (a *activity) get() Activity {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.a
}

func (a *activity) update(f func(*Activity)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	f(&a.a)
}

func now() *time.Time {
	t := time.Now().UTC()
	return &t
}

// Activity returns what the daemon's loops are doing right now
func (d *Daemon) Activity() Activity {
	return d.activity.get()
}

// ActivityReport is the daemon's activity with its queue, caches and recent sync runs
type ActivityReport struct {
	Activity

	// ScanQueue is unset when scanning is disabled
	ScanQueue *ScanQueue `json:"scanQueue,omitempty"`

	Caches   []Cache             `json:"caches"`
	TagLists *store.TagListStats `json:"tagLists"`

	// SyncRuns are the most recent sync runs, newest first
	SyncRuns []*store.SyncRun `json:"syncRuns"`

	// RenovateRuns are the most recent Renovate runs, newest first
	RenovateRuns []*store.RenovateRun `json:"renovateRuns"`
}

// ScanQueue counts the open PRs' image references by what they need next
type ScanQueue struct {
	// Images are the references to scan: PR is both sides of the open PRs' image changes,
	// Running the digests containers run, Newest the newest versions of running image tags
	Images  int `json:"images"`
	PR      int `json:"pr"`
	Running int `json:"running"`
	Newest  int `json:"newest"`

	// Resolve, Generate and Check are waiting to be resolved to a digest, to have their
	// SBOM generated, or to have their SBOM checked against the current database
	Resolve  int `json:"resolve"`
	Generate int `json:"generate"`
	Check    int `json:"check"`

	// Failed failed to resolve or scan and are retried later; Skipped have no linux/amd64 image
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

// Cache is a directory or file the daemon keeps in its data dir
type Cache struct {
	Name  string `json:"name"`
	Bytes int64  `json:"bytes"`
	Files int    `json:"files"`

	// UpdatedAt and NextUpdateAt are set for Trivy's databases: when the database
	// was published, when it was downloaded, and when Trivy will download a newer one
	UpdatedAt    *time.Time `json:"updatedAt,omitempty"`
	DownloadedAt *time.Time `json:"downloadedAt,omitempty"`
	NextUpdateAt *time.Time `json:"nextUpdateAt,omitempty"`
}

// cacheSizes keeps the last measured caches for cacheSizesMaxAge
type cacheSizes struct {
	mu         sync.Mutex
	caches     []Cache
	measuredAt time.Time
}

// ActivityReport returns the daemon's activity, scan queue, caches and recent sync runs
func (d *Daemon) ActivityReport(ctx context.Context) (*ActivityReport, error) {
	r := &ActivityReport{Activity: d.Activity(), Caches: d.caches()}

	var err error
	if d.scanner != nil {
		if r.ScanQueue, err = d.scanQueue(ctx); err != nil {
			return nil, err
		}
	}
	if r.TagLists, err = d.store.TagListStats(ctx); err != nil {
		return nil, err
	}
	if r.SyncRuns, err = d.store.ListSyncRuns(ctx, syncRunHistory); err != nil {
		return nil, err
	}
	if r.RenovateRuns, err = d.store.ListRenovateRuns(ctx, renovateRunHistory); err != nil {
		return nil, err
	}
	return r, nil
}

func (d *Daemon) scanQueue(ctx context.Context) (*ScanQueue, error) {
	targets, err := d.scanTargets(ctx)
	if err != nil {
		return nil, err
	}
	q := &ScanQueue{}
	for _, t := range targets {
		q.Images++
		switch t.kind {
		case scanTargetPR:
			q.PR++
		case scanTargetRunning:
			q.Running++
		case scanTargetNewest:
			q.Newest++
		}
		step, res, sc, err := d.nextScanStep(ctx, t.ref)
		if err != nil {
			return nil, err
		}
		switch step {
		case scanResolve:
			q.Resolve++
		case scanGenerate:
			q.Generate++
		case scanCheck:
			q.Check++
		}
		switch {
		case res != nil && res.Error != "", sc != nil && sc.Error != "":
			q.Failed++
		case sc != nil && sc.Skipped != "":
			q.Skipped++
		}
	}
	return q, nil
}

// caches measures the data dir's caches, reusing the last measurement for cacheSizesMaxAge
func (d *Daemon) caches() []Cache {
	d.cacheSizes.mu.Lock()
	defer d.cacheSizes.mu.Unlock()
	if d.cacheSizes.caches != nil && time.Since(d.cacheSizes.measuredAt) < cacheSizesMaxAge {
		return d.cacheSizes.caches
	}

	dir := d.config.DataDir
	trivy := filepath.Join(dir, "scan", "trivy")
	caches := []Cache{
		trivyDB("Trivy vulnerability database", filepath.Join(trivy, "db")),
		trivyDB("Trivy Java database", filepath.Join(trivy, "java-db")),
		usage("Trivy scan cache", filepath.Join(trivy, "fanal")),
		usage("SBOMs", filepath.Join(dir, "scan", "sbom")),
		usage("Repository checkouts", filepath.Join(dir, "render", "git")),
		usage("Helm cache", filepath.Join(dir, "render", "helm")),
		usage("Database", filepath.Join(dir, "imginq.db*")),
		usage("Data directory total", dir),
	}

	// Leave out what doesn't exist, e.g. the Java database before a Java image is scanned
	d.cacheSizes.caches = []Cache{}
	for _, c := range caches {
		if c.Files > 0 {
			d.cacheSizes.caches = append(d.cacheSizes.caches, c)
		}
	}
	d.cacheSizes.measuredAt = time.Now()
	return d.cacheSizes.caches
}

// usage sums the size of the files under a path, which may be a glob
func usage(name, pattern string) Cache {
	c := Cache{Name: name}
	paths, _ := filepath.Glob(pattern)
	for _, p := range paths {
		filepath.WalkDir(p, func(_ string, e fs.DirEntry, err error) error { //nolint:errcheck
			if err != nil || e.IsDir() {
				return nil
			}
			if info, err := e.Info(); err == nil {
				c.Bytes += info.Size()
				c.Files++
			}
			return nil
		})
	}
	return c
}

// trivyDB measures one of Trivy's databases, with the dates from its metadata.json
func trivyDB(name, dir string) Cache {
	c := usage(name, dir)
	b, err := os.ReadFile(filepath.Join(dir, "metadata.json"))
	if err != nil {
		return c
	}
	var meta struct {
		UpdatedAt    time.Time
		DownloadedAt time.Time
		NextUpdate   time.Time
	}
	if json.Unmarshal(b, &meta) == nil {
		c.UpdatedAt = timePtr(meta.UpdatedAt)
		c.DownloadedAt = timePtr(meta.DownloadedAt)
		c.NextUpdateAt = timePtr(meta.NextUpdate)
	}
	return c
}

func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	t = t.UTC()
	return &t
}
