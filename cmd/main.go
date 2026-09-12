package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/config"
	"github.com/bartlettc22/image-inquisitor/internal/metrics"
	"github.com/bartlettc22/image-inquisitor/internal/registries"
	"github.com/bartlettc22/image-inquisitor/internal/registries/querier"
	"github.com/bartlettc22/image-inquisitor/internal/reports"
	"github.com/bartlettc22/image-inquisitor/internal/sources"
	exportsources "github.com/bartlettc22/image-inquisitor/internal/sources/export"
	importsources "github.com/bartlettc22/image-inquisitor/internal/sources/import"
	sourcetypes "github.com/bartlettc22/image-inquisitor/internal/sources/types"
	"github.com/bartlettc22/image-inquisitor/internal/trivy"
	log "github.com/sirupsen/logrus"
)

func main() {

	start := time.Now()
	ctx := context.Background()

	cfg := config.LoadConfig()

	if cfg.LogJSON {
		log.SetFormatter(&log.JSONFormatter{})
	}
	logLevel, err := log.ParseLevel(cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to parse log level: %v", err)
	}
	log.SetLevel(logLevel)

	inventory, err := sources.GetInventoryFromSources(ctx, &sources.ImageSourcesConfig{
		SourceID:               cfg.SourceID,
		ImageSourceTypes:       cfg.ImageSources,
		ExcludeImageRegistries: cfg.ExcludeImageRegistries,
		KubernetesSourceConfig: &sources.KubernetesSourceConfig{
			IncludeNamespaces: cfg.KubernetesSourceIncludeNamespaces,
			ExcludeNamespaces: cfg.KubernetesSourceExcludeNamespaces,
		},
		FileSourceConfig: &sources.FileSourceConfig{
			SourceFilePath: cfg.ImageSourcesFilePath,
		},
		ImportSourcesConfig: &importsources.ImportSourcesConfig{
			ImportSourcesFrom:             cfg.ImportSourcesFrom,
			ImportSourcesFilePath:         cfg.ImportSourcesFilePath,
			ImportSourcesGCSBucket:        cfg.ImportSourcesGCSBucket,
			ImportSourcesGCSDirectoryPath: cfg.ImportSourcesGCSDirectoryPath,
		},
	})
	if err != nil {
		log.Fatalf("%v", err)
	}
	if len(cfg.ExportSourcesDestinations) > 0 {
		log.Infof("exporting primary sources to: %s", cfg.ExportSourcesDestinations.String())
		err := inventory.Export(ctx, &exportsources.ExporterConfig{
			SourceID:         cfg.SourceID,
			Destinations:     cfg.ExportSourcesDestinations,
			FilePath:         cfg.ExportSourcesFilePath,
			GCSBucket:        cfg.ExportSourcesGCSBucket,
			GCSDirectoryPath: cfg.ExportSourcesGCSDirectoryPath,
		})
		if err != nil {
			log.Fatalf("%v", err)
		}
	}

	wg := &sync.WaitGroup{}
	mu := &sync.Mutex{}

	masterSummaryReportList := reports.NewSummaryReportList(start)
	masterImageReportList := reports.NewImageReportList(start)

	// Shared between the registry and vulnerability passes so the vulnerability
	// report can carry upgrade context. Both are written under mu inside their
	// goroutines and read only after wg.Wait().
	registryReports := make(map[string]*registries.RegistryImageReport)
	var vulnerabilityReports trivy.TrivyReport

	// Maps each image to the Kubernetes workloads running it. This is what turns
	// a CVE list into a work queue -- "this image has 5 criticals" is far less
	// actionable than "this image has 5 criticals and 6 Deployments run it".
	//
	// Previously commented out. It could not simply be uncommented because it
	// referenced masterImageReportList above its declaration; moving it here is
	// the whole fix.
	if cfg.ReportOutputs.Contains(reports.ReportTypeImageKubernetes) {
		// The old GetKubernetesSourceReports() helper no longer exists -- the
		// kubernetes source now folds its data into the inventory as
		// ImageSourceDetails, keyed by source type. The workload detail is still
		// there, so read it back out from the inventory.
		for imageFullName, details := range inventory.ImageDetails {
			for _, sourceDetails := range details.ImageSourceDetailsByID {
				if kubeReport, ok := sourceDetails.SourcesByType[sourcetypes.ImageSourceTypeKubernetes]; ok {
					masterImageReportList.AddImageReport(reports.ReportTypeImageKubernetes, imageFullName, kubeReport)
				}
			}
		}
	}

	if cfg.ReportOutputs.Contains(reports.ReportTypeImageRegistry) ||
		cfg.ReportOutputs.Contains(reports.ReportTypeSummaryRegistry) {
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			registryQueries := querier.NewRegistryQuerier(cfg.HarborRegistries)
			for imageFullName, image := range inventory.ImageComponents() {
				registryReport, err := registryQueries.FetchReport(image)
				if err != nil {
					log.Error(err)
					continue
				}
				mu.Lock()
				registryReports[imageFullName] = registryReport
				masterImageReportList.AddImageReport(reports.ReportTypeImageRegistry, imageFullName, registryReport)
				mu.Unlock()
			}
		}(wg)
	}

	if cfg.ReportOutputs.Contains(reports.ReportTypeImageVulnerabilities) {
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			trivyReport, err := GetTrivyReport(inventory.ImagesAsSlice(), cfg.VulnerabilityDetail)
			if err != nil {
				log.Error(err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			vulnerabilityReports = trivyReport
		}(wg)
	}
	wg.Wait()

	// Enrich AFTER the wait, not inside either goroutine: the registry and
	// vulnerability reports are produced concurrently, so neither can read the
	// other's results without racing. Once both have finished this is a plain
	// single-threaded pass.
	//
	// Images with no registry report keep empty upgrade fields (omitempty), which
	// is honest -- absence of an answer, not a claim that nothing is available.
	for imageFullName, vulnReport := range vulnerabilityReports {
		// Always set the state, even with no registry report. An ABSENT label and
		// a label reading "unknown" are very different to a consumer: Loki drops
		// records whose label does not exist from a label filter, so leaving it
		// unset would silently hide those images from any filtered view rather
		// than showing them as unassessed. Those are
		// exactly the images you least want
		// disappearing from a "what still needs attention" list.
		//
		// The tags stay omitempty: those genuinely have no value to report.
		if registryReport, ok := registryReports[imageFullName]; ok {
			vulnReport.SetUpgradeContext(registryReport.Tag, registryReport.LatestTag)
		} else {
			vulnReport.SetUpgradeContext("", "")
		}
		masterImageReportList.AddImageReport(reports.ReportTypeImageVulnerabilities, imageFullName, vulnReport)
	}

	// Publish the same findings as metrics.
	//
	// The log reports stay the system of record; this is an additional consumer
	// path, not a replacement. Metrics carry last-value semantics, which removes
	// the "which scan is inside this window" question that every log-based panel
	// otherwise has to answer.
	if cfg.MetricsEndpoint != "" {
		pub := metrics.NewPublisher(cfg.MetricsEndpoint)

		for imageFullName, vulnReport := range vulnerabilityReports {
			if vulnReport.ImageIssues != nil && vulnReport.ImageIssues.Total != nil {
				t := vulnReport.ImageIssues.Total
				for severity, count := range map[string]int{
					"critical": t.Critical, "high": t.High, "medium": t.Medium,
					"low": t.Low, "unknown": t.Unknown,
				} {
					pub.Gauge("image_inquisitor_vulnerabilities",
						"Vulnerabilities found in a running image, by severity.",
						map[string]string{"image": imageFullName, "severity": severity},
						float64(count))
				}
			}
			// Age is only emitted when known. A missing series is a blank panel
			// cell; a zero would read as "built today", which is what the old
			// int-valued field wrongly claimed for epoch-pinned images.
			if vulnReport.AgeDays != nil {
				pub.Gauge("image_inquisitor_build_age_days",
					"Days since the image was built, omitted when the created date is unusable.",
					map[string]string{"image": imageFullName}, float64(*vulnReport.AgeDays))
			}
			pub.Gauge("image_inquisitor_image_info",
				"Always 1. Carries the running tag, newest known tag and upgrade state as labels.",
				map[string]string{
					"image":         imageFullName,
					"running_tag":   vulnReport.RunningTag,
					"latest_tag":    vulnReport.LatestTag,
					"upgrade_state": vulnReport.UpgradeState,
				}, 1)
		}

		// Liveness of the scan itself. Nothing currently notices a CronJob that
		// stops running -- there is no pod to alert on, so the data simply stops
		// changing and the dashboards keep rendering the last result.
		pub.Gauge("image_inquisitor_scan_timestamp_seconds",
			"Unix time the scan completed. Alert on this going stale.",
			nil, float64(time.Now().Unix()))
		pub.Gauge("image_inquisitor_images_scanned",
			"Number of images in the completed scan.",
			nil, float64(len(vulnerabilityReports)))
		pub.Gauge("image_inquisitor_scan_duration_seconds",
			"Wall-clock duration of the scan.",
			nil, time.Since(start).Seconds())

		if err := pub.Push(30 * time.Second); err != nil {
			// Never fatal: the log reports are already written and are the system
			// of record. A metrics outage must not cost us the scan.
			log.Errorf("publishing metrics: %v", err)
		} else {
			log.Infof("published metrics for %d images to %s", len(vulnerabilityReports), cfg.MetricsEndpoint)
		}
	}

	// Summary reports are derived from the per-image reports, so only generate
	// them when a summary-type report was actually asked for.
	wantSummary := false
	for _, rt := range cfg.ReportOutputs {
		if rt.IsSummaryReportType() {
			wantSummary = true
			break
		}
	}
	if wantSummary {
		masterSummaryReportList.GenerateSummaryReports(inventory.ImageComponents(), masterImageReportList)
		masterSummaryReportList.Output()
	}

	// Always emit the per-image reports. Previously BOTH Output() calls sat
	// behind a ReportTypeImageSummary check, so asking for any other report on
	// its own produced a run that did all the work, logged "done", and printed
	// nothing -- which reads as a config mistake rather than a bug.
	masterImageReportList.Output()

	log.Infof("done")
}

func GetTrivyReport(images []string, vulnerabilityDetail bool) (trivy.TrivyReport, error) {

	trivyOutputDir, err := os.MkdirTemp("/tmp", "trivy_*")
	if err != nil {
		return nil, fmt.Errorf("error creating Trivy tmp directory: %v", err)
	}
	defer func() {
		err := os.RemoveAll(trivyOutputDir)
		if err != nil {
			log.Errorf("Error removing directory: %v\n", err)
		}
	}()

	trivyRunner := trivy.NewTrivyRunner(trivy.TrivyRunnerConfig{
		VulnerabilityDetail: vulnerabilityDetail,
		NumWorkers:          5,
		Images:              images,
		OutputDir:           trivyOutputDir,
	})

	return trivyRunner.Run(), nil
}
