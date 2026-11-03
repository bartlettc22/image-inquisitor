package api

import (
	"context"
	"net/http"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/scan"
	"github.com/bartlettc22/image-inquisitor/internal/store"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// timeToFixBuckets are the time-to-fix histogram's buckets, from a day to a year
var timeToFixBuckets = func() []float64 {
	buckets := []float64{}
	for _, days := range []float64{1, 3, 7, 14, 30, 60, 90, 180, 365} {
		buckets = append(buckets, days*day.Seconds())
	}
	return buckets
}()

var (
	descOpenFindings = prometheus.NewDesc("imginq_open_findings",
		"CVEs running in workload containers, counted once per container", []string{"severity", "fixable"}, nil)
	descPastSLA = prometheus.NewDesc("imginq_findings_past_sla",
		"Open CVEs that have had a fix available for longer than their severity's SLA", []string{"severity"}, nil)
	descSLA = prometheus.NewDesc("imginq_sla_seconds",
		"How long a CVE of each severity may run once a fix is available", []string{"severity"}, nil)
	descFixed = prometheus.NewDesc("imginq_findings_fixed_total",
		"CVEs removed from a workload container by an image update", []string{"severity"}, nil)
	descTimeToFix = prometheus.NewDesc("imginq_time_to_fix_seconds",
		"Time from a fix being available (or the CVE first seen, if none was reported) to an image update removing it", []string{"severity"}, nil)
	descRunningImages = prometheus.NewDesc("imginq_running_images",
		"Running images by update status: pending, current, pinned, behind or unknown", []string{"status"}, nil)
	descOpenPRs = prometheus.NewDesc("imginq_open_prs",
		"Open dependency update PRs", []string{"repo"}, nil)
	descClusterSync = prometheus.NewDesc("imginq_cluster_last_sync_timestamp_seconds",
		"When the cluster's inventory was last synced", []string{"cluster"}, nil)
	descClusterSuccess = prometheus.NewDesc("imginq_cluster_last_sync_success",
		"Whether the cluster's last inventory sync succeeded", []string{"cluster"}, nil)
	descClusterContainers = prometheus.NewDesc("imginq_cluster_containers",
		"Containers in the cluster's inventory", []string{"cluster"}, nil)
	descSyncStart = prometheus.NewDesc("imginq_last_sync_timestamp_seconds",
		"When the last finished sync started", nil, nil)
	descSyncDuration = prometheus.NewDesc("imginq_last_sync_duration_seconds",
		"How long the last finished sync took", nil, nil)
	descSyncSuccess = prometheus.NewDesc("imginq_last_sync_success",
		"Whether every step of the last finished sync succeeded", nil, nil)
	descScanQueue = prometheus.NewDesc("imginq_scan_queue",
		"Image references waiting for each scan stage: resolve, generate (an SBOM) or check (against the vulnerability database)", []string{"stage"}, nil)
	descRenovateStart = prometheus.NewDesc("imginq_renovate_last_run_timestamp_seconds",
		"When the last finished Renovate run started", nil, nil)
	descRenovateSuccess = prometheus.NewDesc("imginq_renovate_last_run_success",
		"Whether the last finished Renovate run succeeded", nil, nil)
)

// metrics serves Prometheus metrics read from the store at scrape time, so they survive restarts
// and agree with the UI. History is Prometheus's to keep.
func (h *handler) metrics(w http.ResponseWriter, r *http.Request) {
	reg := prometheus.NewRegistry()
	reg.MustRegister(&collector{h: h, ctx: r.Context()}, collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	promhttp.HandlerFor(reg, promhttp.HandlerOpts{ErrorHandling: promhttp.ContinueOnError}).ServeHTTP(w, r)
}

type collector struct {
	h   *handler
	ctx context.Context
}

func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, ch)
}

func (c *collector) Collect(ch chan<- prometheus.Metric) {
	for _, collect := range []func(chan<- prometheus.Metric) error{
		c.collectFindings, c.collectImages, c.collectPullRequests, c.collectSync, c.collectActivity,
	} {
		if err := collect(ch); err != nil {
			svcLog.WithError(err).Error("error collecting metrics")
			ch <- prometheus.NewInvalidMetric(descOpenFindings, err)
		}
	}
}

func (c *collector) collectFindings(ch chan<- prometheus.Metric) error {
	findings, err := c.h.store.ListFindings(c.ctx)
	if err != nil {
		return err
	}
	now := time.Now()

	type openKey struct{ severity, fixable string }
	open := map[openKey]int{}
	pastSLA := scan.Counts{}
	fixed := scan.Counts{}
	durations := map[string][]float64{}
	for _, f := range findings {
		if f.ClosedAt.IsZero() {
			fixable := "false"
			if !f.FixAvailable.IsZero() {
				fixable = "true"
			}
			open[openKey{f.Severity, fixable}]++
			if sla, ok := slaDays[f.Severity]; ok && !f.FixAvailable.IsZero() && now.Sub(f.FixAvailable) > time.Duration(sla)*day {
				pastSLA[f.Severity]++
			}
		} else if f.CloseReason == store.CloseReasonUpdated {
			fixed[f.Severity]++
			durations[f.Severity] = append(durations[f.Severity], f.ClosedAt.Sub(clockStart(f)).Seconds())
		}
	}

	// Every severity and fixability is reported, at zero too, so series don't vanish when cleared
	for _, s := range scan.Severities {
		for _, fixable := range []string{"true", "false"} {
			ch <- prometheus.MustNewConstMetric(descOpenFindings, prometheus.GaugeValue, float64(open[openKey{s, fixable}]), s, fixable)
		}
		ch <- prometheus.MustNewConstMetric(descFixed, prometheus.CounterValue, float64(fixed[s]), s)
		ch <- histogram(descTimeToFix, durations[s], timeToFixBuckets, s)
		if sla, ok := slaDays[s]; ok {
			ch <- prometheus.MustNewConstMetric(descPastSLA, prometheus.GaugeValue, float64(pastSLA[s]), s)
			ch <- prometheus.MustNewConstMetric(descSLA, prometheus.GaugeValue, (time.Duration(sla) * day).Seconds(), s)
		}
	}
	return nil
}

// histogram builds a cumulative histogram of observations
func histogram(desc *prometheus.Desc, observations, buckets []float64, labels ...string) prometheus.Metric {
	// Every bucket is listed, at zero too, or the histogram's buckets would vary between scrapes
	counts := make(map[float64]uint64, len(buckets))
	for _, b := range buckets {
		counts[b] = 0
	}
	sum := 0.0
	for _, o := range observations {
		sum += o
		for _, b := range buckets {
			if o <= b {
				counts[b]++
			}
		}
	}
	return prometheus.MustNewConstHistogram(desc, uint64(len(observations)), sum, counts, labels...)
}

func (c *collector) collectImages(ch chan<- prometheus.Metric) error {
	if len(c.h.clusters) == 0 {
		return nil
	}
	r, err := http.NewRequestWithContext(c.ctx, http.MethodGet, "/", nil)
	if err != nil {
		return err
	}
	inv, err := c.h.inventory(r)
	if err != nil {
		return err
	}
	prs, err := c.h.store.ListPullRequests(c.ctx, store.PullRequestStateOpen)
	if err != nil {
		return err
	}
	byStatus := map[string]int{}
	for _, img := range runningImages(inv.containers, prs, inv.versions) {
		byStatus[img.Status]++
	}
	for _, status := range []string{StatusPending, StatusCurrent, StatusPinned, StatusBehind, StatusUnknown} {
		ch <- prometheus.MustNewConstMetric(descRunningImages, prometheus.GaugeValue, float64(byStatus[status]), status)
	}
	return nil
}

func (c *collector) collectPullRequests(ch chan<- prometheus.Metric) error {
	prs, err := c.h.store.ListPullRequests(c.ctx, store.PullRequestStateOpen)
	if err != nil {
		return err
	}
	byRepo := map[string]int{}
	for _, repo := range c.h.syncer.Repos() {
		byRepo[repo.Name] = 0
	}
	for _, pr := range prs {
		byRepo[pr.Repo]++
	}
	for repo, n := range byRepo {
		ch <- prometheus.MustNewConstMetric(descOpenPRs, prometheus.GaugeValue, float64(n), repo)
	}
	return nil
}

func (c *collector) collectSync(ch chan<- prometheus.Metric) error {
	clusters, err := c.h.store.ListClusters(c.ctx)
	if err != nil {
		return err
	}
	for _, cl := range clusters {
		ch <- prometheus.MustNewConstMetric(descClusterSync, prometheus.GaugeValue, float64(cl.SyncedAt.Unix()), cl.Name)
		ch <- prometheus.MustNewConstMetric(descClusterSuccess, prometheus.GaugeValue, boolValue(cl.Error == ""), cl.Name)
		ch <- prometheus.MustNewConstMetric(descClusterContainers, prometheus.GaugeValue, float64(cl.Containers), cl.Name)
	}

	runs, err := c.h.store.ListSyncRuns(c.ctx, 5)
	if err != nil {
		return err
	}
	for _, run := range runs {
		if run.FinishedAt == nil {
			continue
		}
		ch <- prometheus.MustNewConstMetric(descSyncStart, prometheus.GaugeValue, float64(run.StartedAt.Unix()))
		ch <- prometheus.MustNewConstMetric(descSyncDuration, prometheus.GaugeValue, run.FinishedAt.Sub(run.StartedAt).Seconds())
		ch <- prometheus.MustNewConstMetric(descSyncSuccess, prometheus.GaugeValue, boolValue(run.Error == ""))
		break
	}

	renovateRuns, err := c.h.store.ListRenovateRuns(c.ctx, 5)
	if err != nil {
		return err
	}
	for _, run := range renovateRuns {
		if run.FinishedAt == nil {
			continue
		}
		ch <- prometheus.MustNewConstMetric(descRenovateStart, prometheus.GaugeValue, float64(run.StartedAt.Unix()))
		ch <- prometheus.MustNewConstMetric(descRenovateSuccess, prometheus.GaugeValue, boolValue(run.Status == store.RenovateStatusSucceeded))
		break
	}
	return nil
}

func (c *collector) collectActivity(ch chan<- prometheus.Metric) error {
	report, err := c.h.syncer.ActivityReport(c.ctx)
	if err != nil {
		return err
	}
	if q := report.ScanQueue; q != nil {
		ch <- prometheus.MustNewConstMetric(descScanQueue, prometheus.GaugeValue, float64(q.Resolve), "resolve")
		ch <- prometheus.MustNewConstMetric(descScanQueue, prometheus.GaugeValue, float64(q.Generate), "generate")
		ch <- prometheus.MustNewConstMetric(descScanQueue, prometheus.GaugeValue, float64(q.Check), "check")
	}
	return nil
}

func boolValue(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
