package registries

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewImage(t *testing.T) {
	testMatrix := []struct {
		imageRef                   string
		expectedRef                string
		expectedNormalizedRef      string
		expectedTagRef             string
		expectedIsTagRef           bool
		expectedIsDigestRef        bool
		expectedRegistry           string
		expectedRepository         string
		expectedExpandedRepository string
		expectErrorContains        string
	}{
		{
			imageRef:                   "nginx",
			expectedRef:                "nginx",
			expectedNormalizedRef:      "index.docker.io/library/nginx:latest",
			expectedTagRef:             "latest",
			expectedIsTagRef:           true,
			expectedIsDigestRef:        false,
			expectedRegistry:           "index.docker.io",
			expectedRepository:         "library/nginx",
			expectedExpandedRepository: "index.docker.io/library/nginx",
		},
		{
			imageRef:                   "nginx:latest",
			expectedRef:                "nginx:latest",
			expectedNormalizedRef:      "index.docker.io/library/nginx:latest",
			expectedTagRef:             "latest",
			expectedIsTagRef:           true,
			expectedIsDigestRef:        false,
			expectedRegistry:           "index.docker.io",
			expectedRepository:         "library/nginx",
			expectedExpandedRepository: "index.docker.io/library/nginx",
		},
		{
			imageRef:                   "nginx/nginx:latest",
			expectedRef:                "nginx/nginx:latest",
			expectedNormalizedRef:      "index.docker.io/nginx/nginx:latest",
			expectedTagRef:             "latest",
			expectedIsTagRef:           true,
			expectedIsDigestRef:        false,
			expectedRegistry:           "index.docker.io",
			expectedRepository:         "nginx/nginx",
			expectedExpandedRepository: "index.docker.io/nginx/nginx",
		},
		{
			imageRef:                   "myregistry.io/a/b/c/nginx/nginx:v1.2.3",
			expectedRef:                "myregistry.io/a/b/c/nginx/nginx:v1.2.3",
			expectedNormalizedRef:      "myregistry.io/a/b/c/nginx/nginx:v1.2.3",
			expectedTagRef:             "v1.2.3",
			expectedIsTagRef:           true,
			expectedIsDigestRef:        false,
			expectedRegistry:           "myregistry.io",
			expectedRepository:         "a/b/c/nginx/nginx",
			expectedExpandedRepository: "myregistry.io/a/b/c/nginx/nginx",
		},

		{
			imageRef:                   "myregistry.io/a/b/c/nginx/nginx",
			expectedRef:                "myregistry.io/a/b/c/nginx/nginx",
			expectedNormalizedRef:      "myregistry.io/a/b/c/nginx/nginx:latest",
			expectedTagRef:             "latest",
			expectedIsTagRef:           true,
			expectedIsDigestRef:        false,
			expectedRegistry:           "myregistry.io",
			expectedRepository:         "a/b/c/nginx/nginx",
			expectedExpandedRepository: "myregistry.io/a/b/c/nginx/nginx",
		},
		{
			imageRef:                   "myregistry.io/a/b/c/nginx/nginx@sha256:aeba7989c0cf30d121a8bff080c21b1467147e533488af5e9607db35a42b2566",
			expectedRef:                "myregistry.io/a/b/c/nginx/nginx@sha256:aeba7989c0cf30d121a8bff080c21b1467147e533488af5e9607db35a42b2566",
			expectedNormalizedRef:      "myregistry.io/a/b/c/nginx/nginx@sha256:aeba7989c0cf30d121a8bff080c21b1467147e533488af5e9607db35a42b2566",
			expectedTagRef:             "",
			expectedIsTagRef:           false,
			expectedIsDigestRef:        true,
			expectedRegistry:           "myregistry.io",
			expectedRepository:         "a/b/c/nginx/nginx",
			expectedExpandedRepository: "myregistry.io/a/b/c/nginx/nginx",
		},
		{
			imageRef:            "myregistry.io/a/b/c/nginx/nginx@abc123",
			expectErrorContains: "could not parse reference",
		},
		{
			imageRef:                   "nginx:v1.2.3@sha256:aeba7989c0cf30d121a8bff080c21b1467147e533488af5e9607db35a42b2566",
			expectedRef:                "nginx:v1.2.3@sha256:aeba7989c0cf30d121a8bff080c21b1467147e533488af5e9607db35a42b2566",
			expectedNormalizedRef:      "index.docker.io/library/nginx@sha256:aeba7989c0cf30d121a8bff080c21b1467147e533488af5e9607db35a42b2566",
			expectedTagRef:             "",
			expectedIsTagRef:           false,
			expectedIsDigestRef:        true,
			expectedRegistry:           "index.docker.io",
			expectedRepository:         "library/nginx",
			expectedExpandedRepository: "index.docker.io/library/nginx",
		},
	}

	for _, tc := range testMatrix {
		img, err := NewImage(tc.imageRef)

		if tc.expectErrorContains != "" {
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectErrorContains)
		} else {
			require.NoError(t, err)
			assert.Equal(t, tc.expectedRef, img.Ref())
			assert.Equal(t, tc.expectedNormalizedRef, img.NormalizedRef())
			assert.Equal(t, tc.expectedTagRef, img.TagRef())
			assert.Equal(t, tc.expectedIsTagRef, img.IsTagRef())
			assert.Equal(t, tc.expectedIsDigestRef, img.IsDigestRef())
			assert.Equal(t, tc.expectedRegistry, img.Registry())
			assert.Equal(t, tc.expectedRepository, img.Repository())
			assert.Equal(t, tc.expectedExpandedRepository, img.RefPrefix())
		}
	}
}
