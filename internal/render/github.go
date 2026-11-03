package render

import (
	"context"
	"errors"
	"fmt"

	"github.com/bartlettc22/image-inquisitor/internal/github"
)

// GitHubFetcher reads files from github.com repositories through the GitHub API
type GitHubFetcher struct {
	Client *github.Client
}

func (f *GitHubFetcher) File(ctx context.Context, repoURL, revision, path string) ([]byte, error) {
	repo, ok := github.RepoFromURL(repoURL)
	if !ok {
		return nil, fmt.Errorf("unsupported repository %s: only github.com is supported", repoURL)
	}
	b, err := f.Client.FileContent(ctx, repo, revision, path)
	if errors.Is(err, github.ErrNotFound) {
		return nil, ErrNotFound
	}
	return b, err
}

func (f *GitHubFetcher) Archive(ctx context.Context, repoURL, revision string) ([]byte, error) {
	repo, ok := github.RepoFromURL(repoURL)
	if !ok {
		return nil, fmt.Errorf("unsupported repository %s: only github.com is supported", repoURL)
	}
	return f.Client.Tarball(ctx, repo, revision)
}
