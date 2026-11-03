package daemon

import (
	"context"
	"fmt"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/scan"
	"github.com/bartlettc22/image-inquisitor/internal/store"
	"github.com/bartlettc22/image-inquisitor/internal/workloads"
)

// reconcileFindings records which vulnerabilities run in each workload container, opening
// and closing findings, for the history of how long vulnerabilities take to be fixed.
// Clusters whose inventory failed are skipped, so a failure isn't taken for everything being fixed.
func (d *Daemon) reconcileFindings(ctx context.Context) (string, error) {
	clusters, err := d.store.ListClusters(ctx)
	if err != nil {
		return "", err
	}
	containers, err := d.store.ListContainers(ctx)
	if err != nil {
		return "", err
	}
	scans, err := d.store.ListScans(ctx)
	if err != nil {
		return "", err
	}

	scanned := map[string]bool{}
	for _, sc := range scans {
		scanned[sc.Digest] = sc.Error == "" && sc.Skipped == "" && !sc.SBOMAt.IsZero()
	}
	running, err := runningVulnerabilities(containers, scanned, func(digest string) ([]scan.Vulnerability, error) {
		return d.store.GetScanVulnerabilities(ctx, digest)
	})
	if err != nil {
		return "", err
	}

	syncedOK := map[string]bool{}
	for _, cl := range clusters {
		syncedOK[cl.Name] = cl.Error == ""
	}

	at := time.Now().UTC()
	opened, closed := 0, 0
	for _, cl := range d.clusters {
		name := cl.config.Name
		if !syncedOK[name] {
			continue
		}
		o, c, err := d.store.ReconcileFindings(ctx, name, running[name], at)
		if err != nil {
			return "", err
		}
		opened, closed = opened+o, closed+c
	}

	open, err := d.store.CountOpenFindings(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d open, %d opened, %d closed", open, opened, closed), nil
}

// runningVulnerabilities groups the vulnerabilities of what runs by cluster, slot and digest.
// A slot with a digest that isn't known or cleanly scanned maps to nil: its vulnerabilities aren't known.
func runningVulnerabilities(containers []workloads.Container, scanned map[string]bool, vulns func(digest string) ([]scan.Vulnerability, error)) (map[string]map[store.Slot]map[string][]scan.Vulnerability, error) {
	byDigest := map[string][]scan.Vulnerability{}
	unknown := map[store.Slot]bool{}
	running := map[string]map[store.Slot]map[string][]scan.Vulnerability{}

	for _, c := range containers {
		slot := store.Slot{Cluster: c.Cluster, Namespace: c.Namespace, WorkloadKind: c.WorkloadKind, WorkloadName: c.WorkloadName, Container: c.Container}
		if running[c.Cluster] == nil {
			running[c.Cluster] = map[store.Slot]map[string][]scan.Vulnerability{}
		}
		if running[c.Cluster][slot] == nil {
			running[c.Cluster][slot] = map[string][]scan.Vulnerability{}
		}
		if c.Digest == "" || !scanned[c.Digest] {
			unknown[slot] = true
			continue
		}
		if _, ok := byDigest[c.Digest]; !ok {
			v, err := vulns(c.Digest)
			if err != nil {
				return nil, err
			}
			byDigest[c.Digest] = v
		}
		running[c.Cluster][slot][c.Digest] = byDigest[c.Digest]
	}

	for slot := range unknown {
		running[slot.Cluster][slot] = nil
	}
	return running, nil
}
