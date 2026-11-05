package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// Why a history entry ended
const (
	// EndReasonStopped: the container finished, or its pod did
	EndReasonStopped = "stopped"
	// EndReasonDeleted: the pod was deleted while the daemon was watching
	EndReasonDeleted = "deleted"
	// EndReasonReplaced: the container in the same pod now runs a different image or digest
	EndReasonReplaced = "replaced"
	// EndReasonNeverStarted: the pod went away before the container ran, e.g. a failed image pull
	EndReasonNeverStarted = "never_started"
	// EndReasonUnobserved: it ended while the daemon wasn't watching, so EndedAt is only when
	// that was noticed
	EndReasonUnobserved = "unobserved"
)

// historyLimitMax caps the entries one query returns
const historyLimitMax = 5000

// HistoryEntry is a stretch of time one image digest was present in a container of a pod.
// Entries are never deleted, and a pod's containers keep their identity by the pod's UID.
type HistoryEntry struct {
	ID            int64  `json:"id"`
	Cluster       string `json:"cluster"`
	Namespace     string `json:"namespace"`
	WorkloadKind  string `json:"workloadKind"`
	WorkloadName  string `json:"workloadName"`
	Pod           string `json:"pod"`
	PodUID        string `json:"podUID"`
	Node          string `json:"node,omitempty"`
	Container     string `json:"container"`
	ContainerType string `json:"containerType,omitempty"`

	// Image is the reference from the pod spec, and Repository its normalized repository
	Image      string `json:"image"`
	Repository string `json:"repository"`

	// Digest is empty until the container status reports it
	Digest  string `json:"digest,omitempty"`
	ArgoApp string `json:"argoApp,omitempty"`

	// Ran is whether the container is known to have started
	Ran bool `json:"ran"`

	// StartedAt is when Kubernetes says the container first started, else when it was first seen
	StartedAt time.Time `json:"startedAt"`
	// RecordedAt is when the daemon first saw it
	RecordedAt time.Time `json:"recordedAt"`
	// EndedAt is zero while the entry is open
	EndedAt   time.Time `json:"endedAt"`
	EndReason string    `json:"endReason,omitempty"`
}

// ObservePod records the current state of a pod's containers. Observing the same state again
// changes nothing, so it's safe on every pod update. In each entry StartedAt is zero until the
// container has started, and a non-zero EndedAt means it has finished.
func (s *Store) ObservePod(ctx context.Context, entries []HistoryEntry, at time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	for i := range entries {
		if err := observeContainer(ctx, tx, &entries[i], at); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func observeContainer(ctx context.Context, tx *sql.Tx, e *HistoryEntry, at time.Time) error {
	var id, startedAt int64
	var image, digest, node string
	var ran bool
	err := tx.QueryRowContext(ctx, `
		SELECT id, image, digest, node, ran, started_at FROM image_history
		WHERE cluster = ? AND pod_uid = ? AND container = ? AND ended_at = 0`,
		e.Cluster, e.PodUID, e.Container,
	).Scan(&id, &image, &digest, &node, &ran, &startedAt)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	open := err == nil

	if open && image == e.Image && (digest == e.Digest || digest == "" || e.Digest == "") {
		return updateOpen(ctx, tx, id, e, digest, node, ran, startedAt)
	}
	if open {
		// The digest can change in the same pod: a mutable tag re-pulled on restart, or a patched image
		replacedAt := at
		if !e.StartedAt.IsZero() {
			replacedAt = e.StartedAt
		}
		if _, err := tx.ExecContext(ctx, `UPDATE image_history SET ended_at = ?, end_reason = ? WHERE id = ?`,
			replacedAt.Unix(), EndReasonReplaced, id); err != nil {
			return err
		}
	} else {
		seen, err := alreadyRecorded(ctx, tx, e)
		if err != nil || seen {
			return err
		}
	}

	started, endedAt, endReason := at, int64(0), ""
	if !e.StartedAt.IsZero() {
		started = e.StartedAt
	}
	if !e.EndedAt.IsZero() {
		endedAt, endReason = e.EndedAt.Unix(), EndReasonStopped
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO image_history (cluster, namespace, workload_kind, workload_name, pod, pod_uid, node, container,
			container_type, image, repository, digest, argo_app, ran, started_at, recorded_at, ended_at, end_reason)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.Cluster, e.Namespace, e.WorkloadKind, e.WorkloadName, e.Pod, e.PodUID, e.Node, e.Container,
		e.ContainerType, e.Image, e.Repository, e.Digest, e.ArgoApp, !e.StartedAt.IsZero(), started.Unix(), at.Unix(), endedAt, endReason,
	)
	return err
}

// updateOpen fills in what's newly known about an open entry, writing only if something changed.
// The earliest start is kept, so a restarting container stays one entry.
func updateOpen(ctx context.Context, tx *sql.Tx, id int64, e *HistoryEntry, digest, node string, ran bool, startedAt int64) error {
	newDigest, newNode, newRan, newStarted := digest, node, ran, startedAt
	if digest == "" {
		newDigest = e.Digest
	}
	if node == "" {
		newNode = e.Node
	}
	if !e.StartedAt.IsZero() {
		newRan = true
		if !ran || e.StartedAt.Unix() < startedAt {
			newStarted = e.StartedAt.Unix()
		}
	}
	endedAt, endReason := int64(0), ""
	if !e.EndedAt.IsZero() {
		endedAt, endReason = e.EndedAt.Unix(), EndReasonStopped
	}
	if newDigest == digest && newNode == node && newRan == ran && newStarted == startedAt && endedAt == 0 {
		return nil
	}
	_, err := tx.ExecContext(ctx, `
		UPDATE image_history SET digest = ?, node = ?, ran = ?, started_at = ?, ended_at = ?, end_reason = ? WHERE id = ?`,
		newDigest, newNode, newRan, newStarted, endedAt, endReason, id)
	return err
}

// alreadyRecorded reports whether a container with no open entry is one that has ended, as
// later updates of a finished pod keep reporting it
func alreadyRecorded(ctx context.Context, tx *sql.Tx, e *HistoryEntry) (bool, error) {
	where := `cluster = ? AND pod_uid = ? AND container = ? AND image = ?`
	args := []any{e.Cluster, e.PodUID, e.Container, e.Image}
	switch {
	case e.StartedAt.IsZero():
	case e.EndedAt.IsZero():
		where += ` AND started_at = ?`
		args = append(args, e.StartedAt.Unix())
	default:
		where += ` AND (started_at = ? OR ended_at = ?)`
		args = append(args, e.StartedAt.Unix(), e.EndedAt.Unix())
	}
	var seen int
	err := tx.QueryRowContext(ctx, `SELECT 1 FROM image_history WHERE `+where+` LIMIT 1`, args...).Scan(&seen)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// EndPod ends a pod's open entries. Containers that never ran get EndReasonNeverStarted.
func (s *Store) EndPod(ctx context.Context, cluster, podUID string, at time.Time, reason string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE image_history SET ended_at = ?, end_reason = CASE WHEN ran THEN ? ELSE ? END
		WHERE cluster = ? AND pod_uid = ? AND ended_at = 0`,
		at.Unix(), reason, EndReasonNeverStarted, cluster, podUID)
	return err
}

// EndMissingPods ends the open entries of pods no longer in the cluster, which went away while
// the daemon wasn't watching. Only entries recorded before the live pods were listed are
// considered, as newer ones may be of pods created since. It returns how many pods it ended.
func (s *Store) EndMissingPods(ctx context.Context, cluster string, live map[string]bool, listedAt, at time.Time) (int, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT pod_uid FROM image_history WHERE cluster = ? AND ended_at = 0 AND recorded_at < ?`,
		cluster, listedAt.Unix())
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	missing := []string{}
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			return 0, err
		}
		if !live[uid] {
			missing = append(missing, uid)
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	rows.Close()

	for _, uid := range missing {
		if err := s.EndPod(ctx, cluster, uid, at, EndReasonUnobserved); err != nil {
			return 0, err
		}
	}
	return len(missing), nil
}

// HistoryQuery selects history entries; every field is optional
type HistoryQuery struct {
	// Repository matches a normalized repository exactly
	Repository string
	// Text matches part of the image, digest, namespace, workload, pod or node
	Text string
	// At selects entries present at this time
	At time.Time
	// Limit defaults to, and is capped at, historyLimitMax
	Limit int
}

// ListHistory returns matching entries, most recently started first
func (s *Store) ListHistory(ctx context.Context, q HistoryQuery) ([]HistoryEntry, error) {
	where, args := []string{"1 = 1"}, []any{}
	if q.Repository != "" {
		where = append(where, "repository = ?")
		args = append(args, q.Repository)
	}
	if q.Text != "" {
		text := strings.ToLower(q.Text)
		fields := []string{"image", "digest", "namespace", "workload_name", "pod", "node"}
		match := make([]string, len(fields))
		for i, f := range fields {
			match[i] = "instr(lower(" + f + "), ?) > 0"
			args = append(args, text)
		}
		where = append(where, "("+strings.Join(match, " OR ")+")")
	}
	if !q.At.IsZero() {
		where = append(where, "started_at <= ? AND (ended_at = 0 OR ended_at >= ?)")
		args = append(args, q.At.Unix(), q.At.Unix())
	}
	limit := q.Limit
	if limit <= 0 || limit > historyLimitMax {
		limit = historyLimitMax
	}
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, cluster, namespace, workload_kind, workload_name, pod, pod_uid, node, container, container_type,
			image, repository, digest, argo_app, ran, started_at, recorded_at, ended_at, end_reason
		FROM image_history WHERE `+strings.Join(where, " AND ")+`
		ORDER BY started_at DESC, id DESC LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []HistoryEntry{}
	for rows.Next() {
		var e HistoryEntry
		var startedAt, recordedAt, endedAt int64
		if err := rows.Scan(&e.ID, &e.Cluster, &e.Namespace, &e.WorkloadKind, &e.WorkloadName, &e.Pod, &e.PodUID, &e.Node,
			&e.Container, &e.ContainerType, &e.Image, &e.Repository, &e.Digest, &e.ArgoApp, &e.Ran,
			&startedAt, &recordedAt, &endedAt, &e.EndReason); err != nil {
			return nil, err
		}
		e.StartedAt, e.RecordedAt = time.Unix(startedAt, 0).UTC(), time.Unix(recordedAt, 0).UTC()
		if endedAt != 0 {
			e.EndedAt = time.Unix(endedAt, 0).UTC()
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// HistoryCoverage is a stretch of time a cluster's pods were being watched. Anything outside
// these stretches may have run unrecorded.
type HistoryCoverage struct {
	Cluster   string    `json:"cluster"`
	StartedAt time.Time `json:"startedAt"`
	LastSeen  time.Time `json:"lastSeen"`
}

// StartCoverage opens a stretch of coverage, returning its ID for TouchCoverage
func (s *Store) StartCoverage(ctx context.Context, cluster string, at time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO history_coverage (cluster, started_at, last_seen) VALUES (?, ?, ?)`,
		cluster, at.Unix(), at.Unix())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// TouchCoverage extends a stretch of coverage up to at
func (s *Store) TouchCoverage(ctx context.Context, id int64, at time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE history_coverage SET last_seen = ? WHERE id = ?`, at.Unix(), id)
	return err
}

// ListCoverage returns every stretch of coverage, oldest first per cluster
func (s *Store) ListCoverage(ctx context.Context) ([]HistoryCoverage, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT cluster, started_at, last_seen FROM history_coverage ORDER BY cluster, started_at, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	coverage := []HistoryCoverage{}
	for rows.Next() {
		var c HistoryCoverage
		var startedAt, lastSeen int64
		if err := rows.Scan(&c.Cluster, &startedAt, &lastSeen); err != nil {
			return nil, err
		}
		c.StartedAt, c.LastSeen = time.Unix(startedAt, 0).UTC(), time.Unix(lastSeen, 0).UTC()
		coverage = append(coverage, c)
	}
	return coverage, rows.Err()
}
