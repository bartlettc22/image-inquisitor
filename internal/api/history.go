package api

import (
	"net/http"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/store"
)

const (
	// historyLimit is how many entries one response shows
	historyLimit = 1000

	// recordingWithin is how recently a cluster's watch must have checked in to count as
	// recording: a few of the daemon's one-minute heartbeats
	recordingWithin = 3 * time.Minute
)

// HistoryResponse is the image history matching a query, and how far it can be trusted
type HistoryResponse struct {
	Entries []store.HistoryEntry `json:"entries"`

	// Truncated is whether there were more matches than Entries holds
	Truncated bool `json:"truncated"`

	Coverage []ClusterCoverage `json:"coverage"`
}

// ClusterCoverage is the time a cluster's pods were watched
type ClusterCoverage struct {
	Cluster string `json:"cluster"`

	// Recording is whether the watch is running now
	Recording bool `json:"recording"`

	// Since is when recording first began and LastSeen when it last checked in; both unset
	// if it never has
	Since    *time.Time `json:"since,omitempty"`
	LastSeen *time.Time `json:"lastSeen,omitempty"`

	// Gaps are the times since then that pods weren't watched, oldest first
	Gaps []CoverageGap `json:"gaps"`
}

// CoverageGap is a stretch of time containers may have run without being recorded
type CoverageGap struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// clusterCoverage summarizes the stored coverage, which is sorted by cluster then time. Clusters
// lists the configured clusters, so one that never recorded is still reported; a cluster no
// longer configured keeps its history and its coverage.
func clusterCoverage(clusters []string, stored []store.HistoryCoverage, now time.Time) []ClusterCoverage {
	byCluster := map[string]*ClusterCoverage{}
	result := []*ClusterCoverage{}
	get := func(name string) *ClusterCoverage {
		c, ok := byCluster[name]
		if !ok {
			c = &ClusterCoverage{Cluster: name, Gaps: []CoverageGap{}}
			byCluster[name] = c
			result = append(result, c)
		}
		return c
	}
	for _, name := range clusters {
		get(name)
	}
	for _, s := range stored {
		c := get(s.Cluster)
		if c.Since == nil {
			since := s.StartedAt
			c.Since = &since
		} else if s.StartedAt.After(*c.LastSeen) {
			c.Gaps = append(c.Gaps, CoverageGap{From: *c.LastSeen, To: s.StartedAt})
		}
		if c.LastSeen == nil || s.LastSeen.After(*c.LastSeen) {
			lastSeen := s.LastSeen
			c.LastSeen = &lastSeen
		}
	}

	coverage := make([]ClusterCoverage, 0, len(result))
	for _, c := range result {
		c.Recording = c.LastSeen != nil && now.Sub(*c.LastSeen) < recordingWithin
		coverage = append(coverage, *c)
	}
	return coverage
}

// history returns image history entries, by repository, text and the time they were present
func (h *handler) history(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	query := store.HistoryQuery{Text: params.Get("q"), Limit: historyLimit + 1}
	if repository := params.Get("repository"); repository != "" {
		ref, ok := parseImage(repository)
		if !ok {
			http.Error(w, "invalid image repository", http.StatusBadRequest)
			return
		}
		query.Repository = ref.repository
	}
	if at := params.Get("at"); at != "" {
		t, err := time.Parse(time.RFC3339, at)
		if err != nil {
			http.Error(w, "invalid time, expected RFC 3339", http.StatusBadRequest)
			return
		}
		query.At = t
	}

	entries, err := h.store.ListHistory(r.Context(), query)
	if err != nil {
		writeError(w, err)
		return
	}
	stored, err := h.store.ListCoverage(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	truncated := len(entries) > historyLimit
	if truncated {
		entries = entries[:historyLimit]
	}
	writeJSON(w, http.StatusOK, HistoryResponse{
		Entries:   entries,
		Truncated: truncated,
		Coverage:  clusterCoverage(h.clusters, stored, time.Now().UTC()),
	})
}
