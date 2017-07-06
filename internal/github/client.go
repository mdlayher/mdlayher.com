package github

import (
	"context"
	"net/http"

	"github.com/google/go-github/github"
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

// NewClient creates a GitHub client that retrieves information
// for the user specified by username, using an optional HTTP client.
//
// If httpc is nil, a default client is used.
func NewClient(username string, httpc *http.Client) Client {
	ghc := github.NewClient(httpc)
	ghc.UserAgent = userAgent

	return &client{
		c:        ghc,
		username: username,
	}
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
