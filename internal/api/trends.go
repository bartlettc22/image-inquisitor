package api

import (
	"cmp"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/scan"
	"github.com/bartlettc22/image-inquisitor/internal/store"
)

const (
	day = 24 * time.Hour

	defaultTrendDays = 90
	maxTrendDays     = 365

	// overdueLimit is how many overdue findings are listed, most overdue first
	overdueLimit = 25
)

// slaDays is how long a finding of each severity may run once a fix is available
var slaDays = map[string]int{
	scan.SeverityCritical: 7,
	scan.SeverityHigh:     30,
	scan.SeverityMedium:   90,
	scan.SeverityLow:      180,
}

// Trends is the history of the vulnerabilities running in workload containers, from findings
type Trends struct {
	// Since is the first day of the window; RecordedSince when findings were first recorded (nil if never)
	Since         time.Time  `json:"since"`
	RecordedSince *time.Time `json:"recordedSince,omitempty"`

	SLADays map[string]int `json:"slaDays"`

	Days      []TrendDay      `json:"days"`
	Weeks     []TrendWeek     `json:"weeks"`
	TimeToFix []TimeToFix     `json:"timeToFix"`
	Overdue   OverdueFindings `json:"overdue"`
}

// TrendDay is the number of findings open at the end of a day (now, for today)
type TrendDay struct {
	Date    string      `json:"date"`
	Open    scan.Counts `json:"open"`
	Fixable int         `json:"fixable"`
}

// TrendWeek counts the findings opened and fixed in a week starting Monday. Findings that
// were already running when recording began aren't counted as opened.
type TrendWeek struct {
	Start  string `json:"start"`
	Opened int    `json:"opened"`
	Fixed  int    `json:"fixed"`
}

// TimeToFix is how long the findings of a severity fixed in the window ran, counted from when
// a fix was available (or first seen, if Trivy never reported one), in days
type TimeToFix struct {
	Severity  string  `json:"severity"`
	Fixed     int     `json:"fixed"`
	WithinSLA int     `json:"withinSLA"`
	Median    float64 `json:"median"`
	P90       float64 `json:"p90"`
}

type OverdueFindings struct {
	Total      int              `json:"total"`
	BySeverity scan.Counts      `json:"bySeverity"`
	Findings   []OverdueFinding `json:"findings"`
}

type OverdueFinding struct {
	store.Finding
	DaysOverdue float64 `json:"daysOverdue"`
}

func (h *handler) trends(w http.ResponseWriter, r *http.Request) {
	days := defaultTrendDays
	if s := r.URL.Query().Get("days"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 || n > maxTrendDays {
			http.Error(w, "days must be 1 to "+strconv.Itoa(maxTrendDays), http.StatusBadRequest)
			return
		}
		days = n
	}
	findings, err := h.store.ListFindings(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, trends(findings, time.Now().UTC(), days))
}

func trends(findings []store.Finding, now time.Time, days int) *Trends {
	today := now.Truncate(day)
	since := today.Add(-time.Duration(days-1) * day)
	t := &Trends{Since: since, SLADays: slaDays, Days: []TrendDay{}, Weeks: []TrendWeek{}, TimeToFix: []TimeToFix{}}

	var recordedDay time.Time
	for _, f := range findings {
		if t.RecordedSince == nil || f.FirstSeen.Before(*t.RecordedSince) {
			first := f.FirstSeen
			t.RecordedSince = &first
		}
	}
	if t.RecordedSince != nil {
		recordedDay = t.RecordedSince.Truncate(day)
	}

	for d := since; !d.After(today); d = d.Add(day) {
		at := d.Add(day - time.Second)
		if at.After(now) {
			at = now
		}
		td := TrendDay{Date: d.Format(time.DateOnly), Open: scan.Counts{}}
		for _, f := range findings {
			if openAt(f, at) {
				td.Open[f.Severity]++
				if !f.FixAvailable.IsZero() && !f.FixAvailable.After(at) {
					td.Fixable++
				}
			}
		}
		t.Days = append(t.Days, td)
	}

	weekIdx := map[time.Time]int{}
	for d := since; !d.After(today); d = d.Add(day) {
		if start := weekStart(d); len(t.Weeks) == 0 || t.Weeks[len(t.Weeks)-1].Start != start.Format(time.DateOnly) {
			weekIdx[start] = len(t.Weeks)
			t.Weeks = append(t.Weeks, TrendWeek{Start: start.Format(time.DateOnly)})
		}
	}

	durations := map[string][]float64{}
	withinSLA := map[string]int{}
	overdue := []OverdueFinding{}
	t.Overdue.BySeverity = scan.Counts{}

	for _, f := range findings {
		if i, ok := weekIdx[weekStart(f.FirstSeen)]; ok && !f.FirstSeen.Truncate(day).Equal(recordedDay) {
			t.Weeks[i].Opened++
		}
		if f.CloseReason == store.CloseReasonUpdated {
			if i, ok := weekIdx[weekStart(f.ClosedAt)]; ok {
				t.Weeks[i].Fixed++
			}
			if !f.ClosedAt.Before(since) {
				days := f.ClosedAt.Sub(clockStart(f)).Hours() / 24
				durations[f.Severity] = append(durations[f.Severity], days)
				if sla, ok := slaDays[f.Severity]; ok && days <= float64(sla) {
					withinSLA[f.Severity]++
				}
			}
		}
		if sla, ok := slaDays[f.Severity]; ok && f.ClosedAt.IsZero() && !f.FixAvailable.IsZero() {
			if late := now.Sub(f.FixAvailable).Hours()/24 - float64(sla); late > 0 {
				overdue = append(overdue, OverdueFinding{Finding: f, DaysOverdue: late})
				t.Overdue.BySeverity[f.Severity]++
			}
		}
	}

	for _, s := range scan.Severities {
		d := durations[s]
		if len(d) == 0 {
			continue
		}
		slices.Sort(d)
		t.TimeToFix = append(t.TimeToFix, TimeToFix{
			Severity: s, Fixed: len(d), WithinSLA: withinSLA[s], Median: percentile(d, 0.5), P90: percentile(d, 0.9),
		})
	}

	slices.SortFunc(overdue, func(a, b OverdueFinding) int {
		if c := slices.Index(scan.Severities, a.Severity) - slices.Index(scan.Severities, b.Severity); c != 0 {
			return c
		}
		return cmp.Compare(b.DaysOverdue, a.DaysOverdue)
	})
	t.Overdue.Total = len(overdue)
	t.Overdue.Findings = overdue[:min(len(overdue), overdueLimit)]

	return t
}

func openAt(f store.Finding, at time.Time) bool {
	return !f.FirstSeen.After(at) && (f.ClosedAt.IsZero() || f.ClosedAt.After(at))
}

// clockStart is when the time to fix a finding starts: when a fix was available, if before it closed
func clockStart(f store.Finding) time.Time {
	if !f.FixAvailable.IsZero() && f.FixAvailable.Before(f.ClosedAt) {
		return f.FixAvailable
	}
	return f.FirstSeen
}

func weekStart(t time.Time) time.Time {
	d := t.UTC().Truncate(day)
	return d.Add(-time.Duration((int(d.Weekday())+6)%7) * day)
}

// percentile of sorted values, interpolating between the nearest two
func percentile(sorted []float64, p float64) float64 {
	pos := p * float64(len(sorted)-1)
	lo := int(pos)
	if lo+1 >= len(sorted) {
		return sorted[lo]
	}
	return sorted[lo] + (pos-float64(lo))*(sorted[lo+1]-sorted[lo])
}
