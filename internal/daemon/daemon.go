// Package daemon periodically syncs dependency update PRs from GitHub
// and runs them through the pipeline stages, storing results for the API
package daemon

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/argocd"
	"github.com/bartlettc22/image-inquisitor/internal/github"
	"github.com/bartlettc22/image-inquisitor/internal/prdiff"
	"github.com/bartlettc22/image-inquisitor/internal/render"
	"github.com/bartlettc22/image-inquisitor/internal/scan"
	"github.com/bartlettc22/image-inquisitor/internal/store"
	"github.com/bartlettc22/image-inquisitor/internal/workloads"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// StageExtract parses the PR diff into image and chart changes
	StageExtract = "extract"

	// StageRender renders the apps using a PR's Helm charts at the new chart version
	StageRender = "render"
)

var svcLog = log.WithField("service", "daemon")

type Daemon struct {
	config   *Config
	github   *github.Client
	store    *store.Store
	renderer *render.Renderer
	clusters []*cluster
	trigger  chan struct{}

	// scanner is nil when scanning is disabled
	scanner *scan.Scanner
	// trivyVersion is empty if it couldn't be read
	trivyVersion string
	scanTrigger  chan struct{}

	activity   activity
	cacheSizes cacheSizes

	// repos are the configured repos entries with the repositories they match
	repos      *repoSet
	repoLister repoLister

	soak         soakState
	imageCreated func(ctx context.Context, ref string) (time.Time, error)

	// renovateJobs is nil when Renovate isn't configured
	renovateJobs    jobClient
	renovateTrigger chan RenovateRequest
	renovatePoll    time.Duration
}

type cluster struct {
	config  ClusterConfig
	client  kubernetes.Interface
	dynamic dynamic.Interface

	// capabilities are read on each sync and passed to helm when rendering
	capabilities render.Capabilities
}

func New(c *Config, gh *github.Client, s *store.Store) (*Daemon, error) {
	d := &Daemon{
		config: c,
		github: gh,
		store:  s,
		renderer: &render.Renderer{
			Helm:    c.Helm.Binary,
			Dir:     filepath.Join(c.DataDir, "render"),
			Fetcher: &render.GitHubFetcher{Client: gh},
		},
		trigger:     make(chan struct{}, 1),
		scanTrigger: make(chan struct{}, 1),
	}

	if !c.Scan.Disabled {
		if _, err := exec.LookPath(c.Scan.Trivy); err != nil {
			svcLog.WithError(err).Warn("trivy not found, scanning disabled")
		} else {
			d.scanner = &scan.Scanner{Trivy: c.Scan.Trivy, Dir: filepath.Join(c.DataDir, "scan")}
			if d.trivyVersion, err = d.scanner.Version(context.Background()); err != nil {
				svcLog.WithError(err).Warn("error reading trivy version")
			}
		}
	}

	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if c.Kubernetes.Kubeconfig != "" {
		rules.ExplicitPath = c.Kubernetes.Kubeconfig
	}
	for _, cc := range c.Kubernetes.Clusters {
		var restConfig *rest.Config
		var err error
		if cc.InCluster {
			restConfig, err = rest.InClusterConfig()
		} else {
			restConfig, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
				rules,
				&clientcmd.ConfigOverrides{CurrentContext: cc.Context},
			).ClientConfig()
		}
		if err != nil {
			return nil, fmt.Errorf("kubernetes cluster %s: %w", cc.Name, err)
		}
		client, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			return nil, fmt.Errorf("kubernetes cluster %s: %w", cc.Name, err)
		}
		dyn, err := dynamic.NewForConfig(restConfig)
		if err != nil {
			return nil, fmt.Errorf("kubernetes cluster %s: %w", cc.Name, err)
		}
		d.clusters = append(d.clusters, &cluster{config: cc, client: client, dynamic: dyn})
		if c.Renovate != nil && c.Renovate.Cluster == cc.Name {
			d.renovateJobs = &kubeJobClient{client: client, namespace: c.Renovate.Namespace}
		}
	}
	d.repos = newRepoSet(c.Repos)
	d.imageCreated = scan.ImageCreated
	d.repoLister = gh
	d.renovateTrigger = make(chan RenovateRequest, 1)
	d.renovatePoll = renovatePollInterval

	return d, nil
}

// Run syncs immediately, then on every interval or trigger until ctx is cancelled,
// scanning the synced PRs' images in the background
func (d *Daemon) Run(ctx context.Context) {
	// Runs left unfinished by the last daemon would otherwise look like they're still going
	if err := d.store.InterruptSyncRuns(ctx, time.Now().UTC()); err != nil {
		svcLog.WithError(err).Error("error recording interrupted sync runs")
	}

	interval := d.config.syncInterval
	ticked := time.Now()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	if d.scanner != nil {
		d.activity.update(func(a *Activity) { a.Scan.Enabled = true })
		go d.scanLoop(ctx)
	}
	if d.renovateJobs != nil {
		d.activity.update(func(a *Activity) { a.Renovate.Enabled = true })
		go d.renovateLoop(ctx)
	}

	for _, cl := range d.clusters {
		go d.watchCluster(ctx, cl)
	}

	for {
		d.sync(ctx)
		d.triggerScan()

		// The ticker keeps its schedule however long the sync took
		next := ticked.Add((time.Since(ticked)/interval + 1) * interval).UTC()
		d.activity.update(func(a *Activity) { a.Sync.NextRunAt = &next })

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-d.trigger:
		}
	}
}

// Scanning reports whether PR images are scanned (scanning is enabled and trivy was found)
func (d *Daemon) Scanning() bool {
	return d.scanner != nil
}

// TrivyVersion is the version of the Trivy that scans images, empty if unknown or scanning is off
func (d *Daemon) TrivyVersion() string {
	return d.trivyVersion
}

// Trigger requests a sync as soon as the current one (if any) finishes
func (d *Daemon) Trigger() {
	select {
	case d.trigger <- struct{}{}:
	default:
	}
}

func (d *Daemon) sync(ctx context.Context) {
	started := time.Now().UTC()
	svcLog.Info("starting sync")

	runID, err := d.store.StartSyncRun(ctx, started)
	if err != nil {
		svcLog.WithError(err).Error("error recording sync run")
		return
	}
	d.activity.update(func(a *Activity) { a.Sync = SyncActivity{Running: true} })
	defer d.activity.update(func(a *Activity) { a.Sync = SyncActivity{} })

	errs := []error{}
	// step runs one step of the sync, recording it on the run; run returns a summary of what it did
	step := func(name string, run func() (string, error)) {
		stepStarted := time.Now().UTC()
		d.activity.update(func(a *Activity) { a.Sync.Step, a.Sync.StepStartedAt = name, &stepStarted })

		summary, err := run()
		result := &store.SyncStep{Name: name, StartedAt: stepStarted, FinishedAt: time.Now().UTC(), Summary: summary}
		if err != nil {
			svcLog.WithField("step", name).WithError(err).Error("error syncing")
			result.Error = err.Error()
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
		}
		if err := d.store.AddSyncStep(ctx, runID, result); err != nil {
			svcLog.WithError(err).Error("error recording sync step")
		}
	}

	step("repo list", func() (string, error) {
		return d.syncRepoList(ctx)
	})

	total := 0
	for _, repo := range d.Repos() {
		step("repo "+repo.Name, func() (string, error) {
			n, err := d.syncRepo(ctx, repo)
			total += n
			return plural(n, "PR"), err
		})
	}

	for _, cl := range d.clusters {
		step("cluster "+cl.config.Name, func() (string, error) {
			return d.syncCluster(ctx, cl)
		})
	}

	// After the clusters, so chart changes are compared against freshly rendered apps
	step("render PR charts", func() (string, error) {
		n, err := d.renderPullRequests(ctx)
		return plural(n, "PR") + " rendered", err
	})
	step("latest versions", func() (string, error) {
		return d.lookupVersions(ctx)
	})
	if d.config.Renovate != nil && d.config.Renovate.Soak != nil {
		step("soak policy", func() (string, error) {
			return d.decideSoaks(ctx)
		})
	}
	step("findings", func() (string, error) {
		return d.reconcileFindings(ctx)
	})
	step("prune checkouts", func() (string, error) {
		n, err := d.renderer.PruneCheckouts(checkoutMaxAge)
		return plural(n, "checkout") + " removed", err
	})

	if err := d.store.FinishSyncRun(ctx, runID, time.Now().UTC(), total, errors.Join(errs...)); err != nil {
		svcLog.WithError(err).Error("error recording sync run")
	}
	if err := d.store.PruneSyncRuns(ctx, syncRunHistory); err != nil {
		svcLog.WithError(err).Error("error pruning sync runs")
	}

	svcLog.WithField("pullRequests", total).WithField("duration", time.Since(started).String()).Info("finished sync")
}

// plural formats a count with its noun, e.g. "1 PR", "3 PRs" or "2 repositories"
func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	// "repository" → "repositories", but "day" → "days"
	if len(noun) > 1 && strings.HasSuffix(noun, "y") && !strings.ContainsRune("aeiou", rune(noun[len(noun)-2])) {
		return fmt.Sprintf("%d %sies", n, strings.TrimSuffix(noun, "y"))
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

// syncRepo stores the repo's open dependency update PRs, extracts their changes
// and marks PRs that are no longer open as closed. It returns the number of PRs seen.
func (d *Daemon) syncRepo(ctx context.Context, repo Repo) (int, error) {
	prs, err := d.github.ListOpenPullRequests(ctx, repo.Name)
	if err != nil {
		return 0, err
	}

	open := []int{}
	errs := []error{}
	for _, ghPR := range prs {
		if !strings.HasPrefix(ghPR.Head.Ref, repo.BranchPrefix) {
			continue
		}
		open = append(open, ghPR.Number)

		pr := &store.PullRequest{
			Repo:      repo.Name,
			Number:    ghPR.Number,
			Title:     ghPR.Title,
			URL:       ghPR.HTMLURL,
			Branch:    ghPR.Head.Ref,
			HeadSHA:   ghPR.Head.SHA,
			Author:    ghPR.User.Login,
			State:     store.PullRequestStateOpen,
			CreatedAt: ghPR.CreatedAt,
			UpdatedAt: ghPR.UpdatedAt,
			SyncedAt:  time.Now().UTC(),
		}
		if err := d.store.UpsertPullRequest(ctx, pr); err != nil {
			errs = append(errs, err)
			continue
		}

		if err := d.extract(ctx, pr); err != nil {
			errs = append(errs, fmt.Errorf("#%d: %w", pr.Number, err))
		}
	}

	if err := d.store.CloseMissingPullRequests(ctx, repo.Name, open); err != nil {
		errs = append(errs, err)
	}

	return len(open), errors.Join(errs...)
}

// syncCluster stores the cluster's running containers and Argo CD apps, and renders the apps.
// A failed inventory keeps the previous containers and records the error on the cluster.
// It returns a summary of what it found.
func (d *Daemon) syncCluster(ctx context.Context, cl *cluster) (string, error) {
	filter := workloads.Filter{
		IncludeNamespaces: cl.config.IncludeNamespaces,
		ExcludeNamespaces: cl.config.ExcludeNamespaces,
	}
	containers, collectErr := workloads.Collect(ctx, cl.client, cl.config.Name, filter)
	if err := d.store.ReplaceContainers(ctx, cl.config.Name, containers, time.Now().UTC(), collectErr); err != nil {
		return "", err
	}
	if collectErr != nil {
		return "", collectErr
	}

	apps, err := argocd.List(ctx, cl.dynamic, cl.config.Name)
	if err != nil {
		return "", err
	}
	if err := d.store.ReplaceApplications(ctx, cl.config.Name, apps); err != nil {
		return "", err
	}

	caps, err := capabilities(cl.client.Discovery())
	if err != nil {
		return "", err
	}
	cl.capabilities = caps

	rendered, failed, err := d.renderApps(ctx, cl, apps)
	if err != nil {
		return "", err
	}

	svcLog.WithField("cluster", cl.config.Name).WithField("containers", len(containers)).
		WithField("apps", len(apps)).WithField("failedRenders", failed).Info("synced cluster")
	// Renders are cached by their inputs, so only changed apps are rendered again
	summary := fmt.Sprintf("%s, %s, %d re-rendered", plural(len(containers), "container"), plural(len(apps), "app"), rendered)
	if failed > 0 {
		summary += fmt.Sprintf(" (%d failed)", failed)
	}
	return summary, nil
}

// extract parses the PR diff into changes, skipping PRs whose head hasn't moved since
// the last successful extraction. Extraction failures are recorded on the stage;
// only storage errors are returned.
func (d *Daemon) extract(ctx context.Context, pr *store.PullRequest) error {
	stage, err := d.store.GetStage(ctx, pr.Repo, pr.Number, StageExtract)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return err
	}
	if stage != nil && stage.HeadSHA == pr.HeadSHA && stage.Status == store.StageStatusDone {
		return nil
	}

	logger := svcLog.WithField("repo", pr.Repo).WithField("number", pr.Number)
	logger.Debug("extracting changes")

	result := &store.Stage{
		Name:      StageExtract,
		Status:    store.StageStatusDone,
		HeadSHA:   pr.HeadSHA,
		UpdatedAt: time.Now().UTC(),
	}

	changes, err := d.extractChanges(ctx, pr)
	if err != nil {
		logger.WithError(err).Warn("error extracting changes")
		result.Status = store.StageStatusError
		result.Error = err.Error()
	} else if err := d.store.ReplaceChanges(ctx, pr.Repo, pr.Number, changes); err != nil {
		return err
	}

	return d.store.SetStage(ctx, pr.Repo, pr.Number, result)
}

func (d *Daemon) extractChanges(ctx context.Context, pr *store.PullRequest) ([]prdiff.Change, error) {
	diff, err := d.github.PullRequestDiff(ctx, pr.Repo, pr.Number)
	if err != nil {
		return nil, err
	}
	return prdiff.Parse(diff)
}
