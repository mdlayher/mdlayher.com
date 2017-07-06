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
	// Only need the 10 most recently pushed repos.
	options := &github.RepositoryListOptions{
		Sort:        "pushed",
		ListOptions: github.ListOptions{PerPage: 10},
	}

	// Only need repos belonging to the specified user.
	ghrepos, _, err := c.c.Repositories.List(ctx, c.username, options)
	if err != nil {
		return nil, err
	}

	repos := make([]*Repository, 0, len(ghrepos))
	for _, r := range ghrepos {
		repos = append(repos, &Repository{
			Name:        *r.Name,
			Link:        *r.HTMLURL,
			Description: *r.Description,
		})
	}

	return repos, nil
}
