// Package api serves the daemon's state as JSON for the UI
package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/bartlettc22/image-inquisitor/internal/argocd"
	"github.com/bartlettc22/image-inquisitor/internal/daemon"
	"github.com/bartlettc22/image-inquisitor/internal/scan"
	"github.com/bartlettc22/image-inquisitor/internal/store"
	log "github.com/sirupsen/logrus"
)

var svcLog = log.WithField("service", "api")

// Syncer is the daemon: it triggers an out-of-schedule sync and reports what it's doing
type Syncer interface {
	Trigger()
	RunRenovate(req daemon.RenovateRequest) error
	Repos() []daemon.Repo
	SoakDecisions() []daemon.SoakDecision
	RepoPatterns() []daemon.RepoPattern
	Activity() daemon.Activity
	ActivityReport(ctx context.Context) (*daemon.ActivityReport, error)
}

// Config is the daemon config shown in the UI; it must never include secrets
type Config struct {
	SyncInterval string `json:"syncInterval"`
	GitHubAPIURL string `json:"githubAPIURL,omitempty"`

	// GitHubToken is whether a token is set, not the token
	GitHubToken bool `json:"githubToken"`

	// Scanning is whether the daemon scans PR images
	Scanning bool `json:"scanning"`

	// Repos are filled in per request, as patterns' matches change
	Repos    []daemon.RepoPattern `json:"repos"`
	Clusters []ClusterConfig      `json:"clusters"`

	// Renovate is unset when the daemon doesn't run Renovate
	Renovate *RenovateConfig `json:"renovate,omitempty"`
}

// RenovateConfig is where and how often the daemon runs Renovate
type RenovateConfig struct {
	Image     string `json:"image"`
	Cluster   string `json:"cluster"`
	Namespace string `json:"namespace"`

	// Interval is empty when Renovate only runs from the UI
	Interval string `json:"interval,omitempty"`

	// Soak is unset when the daemon doesn't set Renovate's soak
	Soak *daemon.SoakConfig `json:"soak,omitempty"`
}

type ClusterConfig struct {
	Name              string   `json:"name"`
	Context           string   `json:"context,omitempty"`
	InCluster         bool     `json:"inCluster,omitempty"`
	IncludeNamespaces []string `json:"includeNamespaces,omitempty"`
	ExcludeNamespaces []string `json:"excludeNamespaces,omitempty"`
}

type StatusResponse struct {
	Repos    []string        `json:"repos"`
	Clusters []store.Cluster `json:"clusters"`
	LastRun  *store.SyncRun  `json:"lastRun"`
	Config   *Config         `json:"config"`
	Activity daemon.Activity `json:"activity"`
}

type handler struct {
	store    *store.Store
	syncer   Syncer
	config   *Config
	clusters []string
	scanning bool
	vulns    vulnCache
}

func NewHandler(s *store.Store, syncer Syncer, config *Config) http.Handler {
	h := &handler{store: s, syncer: syncer, config: config, scanning: config.Scanning}
	for _, c := range config.Clusters {
		h.clusters = append(h.clusters, c.Name)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("GET /api/v1/status", h.status)
	mux.HandleFunc("POST /api/v1/sync", h.sync)
	mux.HandleFunc("GET /api/v1/activity", h.activity)
	mux.HandleFunc("POST /api/v1/renovate", h.runRenovate)
	mux.HandleFunc("GET /api/v1/prs", h.listPullRequests)
	mux.HandleFunc("GET /api/v1/prs/{owner}/{repo}/{number}", h.getPullRequest)
	mux.HandleFunc("GET /api/v1/images", h.listRunningImages)
	mux.HandleFunc("GET /api/v1/apps", h.listApps)
	mux.HandleFunc("GET /api/v1/trends", h.trends)
	mux.HandleFunc("GET /metrics", h.metrics)

	return mux
}

func (h *handler) healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *handler) status(w http.ResponseWriter, r *http.Request) {
	run, err := h.store.LastSyncRun(r.Context())
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		writeError(w, err)
		return
	}

	// Report every configured cluster, including ones not yet inventoried
	stored, err := h.store.ListClusters(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	clusters := make([]store.Cluster, 0, len(h.clusters))
	for _, name := range h.clusters {
		c := store.Cluster{Name: name}
		for _, sc := range stored {
			if sc.Name == name {
				c = sc
			}
		}
		clusters = append(clusters, c)
	}

	repos := []string{}
	for _, r := range h.syncer.Repos() {
		repos = append(repos, r.Name)
	}
	config := *h.config
	config.Repos = h.syncer.RepoPatterns()

	writeJSON(w, http.StatusOK, &StatusResponse{
		Repos: repos, Clusters: clusters, LastRun: run, Config: &config, Activity: h.syncer.Activity(),
	})
}

// activity returns what the daemon is doing, its scan queue, caches and recent sync runs
func (h *handler) activity(w http.ResponseWriter, r *http.Request) {
	report, err := h.syncer.ActivityReport(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (h *handler) sync(w http.ResponseWriter, r *http.Request) {
	h.syncer.Trigger()
	w.WriteHeader(http.StatusAccepted)
}

// runRenovate starts a Renovate run on every repo, or on ?repo=owner/name
func (h *handler) runRenovate(w http.ResponseWriter, r *http.Request) {
	err := h.syncer.RunRenovate(daemon.RenovateRequest{Repo: r.URL.Query().Get("repo")})
	switch {
	case err == nil:
		w.WriteHeader(http.StatusAccepted)
	case errors.Is(err, daemon.ErrRenovateDisabled), errors.Is(err, daemon.ErrUnknownRepo):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, daemon.ErrRenovateBusy):
		http.Error(w, err.Error(), http.StatusConflict)
	default:
		writeError(w, err)
	}
}

// listPullRequests returns open PRs by default; ?state=all or ?state=closed for others
func (h *handler) listPullRequests(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	switch state {
	case "":
		state = store.PullRequestStateOpen
	case "all":
		state = ""
	case store.PullRequestStateOpen, store.PullRequestStateClosed:
	default:
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}

	prs, err := h.store.ListPullRequests(r.Context(), state)
	if err != nil {
		writeError(w, err)
		return
	}
	containers, err := h.containers(r)
	if err != nil {
		writeError(w, err)
		return
	}

	scans, err := h.scanIndex(r)
	if err != nil {
		writeError(w, err)
		return
	}

	views := make([]*PullRequestView, 0, len(prs))
	for _, pr := range prs {
		view := pullRequestView(pr, containers, len(h.clusters) > 0)
		if scans != nil {
			if err := attachScans(view, scans, false); err != nil {
				writeError(w, err)
				return
			}
		}
		views = append(views, view)
	}
	writeJSON(w, http.StatusOK, views)
}

func (h *handler) getPullRequest(w http.ResponseWriter, r *http.Request) {
	number, err := strconv.Atoi(r.PathValue("number"))
	if err != nil {
		http.Error(w, "invalid PR number", http.StatusBadRequest)
		return
	}

	pr, err := h.store.GetPullRequest(r.Context(), r.PathValue("owner")+"/"+r.PathValue("repo"), number)
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		writeError(w, err)
		return
	}
	containers, err := h.containers(r)
	if err != nil {
		writeError(w, err)
		return
	}
	scans, err := h.scanIndex(r)
	if err != nil {
		writeError(w, err)
		return
	}
	view := pullRequestView(pr, containers, len(h.clusters) > 0)
	if scans != nil {
		if err := attachScans(view, scans, true); err != nil {
			writeError(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, view)
}

func (h *handler) listRunningImages(w http.ResponseWriter, r *http.Request) {
	inv, err := h.inventory(r)
	if err != nil {
		writeError(w, err)
		return
	}
	prs, err := h.store.ListPullRequests(r.Context(), store.PullRequestStateOpen)
	if err != nil {
		writeError(w, err)
		return
	}
	images := runningImages(inv.containers, prs, inv.versions)
	attachSoaks(images, h.syncer.SoakDecisions())
	idx, err := h.scanIndex(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if idx != nil {
		if err := attachRunningScans(images, idx); err != nil {
			writeError(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, images)
}

func (h *handler) listApps(w http.ResponseWriter, r *http.Request) {
	inv, err := h.inventory(r)
	if err != nil {
		writeError(w, err)
		return
	}
	prs, err := h.store.ListPullRequests(r.Context(), store.PullRequestStateOpen)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, appViews(inv.apps, inv.renders, inv.containers, inv.versions, prs))
}

type inventory struct {
	containers []parsedContainer
	apps       []argocd.Application
	renders    []store.Render
	versions   versionIndex
}

// inventory loads running containers attributed to the Argo CD app sources that deploy them
func (h *handler) inventory(r *http.Request) (*inventory, error) {
	containers, err := h.store.ListContainers(r.Context())
	if err != nil {
		return nil, err
	}
	apps, err := h.store.ListApplications(r.Context())
	if err != nil {
		return nil, err
	}
	renders, err := h.store.ListRenders(r.Context())
	if err != nil {
		return nil, err
	}
	versions, err := h.store.ListVersions(r.Context())
	if err != nil {
		return nil, err
	}
	idx := newVersionIndex(versions)
	return &inventory{
		containers: parseContainers(containers, newAttribution(apps, renders, idx)),
		apps:       apps,
		renders:    renders,
		versions:   idx,
	}, nil
}

// scanIndex loads image resolutions and scans; nil when scanning is disabled
func (h *handler) scanIndex(r *http.Request) (*scanIndex, error) {
	if !h.scanning {
		return nil, nil
	}
	resolutions, err := h.store.ListResolutions(r.Context())
	if err != nil {
		return nil, err
	}
	scans, err := h.store.ListScans(r.Context())
	if err != nil {
		return nil, err
	}

	idx := &scanIndex{
		resolutions: make(map[string]store.Resolution, len(resolutions)),
		scans:       make(map[string]store.Scan, len(scans)),
		vulns: func(sc *store.Scan) ([]scan.Vulnerability, error) {
			return h.vulns.get(r.Context(), h.store, sc)
		},
	}
	for _, res := range resolutions {
		idx.resolutions[res.Ref] = res
	}
	for _, sc := range scans {
		idx.scans[sc.Digest] = sc
	}
	return idx, nil
}

func (h *handler) containers(r *http.Request) ([]parsedContainer, error) {
	inv, err := h.inventory(r)
	if err != nil {
		return nil, err
	}
	return inv.containers, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		svcLog.WithError(err).Error("error writing response")
	}
}

func writeError(w http.ResponseWriter, err error) {
	svcLog.WithError(err).Error("request failed")
	http.Error(w, "internal error", http.StatusInternalServerError)
}
