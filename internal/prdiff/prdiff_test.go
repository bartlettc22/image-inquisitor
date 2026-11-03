package prdiff

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	testMatrix := []struct {
		fixture         string
		expectedChanges []Change
	}{
		{
			fixture: "values_image.diff",
			expectedChanges: []Change{
				{File: "deploy/app/values.app.yaml", Kind: KindImage, Name: "example/app", From: "example/app:3.9.0", To: "example/app:5.4.0"},
			},
		},
		{
			fixture: "values_image_registry.diff",
			expectedChanges: []Change{
				{File: "deploy/nvr/values.yaml", Kind: KindImage, Name: "ghcr.io/example-org/nvr", From: "ghcr.io/example-org/nvr:0.16.4", To: "ghcr.io/example-org/nvr:0.18.2"},
			},
		},
		{
			fixture: "image_digest.diff",
			expectedChanges: []Change{
				{
					File: "deploy/db/values.yaml",
					Kind: KindImage,
					Name: "quay.io/example/db",
					From: "quay.io/example/db:v3.6.11@sha256:1111111111111111111111111111111111111111111111111111111111111111",
					To:   "quay.io/example/db:v3.6.12@sha256:2222222222222222222222222222222222222222222222222222222222222222",
				},
			},
		},
		{
			fixture: "multi_file.diff",
			expectedChanges: []Change{
				{File: "deploy/bot/values.yaml", Kind: KindImage, Name: "ghcr.io/example-org/bot", From: "ghcr.io/example-org/bot:41.112.0", To: "ghcr.io/example-org/bot:41.113.3"},
				{File: "deploy/hub/values.yaml", Kind: KindImage, Name: "example/hub", From: "example/hub:1.38.0", To: "example/hub:1.42.0"},
			},
		},
		{
			fixture: "helm_chart.diff",
			expectedChanges: []Change{
				{File: "deploy/operator/deploy.yaml", Kind: KindHelmChart, Name: "example-operator", RepoURL: "https://charts.example.com/stable", From: "1.92.5", To: "1.102.4"},
			},
		},
		{
			// Renovate reports OCI charts as "Docker tag" updates; they must not be treated as images
			fixture: "helm_oci.diff",
			expectedChanges: []Change{
				{File: "deploy/dashboards/deploy.yaml", Kind: KindHelmChart, Name: "dashboards", RepoURL: "oci://ghcr.io/example-org/helm-charts/dashboards", From: "12.4.2", To: "12.11.2"},
			},
		},
		{
			fixture: "dockerfile_from.diff",
			expectedChanges: []Change{
				{File: "Dockerfile", Kind: KindImage, Name: "golang", From: "golang:1.24-alpine", To: "golang:1.25-alpine"},
				{File: "Dockerfile", Kind: KindImage, Name: "alpine", From: "alpine:3.20", To: "alpine:3.22"},
			},
		},
		{
			fixture:         "github_actions.diff",
			expectedChanges: []Change{},
		},
	}

	for _, tm := range testMatrix {
		t.Run(tm.fixture, func(t *testing.T) {
			diff, err := os.ReadFile(filepath.Join("testdata", tm.fixture))
			require.NoError(t, err)

			changes, err := Parse(diff)
			require.NoError(t, err)
			assert.Equal(t, tm.expectedChanges, changes)
		})
	}
}

func TestImageName(t *testing.T) {
	testMatrix := []struct {
		ref          string
		expectedName string
	}{
		{ref: "nginx", expectedName: "nginx"},
		{ref: "nginx:1.29", expectedName: "nginx"},
		{ref: "registry.local:5000/team/app:1.0", expectedName: "registry.local:5000/team/app"},
		{ref: "registry.local:5000/team/app", expectedName: "registry.local:5000/team/app"},
		{ref: "ghcr.io/org/app:1.0@sha256:abc", expectedName: "ghcr.io/org/app"},
		{ref: "ghcr.io/org/app@sha256:abc", expectedName: "ghcr.io/org/app"},
	}

	for _, tm := range testMatrix {
		assert.Equal(t, tm.expectedName, imageName(tm.ref), tm.ref)
	}
}
