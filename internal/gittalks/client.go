package gittalks

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/mdlayher/mdlayher.com/internal/memocache"

	"gopkg.in/src-d/go-billy.v3/memfs"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

// A Client is a git talks client.
type Client interface {
	ListTalks(ctx context.Context) ([]*Talk, error)
}

// A Talk is a presentation and its metadata.
type Talk struct {
	Title       string
	Description string
	SlidesLink  string
	VideoLink   string
}

const (
	// talksJSON is the file read from the git repository containing talk data.
	talksJSON = "talks.json"
)

// NewClient creates a caching git talks client that retrieves talks
// from the specified repository URL.  Data for subsequent calls
// is cached until the expiration period elapses.
func NewClient(repo string, expire time.Duration) Client {
	return newClient(&gitCloner{}, repo, expire)
}

// A cloner clones git repositories and returns the contents of talks.json.
type cloner interface {
	Clone(ctx context.Context, repo string) (io.ReadCloser, error)
}

// newClient is the internal constructor for a Client.
func newClient(c cloner, repo string, expire time.Duration) Client {
	return &cachingClient{
		cache: memocache.New(expire),
		client: &client{
			c:    c,
			repo: repo,
		},
	}
}

var _ Client = &cachingClient{}

// A cachingClient is a caching git talks client.
type cachingClient struct {
	cache  memocache.Cache
	client Client
}

// ListTalks implements Client.
func (c *cachingClient) ListTalks(ctx context.Context) ([]*Talk, error) {
	talks, err := c.cache.Get(func() (memocache.Object, error) {
		return c.client.ListTalks(ctx)
	})
	if err != nil {
		return nil, err
	}

	return talks.([]*Talk), nil
}

var _ Client = &client{}

// A client is a wrapper around a cloner to clone a repository and decode
// its talks.json file.
type client struct {
	c    cloner
	repo string
}

// ListTalks implements Client.
func (c *client) ListTalks(ctx context.Context) ([]*Talk, error) {
	rc, err := c.c.Clone(ctx, c.repo)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	var talks []*Talk
	if err := json.NewDecoder(rc).Decode(&talks); err != nil {
		return nil, err
	}

	return talks, nil
}

var _ cloner = &gitCloner{}

// A gitCloner is a cloner that uses real git operations.
type gitCloner struct{}

// Clone implements cloner.
func (*gitCloner) Clone(ctx context.Context, repo string) (io.ReadCloser, error) {
	// Clone repo in-memory.
	fs := memfs.New()

	_, err := git.CloneContext(ctx, memory.NewStorage(), fs, &git.CloneOptions{
		// Clone the repo as quickly as possible to just get the latest talks.json.
		URL:               repo,
		ReferenceName:     plumbing.Master,
		SingleBranch:      true,
		Depth:             1,
		RecurseSubmodules: git.NoRecurseSubmodules,
	})
	if err != nil {
		return nil, err
	}

	return fs.Open(talksJSON)
}
