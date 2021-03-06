package github

import (
	"context"

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

// NewClient creates a GitHub client that retrieves information for the user
// specified by username.
func NewClient(username string) Client {
	return newClient(github.NewClient(nil), username)
}

// newClient is the internal constructor for a Client.
func newClient(ghc *github.Client, username string) Client {
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
		// Skip:
		//   - archived repositories
		//   - forks
		if r.GetArchived() || r.GetFork() {
			continue
		}

		repos = append(repos, &Repository{
			Name:        r.GetName(),
			Link:        r.GetHTMLURL(),
			Description: r.GetDescription(),
		})

		// Only return 10 repositories at most.
		if len(repos) == 10 {
			break
		}
	}

	return repos, nil
}
