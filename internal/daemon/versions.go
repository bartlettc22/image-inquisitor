package daemon

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/argocd"
	"github.com/bartlettc22/image-inquisitor/internal/store"
	"github.com/bartlettc22/image-inquisitor/internal/versions"
	"github.com/bartlettc22/image-inquisitor/internal/workloads"
	"github.com/google/go-containerregistry/pkg/name"
	"golang.org/x/sync/errgroup"
)

const (
	// tagListMaxAge is how long a repository's tags are reused before listing them again
	tagListMaxAge = 6 * time.Hour

	// tagListRetryAfter is how long a failed listing waits before it is retried
	tagListRetryAfter = 30 * time.Minute

	tagListConcurrency = 4

	// tagListTimeout bounds one listing; the largest repositories have hundreds of thousands of tags
	tagListTimeout = 5 * time.Minute
)

// versionTarget is an image repository or chart whose tags are listed
type versionTarget struct {
	kind       string
	repository string

	// repoURL and chart locate a chart's versions
	repoURL string
	chart   string
}

// lookupVersions finds the newest version of every running image tag and deployed chart
// version, listing each repository's tags at most every tagListMaxAge. It returns a summary.
func (d *Daemon) lookupVersions(ctx context.Context) (string, error) {
	containers, err := d.store.ListContainers(ctx)
	if err != nil {
		return "", err
	}
	apps, err := d.store.ListApplications(ctx)
	if err != nil {
		return "", err
	}
	currents := versionTargets(containers, apps)

	lister := versions.NewLister()
	mu := sync.Mutex{}
	tagLists := map[versionTarget]*store.TagList{}
	listed := 0

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(tagListConcurrency)
	for target := range currents {
		g.Go(func() error {
			tl, fetched, err := d.tagList(gctx, lister, target)
			if err != nil {
				return err
			}
			mu.Lock()
			tagLists[target] = tl
			if fetched {
				listed++
			}
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return "", err
	}

	now := time.Now().UTC()
	result := []store.Version{}
	failed := 0
	for target, tags := range currents {
		tl := tagLists[target]
		if tl.Error != "" {
			failed++
		}
		for current := range tags {
			v := store.Version{Kind: target.kind, Repository: target.repository, Current: current, Error: tl.Error, CheckedAt: now}
			if tl.Error == "" {
				v.Latest, _ = versions.Latest(current, tl.Tags)
			}
			result = append(result, v)
		}
	}

	svcLog.WithField("repositories", len(currents)).Debug("looked up versions")
	summary := fmt.Sprintf("%s, %d listed", plural(len(currents), "repository"), listed)
	if failed > 0 {
		summary += fmt.Sprintf(", %d failed", failed)
	}
	return summary, d.store.ReplaceVersions(ctx, result)
}

// tagList returns a target's cached tags, listing them again once they are stale, and
// whether it listed them. Listing failures are stored on the tag list; only storage errors
// are returned.
func (d *Daemon) tagList(ctx context.Context, lister *versions.Lister, target versionTarget) (*store.TagList, bool, error) {
	cached, err := d.store.GetTagList(ctx, target.kind, target.repository)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return nil, false, err
	}
	if cached != nil {
		age := time.Since(cached.FetchedAt)
		if (cached.Error == "" && age < tagListMaxAge) || (cached.Error != "" && age < tagListRetryAfter) {
			return cached, false, nil
		}
	}

	listCtx, cancel := context.WithTimeout(ctx, tagListTimeout)
	defer cancel()
	tl := &store.TagList{Kind: target.kind, Repository: target.repository, FetchedAt: time.Now().UTC()}
	if target.kind == store.VersionKindChart {
		tl.Tags, err = lister.ChartVersions(listCtx, target.repoURL, target.chart)
	} else {
		tl.Tags, err = lister.ImageTags(listCtx, target.repository)
	}
	tl.Tags = versions.Versions(tl.Tags)
	if err != nil {
		svcLog.WithField("repository", target.repository).WithError(err).Warn("error listing tags")
		tl.Error = err.Error()
		// Keep comparing against the last good list, and retry after tagListRetryAfter
		if cached != nil && cached.Error == "" {
			tl.Tags = cached.Tags
			tl.Error = ""
			tl.FetchedAt = time.Now().UTC().Add(tagListRetryAfter - tagListMaxAge)
		}
	}

	return tl, true, d.store.SaveTagList(ctx, tl)
}

// versionTargets returns the tags in use per image repository and chart:
// running image tags, and the versions of charts deployed from Helm and OCI repositories
func versionTargets(containers []workloads.Container, apps []argocd.Application) map[versionTarget]map[string]bool {
	currents := map[versionTarget]map[string]bool{}
	add := func(t versionTarget, current string) {
		if currents[t] == nil {
			currents[t] = map[string]bool{}
		}
		currents[t][current] = true
	}

	for _, c := range containers {
		base, _, _ := strings.Cut(c.Image, "@")
		ref, err := name.ParseReference(base, name.WeakValidation)
		if err != nil {
			continue
		}
		// Digest-only references have no tag to compare
		if tag, ok := ref.(name.Tag); ok && strings.Contains(base, ":"+tag.TagStr()) {
			add(versionTarget{kind: store.VersionKindImage, repository: ref.Context().Name()}, tag.TagStr())
		}
	}

	for _, app := range apps {
		for i, s := range app.Sources {
			if !app.IsHelm(i) || !s.IsRepoChart() {
				continue
			}
			add(versionTarget{kind: store.VersionKindChart, repository: s.ChartRef(), repoURL: s.RepoURL, chart: s.Chart}, app.Version(i))
		}
	}

	return currents
}
