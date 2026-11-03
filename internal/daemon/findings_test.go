package daemon

import (
	"testing"

	"github.com/bartlettc22/image-inquisitor/internal/scan"
	"github.com/bartlettc22/image-inquisitor/internal/store"
	"github.com/bartlettc22/image-inquisitor/internal/workloads"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunningVulnerabilities(t *testing.T) {
	container := func(cluster, name, pod, digest string) workloads.Container {
		return workloads.Container{Cluster: cluster, Namespace: "apps", WorkloadKind: "Deployment", WorkloadName: name, Pod: pod, Container: name, Digest: digest}
	}
	slot := func(cluster, name string) store.Slot {
		return store.Slot{Cluster: cluster, Namespace: "apps", WorkloadKind: "Deployment", WorkloadName: name, Container: name}
	}
	cve := []scan.Vulnerability{{ID: "CVE-1", Package: "openssl"}}

	loads := 0
	running, err := runningVulnerabilities([]workloads.Container{
		container("home", "web", "web-a", "sha256:w1"),
		container("home", "web", "web-b", "sha256:w1"),
		// Mid-rollout, the new digest not scanned yet
		container("home", "api", "api-a", "sha256:a1"),
		container("home", "api", "api-b", "sha256:a2"),
		container("home", "backup", "", ""),
		container("work", "web", "web-a", "sha256:w1"),
	}, map[string]bool{"sha256:w1": true, "sha256:a1": true, "sha256:a2": false}, func(digest string) ([]scan.Vulnerability, error) {
		loads++
		return cve, nil
	})
	require.NoError(t, err)

	assert.Equal(t, map[string]map[store.Slot]map[string][]scan.Vulnerability{
		"home": {
			slot("home", "web"):    {"sha256:w1": cve},
			slot("home", "api"):    nil,
			slot("home", "backup"): nil,
		},
		"work": {slot("work", "web"): {"sha256:w1": cve}},
	}, running)
	assert.Equal(t, 2, loads, "each scanned digest's vulnerabilities are loaded once")
}
