package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/config"
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
			trivyReport, err := GetTrivyReport(inventory.ImagesAsSlice())
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

func GetTrivyReport(images []string) (trivy.TrivyReport, error) {

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
		NumWorkers: 5,
		Images:     images,
		OutputDir:  trivyOutputDir,
	})

	return trivyRunner.Run(), nil
}
