// Package argocd reads Argo CD Applications from a cluster: their sources and the
// revisions Argo CD last synced, which is enough to render what each app deploys
package argocd

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var applicationsResource = schema.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: "applications"}

// Application is the subset of an Argo CD Application needed to render it
type Application struct {
	Cluster   string `json:"cluster"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Project   string `json:"project"`

	// DestNamespace is the namespace the app deploys to
	DestNamespace string `json:"destNamespace"`

	Sources []Source `json:"sources"`

	// Revisions are the revisions last synced, one per source (chart versions, commit SHAs)
	Revisions []string `json:"revisions"`

	// SourceTypes are Argo CD's detected type per source (Helm, Kustomize, Directory, ...)
	SourceTypes []string `json:"sourceTypes"`

	SyncStatus string `json:"syncStatus"`
	Health     string `json:"health"`
}

type Source struct {
	RepoURL        string `json:"repoURL"`
	Chart          string `json:"chart,omitempty"`
	Path           string `json:"path,omitempty"`
	TargetRevision string `json:"targetRevision,omitempty"`
	Ref            string `json:"ref,omitempty"`
	Helm           *Helm  `json:"helm,omitempty"`
}

type Helm struct {
	ReleaseName             string          `json:"releaseName,omitempty"`
	Namespace               string          `json:"namespace,omitempty"`
	ValueFiles              []string        `json:"valueFiles,omitempty"`
	IgnoreMissingValueFiles bool            `json:"ignoreMissingValueFiles,omitempty"`
	Values                  string          `json:"values,omitempty"`
	ValuesObject            json.RawMessage `json:"valuesObject,omitempty"`
	Parameters              []Parameter     `json:"parameters,omitempty"`
	FileParameters          []FileParameter `json:"fileParameters,omitempty"`
}

type Parameter struct {
	Name        string `json:"name"`
	Value       string `json:"value"`
	ForceString bool   `json:"forceString,omitempty"`
}

type FileParameter struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// IsHelm reports whether source i is rendered with Helm: a chart from a Helm or OCI
// repository, or a git path Argo CD detected as a Helm chart
func (a *Application) IsHelm(i int) bool {
	s := a.Sources[i]
	if s.Ref != "" && s.Chart == "" && s.Path == "" {
		return false
	}
	if s.Chart != "" || strings.HasPrefix(s.RepoURL, "oci://") {
		return true
	}
	if i < len(a.SourceTypes) && a.SourceTypes[i] != "" {
		return a.SourceTypes[i] == "Helm"
	}
	return s.Helm != nil
}

// Revision returns the revision source i was last synced at, else its target revision
func (a *Application) Revision(i int) string {
	if i < len(a.Revisions) && a.Revisions[i] != "" {
		return a.Revisions[i]
	}
	return a.Sources[i].TargetRevision
}

// Version is the deployed chart version, or commit for git sources. Argo CD records an
// OCI chart's manifest digest as its revision, so for those it is the target revision.
func (a *Application) Version(i int) string {
	if rev := a.Revision(i); !strings.HasPrefix(rev, "sha256:") {
		return rev
	}
	return a.Sources[i].TargetRevision
}

// ReleaseName is the Helm release name Argo CD uses for source i
func (a *Application) ReleaseName(i int) string {
	if h := a.Sources[i].Helm; h != nil && h.ReleaseName != "" {
		return h.ReleaseName
	}
	return a.Name
}

// List returns every Application in the cluster, or none if Argo CD isn't installed
func List(ctx context.Context, client dynamic.Interface, cluster string) ([]Application, error) {
	list, err := client.Resource(applicationsResource).List(ctx, metav1.ListOptions{})
	if apierrors.IsNotFound(err) {
		return []Application{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error listing argo cd applications: %w", err)
	}

	apps := make([]Application, 0, len(list.Items))
	for i := range list.Items {
		app, err := parse(&list.Items[i])
		if err != nil {
			return nil, fmt.Errorf("argo cd application %s/%s: %w", list.Items[i].GetNamespace(), list.Items[i].GetName(), err)
		}
		app.Cluster = cluster
		apps = append(apps, app)
	}
	return apps, nil
}

// manifest is the part of the Application resource that is read
type manifest struct {
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
	Spec struct {
		Project     string `json:"project"`
		Destination struct {
			Namespace string `json:"namespace"`
		} `json:"destination"`
		Source  *Source  `json:"source"`
		Sources []Source `json:"sources"`
	} `json:"spec"`
	Status struct {
		Sync struct {
			Status    string   `json:"status"`
			Revision  string   `json:"revision"`
			Revisions []string `json:"revisions"`
		} `json:"sync"`
		Health struct {
			Status string `json:"status"`
		} `json:"health"`
		SourceType  string   `json:"sourceType"`
		SourceTypes []string `json:"sourceTypes"`
	} `json:"status"`
}

func parse(u *unstructured.Unstructured) (Application, error) {
	b, err := json.Marshal(u.Object)
	if err != nil {
		return Application{}, err
	}
	m := manifest{}
	if err := json.Unmarshal(b, &m); err != nil {
		return Application{}, err
	}

	app := Application{
		Name:          m.Metadata.Name,
		Namespace:     m.Metadata.Namespace,
		Project:       m.Spec.Project,
		DestNamespace: m.Spec.Destination.Namespace,
		Sources:       m.Spec.Sources,
		Revisions:     m.Status.Sync.Revisions,
		SourceTypes:   m.Status.SourceTypes,
		SyncStatus:    m.Status.Sync.Status,
		Health:        m.Status.Health.Status,
	}

	// Single-source apps use spec.source and the singular status fields
	if len(app.Sources) == 0 && m.Spec.Source != nil {
		app.Sources = []Source{*m.Spec.Source}
		app.Revisions = []string{m.Status.Sync.Revision}
		app.SourceTypes = []string{m.Status.SourceType}
	}
	if app.Sources == nil {
		app.Sources = []Source{}
	}

	return app, nil
}

// ChartRef identifies the chart a Helm source installs regardless of how it is written:
// "oci://registry/path/chart" for OCI charts, "<repoURL>/<chart>" for Helm repositories
// and "<repoURL>//<path>" for git charts
func (s Source) ChartRef() string {
	repoURL := strings.TrimSuffix(s.RepoURL, "/")
	switch {
	case s.Chart == "" && strings.HasPrefix(repoURL, "oci://"):
		return repoURL
	case s.Chart == "":
		return repoURL + "//" + strings.Trim(s.Path, "/")
	case !strings.Contains(repoURL, "://"):
		return "oci://" + repoURL + "/" + s.Chart
	default:
		return repoURL + "/" + s.Chart
	}
}

// ChartRefOf is ChartRef for a chart written as a repo URL and name, as in a Renovate
// chart update, where an OCI repo URL may already end in the chart's name
func ChartRefOf(repoURL, chart string) string {
	repoURL = strings.TrimSuffix(repoURL, "/")
	if strings.HasPrefix(repoURL, "oci://") && strings.HasSuffix(repoURL, "/"+chart) {
		return repoURL
	}
	return Source{RepoURL: repoURL, Chart: chart}.ChartRef()
}

// IsRepoChart reports whether a source is a chart from a Helm or OCI repository
// (rather than a git repository), which has published versions
func (s Source) IsRepoChart() bool {
	return s.Chart != "" || strings.HasPrefix(s.RepoURL, "oci://")
}

// ChartName is the short name of the chart a source installs
func (s Source) ChartName() string {
	switch {
	case s.Chart != "":
		return s.Chart
	case strings.HasPrefix(s.RepoURL, "oci://"):
		return path.Base(strings.TrimSuffix(s.RepoURL, "/"))
	default:
		return path.Base(strings.Trim(s.Path, "/"))
	}
}
