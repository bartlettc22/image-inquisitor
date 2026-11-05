package daemon

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/github"
	"github.com/bartlettc22/image-inquisitor/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLister struct {
	repos map[string][]string
	err   error
	calls []string
}

func (f *fakeLister) ListOwnerRepos(_ context.Context, owner string) ([]github.Repository, error) {
	f.calls = append(f.calls, owner)
	if f.err != nil {
		return nil, f.err
	}
	repos := []github.Repository{}
	for _, name := range f.repos[owner] {
		r := github.Repository{FullName: name}
		r.Archived = name == "example/old"
		r.Fork = name == "example/forked"
		repos = append(repos, r)
	}
	return repos, nil
}

func TestValidateRepo(t *testing.T) {
	assert.NoError(t, validateRepo(RepoConfig{Name: "example/deploy"}))
	assert.NoError(t, validateRepo(RepoConfig{Name: "example/*", Exclude: []string{"example/docs", "example/test-*"}}))
	assert.ErrorContains(t, validateRepo(RepoConfig{Name: "example"}), "must be owner/name")
	assert.ErrorContains(t, validateRepo(RepoConfig{Name: "*/deploy"}), "owner can't be a pattern")
	assert.ErrorContains(t, validateRepo(RepoConfig{Name: "example/[x"}), "invalid repo pattern")
	assert.ErrorContains(t, validateRepo(RepoConfig{Name: "example/deploy", Exclude: []string{"example/x"}}), "only applies to patterns")
}

func TestRepoSetRefresh(t *testing.T) {
	ctx := context.Background()
	lister := &fakeLister{repos: map[string][]string{
		"example": {"example/deploy", "example/Docs", "example/webapp", "example/old", "example/forked", "example/test-a"},
	}}
	s := newRepoSet([]RepoConfig{
		{Name: "other/app", BranchPrefix: "deps/"},
		{Name: "example/deploy", BranchPrefix: "deps/"},
		{Name: "example/*", Exclude: []string{"example/docs", "example/test-*"}, BranchPrefix: "renovate/"},
		{Name: "example/web*", BranchPrefix: "renovate/"},
	})
	assert.Equal(t, []Repo{{"other/app", "deps/"}, {"example/deploy", "deps/"}}, s.Repos(), "plain names before the first listing")
	assert.False(t, s.complete())

	summary, err := s.refresh(ctx, lister, time.Hour)
	require.NoError(t, err)
	assert.Equal(t, []string{"example"}, lister.calls, "each owner listed once")
	assert.Equal(t, "3 repos, 1 owner listed", summary)
	assert.Equal(t, []Repo{{"other/app", "deps/"}, {"example/deploy", "deps/"}, {"example/webapp", "renovate/"}}, s.Repos(),
		"archived, forked and excluded (ignoring case) repos are left out; the first entry's branch prefix wins")
	assert.True(t, s.complete())

	// Within maxAge nothing is listed again
	_, err = s.refresh(ctx, lister, time.Hour)
	require.NoError(t, err)
	assert.Len(t, lister.calls, 1)

	// A failed listing keeps the last matches
	lister.err = errors.New("rate limited")
	_, err = s.refresh(ctx, lister, 0)
	assert.ErrorContains(t, err, "rate limited")
	assert.Len(t, s.Repos(), 3)
	assert.False(t, s.complete())
	assert.Equal(t, "rate limited", s.Patterns()[2].Error)
}

func TestSyncRepoList(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "imginq.db"))
	require.NoError(t, err)
	defer st.Close()
	for _, repo := range []string{"example/deploy", "example/archived-since"} {
		require.NoError(t, st.UpsertPullRequest(ctx, &store.PullRequest{Repo: repo, Number: 1, State: store.PullRequestStateOpen}))
	}

	lister := &fakeLister{err: errors.New("unreachable")}
	d := &Daemon{store: st, repoLister: lister, repos: newRepoSet([]RepoConfig{{Name: "example/*"}})}

	_, err = d.syncRepoList(ctx)
	require.Error(t, err)
	repos, err := st.ListOpenPullRequestRepos(ctx)
	require.NoError(t, err)
	assert.Len(t, repos, 2, "nothing closed without a good listing")

	lister.err = nil
	lister.repos = map[string][]string{"example": {"example/deploy"}}
	d.repos = newRepoSet([]RepoConfig{{Name: "example/*"}})
	summary, err := d.syncRepoList(ctx)
	require.NoError(t, err)
	assert.Contains(t, summary, "no longer tracking example/archived-since")
	repos, err = st.ListOpenPullRequestRepos(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"example/deploy"}, repos)
}
