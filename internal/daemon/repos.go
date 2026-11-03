package daemon

import (
	"context"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/github"
)

// repoListMaxAge is how long an owner's repo listing is reused before listing it again
const repoListMaxAge = time.Hour

// Repo is a repository the daemon tracks, named in the config or matched by a pattern
type Repo struct {
	Name         string
	BranchPrefix string
}

// RepoPattern is a configured repos entry and the repositories it currently matches
type RepoPattern struct {
	Name         string   `json:"name"`
	Exclude      []string `json:"exclude,omitempty"`
	BranchPrefix string   `json:"branchPrefix"`

	// Pattern is whether Name has wildcards, so Matches come from listing the owner's repos
	Pattern bool     `json:"pattern"`
	Matches []string `json:"matches"`

	// Error is the last listing's error; Matches are then those of the last good listing
	Error    string     `json:"error,omitempty"`
	ListedAt *time.Time `json:"listedAt,omitempty"`
}

// repoLister lists an owner's repositories (github.Client)
type repoLister interface {
	ListOwnerRepos(ctx context.Context, owner string) ([]github.Repository, error)
}

// repoSet holds the configured repos entries with their matches
type repoSet struct {
	mu       sync.Mutex
	patterns []RepoPattern
}

func isPattern(name string) bool {
	return strings.ContainsAny(name, "*?[")
}

// validateRepo checks a repos entry: "owner/name", where name (not owner) may be a pattern
func validateRepo(r RepoConfig) error {
	owner, name, ok := strings.Cut(r.Name, "/")
	if !ok || owner == "" || name == "" || strings.Contains(name, "/") {
		return fmt.Errorf("invalid repo name %q: must be owner/name", r.Name)
	}
	if isPattern(owner) {
		return fmt.Errorf("invalid repo name %q: the owner can't be a pattern", r.Name)
	}
	if _, err := path.Match(r.Name, ""); err != nil {
		return fmt.Errorf("invalid repo pattern %q: %w", r.Name, err)
	}
	for _, ex := range r.Exclude {
		if _, err := path.Match(ex, ""); err != nil {
			return fmt.Errorf("invalid exclude pattern %q: %w", ex, err)
		}
	}
	if len(r.Exclude) > 0 && !isPattern(r.Name) {
		return fmt.Errorf("repo %q: exclude only applies to patterns", r.Name)
	}
	return nil
}

func newRepoSet(config []RepoConfig) *repoSet {
	s := &repoSet{}
	for _, r := range config {
		p := RepoPattern{Name: r.Name, Exclude: r.Exclude, BranchPrefix: r.BranchPrefix, Pattern: isPattern(r.Name), Matches: []string{}}
		if !p.Pattern {
			p.Matches = []string{r.Name}
		}
		s.patterns = append(s.patterns, p)
	}
	return s
}

// Patterns returns the configured repos entries with their current matches
func (s *repoSet) Patterns() []RepoPattern {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]RepoPattern, len(s.patterns))
	copy(out, s.patterns)
	return out
}

// Repos returns the tracked repos: every entry's matches, in config order, each repo once
// (with the branch prefix of the first entry matching it)
func (s *repoSet) Repos() []Repo {
	s.mu.Lock()
	defer s.mu.Unlock()
	repos := []Repo{}
	seen := map[string]bool{}
	for _, p := range s.patterns {
		for _, name := range p.Matches {
			if key := strings.ToLower(name); !seen[key] {
				seen[key] = true
				repos = append(repos, Repo{Name: name, BranchPrefix: p.BranchPrefix})
			}
		}
	}
	return repos
}

// complete reports whether every pattern has been listed and its last listing succeeded,
// so a repo missing from Repos really is no longer tracked
func (s *repoSet) complete() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.patterns {
		if p.Pattern && (p.ListedAt == nil || p.Error != "") {
			return false
		}
	}
	return true
}

// refresh lists the owners of patterns not listed within maxAge (each owner once) and updates
// their matches. A failed listing keeps the pattern's last matches. It returns a summary.
func (s *repoSet) refresh(ctx context.Context, lister repoLister, maxAge time.Duration) (string, error) {
	s.mu.Lock()
	patterns := make([]RepoPattern, len(s.patterns))
	copy(patterns, s.patterns)
	s.mu.Unlock()

	type listing struct {
		repos []github.Repository
		err   error
	}
	listings := map[string]*listing{}
	listed := 0
	errs := []error{}
	for i := range patterns {
		p := &patterns[i]
		if !p.Pattern || (p.ListedAt != nil && p.Error == "" && time.Since(*p.ListedAt) < maxAge) {
			continue
		}
		owner, _, _ := strings.Cut(p.Name, "/")
		l := listings[strings.ToLower(owner)]
		if l == nil {
			l = &listing{}
			l.repos, l.err = lister.ListOwnerRepos(ctx, owner)
			listings[strings.ToLower(owner)] = l
			listed++
		}

		now := time.Now().UTC()
		p.ListedAt = &now
		if l.err != nil {
			p.Error = l.err.Error()
			errs = append(errs, fmt.Errorf("%s: %w", p.Name, l.err))
			continue
		}
		p.Error = ""
		p.Matches = matchRepos(*p, l.repos)
	}

	s.mu.Lock()
	s.patterns = patterns
	s.mu.Unlock()

	tracked := len(s.Repos())
	summary := plural(tracked, "repo")
	if listed > 0 {
		summary += fmt.Sprintf(", %s listed", plural(listed, "owner"))
	}
	return summary, errors.Join(errs...)
}

// matchRepos returns the names of the active repositories that match a pattern and none of
// its excludes, sorted. Matching ignores case, as GitHub does.
func matchRepos(p RepoPattern, repos []github.Repository) []string {
	matches := []string{}
	for _, r := range repos {
		if r.Archived || r.Fork || !matchName(p.Name, r.FullName) {
			continue
		}
		excluded := false
		for _, ex := range p.Exclude {
			excluded = excluded || matchName(ex, r.FullName)
		}
		if !excluded {
			matches = append(matches, r.FullName)
		}
	}
	sort.Strings(matches)
	return matches
}

func matchName(pattern, name string) bool {
	ok, _ := path.Match(strings.ToLower(pattern), strings.ToLower(name))
	return ok
}

// Repos returns the repos the daemon tracks
func (d *Daemon) Repos() []Repo {
	return d.repos.Repos()
}

// RepoPatterns returns the configured repos entries with the repositories they match
func (d *Daemon) RepoPatterns() []RepoPattern {
	return d.repos.Patterns()
}

// syncRepoList refreshes the patterns' matches and closes the PRs of repos no longer tracked
func (d *Daemon) syncRepoList(ctx context.Context) (string, error) {
	summary, err := d.repos.refresh(ctx, d.repoLister, repoListMaxAge)

	// Only once every pattern has a good listing, so a GitHub outage doesn't close everything
	if d.repos.complete() {
		stored, serr := d.store.ListOpenPullRequestRepos(ctx)
		if serr != nil {
			return summary, errors.Join(err, serr)
		}
		tracked := map[string]bool{}
		for _, r := range d.repos.Repos() {
			tracked[strings.ToLower(r.Name)] = true
		}
		for _, repo := range stored {
			if !tracked[strings.ToLower(repo)] {
				if serr := d.store.CloseMissingPullRequests(ctx, repo, nil); serr != nil {
					return summary, errors.Join(err, serr)
				}
				summary += ", no longer tracking " + repo
			}
		}
	}
	return summary, err
}
