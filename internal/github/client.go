package github

import (
	"context"
	"sort"

	"github.com/google/go-github/v42/github"
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
	Tag         string
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
	// Grab 30 most recently pushed repositories, though we will limit the
	// number of results after filtering.
	options := &github.RepositoryListOptions{
		Sort:        "pushed",
		ListOptions: github.ListOptions{PerPage: 30},
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
		//   - this website (it's already linked)
		if r.GetArchived() || r.GetFork() || r.GetName() == "mdlayher.com" {
			continue
		}

		// Look for the latest tagged release, if one exists.
		releases, _, err := c.c.Repositories.ListReleases(
			ctx,
			c.username,
			r.GetName(), &github.ListOptions{PerPage: 1},
		)
		if err != nil {
			return nil, err
		}

		var tag string
		if len(releases) > 0 {
			tag = releases[0].GetTagName()
		}

		repos = append(repos, &Repository{
			Name:        r.GetName(),
			Link:        r.GetHTMLURL(),
			Description: r.GetDescription(),
			Tag:         tag,
		})

		// Only return X repositories at most.
		if len(repos) == 20 {
			break
		}
	}

	// Repos with tags are regularly maintained, sort those higher in the list.
	sort.SliceStable(repos, func(i, j int) bool {
		return repos[i].Tag != "" && repos[j].Tag == ""
	})

	return repos, nil
}
