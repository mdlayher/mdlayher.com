package httptalks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
// from the specified URL.
func NewClient(addr string) Client {
	return newClient(addr)
}

// newClient is the internal constructor for a Client.
func newClient(addr string) Client {
	return &client{
		client: &http.Client{},
		addr:   addr,
	}
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
	req = req.WithContext(ctx)

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
