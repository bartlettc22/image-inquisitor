package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/store"
	"github.com/bartlettc22/image-inquisitor/internal/workloads"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetImage(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	defer s.Close()
	require.NoError(t, s.ReplaceContainers(ctx, "home", []workloads.Container{
		{Namespace: "db", WorkloadKind: "StatefulSet", WorkloadName: "db", Pod: "db-0", Container: "db", Image: "mariadb:11.6"},
		{Namespace: "web", WorkloadKind: "Deployment", WorkloadName: "app", Pod: "app-1", Container: "app", Image: "ghcr.io/org/app:1.0"},
	}, time.Now(), nil))

	srv := httptest.NewServer(NewHandler(s, fakeSyncer{}, &Config{}))
	defer srv.Close()

	get := func(path string) (int, ImageDetail) {
		resp, err := http.Get(srv.URL + "/api/v1/images/" + path)
		require.NoError(t, err)
		defer resp.Body.Close()
		var detail ImageDetail
		if resp.StatusCode == http.StatusOK {
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&detail))
		}
		return resp.StatusCode, detail
	}

	for path, repository := range map[string]string{
		"mariadb":                         "index.docker.io/library/mariadb",
		"index.docker.io/library/mariadb": "index.docker.io/library/mariadb",
		"ghcr.io/org/app":                 "ghcr.io/org/app",
	} {
		status, detail := get(path)
		require.Equal(t, http.StatusOK, status, path)
		assert.Equal(t, repository, detail.Repository, path)
		require.Len(t, detail.Tags, 1, path)
	}

	status, detail := get("ghcr.io/org/gone")
	assert.Equal(t, http.StatusOK, status, "an image that's gone isn't an error")
	assert.Empty(t, detail.Tags)

	status, _ = get("NOT%20VALID")
	assert.Equal(t, http.StatusBadRequest, status)

	// The list route is still separate
	resp, err := http.Get(srv.URL + "/api/v1/images")
	require.NoError(t, err)
	defer resp.Body.Close()
	var images []RunningImage
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&images))
	assert.Len(t, images, 2)
}
