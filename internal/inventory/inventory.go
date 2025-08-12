package inventory

import (
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/registries"
	"github.com/bartlettc22/image-inquisitor/internal/trivy"
	callbackworker "github.com/bartlettc22/image-inquisitor/internal/worker/callback"
	sourcesapi "github.com/bartlettc22/image-inquisitor/pkg/api/v1alpha1/sources"
	log "github.com/sirupsen/logrus"
)

type InventoryGenerator struct {
	config       *InventoryConfig
	mu           *sync.Mutex
	workerPool   *callbackworker.WorkerPool
	workerPoolWG *sync.WaitGroup
	inventory    Inventory
}

type Inventory map[string]*InventoryRepositoryPrefixDetails

type InventoryRepositoryPrefixDetails struct {
	Registry            string                `yaml:"registry" json:"registry"`
	Repository          string                `yaml:"repository" json:"repository"`
	LatestSemverTag     string                `yaml:"latestSemverTag,omitempty" json:"latestSemverTag,omitempty"`
	LatestSemverDigest  string                `yaml:"latestSemverDigest,omitempty" json:"latestSemverDigest,omitempty"`
	LatestSemverCreated time.Time             `yaml:"latestSemverCreated,omitempty" json:"latestSemverCreated,omitempty"`
	Digests             InventoryImageDigests `yaml:"digests" json:"digests"`
}

type InventoryImageDigests map[string]*InventoryDigestDetails

type InventoryDigestDetails struct {
	Created time.Time            `yaml:"created" json:"created"`
	Sources []*sourcesapi.Source `yaml:"sources" json:"sources"`
	Issues  *trivy.ImageIssues   `yaml:"issues,omitempty" json:"issues,omitempty"`
}

type InventoryConfig struct {
	SkipRegistries              []string
	LatestSemverScanningEnabled bool
	SecurityScanningEnabled     bool
	SecurityScanner             *trivy.TrivyScanner
}

func NewInventoryGenerator(c *InventoryConfig) (*InventoryGenerator, error) {

	if c.SecurityScanningEnabled {
		err := trivy.RefreshTrivyDB()
		if err != nil {
			return nil, fmt.Errorf("unable to refresh Trivy database: %w", err)
		}
	}

	return &InventoryGenerator{
		config:       c,
		mu:           &sync.Mutex{},
		workerPoolWG: &sync.WaitGroup{},
		workerPool:   callbackworker.NewWorkerPool(&callbackworker.WorkerPoolConfig{}),
		inventory:    make(Inventory),
	}, nil
}

func (i *InventoryGenerator) AddSources(sourceList sourcesapi.SourceList) {
	for _, source := range sourceList {
		i.AddSource(source)
	}
}

func (i *InventoryGenerator) AddSource(source *sourcesapi.Source) {

	// Ignore skipped registries
	image, err := registries.NewImage(source.ImageReference)
	if err != nil {
		log.Errorf("error parsing image reference: %v", err)
		return
	}
	if slices.Contains(i.config.SkipRegistries, image.Registry()) {
		log.Debugf("skipping image due to registry: %s", source.ImageReference)
		return
	}

	i.workerPoolWG.Add(1)
	i.workerPool.AddTask(callbackworker.NewCallbackTask(
		newAddSourceFunc(source),
		i.AddSourceCallback,
	))
}

func (i *InventoryGenerator) Wait() {

	// Wait for all the tasks to finish
	i.workerPoolWG.Wait()

	// Close main worker pool
	i.workerPool.Done()
	i.workerPool.Wait()
}

func (i *InventoryGenerator) AddSourceCallback(result any, err error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	defer i.workerPoolWG.Done()

	if err != nil {
		log.Errorf("error adding source: %v", err)
		return
	}

	sourceResult, ok := result.(*AddSourceResult)
	if !ok {
		log.Errorf("invalid result type: %T", result)
		return
	}

	// Make repo structure if it doesn't exist
	if _, ok := i.inventory[sourceResult.ReferencePrefix]; !ok {
		details := &InventoryRepositoryPrefixDetails{
			Registry:   sourceResult.Registry,
			Repository: sourceResult.Repo,
			Digests:    make(InventoryImageDigests),
		}
		i.inventory[sourceResult.ReferencePrefix] = details

		// Kick off job to fetch latest semver tag (async)
		if i.config.LatestSemverScanningEnabled {
			i.workerPoolWG.Add(1)
			i.workerPool.AddTask(callbackworker.NewCallbackTask(
				newGetLatestSemverFunc(sourceResult.ReferencePrefix),
				i.GetLatestSemverCallback,
			))
		}
	}

	// Make digest structure if it doesn't exist
	if _, ok := i.inventory[sourceResult.ReferencePrefix].Digests[sourceResult.Digest]; !ok {
		i.inventory[sourceResult.ReferencePrefix].Digests[sourceResult.Digest] = &InventoryDigestDetails{
			Created: sourceResult.Created,
			Sources: []*sourcesapi.Source{sourceResult.Source},
		}

		// Kick of scan (async)
		if i.config.SecurityScanningEnabled {
			i.workerPoolWG.Add(1)
			i.config.SecurityScanner.ScanImageDigestWithCallback(
				sourceResult.ReferencePrefix,
				sourceResult.Digest,
				i.ScanCallback,
			)
		}
	} else {
		i.inventory[sourceResult.ReferencePrefix].Digests[sourceResult.Digest].Sources = append(
			i.inventory[sourceResult.ReferencePrefix].Digests[sourceResult.Digest].Sources,
			sourceResult.Source,
		)
	}
}

func (i *InventoryGenerator) GetLatestSemverCallback(result interface{}, err error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	defer i.workerPoolWG.Done()

	if err != nil {
		log.Errorf("error fetching latest semver tag: %v", err)
		return
	}

	latestSemverResult, ok := result.(*getLatestSemverResult)
	if !ok {
		log.Errorf("invalid result type: %T", result)
		return
	}

	if _, ok := i.inventory[latestSemverResult.ReferencePrefix]; ok {
		i.inventory[latestSemverResult.ReferencePrefix].LatestSemverTag = latestSemverResult.LatestSemverTag
		i.inventory[latestSemverResult.ReferencePrefix].LatestSemverDigest = latestSemverResult.LatestSemverDigest
		i.inventory[latestSemverResult.ReferencePrefix].LatestSemverCreated = latestSemverResult.LatestSemverCreated

		// Add the latest digest as a source
		i.AddSource(&sourcesapi.Source{
			Type:           sourcesapi.RegistryLatestSemverSourceType,
			ImageReference: fmt.Sprintf("%s@%s", latestSemverResult.ReferencePrefix, latestSemverResult.LatestSemverDigest),
			SourceDetails: struct {
				Tag string `yaml:"tag" json:"tag"`
			}{
				Tag: latestSemverResult.LatestSemverTag,
			},
		})
	} else {
		log.Errorf("tried adding latest semver to repository that doesn't exist")
	}
}

func (i *InventoryGenerator) ScanCallback(result interface{}, err error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	defer i.workerPoolWG.Done()

	if err != nil {
		log.Errorf("error scanning image: %v", err)
		return
	}

	scanResult, ok := result.(*trivy.TrivyScanImageCallbackResult)
	if !ok {
		log.Errorf("invalid result type: %T", result)
		return
	}

	if repo, ok := i.inventory[scanResult.RefPrefix]; ok {
		if digest, ok := repo.Digests[scanResult.Digest]; ok {
			digest.Issues = scanResult.Issues
		} else {
			log.Errorf("tried adding scan results to digest that doesn't exist")
		}
	} else {
		log.Errorf("tried adding scan results to repository that doesn't exist")
	}
}

// Inventory returns the inventory
// Blocks until all tasks have completed
func (i *InventoryGenerator) Inventory() Inventory {
	i.Wait()
	return i.inventory
}
