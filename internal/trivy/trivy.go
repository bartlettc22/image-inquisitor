package trivy

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	trivyTypes "github.com/aquasecurity/trivy/pkg/types"
	log "github.com/sirupsen/logrus"
)

const (
	defaultNumWorkers = 10
)

type TrivyRunnerConfig struct {
	Images     []string
	NumWorkers int
	OutputDir  string
}

type TrivyRunner struct {
	TrivyRunnerConfig
}

type RunResults map[string]*ScanResult

type ScanResult struct {
	Worker int
	Image  string
	Report *trivyTypes.Report
	Output string
	Err    error
}

func NewTrivyRunner(config TrivyRunnerConfig) *TrivyRunner {
	if config.NumWorkers == 0 {
		config.NumWorkers = defaultNumWorkers
	}
	return &TrivyRunner{
		TrivyRunnerConfig: config,
	}
}

func (tr *TrivyRunner) Run() TrivyReport {

	var wg sync.WaitGroup

	// Create a channel for tasks
	imageQueue := make(chan string, 100000)
	scanResultQueue := make(chan *ScanResult, 100000)

	// Start worker goroutines
	for i := 0; i < tr.NumWorkers; i++ {
		go worker(i, tr.OutputDir, imageQueue, scanResultQueue)
	}

	wg.Add(len(tr.Images))
	for _, image := range tr.Images {
		imageQueue <- image
	}

	runResults := make(RunResults)
	go func() {
		for {
			scanResult := <-scanResultQueue
			runResults[scanResult.Image] = scanResult
			wg.Done()
		}
	}()

	wg.Wait()

	return formatReport(runResults)
}

func worker(id int, outputDir string, images <-chan string, results chan *ScanResult) {
	for image := range images {

		log.Debugf("starting Trivy scan on: %s", image)

		// Prepare the output file name
		outputFile := fmt.Sprintf("%s/%s.json", outputDir, strings.ReplaceAll(image, "/", "_"))

		result := &ScanResult{
			Worker: id,
			Image:  image,
		}

		// Create the command to run Trivy
		cmd := exec.Command("trivy", "image", "--quiet", "--format", "json", "--output", outputFile, image)

		// Execute the command and capture its output
		output, err := cmd.CombinedOutput()
		result.Output = string(output)
		if err != nil {
			result.Err = fmt.Errorf("failed to run Trivy for image %s: %w", image, err)
			results <- result
			continue
		}

		scanContent, err := os.ReadFile(outputFile)
		if err != nil {
			result.Err = fmt.Errorf("failed reading Trivy output file (%s): %v", outputFile, err)
			results <- result
			continue
		}

		report := &trivyTypes.Report{}
		err = json.Unmarshal(scanContent, report)
		if err != nil {
			result.Err = fmt.Errorf("failed parsing Trivy output file: %v", err)
			results <- result
			continue
		}

		result.Report = report

		log.Debugf("completed Trivy scan on: %s", image)

		results <- result
	}
}
