package httptalks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mdlayher/mdlayher.com/internal/memocache"
)

// userAgent identifies this client.
const userAgent = "github.com/mdlayher/mdlayher.com/internal/httptalks"

// A Client is an HTTP talks client.
type Client interface {
	ListTalks(ctx context.Context) ([]*Talk, error)
}

// A Talk is a presentation and its metadata.
type Talk struct {
	Title       string
	Description string
	AudioLink   string
	BlogLink    string
	SlidesLink  string
	VideoLink   string
}

// NewClient creates a caching HTTP talks client that retrieves talks
// from the specified  URL.  Data for subsequent calls is cached until
// the expiration period elapses.
func NewClient(addr string, expire time.Duration) Client {
	return newClient(addr, expire)
}

// newClient is the internal constructor for a Client.
func newClient(addr string, expire time.Duration) Client {
	return &cachingClient{
		cache: memocache.New(expire),
		client: &client{
			client: &http.Client{},
			addr:   addr,
		},
	}
}

var _ Client = &cachingClient{}

// A cachingClient is a caching HTTP talks client.
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

// A client fetches an HTTP URL and decodes its JSON output.
type client struct {
	client *http.Client
	addr   string
}

// ListTalks implements Client.
func (c *client) ListTalks(ctx context.Context) ([]*Talk, error) {
	req, err := http.NewRequest(http.MethodGet, c.addr, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("User-Agent", userAgent)

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// Check for 200-range status code.
	if c := res.StatusCode; c < 200 || c > 299 {
		return nil, fmt.Errorf("unexpected HTTP status while retrieving talks: %d", c)
	}

	var talks []*Talk
	if err := json.NewDecoder(res.Body).Decode(&talks); err != nil {
		return nil, err
	}

	return talks, nil
}
