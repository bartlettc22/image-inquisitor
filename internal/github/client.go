// Package github is a minimal GitHub REST client for reading pull requests
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultAPIURL = "https://api.github.com"

	pageSize = 100
)

// ErrNotFound is returned for a 404 response
var ErrNotFound = errors.New("not found")

type Client struct {
	apiURL string
	token  string
	http   *http.Client
}

type PullRequest struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	HTMLURL   string    `json:"html_url"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Head      struct {
		Ref string `json:"ref"`
		SHA string `json:"sha"`
	} `json:"head"`
	User struct {
		Login string `json:"login"`
	} `json:"user"`
}

func NewClient(apiURL, token string) *Client {
	if apiURL == "" {
		apiURL = DefaultAPIURL
	}
	return &Client{
		apiURL: strings.TrimSuffix(apiURL, "/"),
		token:  token,
		http:   &http.Client{Timeout: 2 * time.Minute},
	}
}

// ListOpenPullRequests returns all open pull requests in a repository ("owner/name")
func (c *Client) ListOpenPullRequests(ctx context.Context, repo string) ([]PullRequest, error) {
	prs := []PullRequest{}
	for page := 1; ; page++ {
		url := fmt.Sprintf("%s/repos/%s/pulls?state=open&per_page=%d&page=%d", c.apiURL, repo, pageSize, page)
		body, err := c.get(ctx, url, "application/vnd.github+json")
		if err != nil {
			return nil, err
		}

		pagePRs := []PullRequest{}
		if err := json.Unmarshal(body, &pagePRs); err != nil {
			return nil, fmt.Errorf("error decoding pull requests for %s: %w", repo, err)
		}
		prs = append(prs, pagePRs...)

		if len(pagePRs) < pageSize {
			return prs, nil
		}
	}
}

// Repository is a repository in a listing of an owner's repositories
type Repository struct {
	FullName string `json:"full_name"`
	Archived bool   `json:"archived"`
	Fork     bool   `json:"fork"`
	Owner    struct {
		Login string `json:"login"`
	} `json:"owner"`
}

// ListOwnerRepos returns the repositories of an organization or user that the token can see
func (c *Client) ListOwnerRepos(ctx context.Context, owner string) ([]Repository, error) {
	repos, err := c.listRepos(ctx, fmt.Sprintf("%s/orgs/%s/repos?type=all", c.apiURL, url.PathEscape(owner)))
	if !errors.Is(err, ErrNotFound) {
		return repos, err
	}

	// A user: their private repositories are only listed for the token's own user
	if c.token != "" {
		all, err := c.listRepos(ctx, c.apiURL+"/user/repos?affiliation=owner")
		if err != nil {
			return nil, err
		}
		own := []Repository{}
		for _, r := range all {
			if strings.EqualFold(r.Owner.Login, owner) {
				own = append(own, r)
			}
		}
		if len(own) > 0 {
			return own, nil
		}
	}
	return c.listRepos(ctx, fmt.Sprintf("%s/users/%s/repos?type=owner", c.apiURL, url.PathEscape(owner)))
}

func (c *Client) listRepos(ctx context.Context, listURL string) ([]Repository, error) {
	repos := []Repository{}
	for page := 1; ; page++ {
		body, err := c.get(ctx, fmt.Sprintf("%s&per_page=%d&page=%d", listURL, pageSize, page), "application/vnd.github+json")
		if err != nil {
			return nil, err
		}
		pageRepos := []Repository{}
		if err := json.Unmarshal(body, &pageRepos); err != nil {
			return nil, fmt.Errorf("error decoding repositories: %w", err)
		}
		repos = append(repos, pageRepos...)
		if len(pageRepos) < pageSize {
			return repos, nil
		}
	}
}

// PullRequestDiff returns the unified diff of a pull request
func (c *Client) PullRequestDiff(ctx context.Context, repo string, number int) ([]byte, error) {
	url := fmt.Sprintf("%s/repos/%s/pulls/%d", c.apiURL, repo, number)
	return c.get(ctx, url, "application/vnd.github.diff")
}

// FileContent returns a file's content at a ref, or ErrNotFound
func (c *Client) FileContent(ctx context.Context, repo, ref, path string) ([]byte, error) {
	u := fmt.Sprintf("%s/repos/%s/contents/%s?ref=%s", c.apiURL, repo, escapePath(path), url.QueryEscape(ref))
	return c.get(ctx, u, "application/vnd.github.raw+json")
}

// Tarball returns a gzipped tarball of the repository at a ref
func (c *Client) Tarball(ctx context.Context, repo, ref string) ([]byte, error) {
	u := fmt.Sprintf("%s/repos/%s/tarball/%s", c.apiURL, repo, url.PathEscape(ref))
	return c.get(ctx, u, "application/vnd.github+json")
}

// RepoFromURL returns "owner/name" for a github.com clone URL
// (https://github.com/owner/name.git, git@github.com:owner/name.git, ...)
func RepoFromURL(repoURL string) (string, bool) {
	rest := ""
	for _, prefix := range []string{"https://github.com/", "http://github.com/", "https://www.github.com/", "ssh://git@github.com/", "git@github.com:"} {
		if strings.HasPrefix(repoURL, prefix) {
			rest = strings.TrimPrefix(repoURL, prefix)
			break
		}
	}
	rest = strings.TrimSuffix(strings.TrimSuffix(rest, "/"), ".git")
	if strings.Count(rest, "/") != 1 || strings.HasPrefix(rest, "/") || strings.HasSuffix(rest, "/") {
		return "", false
	}
	return rest, true
}

func escapePath(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
}

func (c *Client) get(ctx context.Context, url string, accept string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("GET %s: %w", url, ErrNotFound)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s: %s", url, resp.Status, strings.TrimSpace(string(body)))
	}

	return body, nil
}
