// Package prdiff extracts container image and Helm chart version changes
// from the unified diff of a dependency update PR (e.g. one opened by Renovate).
package prdiff

import (
	"bufio"
	"bytes"
	"path"
	"regexp"
	"strings"
)

type Kind string

const (
	KindImage     Kind = "image"
	KindHelmChart Kind = "helmChart"
)

// Change is a single dependency update found in a diff
type Change struct {
	File string `json:"file" yaml:"file"`
	Kind Kind   `json:"kind" yaml:"kind"`

	// Name is the image repository (without tag or digest) or the Helm chart name.
	// For Helm charts it is empty if the repoURL line is outside the hunk's context.
	Name string `json:"name" yaml:"name"`

	// RepoURL is the Helm chart repository (Helm charts only)
	RepoURL string `json:"repoURL,omitempty" yaml:"repoURL,omitempty"`

	// From and To are the old and new image references (images)
	// or the old and new chart versions (Helm charts)
	From string `json:"from" yaml:"from"`
	To   string `json:"to" yaml:"to"`
}

var (
	// image: <ref> in YAML (Kubernetes manifests, Helm values)
	imageLineRe = regexp.MustCompile(`^\s*(?:-\s+)?image:\s*["']?([^\s"']+)["']?\s*$`)

	// FROM [--platform=...] <ref> [AS <name>] in Dockerfiles
	fromLineRe = regexp.MustCompile(`(?i)^FROM\s+(?:--platform=\S+\s+)?(\S+)`)

	// Argo CD style Helm sources: repoURL, chart, targetRevision
	repoURLLineRe        = regexp.MustCompile(`^\s*(?:-\s+)?repoURL:\s*["']?([^\s"']+)["']?\s*$`)
	chartLineRe          = regexp.MustCompile(`^\s*(?:-\s+)?chart:\s*["']?([^\s"']+)["']?\s*$`)
	targetRevisionLineRe = regexp.MustCompile(`^\s*(?:-\s+)?targetRevision:\s*["']?([^\s"']+?)["']?\s*$`)
)

type helmSource struct {
	repoURL string
	chart   string
}

type helmRevision struct {
	source   helmSource
	revision string
}

// hunk accumulates the removed and added values of a single diff hunk.
// Old and new sides track their own Helm source context.
type hunk struct {
	file                       string
	removedImages, addedImages []string
	removedHelm, addedHelm     []helmRevision
	oldSource, newSource       helmSource
}

// Parse returns the image and Helm chart changes in a unified diff, in diff order
func Parse(diff []byte) ([]Change, error) {
	changes := []Change{}

	var file string
	var current *hunk
	inHeader := false

	flush := func() {
		if current != nil {
			changes = append(changes, current.changes()...)
		}
		current = nil
	}

	scanner := bufio.NewScanner(bytes.NewReader(diff))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "diff --git "):
			flush()
			file = ""
			inHeader = true
			continue
		case inHeader && strings.HasPrefix(line, "+++ "):
			file = strings.TrimPrefix(strings.TrimPrefix(line, "+++ "), "b/")
			if file == "/dev/null" {
				file = ""
			}
			continue
		case strings.HasPrefix(line, "@@"):
			flush()
			inHeader = false
			if file != "" {
				current = &hunk{file: file}
			}
			continue
		}

		if inHeader || current == nil {
			continue
		}

		// An empty line is a context line whose leading space was stripped
		op, body := byte(' '), ""
		if line != "" {
			op, body = line[0], line[1:]
		}
		current.addLine(op, body)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	flush()

	return changes, nil
}

func (h *hunk) addLine(op byte, body string) {
	if op != ' ' && op != '-' && op != '+' {
		return
	}

	if isDockerfile(h.file) {
		if m := fromLineRe.FindStringSubmatch(body); m != nil && !strings.Contains(m[1], "$") {
			h.addImage(op, m[1])
		}
		return
	}

	if !isYAML(h.file) {
		return
	}

	if m := imageLineRe.FindStringSubmatch(body); m != nil {
		h.addImage(op, m[1])
		return
	}

	if m := repoURLLineRe.FindStringSubmatch(body); m != nil {
		h.setSource(op, func(s *helmSource) { *s = helmSource{repoURL: m[1]} })
		return
	}

	if m := chartLineRe.FindStringSubmatch(body); m != nil {
		h.setSource(op, func(s *helmSource) { s.chart = m[1] })
		return
	}

	if m := targetRevisionLineRe.FindStringSubmatch(body); m != nil {
		switch op {
		case '-':
			h.removedHelm = append(h.removedHelm, helmRevision{h.oldSource, m[1]})
		case '+':
			h.addedHelm = append(h.addedHelm, helmRevision{h.newSource, m[1]})
		}
	}
}

func (h *hunk) addImage(op byte, ref string) {
	switch op {
	case '-':
		h.removedImages = append(h.removedImages, ref)
	case '+':
		h.addedImages = append(h.addedImages, ref)
	}
}

// setSource applies a Helm source update to the side(s) of the diff the line belongs to
func (h *hunk) setSource(op byte, update func(*helmSource)) {
	if op != '+' {
		update(&h.oldSource)
	}
	if op != '-' {
		update(&h.newSource)
	}
}

// changes pairs removed and added values: images by repository name,
// Helm revisions by chart source
func (h *hunk) changes() []Change {
	changes := []Change{}

	usedImages := make([]bool, len(h.addedImages))
	for _, from := range h.removedImages {
		name := imageName(from)
		for i, to := range h.addedImages {
			if usedImages[i] || imageName(to) != name || to == from {
				continue
			}
			usedImages[i] = true
			changes = append(changes, Change{
				File: h.file,
				Kind: KindImage,
				Name: name,
				From: from,
				To:   to,
			})
			break
		}
	}

	usedHelm := make([]bool, len(h.addedHelm))
	for _, from := range h.removedHelm {
		for i, to := range h.addedHelm {
			if usedHelm[i] || to.source != from.source || to.revision == from.revision {
				continue
			}
			usedHelm[i] = true
			changes = append(changes, Change{
				File:    h.file,
				Kind:    KindHelmChart,
				Name:    chartName(from.source),
				RepoURL: from.source.repoURL,
				From:    from.revision,
				To:      to.revision,
			})
			break
		}
	}

	return changes
}

// imageName strips the tag and digest from an image reference
func imageName(ref string) string {
	if i := strings.Index(ref, "@"); i >= 0 {
		ref = ref[:i]
	}
	if i := strings.LastIndex(ref, ":"); i > strings.LastIndex(ref, "/") {
		ref = ref[:i]
	}
	return ref
}

// chartName returns the chart name, which for OCI sources is the last path element of the repoURL
func chartName(s helmSource) string {
	if s.chart != "" {
		return s.chart
	}
	if strings.HasPrefix(s.repoURL, "oci://") {
		return path.Base(s.repoURL)
	}
	return ""
}

func isYAML(file string) bool {
	ext := path.Ext(file)
	return ext == ".yaml" || ext == ".yml"
}

func isDockerfile(file string) bool {
	base := strings.ToLower(path.Base(file))
	return strings.HasPrefix(base, "dockerfile") ||
		strings.HasPrefix(base, "containerfile") ||
		strings.HasSuffix(base, ".dockerfile")
}
