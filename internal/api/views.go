package api

import (
	"slices"
	"sort"
	"strings"

	"github.com/bartlettc22/image-inquisitor/internal/daemon"
	"github.com/bartlettc22/image-inquisitor/internal/prdiff"
	"github.com/bartlettc22/image-inquisitor/internal/render"
	"github.com/bartlettc22/image-inquisitor/internal/scan"
	"github.com/bartlettc22/image-inquisitor/internal/store"
	"github.com/bartlettc22/image-inquisitor/internal/workloads"
	"github.com/google/go-containerregistry/pkg/name"
)

// Workload is a workload running an image, with the containers that run it
type Workload struct {
	Cluster    string   `json:"cluster"`
	Namespace  string   `json:"namespace"`
	Kind       string   `json:"kind"`
	Name       string   `json:"name"`
	ArgoApp    string   `json:"argoApp,omitempty"`
	Containers []string `json:"containers"`
	Digests    []string `json:"digests"`

	// Source is the Argo CD app source that deploys the workload, if known
	Source *Source `json:"source,omitempty"`
}

// renderStage is the daemon's stage that renders chart changes
const renderStage = "render"

type ChangeView struct {
	prdiff.Change

	// Running lists the workloads running the change's From version: those running the From
	// image, or deployed by the apps installing the From chart. Null when unknown (no clusters
	// configured, or the chart change not rendered yet).
	Running []Workload `json:"running"`

	// Renders are the image changes of a chart change, per app installing the chart
	Renders []ChartRenderView `json:"renders,omitempty"`

	// Scan compares the vulnerabilities of an image change's From and To images
	Scan *ScanDelta `json:"scan,omitempty"`
}

type ChartRenderView struct {
	Cluster string            `json:"cluster"`
	App     string            `json:"app"`
	Error   string            `json:"error,omitempty"`
	Images  []ImageChangeView `json:"images"`
}

// ImageChangeView is a container whose image a chart change updates, with the workloads running it
type ImageChangeView struct {
	render.ImageDiff
	Running []Workload `json:"running"`
	Scan    *ScanDelta `json:"scan,omitempty"`
}

type PullRequestView struct {
	*store.PullRequest
	Changes []ChangeView `json:"changes"`

	// Stages are the stored stages plus the scan stage, when the PR has images to scan
	Stages []store.Stage `json:"stages"`

	Scan *PullRequestScan `json:"scan,omitempty"`
}

// RunningImage is one image repository and tag running somewhere in the inventoried clusters
type RunningImage struct {
	Repository string     `json:"repository"`
	Tag        string     `json:"tag,omitempty"`
	Digests    []string   `json:"digests"`
	Workloads  []Workload `json:"workloads"`
	CoveredBy  []Coverage `json:"coveredBy"`

	// Version is the newest tag of the same variant, if looked up
	Version *store.Version `json:"version,omitempty"`

	// Status is whether the image needs an update: pending, current, pinned, behind or unknown
	Status string `json:"status"`

	// Scan is the scan of the running digest; unset when scanning is off or no digest is known
	Scan *ScanSide `json:"scan,omitempty"`

	// Update compares the running digest with the newest version, when there is a newer one
	Update *ScanDelta `json:"update,omitempty"`

	// Soak is how long the update to the newest version soaks before Renovate may propose it
	Soak *daemon.SoakDecision `json:"soak,omitempty"`

	// Vulnerabilities are the running digest's CVEs, only included for a single image
	Vulnerabilities []scan.Vulnerability `json:"vulnerabilities,omitempty"`

	// DigestScans are the scans of each digest when the tag runs as several, as Scan only covers
	// one of them; only included for a single image
	DigestScans []ScanSide `json:"digestScans,omitempty"`

	// runningRef is the reference the daemon scans for the image (scan.RunningRef), and
	// digestRefs that of each digest
	runningRef string
	digestRefs map[string]string
}

// Coverage is an open PR updating a running image's repository
type Coverage struct {
	Repo   string `json:"repo"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	From   string `json:"from"`
	To     string `json:"to"`

	// MatchesRunning is false when the PR updates a different version than the one running
	MatchesRunning bool `json:"matchesRunning"`

	// Via is the chart change that updates the image, e.g. "cert-manager v1.18.2 → v1.19.0";
	// empty when the PR updates the image directly
	Via string `json:"via,omitempty"`
}

// ImageDetail is everything known about one image repository
type ImageDetail struct {
	Repository string `json:"repository"`

	// Tags are the versions of the image running in the inventoried clusters
	Tags []RunningImage `json:"tags"`

	// PullRequests are the open PRs changing the image, whether or not it's running
	PullRequests []Coverage `json:"pullRequests"`
}

// imageRef is a normalized image reference: registry/repository plus tag and/or digest
type imageRef struct {
	repository string
	tag        string
	digest     string
}

// parseImage normalizes a reference, e.g. "mariadb:11.7" to "index.docker.io/library/mariadb" + "11.7".
// A reference with both a tag and a digest keeps both.
func parseImage(ref string) (imageRef, bool) {
	base, digest, _ := strings.Cut(ref, "@")
	r, err := name.ParseReference(base, name.WeakValidation)
	if err != nil {
		return imageRef{}, false
	}
	parsed := imageRef{repository: r.Context().Name(), digest: digest}
	if t, ok := r.(name.Tag); ok && (digest == "" || strings.Contains(base, ":"+t.TagStr())) {
		parsed.tag = t.TagStr()
	}
	return parsed, true
}

// matches reports whether a container runs this version of the image: same repository,
// and the same digest if both are known, otherwise the same tag
func (r imageRef) matches(c imageRef, runningDigest string) bool {
	if r.repository != c.repository {
		return false
	}
	if r.digest != "" && (runningDigest != "" || c.digest != "") {
		return r.digest == runningDigest || r.digest == c.digest
	}
	return r.tag != "" && r.tag == c.tag
}

type parsedContainer struct {
	workloads.Container
	ref    imageRef
	source *Source
}

// parseContainers parses containers' images and attributes them to their app sources
func parseContainers(containers []workloads.Container, a *attribution) []parsedContainer {
	parsed := make([]parsedContainer, 0, len(containers))
	for _, c := range containers {
		if ref, ok := parseImage(c.Image); ok {
			pc := parsedContainer{Container: c, ref: ref}
			pc.source = a.source(pc)
			parsed = append(parsed, pc)
		}
	}
	return parsed
}

// groupWorkloads collapses containers into one entry per workload
func groupWorkloads(containers []parsedContainer) []Workload {
	byKey := map[string]*Workload{}
	keys := []string{}
	for _, c := range containers {
		key := strings.Join([]string{c.Cluster, c.Namespace, c.WorkloadKind, c.WorkloadName}, "/")
		w, ok := byKey[key]
		if !ok {
			w = &Workload{
				Cluster: c.Cluster, Namespace: c.Namespace, Kind: c.WorkloadKind, Name: c.WorkloadName,
				ArgoApp: c.ArgoApp, Containers: []string{}, Digests: []string{},
			}
			byKey[key] = w
			keys = append(keys, key)
		}
		if !slices.Contains(w.Containers, c.Container.Container) {
			w.Containers = append(w.Containers, c.Container.Container)
		}
		if c.Digest != "" && !slices.Contains(w.Digests, c.Digest) {
			w.Digests = append(w.Digests, c.Digest)
		}
		if w.Source == nil {
			w.Source = c.source
		}
	}

	sort.Strings(keys)
	result := make([]Workload, 0, len(keys))
	for _, k := range keys {
		result = append(result, *byKey[k])
	}
	return result
}

// pullRequestView attaches the workloads running each change's From version, and the image
// changes of chart changes. haveInventory is false when no clusters are configured, leaving
// Running null.
func pullRequestView(pr *store.PullRequest, containers []parsedContainer, haveInventory bool) *PullRequestView {
	rendered := slices.ContainsFunc(pr.Stages, func(s store.Stage) bool { return s.Name == renderStage })

	view := &PullRequestView{PullRequest: pr, Changes: make([]ChangeView, 0, len(pr.Changes)), Stages: pr.Stages}
	for idx, change := range pr.Changes {
		cv := ChangeView{Change: change}
		switch change.Kind {
		case prdiff.KindImage:
			if from, ok := parseImage(change.From); ok && haveInventory {
				matching := []parsedContainer{}
				for _, c := range containers {
					if from.matches(c.ref, c.Digest) {
						matching = append(matching, c)
					}
				}
				cv.Running = groupWorkloads(matching)
			}

		case prdiff.KindHelmChart:
			if !haveInventory || !rendered {
				break
			}
			cv.Renders = []ChartRenderView{}
			matching := []parsedContainer{}
			for _, cr := range pr.ChartRenders {
				if cr.Change != idx {
					continue
				}
				for _, c := range containers {
					if c.source != nil && c.source.Cluster == cr.Cluster && c.source.App == cr.App && c.source.Index == cr.Source {
						matching = append(matching, c)
					}
				}
				cv.Renders = append(cv.Renders, chartRenderView(cr, containers))
			}
			cv.Running = groupWorkloads(matching)
		}
		view.Changes = append(view.Changes, cv)
	}
	return view
}

func chartRenderView(cr store.ChartRender, containers []parsedContainer) ChartRenderView {
	view := ChartRenderView{Cluster: cr.Cluster, App: cr.App, Error: cr.Error, Images: make([]ImageChangeView, 0, len(cr.Images))}
	for _, diff := range cr.Images {
		matching := []parsedContainer{}
		for _, c := range containers {
			if c.Cluster == cr.Cluster && c.Namespace == diff.Namespace && c.WorkloadKind == diff.Kind && c.WorkloadName == diff.Name &&
				(diff.Container == "" || c.Container.Container == diff.Container) {
				matching = append(matching, c)
			}
		}
		view.Images = append(view.Images, ImageChangeView{ImageDiff: diff, Running: groupWorkloads(matching)})
	}
	return view
}

// imageStatus is pending when an open PR updates the running version, else the version status,
// where an outdated image set by charts from Helm or OCI repositories is pinned by those charts
func imageStatus(image RunningImage) string {
	for _, c := range image.CoveredBy {
		if c.MatchesRunning {
			return StatusPending
		}
	}
	status := versionStatus(image.Version)
	if status != StatusBehind {
		return status
	}
	for _, w := range image.Workloads {
		if w.Source == nil || w.Source.Kind != "chart" {
			return StatusBehind
		}
	}
	return StatusPinned
}

// runningImages groups containers by repository and tag, and finds the open PRs updating each
// and whether a newer version exists
func runningImages(containers []parsedContainer, openPRs []*store.PullRequest, versions versionIndex) []RunningImage {
	type group struct {
		ref        imageRef
		containers []parsedContainer
	}
	groups := map[string]*group{}
	for _, c := range containers {
		key := c.ref.repository + ":" + c.ref.tag
		if c.ref.tag == "" {
			key = c.ref.repository + "@" + c.ref.digest
		}
		g, ok := groups[key]
		if !ok {
			g = &group{ref: c.ref}
			groups[key] = g
		}
		g.containers = append(g.containers, c)
	}

	images := make([]RunningImage, 0, len(groups))
	for _, g := range groups {
		image := RunningImage{
			Repository: g.ref.repository,
			Tag:        g.ref.tag,
			Digests:    []string{},
			Workloads:  groupWorkloads(g.containers),
			CoveredBy:  []Coverage{},
			digestRefs: map[string]string{},
		}
		for _, w := range image.Workloads {
			for _, d := range w.Digests {
				if !slices.Contains(image.Digests, d) {
					image.Digests = append(image.Digests, d)
				}
			}
		}

		for _, pr := range openPRs {
			for _, u := range imageUpdates(pr) {
				if imageRepository(u.from) != g.ref.repository {
					continue
				}
				if coverage := newCoverage(pr, u, g.containers); !slices.Contains(image.CoveredBy, coverage) {
					image.CoveredBy = append(image.CoveredBy, coverage)
				}
			}
		}

		for _, c := range g.containers {
			if c.Digest == "" {
				continue
			}
			ref := scan.RunningRef(c.Image, c.Digest)
			if image.runningRef == "" {
				image.runningRef = ref
			}
			image.digestRefs[c.Digest] = ref
		}

		image.Version = versions.get(store.VersionKindImage, image.Repository, image.Tag)
		image.Status = imageStatus(image)
		images = append(images, image)
	}

	sort.Slice(images, func(i, j int) bool {
		if images[i].Repository != images[j].Repository {
			return images[i].Repository < images[j].Repository
		}
		return images[i].Tag < images[j].Tag
	})
	return images
}

// imageUpdate is an image a PR changes, directly or through a chart. From is empty for a
// container a chart adds and To for one it removes.
type imageUpdate struct {
	from, to string

	// via is the chart change making the update, e.g. "cert-manager v1.18.2 → v1.19.0"
	via string
}

// imageUpdates lists the images a PR changes
func imageUpdates(pr *store.PullRequest) []imageUpdate {
	updates := []imageUpdate{}
	for _, change := range pr.Changes {
		if change.Kind == prdiff.KindImage {
			updates = append(updates, imageUpdate{from: change.From, to: change.To})
		}
	}
	for _, cr := range pr.ChartRenders {
		if cr.Change >= len(pr.Changes) {
			continue
		}
		change := pr.Changes[cr.Change]
		for _, diff := range cr.Images {
			updates = append(updates, imageUpdate{diff.From, diff.To, change.Name + " " + change.From + " → " + change.To})
		}
	}
	return updates
}

// imageRepository returns a reference's normalized repository, or "" if it isn't a valid one
func imageRepository(ref string) string {
	parsed, ok := parseImage(ref)
	if !ok {
		return ""
	}
	return parsed.repository
}

// newCoverage describes a PR's update of an image, checking its From version against containers
func newCoverage(pr *store.PullRequest, u imageUpdate, containers []parsedContainer) Coverage {
	coverage := Coverage{Repo: pr.Repo, Number: pr.Number, Title: pr.Title, From: u.from, To: u.to, Via: u.via}
	if from, ok := parseImage(u.from); ok {
		coverage.MatchesRunning = slices.ContainsFunc(containers, func(c parsedContainer) bool {
			return from.matches(c.ref, c.Digest)
		})
	}
	return coverage
}

// imageDetail collects the open PRs changing a repository, alongside its running tags
func imageDetail(repository string, tags []RunningImage, containers []parsedContainer, openPRs []*store.PullRequest) *ImageDetail {
	detail := &ImageDetail{Repository: repository, Tags: tags, PullRequests: []Coverage{}}
	for _, pr := range openPRs {
		for _, u := range imageUpdates(pr) {
			if imageRepository(u.from) != repository && imageRepository(u.to) != repository {
				continue
			}
			if coverage := newCoverage(pr, u, containers); !slices.Contains(detail.PullRequests, coverage) {
				detail.PullRequests = append(detail.PullRequests, coverage)
			}
		}
	}
	return detail
}

// attachRunningScans adds the scan of each image's running digest and, when a newer version
// exists, the CVEs updating to it fixes and introduces. Vulnerability lists are only included
// when withVulnerabilities is set.
func attachRunningScans(images []RunningImage, idx *scanIndex, withVulnerabilities bool) error {
	for i := range images {
		img := &images[i]
		if img.runningRef == "" {
			continue
		}
		side, vulns, err := idx.side(img.runningRef)
		if err != nil {
			return err
		}
		img.Scan = side
		if withVulnerabilities {
			img.Vulnerabilities = vulns
			if img.DigestScans, err = digestScans(img, idx); err != nil {
				return err
			}
		}
		if v := img.Version; v != nil && v.Error == "" && v.Latest != "" && v.Latest != v.Current {
			if img.Update, err = idx.delta(img.runningRef, scan.TagRef(img.Repository, v.Latest)); err != nil {
				return err
			}
			if withVulnerabilities {
				img.Update.withVulnerabilities()
			}
		}
	}
	return nil
}

// digestScans returns the scan of each digest of an image running as several, in image.Digests'
// order; nil when there's only one, which Scan already covers
func digestScans(image *RunningImage, idx *scanIndex) ([]ScanSide, error) {
	if len(image.Digests) < 2 {
		return nil, nil
	}
	sides := make([]ScanSide, 0, len(image.Digests))
	for _, digest := range image.Digests {
		side, _, err := idx.side(image.digestRefs[digest])
		if err != nil {
			return nil, err
		}
		side.Digest = digest
		sides = append(sides, *side)
	}
	return sides, nil
}

// attachSoaks adds each image's soak decision, for the update from its tag to the newest version
func attachSoaks(images []RunningImage, decisions []daemon.SoakDecision) {
	byTag := map[string]*daemon.SoakDecision{}
	for i := range decisions {
		byTag[decisions[i].Repository+":"+decisions[i].Current] = &decisions[i]
	}
	for i := range images {
		images[i].Soak = byTag[images[i].Repository+":"+images[i].Tag]
	}
}
