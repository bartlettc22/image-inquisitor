package daemon

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/argocd"
	"github.com/bartlettc22/image-inquisitor/internal/github"
	"github.com/bartlettc22/image-inquisitor/internal/prdiff"
	"github.com/bartlettc22/image-inquisitor/internal/render"
	"github.com/bartlettc22/image-inquisitor/internal/store"
	"k8s.io/client-go/discovery"
)

// capabilities reads the cluster version and API versions that helm templates can check,
// as Argo CD passes them
func capabilities(client discovery.DiscoveryInterface) (render.Capabilities, error) {
	version, err := client.ServerVersion()
	if err != nil {
		return render.Capabilities{}, fmt.Errorf("error reading server version: %w", err)
	}

	// Discovery of some aggregated APIs failing still leaves the rest usable
	_, resources, err := client.ServerGroupsAndResources()
	if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
		return render.Capabilities{}, fmt.Errorf("error discovering api versions: %w", err)
	}

	seen := map[string]bool{}
	for _, list := range resources {
		seen[list.GroupVersion] = true
		for _, r := range list.APIResources {
			if !strings.Contains(r.Name, "/") {
				seen[list.GroupVersion+"/"+r.Kind] = true
			}
		}
	}
	apiVersions := make([]string, 0, len(seen))
	for v := range seen {
		apiVersions = append(apiVersions, v)
	}
	sort.Strings(apiVersions)

	return render.Capabilities{KubeVersion: version.GitVersion, APIVersions: apiVersions}, nil
}

// renderApps renders every Helm source of the apps whose inputs changed since their last
// render, so each source's images are known. It returns the number of renders and of failures.
func (d *Daemon) renderApps(ctx context.Context, cl *cluster, apps []argocd.Application) (rendered, failed int, err error) {
	for i := range apps {
		app := &apps[i]
		for src := range app.Sources {
			if !app.IsHelm(src) {
				continue
			}
			req := &render.Request{App: app, Source: src, Capabilities: cl.capabilities}
			key := req.Key()

			previous, err := d.store.GetRender(ctx, cl.config.Name, app.Name, src)
			if err != nil && !errors.Is(err, store.ErrNotFound) {
				return rendered, failed, err
			}
			if previous != nil && previous.Key == key && previous.Error == "" {
				continue
			}

			logger := svcLog.WithField("cluster", cl.config.Name).WithField("app", app.Name).WithField("source", app.Sources[src].ChartName())
			logger.Debug("rendering app source")

			result := &store.Render{
				Cluster:    cl.config.Name,
				App:        app.Name,
				Source:     src,
				Key:        key,
				Revision:   app.Version(src),
				RenderedAt: time.Now().UTC(),
			}
			rendered++
			result.Images, err = d.renderer.Render(ctx, req)
			if err != nil {
				logger.WithError(err).Warn("error rendering app source")
				result.Error = err.Error()
				failed++
			}
			if err := d.store.SaveRender(ctx, result); err != nil {
				return rendered, failed, err
			}
		}
		if err := d.store.DeleteRenders(ctx, cl.config.Name, app.Name, len(app.Sources)); err != nil {
			return rendered, failed, err
		}
	}
	return rendered, failed, nil
}

// checkoutMaxAge is how long an unused repository checkout is kept
const checkoutMaxAge = 7 * 24 * time.Hour

// renderPullRequests runs the render stage for open PRs with Helm chart changes,
// returning how many PRs were rendered
func (d *Daemon) renderPullRequests(ctx context.Context) (int, error) {
	prs, err := d.store.ListPullRequests(ctx, store.PullRequestStateOpen)
	if err != nil {
		return 0, err
	}
	apps, err := d.store.ListApplications(ctx)
	if err != nil {
		return 0, err
	}

	rendered := 0
	errs := []error{}
	for _, pr := range prs {
		ok, err := d.renderPullRequest(ctx, pr, apps)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s#%d: %w", pr.Repo, pr.Number, err))
		}
		if ok {
			rendered++
		}
	}
	return rendered, errors.Join(errs...)
}

// renderPullRequest renders each app that runs the From version of one of the PR's charts at
// the To version, with the PR head's values, and stores the images that change. It is skipped
// while the head SHA is unchanged; render failures are recorded, only storage errors returned.
// It reports whether the PR was rendered.
func (d *Daemon) renderPullRequest(ctx context.Context, pr *store.PullRequest, apps []argocd.Application) (bool, error) {
	hasCharts := false
	for _, c := range pr.Changes {
		hasCharts = hasCharts || c.Kind == prdiff.KindHelmChart
	}
	if !hasCharts {
		return false, nil
	}

	stage, err := d.store.GetStage(ctx, pr.Repo, pr.Number, StageRender)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return false, err
	}
	if stage != nil && stage.HeadSHA == pr.HeadSHA && stage.Status == store.StageStatusDone {
		return false, nil
	}

	logger := svcLog.WithField("repo", pr.Repo).WithField("number", pr.Number)
	logger.Debug("rendering chart changes")

	results := []store.ChartRender{}
	failed := 0
	for idx, change := range pr.Changes {
		if change.Kind != prdiff.KindHelmChart {
			continue
		}
		for _, target := range chartTargets(change, apps) {
			cr := d.renderChartChange(ctx, pr, change, target)
			cr.Change = idx
			if cr.Error != "" {
				logger.WithField("app", cr.App).WithField("error", cr.Error).Warn("error rendering chart change")
				failed++
			}
			results = append(results, cr)
		}
	}

	if err := d.store.ReplaceChartRenders(ctx, pr.Repo, pr.Number, results); err != nil {
		return false, err
	}

	result := &store.Stage{Name: StageRender, Status: store.StageStatusDone, HeadSHA: pr.HeadSHA, UpdatedAt: time.Now().UTC()}
	if failed > 0 {
		result.Status = store.StageStatusError
		result.Error = fmt.Sprintf("%d of %d app renders failed", failed, len(results))
	}
	return true, d.store.SetStage(ctx, pr.Repo, pr.Number, result)
}

type chartTarget struct {
	app    *argocd.Application
	source int
}

// chartTargets are the app sources that install a chart change's chart at its From version
func chartTargets(change prdiff.Change, apps []argocd.Application) []chartTarget {
	ref := argocd.ChartRefOf(change.RepoURL, change.Name)
	targets := []chartTarget{}
	for i := range apps {
		app := &apps[i]
		for src, s := range app.Sources {
			if !app.IsHelm(src) || s.ChartRef() != ref {
				continue
			}
			if s.TargetRevision == change.From || app.Version(src) == change.From {
				targets = append(targets, chartTarget{app, src})
			}
		}
	}
	return targets
}

func (d *Daemon) renderChartChange(ctx context.Context, pr *store.PullRequest, change prdiff.Change, t chartTarget) store.ChartRender {
	cr := store.ChartRender{Cluster: t.app.Cluster, App: t.app.Name, Source: t.source, Images: []render.ImageDiff{}}

	current, err := d.store.GetRender(ctx, t.app.Cluster, t.app.Name, t.source)
	if err != nil || current.Error != "" {
		cr.Error = "the deployed version hasn't rendered"
		if current != nil {
			cr.Error += ": " + current.Error
		}
		return cr
	}

	var cl *cluster
	for _, c := range d.clusters {
		if c.config.Name == t.app.Cluster {
			cl = c
		}
	}
	if cl == nil {
		cr.Error = "cluster is no longer configured"
		return cr
	}

	// The PR may change values alongside the chart, so read sources from its repo at its head
	repoRevisions := map[string]string{}
	for _, s := range t.app.Sources {
		if repo, ok := github.RepoFromURL(s.RepoURL); ok && strings.EqualFold(repo, pr.Repo) {
			repoRevisions[s.RepoURL] = pr.HeadSHA
		}
	}

	images, err := d.renderer.Render(ctx, &render.Request{
		App:           t.app,
		Source:        t.source,
		Capabilities:  cl.capabilities,
		Revision:      change.To,
		RepoRevisions: repoRevisions,
	})
	if err != nil {
		cr.Error = err.Error()
		return cr
	}

	cr.Images = render.Diff(current.Images, images)
	return cr
}
