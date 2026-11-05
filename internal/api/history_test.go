package api

import (
	"testing"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClusterCoverage(t *testing.T) {
	at := time.Unix(1_700_000_000, 0).UTC()
	now := at.Add(10 * time.Hour)
	stored := []store.HistoryCoverage{
		{Cluster: "down", StartedAt: at, LastSeen: at.Add(time.Hour)},
		{Cluster: "home", StartedAt: at, LastSeen: at.Add(time.Hour)},
		{Cluster: "home", StartedAt: at.Add(3 * time.Hour), LastSeen: now.Add(-time.Minute)},
	}

	coverage := clusterCoverage([]string{"down", "home", "never"}, stored, now)
	require.Len(t, coverage, 3)

	assert.False(t, coverage[0].Recording, "last checked in hours ago")
	assert.Empty(t, coverage[0].Gaps)

	assert.True(t, coverage[1].Recording)
	assert.Equal(t, at, *coverage[1].Since)
	assert.Equal(t, []CoverageGap{{From: at.Add(time.Hour), To: at.Add(3 * time.Hour)}}, coverage[1].Gaps)

	assert.Equal(t, ClusterCoverage{Cluster: "never", Gaps: []CoverageGap{}}, coverage[2])
}
