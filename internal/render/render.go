// Package render renders the Helm sources of Argo CD Applications with `helm template`,
// the way Argo CD does, to find the images each source deploys
package render

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/argocd"
)

const helmTimeout = 3 * time.Minute

// ErrNotFound is returned by a Fetcher for a file that doesn't exist
var ErrNotFound = errors.New("not found")

var commitSHA = regexp.MustCompile(`^[0-9a-f]{40}$`)

// Fetcher reads files from the git repositories app sources point at
type Fetcher interface {
	// File returns a file at a revision, or ErrNotFound
	File(ctx context.Context, repoURL, revision, path string) ([]byte, error)

	// Archive returns a gzipped tarball of the repository at a revision,
	// with everything under a single top-level directory
	Archive(ctx context.Context, repoURL, revision string) ([]byte, error)
}

// Capabilities describe the cluster to helm, as Argo CD passes them
type Capabilities struct {
	KubeVersion string   `json:"kubeVersion,omitempty"`
	APIVersions []string `json:"apiVersions,omitempty"`
}

// Request renders one source of an app
type Request struct {
	App          *argocd.Application
	Source       int
	Capabilities Capabilities

	// Revision overrides the source's revision, e.g. the chart version a PR updates to
	Revision string

	// RepoRevisions override the revision of any source (including value file refs)
	// from these repositories, keyed by repo URL, e.g. the values repo at a PR's head
	RepoRevisions map[string]string
}

func (r *Request) revision(i int) string {
	if i == r.Source && r.Revision != "" {
		return r.Revision
	}
	if rev, ok := r.RepoRevisions[r.App.Sources[i].RepoURL]; ok {
		return rev
	}
	return r.App.Version(i)
}

// Key identifies the inputs of a render, so an unchanged source isn't rendered again.
// Git revisions are commit SHAs once Argo CD has synced, so equal keys render equally.
func (r *Request) Key() string {
	type ref struct {
		Ref, RepoURL, Revision string
	}
	refs := []ref{}
	for i, s := range r.App.Sources {
		if s.Ref != "" {
			refs = append(refs, ref{s.Ref, s.RepoURL, r.revision(i)})
		}
	}
	// The synced revision also covers an OCI chart re-pushed under the same version
	synced := ""
	if r.Revision == "" {
		synced = r.App.Revision(r.Source)
	}
	b, _ := json.Marshal(struct {
		Source       argocd.Source
		Revision     string
		Synced       string
		Refs         []ref
		Release      string
		Namespace    string
		Capabilities Capabilities
	}{r.App.Sources[r.Source], r.revision(r.Source), synced, refs, r.App.ReleaseName(r.Source), r.App.DestNamespace, r.Capabilities})
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

type Renderer struct {
	// Helm is the helm binary
	Helm string

	// Dir holds helm's cache and repository checkouts
	Dir string

	Fetcher Fetcher
}

// Render runs helm template for the request's source and returns the images it declares
func (r *Renderer) Render(ctx context.Context, req *Request) ([]Image, error) {
	manifests, namespace, err := r.template(ctx, req)
	if err != nil {
		return nil, err
	}
	return Images(manifests, namespace)
}

func (r *Renderer) template(ctx context.Context, req *Request) ([]byte, string, error) {
	app := req.App
	src := app.Sources[req.Source]
	revision := req.revision(req.Source)

	tmp, err := os.MkdirTemp(r.dir("tmp"), "render-")
	if err != nil {
		return nil, "", err
	}
	defer os.RemoveAll(tmp)

	args := []string{"template", app.ReleaseName(req.Source)}
	chartDir := ""
	switch {
	case src.Chart != "" && strings.HasPrefix(src.RepoURL, "oci://"):
		args = append(args, strings.TrimSuffix(src.RepoURL, "/")+"/"+src.Chart, "--version", revision)
	case src.Chart != "" && !strings.Contains(src.RepoURL, "://"):
		// Argo CD's OCI Helm repositories are written without a scheme
		args = append(args, "oci://"+strings.TrimSuffix(src.RepoURL, "/")+"/"+src.Chart, "--version", revision)
	case src.Chart != "":
		args = append(args, src.Chart, "--repo", src.RepoURL, "--version", revision)
	case strings.HasPrefix(src.RepoURL, "oci://"):
		args = append(args, src.RepoURL, "--version", revision)
	default:
		root, err := r.checkout(ctx, src.RepoURL, revision)
		if err != nil {
			return nil, "", err
		}
		chartDir = filepath.Join(root, src.Path)
		if err := r.buildDependencies(ctx, chartDir); err != nil {
			return nil, "", err
		}
		args = append(args, chartDir)
	}

	namespace := app.DestNamespace
	if src.Helm != nil && src.Helm.Namespace != "" {
		namespace = src.Helm.Namespace
	}
	args = append(args, "--namespace", namespace, "--skip-tests")
	if req.Capabilities.KubeVersion != "" {
		args = append(args, "--kube-version", req.Capabilities.KubeVersion)
	}
	for _, v := range req.Capabilities.APIVersions {
		args = append(args, "--api-versions", v)
	}

	if h := src.Helm; h != nil {
		files := &valueFiles{renderer: r, req: req, chartDir: chartDir, dir: tmp}
		for _, vf := range h.ValueFiles {
			path, err := files.resolve(ctx, vf)
			if errors.Is(err, ErrNotFound) && h.IgnoreMissingValueFiles {
				continue
			}
			if err != nil {
				return nil, "", fmt.Errorf("value file %s: %w", vf, err)
			}
			args = append(args, "--values", path)
		}

		// valuesObject replaces values; JSON is valid YAML
		inline := []byte(h.Values)
		if len(h.ValuesObject) > 0 && string(h.ValuesObject) != "null" {
			inline = h.ValuesObject
		}
		if len(bytes.TrimSpace(inline)) > 0 {
			path, err := files.write(inline)
			if err != nil {
				return nil, "", err
			}
			args = append(args, "--values", path)
		}

		for _, p := range h.Parameters {
			flag := "--set"
			if p.ForceString {
				flag = "--set-string"
			}
			args = append(args, flag, p.Name+"="+p.Value)
		}
		for _, fp := range h.FileParameters {
			path, err := files.resolve(ctx, fp.Path)
			if err != nil {
				return nil, "", fmt.Errorf("file parameter %s: %w", fp.Name, err)
			}
			args = append(args, "--set-file", fp.Name+"="+path)
		}
	}

	out, err := r.helm(ctx, args...)
	return out, namespace, err
}

func (r *Renderer) helm(ctx context.Context, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, helmTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, r.Helm, args...)
	cmd.Env = append(os.Environ(),
		"HELM_CACHE_HOME="+r.dir("helm", "cache"),
		"HELM_CONFIG_HOME="+r.dir("helm", "config"),
		"HELM_DATA_HOME="+r.dir("helm", "data"),
	)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("helm %s: %s", args[0], msg)
	}
	return stdout.Bytes(), nil
}

// buildDependencies fetches a git chart's dependencies, as Argo CD does
func (r *Renderer) buildDependencies(ctx context.Context, chartDir string) error {
	chart, err := os.ReadFile(filepath.Join(chartDir, "Chart.yaml"))
	if err != nil {
		return fmt.Errorf("error reading chart: %w", err)
	}
	if !regexp.MustCompile(`(?m)^dependencies:`).Match(chart) {
		return nil
	}
	_, err = r.helm(ctx, "dependency", "build", chartDir)
	return err
}

// checkout extracts a repository at a revision. Commit SHAs are kept for reuse;
// other revisions (branches, tags) move, so they are extracted fresh.
func (r *Renderer) checkout(ctx context.Context, repoURL, revision string) (string, error) {
	sum := sha256.Sum256([]byte(repoURL))
	dir := filepath.Join(r.dir("git"), hex.EncodeToString(sum[:6]), revision)
	if commitSHA.MatchString(revision) {
		if _, err := os.Stat(dir); err == nil {
			// Mark it used, for PruneCheckouts
			now := time.Now()
			return dir, os.Chtimes(dir, now, now)
		}
	} else {
		dir = filepath.Join(r.dir("git"), hex.EncodeToString(sum[:6]), fmt.Sprintf("ref-%d", time.Now().UnixNano()))
	}

	archive, err := r.Fetcher.Archive(ctx, repoURL, revision)
	if err != nil {
		return "", fmt.Errorf("error fetching %s at %s: %w", repoURL, revision, err)
	}

	staging := dir + ".tmp"
	os.RemoveAll(staging)
	if err := extract(archive, staging); err != nil {
		os.RemoveAll(staging)
		return "", fmt.Errorf("error extracting %s at %s: %w", repoURL, revision, err)
	}
	os.RemoveAll(dir)
	if err := os.Rename(staging, dir); err != nil {
		return "", err
	}
	return dir, nil
}

// PruneCheckouts removes repository checkouts not used within maxAge and returns how many
func (r *Renderer) PruneCheckouts(maxAge time.Duration) (int, error) {
	checkouts, err := filepath.Glob(filepath.Join(r.dir("git"), "*", "*"))
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, dir := range checkouts {
		info, err := os.Stat(dir)
		if err != nil {
			return removed, err
		}
		if time.Since(info.ModTime()) > maxAge {
			if err := os.RemoveAll(dir); err != nil {
				return removed, err
			}
			removed++
		}
	}
	return removed, nil
}

// extract unpacks a gzipped tarball into dir, dropping its top-level directory.
// Only regular files and directories are written.
func extract(archive []byte, dir string) error {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}

		_, rel, ok := strings.Cut(h.Name, "/")
		if !ok || rel == "" {
			continue
		}
		target := filepath.Join(dir, rel)
		if !strings.HasPrefix(target, filepath.Clean(dir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid path in archive: %s", h.Name)
		}

		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return err
			}
			_, err = io.Copy(f, tr)
			// Close can be the first to report a failed write
			if cerr := f.Close(); err == nil {
				err = cerr
			}
			if err != nil {
				return err
			}
		}
	}
}

func (r *Renderer) dir(elem ...string) string {
	dir := filepath.Join(append([]string{r.Dir}, elem...)...)
	os.MkdirAll(dir, 0o755) //nolint:errcheck
	return dir
}

// valueFiles resolves value file paths to local files for one render
type valueFiles struct {
	renderer *Renderer
	req      *Request
	chartDir string
	dir      string
	n        int
}

// resolve returns a local path (or URL) for a value file: "$ref/path" is read from the
// ref source's repository, a URL is passed to helm as is, and a relative path is
// resolved against a git chart's directory
func (v *valueFiles) resolve(ctx context.Context, file string) (string, error) {
	if strings.HasPrefix(file, "http://") || strings.HasPrefix(file, "https://") {
		return file, nil
	}

	if strings.HasPrefix(file, "$") {
		ref, path, _ := strings.Cut(file[1:], "/")
		for i, s := range v.req.App.Sources {
			if s.Ref != ref {
				continue
			}
			content, err := v.renderer.Fetcher.File(ctx, s.RepoURL, v.req.revision(i), path)
			if err != nil {
				return "", err
			}
			return v.write(content)
		}
		return "", fmt.Errorf("no source with ref %q", ref)
	}

	if v.chartDir == "" {
		return "", fmt.Errorf("relative value files are only supported for git charts")
	}
	path := filepath.Join(v.chartDir, file)
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return "", ErrNotFound
	}
	return path, nil
}

func (v *valueFiles) write(content []byte) (string, error) {
	v.n++
	path := filepath.Join(v.dir, fmt.Sprintf("values-%d.yaml", v.n))
	return path, os.WriteFile(path, content, 0o600)
}
