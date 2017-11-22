package github

import (
	"context"
	"time"

	"github.com/google/go-github/github"
	"github.com/mdlayher/mdlayher.com/internal/memocache"
)

// A Client is a GitHub client.
type Client interface {
	ListRepositories(ctx context.Context) ([]*Repository, error)
}

// A Repository contains metadata about a GitHub repository.
type Repository struct {
	Name        string
	Link        string
	Description string
}

// userAgent identifies this client to GitHub's API.
const userAgent = "github.com/mdlayher/mdlayher.com/internal/github"

// NewClient creates a caching GitHub client that retrieves information
// for the user specified by username.  Data for subsequent calls
// is cached until the expiration period elapses.
func NewClient(username string, expire time.Duration) Client {
	return newClient(github.NewClient(nil), username, expire)
}

// newClient is the internal constructor for a Client.
func newClient(ghc *github.Client, username string, expire time.Duration) Client {
	ghc.UserAgent = userAgent

	return &cachingClient{
		cache: memocache.New(expire),
		client: &client{
			c:        ghc,
			username: username,
		},
	}
}

// A cachingClient is a caching GitHub client.
type cachingClient struct {
	cache  memocache.Cache
	client Client
}

// ListRepositories implements Client.
func (c *cachingClient) ListRepositories(ctx context.Context) ([]*Repository, error) {
	repos, err := c.cache.Get(func() (memocache.Object, error) {
		return c.client.ListRepositories(ctx)
	})
	if err != nil {
		return nil, err
	}

	return repos.([]*Repository), nil
}

var _ Client = &client{}

// A client is a simplified wrapper around the official go-github.Client.
type client struct {
	c        *github.Client
	username string
}

// ListRepositories implements Client.
func (c *client) ListRepositories(ctx context.Context) ([]*Repository, error) {
	// Grab 15 most recently pushed repositories, though we will limit the
	// number of results after filtering.
	options := &github.RepositoryListOptions{
		Sort:        "pushed",
		ListOptions: github.ListOptions{PerPage: 15},
	}

	// Only need repos belonging to the specified user.
	ghrepos, _, err := c.c.Repositories.List(ctx, c.username, options)
	if err != nil {
		return nil, err
	}

	var repos []*Repository
	for _, r := range ghrepos {
		// Skip archived repositories.
		if r.GetArchived() {
			continue
		}

		repos = append(repos, &Repository{
			Name:        *r.Name,
			Link:        *r.HTMLURL,
			Description: ptrString(r.Description),
		})

		// Only return 10 repositories at most.
		if len(repos) == 10 {
			break
		}
	}

	return repos, nil
}

// ptrString returns the string contents of a *string, or empty string
// if the pointer is nil.
func ptrString(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}
