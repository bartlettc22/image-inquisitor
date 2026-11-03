package api

import (
	"testing"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/scan"
	"github.com/bartlettc22/image-inquisitor/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrends(t *testing.T) {
	// Wednesday noon; the window is Monday to today
	now := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	monday := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	at := func(days, hours int) time.Time {
		return monday.Add(time.Duration(days)*day + time.Duration(hours)*time.Hour)
	}
	finding := func(sev string, first, fix, closed time.Time, reason string) store.Finding {
		return store.Finding{Severity: sev, FirstSeen: first, FixAvailable: fix, LastSeen: first, ClosedAt: closed, CloseReason: reason}
	}

	findings := []store.Finding{
		// Running when recording began, Sunday: not counted as opened
		finding(scan.SeverityCritical, at(-1, 6), at(-1, 6), at(1, 6), store.CloseReasonUpdated),
		finding(scan.SeverityHigh, at(-1, 6), time.Time{}, time.Time{}, ""),
		// Opened Monday with a fix, still open but within the 30-day SLA
		finding(scan.SeverityHigh, at(0, 12), at(0, 12), time.Time{}, ""),
		// Critical with a fix for 9 days: 2 days overdue
		finding(scan.SeverityCritical, at(-1, 6), at(-7, 12), time.Time{}, ""),
		// Removed, not fixed
		finding(scan.SeverityMedium, at(0, 1), time.Time{}, at(2, 1), store.CloseReasonRemoved),
		// Fixed without Trivy ever reporting a fix: counted from first seen
		finding(scan.SeverityCritical, at(0, 0), time.Time{}, at(1, 0), store.CloseReasonUpdated),
	}

	tr := trends(findings, now, 3)

	require.NotNil(t, tr.RecordedSince)
	assert.Equal(t, at(-1, 6), *tr.RecordedSince)
	assert.Equal(t, monday, tr.Since)

	require.Len(t, tr.Days, 3)
	assert.Equal(t, "2026-11-02", tr.Days[0].Date)
	assert.Equal(t, scan.Counts{"CRITICAL": 3, "HIGH": 2, "MEDIUM": 1}, tr.Days[0].Open)
	assert.Equal(t, 3, tr.Days[0].Fixable)
	assert.Equal(t, scan.Counts{"CRITICAL": 1, "HIGH": 2, "MEDIUM": 1}, tr.Days[1].Open, "two criticals fixed Tuesday")
	assert.Equal(t, scan.Counts{"CRITICAL": 1, "HIGH": 2}, tr.Days[2].Open, "today is counted at now")

	assert.Equal(t, []TrendWeek{{Start: "2026-11-02", Opened: 3, Fixed: 2}}, tr.Weeks)

	require.Len(t, tr.TimeToFix, 1)
	ttf := tr.TimeToFix[0]
	assert.Equal(t, scan.SeverityCritical, ttf.Severity)
	assert.Equal(t, 2, ttf.Fixed)
	assert.Equal(t, 2, ttf.WithinSLA)
	assert.InDelta(t, 1.5, ttf.Median, 0.001, "two days from fix available and one from first seen")
	assert.InDelta(t, 1.9, ttf.P90, 0.001)

	assert.Equal(t, 1, tr.Overdue.Total)
	assert.Equal(t, scan.Counts{"CRITICAL": 1}, tr.Overdue.BySeverity)
	assert.InDelta(t, 2.0, tr.Overdue.Findings[0].DaysOverdue, 0.001)
}

func TestWeekStart(t *testing.T) {
	assert.Equal(t, "2026-11-02", weekStart(time.Date(2026, 11, 8, 23, 0, 0, 0, time.UTC)).Format(time.DateOnly), "Sunday")
	assert.Equal(t, "2026-11-02", weekStart(time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)).Format(time.DateOnly), "Monday")
}
