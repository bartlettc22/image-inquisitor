package scan

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const trivyReport = `{
  "Results": [
    {"Target": "img (alpine 3.21)", "Vulnerabilities": [
      {"VulnerabilityID": "CVE-1", "PkgName": "zlib", "InstalledVersion": "1.0", "FixedVersion": "1.1", "Severity": "LOW"},
      {"VulnerabilityID": "CVE-2", "PkgName": "openssl", "InstalledVersion": "3.0", "FixedVersion": "3.1", "Severity": "CRITICAL", "Title": "bad", "PrimaryURL": "https://example.com/CVE-2"}
    ]},
    {"Target": "app", "Vulnerabilities": null}
  ]
}`

func TestParseReport(t *testing.T) {
	vulns, err := parseReport([]byte(trivyReport))
	require.NoError(t, err)
	assert.Equal(t, []Vulnerability{
		{ID: "CVE-2", Package: "openssl", Installed: "3.0", Fixed: "3.1", Severity: SeverityCritical, Title: "bad", URL: "https://example.com/CVE-2"},
		{ID: "CVE-1", Package: "zlib", Installed: "1.0", Fixed: "1.1", Severity: SeverityLow},
	}, vulns)
	assert.Equal(t, Counts{SeverityCritical: 1, SeverityLow: 1}, CountBySeverity(vulns))
}

func TestDiff(t *testing.T) {
	from := []Vulnerability{
		{ID: "CVE-1", Package: "zlib", Severity: SeverityLow},
		{ID: "CVE-2", Package: "openssl", Severity: SeverityCritical},
		{ID: "CVE-3", Package: "curl", Severity: SeverityHigh},
	}
	to := []Vulnerability{
		{ID: "CVE-1", Package: "zlib", Severity: SeverityLow},
		{ID: "CVE-4", Package: "libxml2", Severity: SeverityMedium},
		{ID: "CVE-4", Package: "libxml2", Severity: SeverityMedium},
	}

	fixed, introduced := Diff(from, to)
	assert.Equal(t, []Vulnerability{from[1], from[2]}, fixed, "most severe first")
	assert.Equal(t, []Vulnerability{to[1]}, introduced, "duplicates collapse")
}

func TestScannerRunsTrivy(t *testing.T) {
	dir := t.TempDir()
	trivy := filepath.Join(dir, "trivy")
	args := filepath.Join(dir, "args")
	script := `#!/bin/sh
echo "$@" >> ` + args + `
echo "cache=$TRIVY_CACHE_DIR" >> ` + args + `
if [ "$1" = image ]; then
  while [ "$1" != "--output" ]; do shift; done
  echo '{}' > "$2"
else
  cat <<'EOF'
` + trivyReport + `
EOF
fi
`
	require.NoError(t, os.WriteFile(trivy, []byte(script), 0o755))

	s := &Scanner{Trivy: trivy, Dir: filepath.Join(dir, "scan")}
	require.NoError(t, s.GenerateSBOM(context.Background(), "ghcr.io/org/app", "sha256:abc"))
	assert.FileExists(t, filepath.Join(dir, "scan", "sbom", "sha256-abc.cdx.json"))

	vulns, err := s.Vulnerabilities(context.Background(), "sha256:abc")
	require.NoError(t, err)
	assert.Len(t, vulns, 2)

	recorded, err := os.ReadFile(args)
	require.NoError(t, err)
	assert.Contains(t, string(recorded), "image --quiet --image-src remote --platform linux/amd64 --format cyclonedx --output "+filepath.Join(dir, "scan", "sbom", "sha256-abc.cdx.json.tmp")+" ghcr.io/org/app@sha256:abc")
	assert.Contains(t, string(recorded), "sbom --quiet --format json --scanners vuln "+filepath.Join(dir, "scan", "sbom", "sha256-abc.cdx.json"))
	assert.Contains(t, string(recorded), "cache="+filepath.Join(dir, "scan", "trivy"))
}

func TestResolveDigestReference(t *testing.T) {
	repo, digest, err := Resolve(context.Background(), "nginx:1.27@sha256:abc")
	require.NoError(t, err)
	assert.Equal(t, "index.docker.io/library/nginx", repo)
	assert.Equal(t, "sha256:abc", digest)
}

func TestVersion(t *testing.T) {
	dir := t.TempDir()
	script := func(body string) *Scanner {
		path := filepath.Join(dir, "trivy")
		require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755))
		return &Scanner{Trivy: path, Dir: dir}
	}

	version, err := script(`[ "$1 $2 $3" = "version --format json" ] && echo '{"Version":"0.65.0","VulnerabilityDB":{"Version":2}}'`).Version(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "0.65.0", version)

	_, err = script(`echo boom >&2; exit 1`).Version(context.Background())
	assert.ErrorContains(t, err, "boom")

	_, err = script(`echo not json`).Version(context.Background())
	assert.ErrorContains(t, err, "error parsing trivy version")
}
