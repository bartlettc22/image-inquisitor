// Package store persists daemon state (pull requests, extracted changes, pipeline
// stage status, cluster inventory, rendered apps and sync runs) in SQLite
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/argocd"
	"github.com/bartlettc22/image-inquisitor/internal/prdiff"
	"github.com/bartlettc22/image-inquisitor/internal/render"
	"github.com/bartlettc22/image-inquisitor/internal/scan"
	"github.com/bartlettc22/image-inquisitor/internal/workloads"
	_ "modernc.org/sqlite"
)

const (
	PullRequestStateOpen   = "open"
	PullRequestStateClosed = "closed"

	StageStatusDone  = "done"
	StageStatusError = "error"
)

var ErrNotFound = errors.New("not found")

const schema = `
CREATE TABLE IF NOT EXISTS pull_requests (
	repo       TEXT    NOT NULL,
	number     INTEGER NOT NULL,
	title      TEXT    NOT NULL,
	url        TEXT    NOT NULL,
	branch     TEXT    NOT NULL,
	head_sha   TEXT    NOT NULL,
	author     TEXT    NOT NULL,
	state      TEXT    NOT NULL,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL,
	synced_at  INTEGER NOT NULL,
	PRIMARY KEY (repo, number)
);

CREATE TABLE IF NOT EXISTS changes (
	repo     TEXT    NOT NULL,
	number   INTEGER NOT NULL,
	idx      INTEGER NOT NULL,
	file     TEXT    NOT NULL,
	kind     TEXT    NOT NULL,
	name     TEXT    NOT NULL,
	repo_url TEXT    NOT NULL,
	from_ref TEXT    NOT NULL,
	to_ref   TEXT    NOT NULL,
	PRIMARY KEY (repo, number, idx)
);

CREATE TABLE IF NOT EXISTS stages (
	repo       TEXT    NOT NULL,
	number     INTEGER NOT NULL,
	stage      TEXT    NOT NULL,
	status     TEXT    NOT NULL,
	error      TEXT    NOT NULL,
	head_sha   TEXT    NOT NULL,
	updated_at INTEGER NOT NULL,
	PRIMARY KEY (repo, number, stage)
);

CREATE TABLE IF NOT EXISTS containers (
	cluster       TEXT    NOT NULL,
	namespace     TEXT    NOT NULL,
	workload_kind TEXT    NOT NULL,
	workload_name TEXT    NOT NULL,
	pod           TEXT    NOT NULL,
	container     TEXT    NOT NULL,
	init          INTEGER NOT NULL,
	image         TEXT    NOT NULL,
	digest        TEXT    NOT NULL,
	argo_app      TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS containers_cluster ON containers (cluster);

CREATE TABLE IF NOT EXISTS clusters (
	name       TEXT    PRIMARY KEY,
	synced_at  INTEGER NOT NULL,
	containers INTEGER NOT NULL,
	error      TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS applications (
	cluster TEXT NOT NULL,
	name    TEXT NOT NULL,
	data    TEXT NOT NULL,
	PRIMARY KEY (cluster, name)
);

CREATE TABLE IF NOT EXISTS renders (
	cluster     TEXT    NOT NULL,
	app         TEXT    NOT NULL,
	source      INTEGER NOT NULL,
	key         TEXT    NOT NULL,
	revision    TEXT    NOT NULL,
	error       TEXT    NOT NULL,
	images      TEXT    NOT NULL,
	rendered_at INTEGER NOT NULL,
	PRIMARY KEY (cluster, app, source)
);

CREATE TABLE IF NOT EXISTS chart_renders (
	repo       TEXT    NOT NULL,
	number     INTEGER NOT NULL,
	change_idx INTEGER NOT NULL,
	cluster    TEXT    NOT NULL,
	app        TEXT    NOT NULL,
	source     INTEGER NOT NULL,
	error      TEXT    NOT NULL,
	images     TEXT    NOT NULL,
	PRIMARY KEY (repo, number, change_idx, cluster, app, source)
);

CREATE TABLE IF NOT EXISTS tag_lists (
	kind       TEXT    NOT NULL,
	repository TEXT    NOT NULL,
	tags       TEXT    NOT NULL,
	error      TEXT    NOT NULL,
	fetched_at INTEGER NOT NULL,
	PRIMARY KEY (kind, repository)
);

CREATE TABLE IF NOT EXISTS versions (
	kind       TEXT    NOT NULL,
	repository TEXT    NOT NULL,
	current    TEXT    NOT NULL,
	latest     TEXT    NOT NULL,
	error      TEXT    NOT NULL,
	checked_at INTEGER NOT NULL,
	PRIMARY KEY (kind, repository, current)
);

CREATE TABLE IF NOT EXISTS resolutions (
	ref         TEXT    PRIMARY KEY,
	repository  TEXT    NOT NULL,
	digest      TEXT    NOT NULL,
	error       TEXT    NOT NULL,
	resolved_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS scans (
	digest          TEXT    PRIMARY KEY,
	repository      TEXT    NOT NULL,
	counts          TEXT    NOT NULL,
	vulnerabilities TEXT    NOT NULL,
	error           TEXT    NOT NULL,
	skipped         TEXT    NOT NULL,
	sbom_at         INTEGER NOT NULL,
	scanned_at      INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS sync_runs (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	started_at    INTEGER NOT NULL,
	finished_at   INTEGER,
	pull_requests INTEGER NOT NULL DEFAULT 0,
	error         TEXT    NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS sync_steps (
	run_id      INTEGER NOT NULL,
	position    INTEGER NOT NULL,
	name        TEXT    NOT NULL,
	started_at  INTEGER NOT NULL,
	finished_at INTEGER NOT NULL,
	summary     TEXT    NOT NULL,
	error       TEXT    NOT NULL,
	PRIMARY KEY (run_id, position)
);

CREATE TABLE IF NOT EXISTS release_dates (
	ref        TEXT    PRIMARY KEY,
	created    INTEGER NOT NULL,
	first_seen INTEGER NOT NULL,
	error      TEXT    NOT NULL,
	checked_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS renovate_runs (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	started_at  INTEGER NOT NULL,
	finished_at INTEGER,
	trigger     TEXT    NOT NULL,
	repos       TEXT    NOT NULL,
	job         TEXT    NOT NULL DEFAULT '',
	status      TEXT    NOT NULL,
	error       TEXT    NOT NULL DEFAULT '',
	log         TEXT    NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS findings (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	cluster       TEXT    NOT NULL,
	namespace     TEXT    NOT NULL,
	workload_kind TEXT    NOT NULL,
	workload_name TEXT    NOT NULL,
	container     TEXT    NOT NULL,
	vuln_id       TEXT    NOT NULL,
	package       TEXT    NOT NULL,
	severity      TEXT    NOT NULL,
	digest        TEXT    NOT NULL,
	first_seen    INTEGER NOT NULL,
	fix_available INTEGER NOT NULL,
	last_seen     INTEGER NOT NULL,
	closed_at     INTEGER NOT NULL,
	close_reason  TEXT    NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS findings_open ON findings (cluster, namespace, workload_kind, workload_name, container, vuln_id, package) WHERE closed_at = 0;

CREATE TABLE IF NOT EXISTS image_history (
	id             INTEGER PRIMARY KEY AUTOINCREMENT,
	cluster        TEXT    NOT NULL,
	namespace      TEXT    NOT NULL,
	workload_kind  TEXT    NOT NULL,
	workload_name  TEXT    NOT NULL,
	pod            TEXT    NOT NULL,
	pod_uid        TEXT    NOT NULL,
	node           TEXT    NOT NULL,
	container      TEXT    NOT NULL,
	container_type TEXT    NOT NULL,
	image          TEXT    NOT NULL,
	repository     TEXT    NOT NULL,
	digest         TEXT    NOT NULL,
	argo_app       TEXT    NOT NULL,
	ran            INTEGER NOT NULL,
	started_at     INTEGER NOT NULL,
	recorded_at    INTEGER NOT NULL,
	ended_at       INTEGER NOT NULL,
	end_reason     TEXT    NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS image_history_open ON image_history (cluster, pod_uid, container) WHERE ended_at = 0;
CREATE INDEX IF NOT EXISTS image_history_pod ON image_history (cluster, pod_uid, container);
CREATE INDEX IF NOT EXISTS image_history_repository ON image_history (repository);
CREATE INDEX IF NOT EXISTS image_history_digest ON image_history (digest);
CREATE INDEX IF NOT EXISTS image_history_started ON image_history (started_at);

CREATE TABLE IF NOT EXISTS history_coverage (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	cluster    TEXT    NOT NULL,
	started_at INTEGER NOT NULL,
	last_seen  INTEGER NOT NULL
);
`

type Store struct {
	db *sql.DB
}

type PullRequest struct {
	Repo      string          `json:"repo"`
	Number    int             `json:"number"`
	Title     string          `json:"title"`
	URL       string          `json:"url"`
	Branch    string          `json:"branch"`
	HeadSHA   string          `json:"headSHA"`
	Author    string          `json:"author"`
	State     string          `json:"state"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
	SyncedAt  time.Time       `json:"syncedAt"`
	Changes   []prdiff.Change `json:"changes"`
	Stages    []Stage         `json:"stages"`

	// ChartRenders are the image changes of the PR's Helm chart bumps, per app using the chart
	ChartRenders []ChartRender `json:"chartRenders"`
}

// ChartRender is the result of rendering one app at a PR's chart version
type ChartRender struct {
	// Change is the index of the chart change in the PR's changes
	Change  int                `json:"change"`
	Cluster string             `json:"cluster"`
	App     string             `json:"app"`
	Source  int                `json:"source"`
	Error   string             `json:"error,omitempty"`
	Images  []render.ImageDiff `json:"images"`
}

// Render is the last render of one Helm source of an Argo CD app
type Render struct {
	Cluster string `json:"cluster"`
	App     string `json:"app"`
	Source  int    `json:"source"`

	// Key identifies the render's inputs (render.Request.Key)
	Key        string         `json:"-"`
	Revision   string         `json:"revision"`
	Error      string         `json:"error,omitempty"`
	Images     []render.Image `json:"images"`
	RenderedAt time.Time      `json:"renderedAt"`
}

// Stage is the status of one pipeline stage (e.g. extract, scan) for a pull request
type Stage struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
	HeadSHA   string    `json:"headSHA"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Cluster is the result of the last inventory of a cluster
type Cluster struct {
	Name       string    `json:"name"`
	SyncedAt   time.Time `json:"syncedAt"`
	Containers int       `json:"containers"`
	Error      string    `json:"error,omitempty"`
}

const (
	VersionKindImage = "image"
	VersionKindChart = "chart"
)

// TagList is a cached list of an image repository's tags or a chart's versions
type TagList struct {
	Kind string

	// Repository is the normalized image repository, or the chart's argocd.Source.ChartRef
	Repository string
	Tags       []string
	Error      string
	FetchedAt  time.Time
}

// Version is the newest version of the same variant as a running image tag or deployed
// chart version. Latest is empty when Current isn't a version (e.g. "latest").
type Version struct {
	Kind       string    `json:"kind"`
	Repository string    `json:"repository"`
	Current    string    `json:"current"`
	Latest     string    `json:"latest,omitempty"`
	Error      string    `json:"error,omitempty"`
	CheckedAt  time.Time `json:"checkedAt"`
}

// Resolution is the digest an image reference pointed at when it was last resolved
type Resolution struct {
	Ref        string    `json:"ref"`
	Repository string    `json:"repository"`
	Digest     string    `json:"digest,omitempty"`
	Error      string    `json:"error,omitempty"`
	ResolvedAt time.Time `json:"resolvedAt"`
}

// Scan is the result of scanning an image digest. SBOMAt is when its SBOM was generated
// (zero if that failed); ScannedAt when the SBOM was last checked for vulnerabilities.
type Scan struct {
	Digest     string      `json:"digest"`
	Repository string      `json:"repository"`
	Counts     scan.Counts `json:"counts"`
	Error      string      `json:"error,omitempty"`

	// Skipped is why the digest isn't scanned, e.g. it has no linux/amd64 image
	Skipped   string    `json:"skipped,omitempty"`
	SBOMAt    time.Time `json:"sbomAt"`
	ScannedAt time.Time `json:"scannedAt"`

	// Vulnerabilities are only loaded by GetScanVulnerabilities
	Vulnerabilities []scan.Vulnerability `json:"-"`
}

type SyncRun struct {
	ID           int64      `json:"id"`
	StartedAt    time.Time  `json:"startedAt"`
	FinishedAt   *time.Time `json:"finishedAt,omitempty"`
	PullRequests int        `json:"pullRequests"`
	Error        string     `json:"error,omitempty"`

	// Steps are only loaded by ListSyncRuns
	Steps []SyncStep `json:"steps,omitempty"`
}

// SyncStep is one step of a sync run, e.g. syncing one repo or one cluster
type SyncStep struct {
	Name       string    `json:"name"`
	StartedAt  time.Time `json:"startedAt"`
	FinishedAt time.Time `json:"finishedAt"`

	// Summary says what the step did, e.g. "84 containers, 9 apps"
	Summary string `json:"summary"`
	Error   string `json:"error,omitempty"`
}

// SyncRunInterrupted is the error recorded on sync runs the daemon stopped in the middle of
const SyncRunInterrupted = "interrupted: the daemon stopped during this sync"

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", path))
	if err != nil {
		return nil, err
	}

	// A single connection avoids SQLITE_BUSY between concurrent writers
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("error creating schema: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) UpsertPullRequest(ctx context.Context, pr *PullRequest) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pull_requests (repo, number, title, url, branch, head_sha, author, state, created_at, updated_at, synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (repo, number) DO UPDATE SET
			title = excluded.title, url = excluded.url, branch = excluded.branch, head_sha = excluded.head_sha,
			author = excluded.author, state = excluded.state, updated_at = excluded.updated_at, synced_at = excluded.synced_at`,
		pr.Repo, pr.Number, pr.Title, pr.URL, pr.Branch, pr.HeadSHA, pr.Author, pr.State,
		pr.CreatedAt.Unix(), pr.UpdatedAt.Unix(), pr.SyncedAt.Unix(),
	)
	return err
}

// ListOpenPullRequestRepos returns the repos with open pull requests
func (s *Store) ListOpenPullRequestRepos(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT DISTINCT repo FROM pull_requests WHERE state = ? ORDER BY repo`, PullRequestStateOpen)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	repos := []string{}
	for rows.Next() {
		var repo string
		if err := rows.Scan(&repo); err != nil {
			return nil, err
		}
		repos = append(repos, repo)
	}
	return repos, rows.Err()
}

// CloseMissingPullRequests marks open pull requests of a repo that are not in openNumbers as closed
func (s *Store) CloseMissingPullRequests(ctx context.Context, repo string, openNumbers []int) error {
	rows, err := s.db.QueryContext(ctx, `SELECT number FROM pull_requests WHERE repo = ? AND state = ?`, repo, PullRequestStateOpen)
	if err != nil {
		return err
	}
	stored := []int{}
	for rows.Next() {
		var n int
		if err := rows.Scan(&n); err != nil {
			rows.Close()
			return err
		}
		stored = append(stored, n)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	open := make(map[int]bool, len(openNumbers))
	for _, n := range openNumbers {
		open[n] = true
	}
	for _, n := range stored {
		if open[n] {
			continue
		}
		if _, err := s.db.ExecContext(ctx, `UPDATE pull_requests SET state = ? WHERE repo = ? AND number = ?`, PullRequestStateClosed, repo, n); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) ReplaceChanges(ctx context.Context, repo string, number int, changes []prdiff.Change) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx, `DELETE FROM changes WHERE repo = ? AND number = ?`, repo, number); err != nil {
		return err
	}
	for i, c := range changes {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO changes (repo, number, idx, file, kind, name, repo_url, from_ref, to_ref)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			repo, number, i, c.File, string(c.Kind), c.Name, c.RepoURL, c.From, c.To,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) SetStage(ctx context.Context, repo string, number int, stage *Stage) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO stages (repo, number, stage, status, error, head_sha, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (repo, number, stage) DO UPDATE SET
			status = excluded.status, error = excluded.error, head_sha = excluded.head_sha, updated_at = excluded.updated_at`,
		repo, number, stage.Name, stage.Status, stage.Error, stage.HeadSHA, stage.UpdatedAt.Unix(),
	)
	return err
}

// GetStage returns a pull request's stage, or ErrNotFound if it hasn't run
func (s *Store) GetStage(ctx context.Context, repo string, number int, name string) (*Stage, error) {
	stage := &Stage{Name: name}
	var updatedAt int64
	err := s.db.QueryRowContext(ctx, `
		SELECT status, error, head_sha, updated_at FROM stages WHERE repo = ? AND number = ? AND stage = ?`,
		repo, number, name,
	).Scan(&stage.Status, &stage.Error, &stage.HeadSHA, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	stage.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return stage, nil
}

// ListPullRequests returns pull requests with their changes and stages, most recently updated first.
// An empty state returns all pull requests.
func (s *Store) ListPullRequests(ctx context.Context, state string) ([]*PullRequest, error) {
	return s.queryPullRequests(ctx, `WHERE ? = '' OR state = ?`, state, state)
}

// GetPullRequest returns a single pull request, or ErrNotFound
func (s *Store) GetPullRequest(ctx context.Context, repo string, number int) (*PullRequest, error) {
	prs, err := s.queryPullRequests(ctx, `WHERE repo = ? AND number = ?`, repo, number)
	if err != nil {
		return nil, err
	}
	if len(prs) == 0 {
		return nil, ErrNotFound
	}
	return prs[0], nil
}

func (s *Store) queryPullRequests(ctx context.Context, where string, args ...any) ([]*PullRequest, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT repo, number, title, url, branch, head_sha, author, state, created_at, updated_at, synced_at
		FROM pull_requests `+where+` ORDER BY updated_at DESC`, args...)
	if err != nil {
		return nil, err
	}

	prs := []*PullRequest{}
	byKey := map[string]*PullRequest{}
	for rows.Next() {
		pr := &PullRequest{Changes: []prdiff.Change{}, Stages: []Stage{}, ChartRenders: []ChartRender{}}
		var createdAt, updatedAt, syncedAt int64
		if err := rows.Scan(&pr.Repo, &pr.Number, &pr.Title, &pr.URL, &pr.Branch, &pr.HeadSHA, &pr.Author, &pr.State, &createdAt, &updatedAt, &syncedAt); err != nil {
			rows.Close()
			return nil, err
		}
		pr.CreatedAt = time.Unix(createdAt, 0).UTC()
		pr.UpdatedAt = time.Unix(updatedAt, 0).UTC()
		pr.SyncedAt = time.Unix(syncedAt, 0).UTC()
		prs = append(prs, pr)
		byKey[prKey(pr.Repo, pr.Number)] = pr
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := s.attachChanges(ctx, byKey); err != nil {
		return nil, err
	}
	if err := s.attachStages(ctx, byKey); err != nil {
		return nil, err
	}
	if err := s.attachChartRenders(ctx, byKey); err != nil {
		return nil, err
	}

	return prs, nil
}

func (s *Store) attachChanges(ctx context.Context, byKey map[string]*PullRequest) error {
	rows, err := s.db.QueryContext(ctx, `
		SELECT repo, number, file, kind, name, repo_url, from_ref, to_ref FROM changes ORDER BY repo, number, idx`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var repo, kind string
		var number int
		c := prdiff.Change{}
		if err := rows.Scan(&repo, &number, &c.File, &kind, &c.Name, &c.RepoURL, &c.From, &c.To); err != nil {
			return err
		}
		c.Kind = prdiff.Kind(kind)
		if pr, ok := byKey[prKey(repo, number)]; ok {
			pr.Changes = append(pr.Changes, c)
		}
	}

	return rows.Err()
}

func (s *Store) attachStages(ctx context.Context, byKey map[string]*PullRequest) error {
	rows, err := s.db.QueryContext(ctx, `
		SELECT repo, number, stage, status, error, head_sha, updated_at FROM stages ORDER BY repo, number, stage`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var repo string
		var number int
		var updatedAt int64
		st := Stage{}
		if err := rows.Scan(&repo, &number, &st.Name, &st.Status, &st.Error, &st.HeadSHA, &updatedAt); err != nil {
			return err
		}
		st.UpdatedAt = time.Unix(updatedAt, 0).UTC()
		if pr, ok := byKey[prKey(repo, number)]; ok {
			pr.Stages = append(pr.Stages, st)
		}
	}

	return rows.Err()
}

func (s *Store) attachChartRenders(ctx context.Context, byKey map[string]*PullRequest) error {
	rows, err := s.db.QueryContext(ctx, `
		SELECT repo, number, change_idx, cluster, app, source, error, images FROM chart_renders
		ORDER BY repo, number, change_idx, cluster, app, source`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var repo, images string
		var number int
		cr := ChartRender{}
		if err := rows.Scan(&repo, &number, &cr.Change, &cr.Cluster, &cr.App, &cr.Source, &cr.Error, &images); err != nil {
			return err
		}
		if err := json.Unmarshal([]byte(images), &cr.Images); err != nil {
			return err
		}
		if pr, ok := byKey[prKey(repo, number)]; ok {
			pr.ChartRenders = append(pr.ChartRenders, cr)
		}
	}

	return rows.Err()
}

// ReplaceChartRenders stores the results of rendering a PR's chart changes
func (s *Store) ReplaceChartRenders(ctx context.Context, repo string, number int, renders []ChartRender) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx, `DELETE FROM chart_renders WHERE repo = ? AND number = ?`, repo, number); err != nil {
		return err
	}
	for _, cr := range renders {
		images, err := json.Marshal(nonNil(cr.Images))
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO chart_renders (repo, number, change_idx, cluster, app, source, error, images)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			repo, number, cr.Change, cr.Cluster, cr.App, cr.Source, cr.Error, string(images),
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ReplaceApplications stores a cluster's Argo CD apps and drops the renders of apps that are gone
func (s *Store) ReplaceApplications(ctx context.Context, cluster string, apps []argocd.Application) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx, `DELETE FROM applications WHERE cluster = ?`, cluster); err != nil {
		return err
	}
	for _, app := range apps {
		data, err := json.Marshal(app)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO applications (cluster, name, data) VALUES (?, ?, ?)`, cluster, app.Name, string(data)); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM renders WHERE cluster = ? AND app NOT IN (SELECT name FROM applications WHERE cluster = ?)`,
		cluster, cluster,
	); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) ListApplications(ctx context.Context) ([]argocd.Application, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM applications ORDER BY cluster, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	apps := []argocd.Application{}
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		app := argocd.Application{}
		if err := json.Unmarshal([]byte(data), &app); err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}

	return apps, rows.Err()
}

// GetRender returns the last render of an app source, or ErrNotFound
func (s *Store) GetRender(ctx context.Context, cluster, app string, source int) (*Render, error) {
	renders, err := s.queryRenders(ctx, `WHERE cluster = ? AND app = ? AND source = ?`, cluster, app, source)
	if err != nil {
		return nil, err
	}
	if len(renders) == 0 {
		return nil, ErrNotFound
	}
	return &renders[0], nil
}

func (s *Store) ListRenders(ctx context.Context) ([]Render, error) {
	return s.queryRenders(ctx, ``)
}

func (s *Store) SaveRender(ctx context.Context, r *Render) error {
	images, err := json.Marshal(nonNil(r.Images))
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO renders (cluster, app, source, key, revision, error, images, rendered_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (cluster, app, source) DO UPDATE SET
			key = excluded.key, revision = excluded.revision, error = excluded.error,
			images = excluded.images, rendered_at = excluded.rendered_at`,
		r.Cluster, r.App, r.Source, r.Key, r.Revision, r.Error, string(images), r.RenderedAt.Unix(),
	)
	return err
}

// DeleteRenders removes an app's renders for sources at or beyond index from
func (s *Store) DeleteRenders(ctx context.Context, cluster, app string, from int) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM renders WHERE cluster = ? AND app = ? AND source >= ?`, cluster, app, from)
	return err
}

func (s *Store) queryRenders(ctx context.Context, where string, args ...any) ([]Render, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT cluster, app, source, key, revision, error, images, rendered_at FROM renders `+where+`
		ORDER BY cluster, app, source`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	renders := []Render{}
	for rows.Next() {
		r := Render{}
		var images string
		var renderedAt int64
		if err := rows.Scan(&r.Cluster, &r.App, &r.Source, &r.Key, &r.Revision, &r.Error, &images, &renderedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(images), &r.Images); err != nil {
			return nil, err
		}
		r.RenderedAt = time.Unix(renderedAt, 0).UTC()
		renders = append(renders, r)
	}

	return renders, rows.Err()
}

// GetTagList returns a cached tag list, or ErrNotFound
func (s *Store) GetTagList(ctx context.Context, kind, repository string) (*TagList, error) {
	tl := &TagList{Kind: kind, Repository: repository}
	var tags string
	var fetchedAt int64
	err := s.db.QueryRowContext(ctx, `
		SELECT tags, error, fetched_at FROM tag_lists WHERE kind = ? AND repository = ?`, kind, repository,
	).Scan(&tags, &tl.Error, &fetchedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(tags), &tl.Tags); err != nil {
		return nil, err
	}
	tl.FetchedAt = time.Unix(fetchedAt, 0).UTC()
	return tl, nil
}

func (s *Store) SaveTagList(ctx context.Context, tl *TagList) error {
	tags, err := json.Marshal(nonNil(tl.Tags))
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO tag_lists (kind, repository, tags, error, fetched_at) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (kind, repository) DO UPDATE SET
			tags = excluded.tags, error = excluded.error, fetched_at = excluded.fetched_at`,
		tl.Kind, tl.Repository, string(tags), tl.Error, tl.FetchedAt.Unix(),
	)
	return err
}

// ReplaceVersions stores the version status of everything running, replacing the previous set
func (s *Store) ReplaceVersions(ctx context.Context, versions []Version) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx, `DELETE FROM versions`); err != nil {
		return err
	}
	for _, v := range versions {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO versions (kind, repository, current, latest, error, checked_at) VALUES (?, ?, ?, ?, ?, ?)`,
			v.Kind, v.Repository, v.Current, v.Latest, v.Error, v.CheckedAt.Unix(),
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) ListVersions(ctx context.Context) ([]Version, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT kind, repository, current, latest, error, checked_at FROM versions ORDER BY kind, repository, current`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	versions := []Version{}
	for rows.Next() {
		v := Version{}
		var checkedAt int64
		if err := rows.Scan(&v.Kind, &v.Repository, &v.Current, &v.Latest, &v.Error, &checkedAt); err != nil {
			return nil, err
		}
		v.CheckedAt = time.Unix(checkedAt, 0).UTC()
		versions = append(versions, v)
	}

	return versions, rows.Err()
}

// GetResolution returns an image reference's last resolution, or ErrNotFound
func (s *Store) GetResolution(ctx context.Context, ref string) (*Resolution, error) {
	resolutions, err := s.queryResolutions(ctx, `WHERE ref = ?`, ref)
	if err != nil {
		return nil, err
	}
	if len(resolutions) == 0 {
		return nil, ErrNotFound
	}
	return &resolutions[0], nil
}

func (s *Store) ListResolutions(ctx context.Context) ([]Resolution, error) {
	return s.queryResolutions(ctx, ``)
}

func (s *Store) SaveResolution(ctx context.Context, r *Resolution) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO resolutions (ref, repository, digest, error, resolved_at) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (ref) DO UPDATE SET
			repository = excluded.repository, digest = excluded.digest, error = excluded.error, resolved_at = excluded.resolved_at`,
		r.Ref, r.Repository, r.Digest, r.Error, r.ResolvedAt.Unix(),
	)
	return err
}

func (s *Store) queryResolutions(ctx context.Context, where string, args ...any) ([]Resolution, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT ref, repository, digest, error, resolved_at FROM resolutions `+where+` ORDER BY ref`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	resolutions := []Resolution{}
	for rows.Next() {
		r := Resolution{}
		var resolvedAt int64
		if err := rows.Scan(&r.Ref, &r.Repository, &r.Digest, &r.Error, &resolvedAt); err != nil {
			return nil, err
		}
		r.ResolvedAt = time.Unix(resolvedAt, 0).UTC()
		resolutions = append(resolutions, r)
	}
	return resolutions, rows.Err()
}

// GetScan returns a digest's scan without its vulnerabilities, or ErrNotFound
func (s *Store) GetScan(ctx context.Context, digest string) (*Scan, error) {
	scans, err := s.queryScans(ctx, `WHERE digest = ?`, digest)
	if err != nil {
		return nil, err
	}
	if len(scans) == 0 {
		return nil, ErrNotFound
	}
	return &scans[0], nil
}

// ListScans returns every scan without its vulnerabilities
func (s *Store) ListScans(ctx context.Context) ([]Scan, error) {
	return s.queryScans(ctx, ``)
}

// GetScanVulnerabilities returns a scanned digest's vulnerabilities
func (s *Store) GetScanVulnerabilities(ctx context.Context, digest string) ([]scan.Vulnerability, error) {
	var data string
	err := s.db.QueryRowContext(ctx, `SELECT vulnerabilities FROM scans WHERE digest = ?`, digest).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	vulns := []scan.Vulnerability{}
	return vulns, json.Unmarshal([]byte(data), &vulns)
}

func (s *Store) SaveScan(ctx context.Context, sc *Scan) error {
	counts, err := json.Marshal(sc.Counts)
	if err != nil {
		return err
	}
	vulns, err := json.Marshal(nonNil(sc.Vulnerabilities))
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO scans (digest, repository, counts, vulnerabilities, error, skipped, sbom_at, scanned_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (digest) DO UPDATE SET
			repository = excluded.repository, counts = excluded.counts, vulnerabilities = excluded.vulnerabilities,
			error = excluded.error, skipped = excluded.skipped, sbom_at = excluded.sbom_at, scanned_at = excluded.scanned_at`,
		sc.Digest, sc.Repository, string(counts), string(vulns), sc.Error, sc.Skipped, unixOrZero(sc.SBOMAt), unixOrZero(sc.ScannedAt),
	)
	return err
}

func (s *Store) queryScans(ctx context.Context, where string, args ...any) ([]Scan, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT digest, repository, counts, error, skipped, sbom_at, scanned_at FROM scans `+where+` ORDER BY digest`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	scans := []Scan{}
	for rows.Next() {
		sc := Scan{}
		var counts string
		var sbomAt, scannedAt int64
		if err := rows.Scan(&sc.Digest, &sc.Repository, &counts, &sc.Error, &sc.Skipped, &sbomAt, &scannedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(counts), &sc.Counts); err != nil {
			return nil, err
		}
		sc.SBOMAt = timeOrZero(sbomAt)
		sc.ScannedAt = timeOrZero(scannedAt)
		scans = append(scans, sc)
	}
	return scans, rows.Err()
}

func unixOrZero(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

func timeOrZero(unix int64) time.Time {
	if unix == 0 {
		return time.Time{}
	}
	return time.Unix(unix, 0).UTC()
}

// ReplaceContainers stores a cluster's inventory. On a failed inventory (syncErr != nil)
// the previous containers are kept and only the error is recorded.
func (s *Store) ReplaceContainers(ctx context.Context, cluster string, containers []workloads.Container, syncedAt time.Time, syncErr error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	errStr := ""
	if syncErr != nil {
		errStr = syncErr.Error()
	} else {
		if _, err := tx.ExecContext(ctx, `DELETE FROM containers WHERE cluster = ?`, cluster); err != nil {
			return err
		}
		for _, c := range containers {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO containers (cluster, namespace, workload_kind, workload_name, pod, container, init, image, digest, argo_app)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				cluster, c.Namespace, c.WorkloadKind, c.WorkloadName, c.Pod, c.Container, c.Init, c.Image, c.Digest, c.ArgoApp,
			); err != nil {
				return err
			}
		}
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO clusters (name, synced_at, containers, error) VALUES (?, ?, ?, ?)
		ON CONFLICT (name) DO UPDATE SET
			synced_at = excluded.synced_at, error = excluded.error,
			containers = CASE WHEN excluded.error = '' THEN excluded.containers ELSE clusters.containers END`,
		cluster, syncedAt.Unix(), len(containers), errStr,
	); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) ListContainers(ctx context.Context) ([]workloads.Container, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT cluster, namespace, workload_kind, workload_name, pod, container, init, image, digest, argo_app
		FROM containers ORDER BY cluster, namespace, workload_kind, workload_name, pod, container`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	containers := []workloads.Container{}
	for rows.Next() {
		c := workloads.Container{}
		if err := rows.Scan(&c.Cluster, &c.Namespace, &c.WorkloadKind, &c.WorkloadName, &c.Pod, &c.Container, &c.Init, &c.Image, &c.Digest, &c.ArgoApp); err != nil {
			return nil, err
		}
		containers = append(containers, c)
	}

	return containers, rows.Err()
}

func (s *Store) ListClusters(ctx context.Context) ([]Cluster, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT name, synced_at, containers, error FROM clusters ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	clusters := []Cluster{}
	for rows.Next() {
		c := Cluster{}
		var syncedAt int64
		if err := rows.Scan(&c.Name, &syncedAt, &c.Containers, &c.Error); err != nil {
			return nil, err
		}
		c.SyncedAt = time.Unix(syncedAt, 0).UTC()
		clusters = append(clusters, c)
	}

	return clusters, rows.Err()
}

func (s *Store) StartSyncRun(ctx context.Context, startedAt time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO sync_runs (started_at) VALUES (?)`, startedAt.Unix())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) FinishSyncRun(ctx context.Context, id int64, finishedAt time.Time, pullRequests int, runErr error) error {
	errStr := ""
	if runErr != nil {
		errStr = runErr.Error()
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE sync_runs SET finished_at = ?, pull_requests = ?, error = ? WHERE id = ?`,
		finishedAt.Unix(), pullRequests, errStr, id,
	)
	return err
}

// InterruptSyncRuns finishes the runs left unfinished by a daemon that stopped mid-sync
func (s *Store) InterruptSyncRuns(ctx context.Context, finishedAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE sync_runs SET finished_at = ?, error = ? WHERE finished_at IS NULL`,
		finishedAt.Unix(), SyncRunInterrupted,
	)
	return err
}

// AddSyncStep records a finished step of a sync run, after its earlier steps
func (s *Store) AddSyncStep(ctx context.Context, runID int64, step *SyncStep) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sync_steps (run_id, position, name, started_at, finished_at, summary, error)
		VALUES (?, (SELECT COUNT(*) FROM sync_steps WHERE run_id = ?), ?, ?, ?, ?, ?)`,
		runID, runID, step.Name, step.StartedAt.UnixMilli(), step.FinishedAt.UnixMilli(), step.Summary, step.Error,
	)
	return err
}

// PruneSyncRuns deletes all but the newest keep sync runs and their steps
func (s *Store) PruneSyncRuns(ctx context.Context, keep int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	var oldest sql.NullInt64
	err = tx.QueryRowContext(ctx, `SELECT id FROM sync_runs ORDER BY id DESC LIMIT 1 OFFSET ?`, keep-1).Scan(&oldest)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM sync_steps WHERE run_id < ?`, oldest.Int64); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM sync_runs WHERE id < ?`, oldest.Int64); err != nil {
		return err
	}
	return tx.Commit()
}

const syncRunColumns = `id, started_at, finished_at, pull_requests, error`

func scanSyncRun(row interface{ Scan(...any) error }) (*SyncRun, error) {
	run := &SyncRun{}
	var startedAt int64
	var finishedAt sql.NullInt64
	if err := row.Scan(&run.ID, &startedAt, &finishedAt, &run.PullRequests, &run.Error); err != nil {
		return nil, err
	}
	run.StartedAt = time.Unix(startedAt, 0).UTC()
	if finishedAt.Valid {
		t := time.Unix(finishedAt.Int64, 0).UTC()
		run.FinishedAt = &t
	}
	return run, nil
}

// LastSyncRun returns the most recent sync run, or ErrNotFound if none has started
func (s *Store) LastSyncRun(ctx context.Context) (*SyncRun, error) {
	run, err := scanSyncRun(s.db.QueryRowContext(ctx, `SELECT `+syncRunColumns+` FROM sync_runs ORDER BY id DESC LIMIT 1`))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return run, err
}

// ListSyncRuns returns the newest limit sync runs, newest first, with their steps
func (s *Store) ListSyncRuns(ctx context.Context, limit int) ([]*SyncRun, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+syncRunColumns+` FROM sync_runs ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	runs := []*SyncRun{}
	byID := map[int64]*SyncRun{}
	for rows.Next() {
		run, err := scanSyncRun(rows)
		if err != nil {
			return nil, err
		}
		run.Steps = []SyncStep{}
		runs = append(runs, run)
		byID[run.ID] = run
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(runs) == 0 {
		return runs, nil
	}

	steps, err := s.db.QueryContext(ctx, `
		SELECT run_id, name, started_at, finished_at, summary, error FROM sync_steps
		WHERE run_id >= ? ORDER BY run_id, position`, runs[len(runs)-1].ID)
	if err != nil {
		return nil, err
	}
	defer steps.Close()
	for steps.Next() {
		var runID, startedAt, finishedAt int64
		step := SyncStep{}
		if err := steps.Scan(&runID, &step.Name, &startedAt, &finishedAt, &step.Summary, &step.Error); err != nil {
			return nil, err
		}
		step.StartedAt = time.UnixMilli(startedAt).UTC()
		step.FinishedAt = time.UnixMilli(finishedAt).UTC()
		if run := byID[runID]; run != nil {
			run.Steps = append(run.Steps, step)
		}
	}
	return runs, steps.Err()
}

// TagListStats summarizes the cached tag lists of image repositories and charts
type TagListStats struct {
	Repositories int        `json:"repositories"`
	Failed       int        `json:"failed"`
	Oldest       *time.Time `json:"oldest,omitempty"`

	// Failures are the first tag lists that failed, by repository
	Failures []TagListFailure `json:"failures"`
}

type TagListFailure struct {
	Kind       string `json:"kind"`
	Repository string `json:"repository"`
	Error      string `json:"error"`
}

// tagListFailuresShown is how many failures TagListStats lists
const tagListFailuresShown = 10

func (s *Store) TagListStats(ctx context.Context) (*TagListStats, error) {
	stats := &TagListStats{}
	var oldest sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(error != ''), 0), MIN(fetched_at) FROM tag_lists`,
	).Scan(&stats.Repositories, &stats.Failed, &oldest)
	if err != nil {
		return nil, err
	}
	if oldest.Valid {
		t := time.Unix(oldest.Int64, 0).UTC()
		stats.Oldest = &t
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT kind, repository, error FROM tag_lists WHERE error != '' ORDER BY repository LIMIT ?`, tagListFailuresShown)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	stats.Failures = []TagListFailure{}
	for rows.Next() {
		f := TagListFailure{}
		if err := rows.Scan(&f.Kind, &f.Repository, &f.Error); err != nil {
			return nil, err
		}
		stats.Failures = append(stats.Failures, f)
	}
	return stats, rows.Err()
}

// nonNil keeps empty lists as [] rather than null in stored JSON
func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func prKey(repo string, number int) string {
	return fmt.Sprintf("%s#%d", repo, number)
}

const (
	RenovateTriggerSchedule = "schedule"
	RenovateTriggerManual   = "manual"

	RenovateStatusRunning     = "running"
	RenovateStatusSucceeded   = "succeeded"
	RenovateStatusFailed      = "failed"
	RenovateStatusInterrupted = "interrupted"
)

// RenovateRun is one run of Renovate as a Kubernetes Job
type RenovateRun struct {
	ID         int64      `json:"id"`
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`

	// Trigger is RenovateTriggerSchedule or RenovateTriggerManual
	Trigger string   `json:"trigger"`
	Repos   []string `json:"repos"`
	Job     string   `json:"job,omitempty"`
	Status  string   `json:"status"`
	Error   string   `json:"error,omitempty"`

	// Log is the end of the run's log
	Log string `json:"log,omitempty"`
}

// StartRenovateRun records a new run and sets its ID
func (s *Store) StartRenovateRun(ctx context.Context, run *RenovateRun) error {
	repos, err := json.Marshal(nonNil(run.Repos))
	if err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO renovate_runs (started_at, trigger, repos, status) VALUES (?, ?, ?, ?)`,
		run.StartedAt.Unix(), run.Trigger, string(repos), run.Status,
	)
	if err != nil {
		return err
	}
	run.ID, err = res.LastInsertId()
	return err
}

func (s *Store) SetRenovateRunJob(ctx context.Context, id int64, job string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE renovate_runs SET job = ? WHERE id = ?`, job, id)
	return err
}

// FinishRenovateRun records a run's result and deletes all but the newest renovateRunsKept runs
func (s *Store) FinishRenovateRun(ctx context.Context, run *RenovateRun) error {
	var finishedAt *int64
	if run.FinishedAt != nil {
		t := run.FinishedAt.Unix()
		finishedAt = &t
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE renovate_runs SET finished_at = ?, status = ?, error = ?, log = ? WHERE id = ?`,
		finishedAt, run.Status, run.Error, run.Log, run.ID,
	)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		DELETE FROM renovate_runs WHERE id <= (SELECT id FROM renovate_runs ORDER BY id DESC LIMIT 1 OFFSET ?)`,
		renovateRunsKept,
	)
	return err
}

// renovateRunsKept is how many Renovate runs FinishRenovateRun keeps
const renovateRunsKept = 50

// InterruptRenovateRuns finishes the runs left unfinished by a daemon that stopped mid-run.
// Their Jobs may still be running; they are left to finish on their own.
func (s *Store) InterruptRenovateRuns(ctx context.Context, finishedAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE renovate_runs SET finished_at = ?, status = ?, error = ? WHERE finished_at IS NULL`,
		finishedAt.Unix(), RenovateStatusInterrupted, "the daemon stopped during this run; its job was left to finish",
	)
	return err
}

// ListRenovateRuns returns the newest limit runs, newest first
func (s *Store) ListRenovateRuns(ctx context.Context, limit int) ([]*RenovateRun, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, started_at, finished_at, trigger, repos, job, status, error, log
		FROM renovate_runs ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	runs := []*RenovateRun{}
	for rows.Next() {
		run := &RenovateRun{}
		var startedAt int64
		var finishedAt sql.NullInt64
		var repos string
		if err := rows.Scan(&run.ID, &startedAt, &finishedAt, &run.Trigger, &repos, &run.Job, &run.Status, &run.Error, &run.Log); err != nil {
			return nil, err
		}
		run.StartedAt = time.Unix(startedAt, 0).UTC()
		if finishedAt.Valid {
			t := time.Unix(finishedAt.Int64, 0).UTC()
			run.FinishedAt = &t
		}
		if err := json.Unmarshal([]byte(repos), &run.Repos); err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

// ReleaseDate is when an image tag was released: its build date from the registry, if real,
// and when the daemon first saw it
type ReleaseDate struct {
	Ref string `json:"ref"`

	// Created is zero when unknown, e.g. for reproducible builds dated 1970
	Created   time.Time `json:"created,omitempty"`
	FirstSeen time.Time `json:"firstSeen"`
	Error     string    `json:"error,omitempty"`
	CheckedAt time.Time `json:"checkedAt"`
}

// Released is the build date when known, else when the tag was first seen
func (r *ReleaseDate) Released() time.Time {
	if !r.Created.IsZero() {
		return r.Created
	}
	return r.FirstSeen
}

// SaveReleaseDate stores a release date, keeping the first time the ref was seen
func (s *Store) SaveReleaseDate(ctx context.Context, r *ReleaseDate) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO release_dates (ref, created, first_seen, error, checked_at) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (ref) DO UPDATE SET created = excluded.created, error = excluded.error, checked_at = excluded.checked_at`,
		r.Ref, unixOrZero(r.Created), r.FirstSeen.Unix(), r.Error, r.CheckedAt.Unix(),
	)
	return err
}

// ListReleaseDates returns the release dates by ref
func (s *Store) ListReleaseDates(ctx context.Context) (map[string]*ReleaseDate, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT ref, created, first_seen, error, checked_at FROM release_dates`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	dates := map[string]*ReleaseDate{}
	for rows.Next() {
		r := &ReleaseDate{}
		var created, firstSeen, checkedAt int64
		if err := rows.Scan(&r.Ref, &created, &firstSeen, &r.Error, &checkedAt); err != nil {
			return nil, err
		}
		if created != 0 {
			r.Created = time.Unix(created, 0).UTC()
		}
		r.FirstSeen = time.Unix(firstSeen, 0).UTC()
		r.CheckedAt = time.Unix(checkedAt, 0).UTC()
		dates[r.Ref] = r
	}
	return dates, rows.Err()
}
