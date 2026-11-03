package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/scan"
)

// Reasons a finding was closed
const (
	// CloseReasonUpdated: the workload now runs a digest without the vulnerability
	CloseReasonUpdated = "updated"
	// CloseReasonRemoved: the workload container no longer runs
	CloseReasonRemoved = "workload_removed"
	// CloseReasonWithdrawn: the same digest no longer has the vulnerability, i.e. the
	// vulnerability database dropped or changed it
	CloseReasonWithdrawn = "withdrawn"
)

// Slot is a workload container, which keeps its identity as the digest it runs changes
type Slot struct {
	Cluster      string `json:"cluster"`
	Namespace    string `json:"namespace"`
	WorkloadKind string `json:"workloadKind"`
	WorkloadName string `json:"workloadName"`
	Container    string `json:"container"`
}

// Finding is a vulnerability running in a slot, from when it was first seen there until it
// wasn't. Once closed it stays closed: a vulnerability that comes back opens a new finding.
type Finding struct {
	ID int64 `json:"id"`
	Slot
	VulnID   string `json:"vulnId"`
	Package  string `json:"package"`
	Severity string `json:"severity"`

	// Digest is the last digest the vulnerability was seen in
	Digest string `json:"digest"`

	FirstSeen time.Time `json:"firstSeen"`
	// FixAvailable is when a fixed version was first reported (zero if none yet)
	FixAvailable time.Time `json:"fixAvailable"`
	LastSeen     time.Time `json:"lastSeen"`
	// ClosedAt is zero while the finding is open
	ClosedAt    time.Time `json:"closedAt"`
	CloseReason string    `json:"closeReason,omitempty"`
}

func (f *Finding) key() string {
	return findingKey(f.Slot, f.VulnID, f.Package)
}

func findingKey(slot Slot, vulnID, pkg string) string {
	return slot.Namespace + "|" + slot.WorkloadKind + "|" + slot.WorkloadName + "|" + slot.Container + "|" + vulnID + "|" + pkg
}

// ReconcileFindings brings a cluster's open findings in line with what runs in it: the
// vulnerabilities of each digest running in each of its slots. A slot mapped to nil isn't
// known yet (a digest not scanned, say) and its findings are left as they are; open findings
// of slots missing from running are closed. The slots must all be of the cluster. It returns the number of findings opened and closed.
func (s *Store) ReconcileFindings(ctx context.Context, cluster string, running map[Slot]map[string][]scan.Vulnerability, at time.Time) (opened, closed int, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback() //nolint:errcheck

	open, err := queryFindings(ctx, tx, `WHERE cluster = ? AND closed_at = 0`, cluster)
	if err != nil {
		return 0, 0, err
	}
	byKey := map[string]*Finding{}
	for i := range open {
		byKey[open[i].key()] = &open[i]
	}

	seen := map[string]bool{}
	for slot, digests := range running {
		for digest, vulns := range digests {
			for _, v := range vulns {
				key := findingKey(slot, v.ID, v.Package)
				if seen[key] {
					continue
				}
				seen[key] = true

				fixAvailable := int64(0)
				if v.Fixed != "" {
					fixAvailable = at.Unix()
				}
				if f, ok := byKey[key]; ok {
					// Keep the digest it was seen in while that still runs, so a closing can tell an update from a withdrawal
					if _, ok := digests[f.Digest]; !ok {
						f.Digest = digest
					}
					if _, err := tx.ExecContext(ctx, `
						UPDATE findings SET severity = ?, digest = ?, last_seen = ?,
							fix_available = CASE WHEN fix_available = 0 THEN ? ELSE fix_available END
						WHERE id = ?`,
						v.Severity, f.Digest, at.Unix(), fixAvailable, f.ID,
					); err != nil {
						return 0, 0, err
					}
					continue
				}
				if _, err := tx.ExecContext(ctx, `
					INSERT INTO findings (cluster, namespace, workload_kind, workload_name, container, vuln_id, package, severity, digest,
						first_seen, fix_available, last_seen, closed_at, close_reason)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, '')`,
					cluster, slot.Namespace, slot.WorkloadKind, slot.WorkloadName, slot.Container, v.ID, v.Package, v.Severity, digest,
					at.Unix(), fixAvailable, at.Unix(),
				); err != nil {
					return 0, 0, err
				}
				opened++
			}
		}
	}

	for _, f := range open {
		if seen[f.key()] {
			continue
		}
		digests, ok := running[f.Slot]
		reason := CloseReasonRemoved
		switch {
		case ok && digests == nil:
			continue
		case ok:
			reason = CloseReasonUpdated
			if _, ok := digests[f.Digest]; ok {
				reason = CloseReasonWithdrawn
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE findings SET closed_at = ?, close_reason = ? WHERE id = ?`, at.Unix(), reason, f.ID); err != nil {
			return 0, 0, err
		}
		closed++
	}

	return opened, closed, tx.Commit()
}

// ListFindings returns every finding, open and closed
func (s *Store) ListFindings(ctx context.Context) ([]Finding, error) {
	return queryFindings(ctx, s.db, ``)
}

// CountOpenFindings returns the number of open findings
func (s *Store) CountOpenFindings(ctx context.Context) (int, error) {
	n := 0
	return n, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM findings WHERE closed_at = 0`).Scan(&n)
}

func queryFindings(ctx context.Context, q querier, where string, args ...any) ([]Finding, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT id, cluster, namespace, workload_kind, workload_name, container, vuln_id, package, severity, digest,
			first_seen, fix_available, last_seen, closed_at, close_reason
		FROM findings `+where+` ORDER BY id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	findings := []Finding{}
	for rows.Next() {
		f := Finding{}
		var firstSeen, fixAvailable, lastSeen, closedAt int64
		if err := rows.Scan(&f.ID, &f.Cluster, &f.Namespace, &f.WorkloadKind, &f.WorkloadName, &f.Container, &f.VulnID, &f.Package,
			&f.Severity, &f.Digest, &firstSeen, &fixAvailable, &lastSeen, &closedAt, &f.CloseReason); err != nil {
			return nil, err
		}
		f.FirstSeen = timeOrZero(firstSeen)
		f.FixAvailable = timeOrZero(fixAvailable)
		f.LastSeen = timeOrZero(lastSeen)
		f.ClosedAt = timeOrZero(closedAt)
		findings = append(findings, f)
	}
	return findings, rows.Err()
}

// querier is a *sql.DB or *sql.Tx
type querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}
