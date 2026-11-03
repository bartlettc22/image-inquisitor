package api

import (
	"strings"

	"github.com/bartlettc22/image-inquisitor/internal/argocd"
	"github.com/bartlettc22/image-inquisitor/internal/prdiff"
	"github.com/bartlettc22/image-inquisitor/internal/store"
)

// Update statuses of a running image or deployed chart
const (
	// StatusPending: an open PR updates it
	StatusPending = "pending"

	// StatusCurrent: no newer version of the same variant
	StatusCurrent = "current"

	// StatusPinned: a newer image exists, but a chart from a Helm or OCI repository sets
	// the image, so the chart's own updates decide when it changes
	StatusPinned = "pinned"

	// StatusBehind: a newer version exists and no open PR updates it, i.e. a gap in Renovate's coverage
	StatusBehind = "behind"

	// StatusUnknown: the tag isn't a version (e.g. "latest") or its tags couldn't be listed
	StatusUnknown = "unknown"
)

// Source is the Argo CD app source that deploys a workload
type Source struct {
	Cluster string `json:"-"`
	App     string `json:"app"`
	Index   int    `json:"index"`

	// Kind is "chart" (Helm or OCI repository) or "git" (chart in a git repo)
	Kind    string `json:"kind"`
	Chart   string `json:"chart"`
	RepoURL string `json:"repoURL"`
	Path    string `json:"path,omitempty"`

	// Version is the deployed chart version, or commit for git charts
	Version string `json:"version"`

	// Latest is the newest chart version, if known
	Latest string `json:"latest,omitempty"`
}

// AppView is an Argo CD app with its sources and how much of it is running
type AppView struct {
	Cluster       string       `json:"cluster"`
	Name          string       `json:"name"`
	Project       string       `json:"project"`
	DestNamespace string       `json:"destNamespace"`
	SyncStatus    string       `json:"syncStatus"`
	Health        string       `json:"health"`
	Sources       []SourceView `json:"sources"`

	// Containers are the running containers labelled as this app's;
	// Unattributed those not matched to any of its rendered sources
	Containers   int `json:"containers"`
	Unattributed int `json:"unattributed"`
}

type SourceView struct {
	Index int `json:"index"`

	// Kind is "chart" (Helm or OCI repository), "git" (chart in a git repo),
	// "ref" (values only) or "other" (not rendered, e.g. Kustomize)
	Kind           string `json:"kind"`
	Chart          string `json:"chart,omitempty"`
	RepoURL        string `json:"repoURL"`
	Path           string `json:"path,omitempty"`
	TargetRevision string `json:"targetRevision,omitempty"`
	Revision       string `json:"revision,omitempty"`

	Render *store.Render `json:"render,omitempty"`

	// Version, UpdatePRs and Status are set for charts from Helm and OCI repositories
	Version   *store.Version `json:"version,omitempty"`
	UpdatePRs []Coverage     `json:"updatePRs,omitempty"`
	Status    string         `json:"status,omitempty"`
}

// versionIndex looks up the version status of image tags and chart versions
type versionIndex map[string]*store.Version

func newVersionIndex(versions []store.Version) versionIndex {
	idx := versionIndex{}
	for i := range versions {
		v := &versions[i]
		idx[v.Kind+"|"+v.Repository+"|"+v.Current] = v
	}
	return idx
}

func (idx versionIndex) get(kind, repository, current string) *store.Version {
	return idx[kind+"|"+repository+"|"+current]
}

// versionStatus is the status of a version that no open PR updates
func versionStatus(v *store.Version) string {
	switch {
	case v == nil || v.Error != "" || v.Latest == "":
		return StatusUnknown
	case v.Latest == v.Current:
		return StatusCurrent
	default:
		return StatusBehind
	}
}

func sourceKind(app *argocd.Application, idx int) string {
	s := app.Sources[idx]
	switch {
	case !app.IsHelm(idx) && s.Ref != "" && s.Path == "":
		return "ref"
	case !app.IsHelm(idx):
		return "other"
	case s.IsRepoChart():
		return "chart"
	default:
		return "git"
	}
}

// attribution maps rendered workloads to the app sources that declare them
type attribution struct {
	byContainer map[string]*Source
	byWorkload  map[string]*Source
}

func workloadKey(cluster, namespace, kind, name string) string {
	return strings.Join([]string{cluster, namespace, kind, name}, "/")
}

func newAttribution(apps []argocd.Application, renders []store.Render, versions versionIndex) *attribution {
	appsByKey := map[string]*argocd.Application{}
	for i := range apps {
		appsByKey[apps[i].Cluster+"/"+apps[i].Name] = &apps[i]
	}

	a := &attribution{byContainer: map[string]*Source{}, byWorkload: map[string]*Source{}}
	for _, r := range renders {
		app, ok := appsByKey[r.Cluster+"/"+r.App]
		if !ok || r.Source >= len(app.Sources) || r.Error != "" {
			continue
		}
		s := app.Sources[r.Source]
		src := &Source{
			Cluster: r.Cluster, App: r.App, Index: r.Source, Kind: sourceKind(app, r.Source),
			Chart: s.ChartName(), RepoURL: s.RepoURL, Path: s.Path, Version: r.Revision,
		}
		if v := versions.get(store.VersionKindChart, s.ChartRef(), r.Revision); v != nil {
			src.Latest = v.Latest
		}
		for _, img := range r.Images {
			key := workloadKey(r.Cluster, img.Namespace, img.Kind, img.Name)
			a.byContainer[key+"/"+img.Container] = src
			if _, ok := a.byWorkload[key]; !ok {
				a.byWorkload[key] = src
			}
		}
	}
	return a
}

// source returns the app source that declares a running container: the same workload and
// container, else the same workload (custom resources whose operator names the containers)
func (a *attribution) source(c parsedContainer) *Source {
	key := workloadKey(c.Cluster, c.Namespace, c.WorkloadKind, c.WorkloadName)
	if s, ok := a.byContainer[key+"/"+c.Container.Container]; ok {
		return s
	}
	return a.byWorkload[key]
}

func appViews(apps []argocd.Application, renders []store.Render, containers []parsedContainer, versions versionIndex, openPRs []*store.PullRequest) []AppView {
	type renderKey struct {
		cluster, app string
		source       int
	}
	rendersByKey := map[renderKey]*store.Render{}
	for i := range renders {
		r := &renders[i]
		rendersByKey[renderKey{r.Cluster, r.App, r.Source}] = r
	}

	views := make([]AppView, 0, len(apps))
	for i := range apps {
		app := &apps[i]
		v := AppView{
			Cluster: app.Cluster, Name: app.Name, Project: app.Project, DestNamespace: app.DestNamespace,
			SyncStatus: app.SyncStatus, Health: app.Health, Sources: []SourceView{},
		}
		for idx, s := range app.Sources {
			sv := SourceView{
				Index: idx, Kind: sourceKind(app, idx), RepoURL: s.RepoURL, Path: s.Path,
				TargetRevision: s.TargetRevision, Revision: app.Version(idx),
			}
			if sv.Kind == "chart" || sv.Kind == "git" {
				sv.Chart = s.ChartName()
				sv.Render = rendersByKey[renderKey{app.Cluster, app.Name, idx}]
			}
			if sv.Kind == "chart" {
				sv.Version = versions.get(store.VersionKindChart, s.ChartRef(), sv.Revision)
				sv.UpdatePRs = chartUpdatePRs(s.ChartRef(), sv.Revision, openPRs)
				sv.Status = versionStatus(sv.Version)
				if len(sv.UpdatePRs) > 0 {
					sv.Status = StatusPending
				}
			}
			v.Sources = append(v.Sources, sv)
		}
		for _, c := range containers {
			if c.Cluster == app.Cluster && c.ArgoApp == app.Name {
				v.Containers++
				if c.source == nil || c.source.App != app.Name {
					v.Unattributed++
				}
			}
		}
		views = append(views, v)
	}
	return views
}

// chartUpdatePRs are the open PRs updating a chart from the given version
func chartUpdatePRs(chartRef, version string, openPRs []*store.PullRequest) []Coverage {
	prs := []Coverage{}
	for _, pr := range openPRs {
		for _, c := range pr.Changes {
			if c.Kind == prdiff.KindHelmChart && c.From == version && argocd.ChartRefOf(c.RepoURL, c.Name) == chartRef {
				prs = append(prs, Coverage{Repo: pr.Repo, Number: pr.Number, Title: pr.Title, From: c.From, To: c.To, MatchesRunning: true})
			}
		}
	}
	return prs
}
