package trivy

import (
	"fmt"
	"os"

	callbackworker "github.com/bartlettc22/image-inquisitor/internal/worker/callback"
	log "github.com/sirupsen/logrus"
)

const (
	defaultNumWorkers = 3
	defaultOutputDir  = "/tmp/trivy-results"
)

var svcLog = log.WithField("service", "trivy")

type TrivyScannerConfig struct {
	NumWorkers int
	OutputDir  string
}

type TrivyScanner struct {
	*TrivyScannerConfig
	workPool *callbackworker.WorkerPool
}

func NewTrivyScanner(config *TrivyScannerConfig) (*TrivyScanner, error) {

	if config.OutputDir == "" {
		config.OutputDir = defaultOutputDir
	}

	if config.NumWorkers == 0 {
		config.NumWorkers = defaultNumWorkers
	}

	err := os.MkdirAll(config.OutputDir, 0755)
	if err != nil {
		log.Fatalf("failed to create Trivy output directory: %v", err)
	}

	return &TrivyScanner{
		TrivyScannerConfig: config,
		workPool:           callbackworker.NewWorkerPool(&callbackworker.WorkerPoolConfig{}),
	}, nil
}

type TrivyScanImageCallbackResult struct {
	RefPrefix string
	Digest    string
	Issues    ImageIssues
}

func (tr *TrivyScanner) ScanImageDigestWithCallback(referencePrefix string, digest string, callback func(result interface{}, err error)) {
	tr.workPool.AddTask(callbackworker.NewCallbackTask(
		func() (interface{}, error) {
			ref := fmt.Sprintf("%s@%s", referencePrefix, digest)
			issues, err := Scan(ref, tr.OutputDir)
			if err != nil {
				return nil, fmt.Errorf("failed to run Trivy for image %s: %w", ref, err)
			}

			return &TrivyScanImageCallbackResult{
				RefPrefix: referencePrefix,
				Digest:    digest,
				Issues:    issues,
			}, nil
		},
		callback,
	))
}

func (tr *TrivyScanner) Done() {
	tr.workPool.Done()
}
