// Package scan resolves images to digests and scans them with Trivy in two steps: an SBOM
// is generated once per digest (which pulls the image), then checked for vulnerabilities
// against Trivy's current database as often as needed (which takes well under a second)
package scan

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

const (
	SeverityCritical = "CRITICAL"
	SeverityHigh     = "HIGH"
	SeverityMedium   = "MEDIUM"
	SeverityLow      = "LOW"
	SeverityUnknown  = "UNKNOWN"

	// platform is the image platform scanned
	platform = "linux/amd64"

	sbomTimeout  = 20 * time.Minute
	vulnsTimeout = 10 * time.Minute
)

// ErrNoPlatform is returned for an image with no linux/amd64 variant (e.g. a Windows image),
// which can't run on the clusters being scanned for
var ErrNoPlatform = errors.New("no " + platform + " image")

// Severities are ordered most severe first
var Severities = []string{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, SeverityUnknown}

type Vulnerability struct {
	ID        string `json:"id"`
	Package   string `json:"package"`
	Installed string `json:"installed"`
	Fixed     string `json:"fixed,omitempty"`
	Severity  string `json:"severity"`
	Title     string `json:"title,omitempty"`
	URL       string `json:"url,omitempty"`
}

// Key identifies a vulnerability in a package, for comparing two scans
func (v Vulnerability) Key() string {
	return v.ID + "|" + v.Package
}

// Counts are vulnerabilities by severity
type Counts map[string]int

func CountBySeverity(vulns []Vulnerability) Counts {
	counts := Counts{}
	for _, v := range vulns {
		counts[v.Severity]++
	}
	return counts
}

// Diff returns the vulnerabilities of from that to doesn't have (fixed) and those of
// to that from doesn't have (introduced), most severe first
func Diff(from, to []Vulnerability) (fixed, introduced []Vulnerability) {
	return subtract(from, to), subtract(to, from)
}

func subtract(a, b []Vulnerability) []Vulnerability {
	inB := map[string]bool{}
	for _, v := range b {
		inB[v.Key()] = true
	}
	result := []Vulnerability{}
	seen := map[string]bool{}
	for _, v := range a {
		if !inB[v.Key()] && !seen[v.Key()] {
			seen[v.Key()] = true
			result = append(result, v)
		}
	}
	Sort(result)
	return result
}

// Sort orders vulnerabilities most severe first, then by ID
func Sort(vulns []Vulnerability) {
	rank := map[string]int{}
	for i, s := range Severities {
		rank[s] = i
	}
	sort.SliceStable(vulns, func(i, j int) bool {
		ri, ok := rank[vulns[i].Severity]
		if !ok {
			ri = len(Severities)
		}
		rj, ok := rank[vulns[j].Severity]
		if !ok {
			rj = len(Severities)
		}
		if ri != rj {
			return ri < rj
		}
		return vulns[i].Key() < vulns[j].Key()
	})
}

// RunningRef is the reference scanned for a container: its spec image (without any digest)
// pinned to the digest it runs
func RunningRef(image, digest string) string {
	base, _, _ := strings.Cut(image, "@")
	return base + "@" + digest
}

// versionTimeout bounds reading the version, which happens at startup
const versionTimeout = 10 * time.Second

// Version returns the version of the trivy binary
func (s *Scanner) Version(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, versionTimeout)
	defer cancel()

	out, err := s.trivy(ctx, "version", "--format", "json")
	if err != nil {
		return "", err
	}
	var version struct {
		Version string `json:"Version"`
	}
	if err := json.Unmarshal(out, &version); err != nil {
		return "", fmt.Errorf("error parsing trivy version: %w", err)
	}
	return version.Version, nil
}

// TagRef is the reference of a tag of a repository, e.g. the newest version of a running image
func TagRef(repository, tag string) string {
	return repository + ":" + tag
}

// ImageCreated returns when the linux/amd64 image of a reference was built, from its config.
// The zero time means unknown: reproducible builds (e.g. ko, distroless) date images 1970.
func ImageCreated(ctx context.Context, ref string) (time.Time, error) {
	r, err := name.ParseReference(ref, name.WeakValidation)
	if err != nil {
		return time.Time{}, err
	}
	p, err := v1.ParsePlatform(platform)
	if err != nil {
		return time.Time{}, err
	}
	img, err := remote.Image(r, remote.WithContext(ctx), remote.WithAuthFromKeychain(authn.DefaultKeychain), remote.WithPlatform(*p))
	if err != nil {
		return time.Time{}, err
	}
	cfg, err := img.ConfigFile()
	if err != nil {
		return time.Time{}, err
	}
	created := cfg.Created.UTC()
	if created.Year() < 2000 {
		return time.Time{}, nil
	}
	return created, nil
}

// Resolve returns the repository and digest an image reference points at.
// A reference that includes a digest isn't looked up.
func Resolve(ctx context.Context, ref string) (repository, digest string, err error) {
	base, d, _ := strings.Cut(ref, "@")
	r, err := name.ParseReference(base, name.WeakValidation)
	if err != nil {
		return "", "", err
	}
	if d != "" {
		return r.Context().Name(), d, nil
	}

	desc, err := remote.Head(r, remote.WithContext(ctx), remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return "", "", err
	}
	return r.Context().Name(), desc.Digest.String(), nil
}

type Scanner struct {
	// Trivy is the trivy binary
	Trivy string

	// Dir holds Trivy's cache and the SBOMs
	Dir string
}

// SBOMPath is where the SBOM of a digest is kept
func (s *Scanner) SBOMPath(digest string) string {
	return filepath.Join(s.Dir, "sbom", strings.ReplaceAll(digest, ":", "-")+".cdx.json")
}

// GenerateSBOM writes a CycloneDX SBOM of repository@digest to SBOMPath(digest)
func (s *Scanner) GenerateSBOM(ctx context.Context, repository, digest string) error {
	path := s.SBOMPath(digest)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, sbomTimeout)
	defer cancel()

	tmp := path + ".tmp"
	// Only the registry: an image of the same name in a local runtime may differ
	_, err := s.trivy(ctx, "image", "--quiet", "--image-src", "remote", "--platform", platform, "--format", "cyclonedx", "--output", tmp, repository+"@"+digest)
	if err != nil {
		os.Remove(tmp)
		if strings.Contains(err.Error(), "no child with platform") {
			return ErrNoPlatform
		}
		return err
	}
	return os.Rename(tmp, path)
}

// Vulnerabilities checks a digest's SBOM against Trivy's vulnerability database
func (s *Scanner) Vulnerabilities(ctx context.Context, digest string) ([]Vulnerability, error) {
	ctx, cancel := context.WithTimeout(ctx, vulnsTimeout)
	defer cancel()

	out, err := s.trivy(ctx, "sbom", "--quiet", "--format", "json", "--scanners", "vuln", s.SBOMPath(digest))
	if err != nil {
		return nil, err
	}
	return parseReport(out)
}

// report is the part of Trivy's JSON report that is read
type report struct {
	Results []struct {
		Vulnerabilities []struct {
			VulnerabilityID  string `json:"VulnerabilityID"`
			PkgName          string `json:"PkgName"`
			InstalledVersion string `json:"InstalledVersion"`
			FixedVersion     string `json:"FixedVersion"`
			Severity         string `json:"Severity"`
			Title            string `json:"Title"`
			PrimaryURL       string `json:"PrimaryURL"`
		} `json:"Vulnerabilities"`
	} `json:"Results"`
}

func parseReport(out []byte) ([]Vulnerability, error) {
	r := report{}
	if err := json.Unmarshal(out, &r); err != nil {
		return nil, fmt.Errorf("error parsing trivy report: %w", err)
	}
	vulns := []Vulnerability{}
	for _, result := range r.Results {
		for _, v := range result.Vulnerabilities {
			vulns = append(vulns, Vulnerability{
				ID: v.VulnerabilityID, Package: v.PkgName, Installed: v.InstalledVersion, Fixed: v.FixedVersion,
				Severity: v.Severity, Title: v.Title, URL: v.PrimaryURL,
			})
		}
	}
	Sort(vulns)
	return vulns, nil
}

func (s *Scanner) trivy(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, s.Trivy, args...)
	cmd.Env = append(os.Environ(), "TRIVY_CACHE_DIR="+filepath.Join(s.Dir, "trivy"))
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("trivy %s: %s", args[0], lastLines(msg, 5))
	}
	return stdout.Bytes(), nil
}

// lastLines keeps the end of trivy's output, where the error is
func lastLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
