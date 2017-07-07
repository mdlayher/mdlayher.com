package medium

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/mdlayher/mdlayher.com/internal/memocache"
)

const (
	// Medium's undocumented API requires application/json without the
	// charset parameter, but the API returns JSON with the charset
	// parameter set to UTF-8.
	applicationJSON     = "application/json"
	applicationJSONUTF8 = "application/json; charset=utf-8"

	// userAgent identifies this client to Medium's API.
	userAgent = "github.com/mdlayher/mdlayher.com/internal/medium"
)

// baseURL is the base URL for medium.com.
var baseURL = func() *url.URL {
	u, err := url.Parse("https://medium.com/")
	if err != nil {
		panic(fmt.Sprintf("failed to parse medium URL: %v", err))
	}

	return u
}()

// jsonHijackingPrefix is a prefix returned by Medium that must
// be removed.
var jsonHijackingPrefix = []byte("])}while(1);</x>")

// A Client is a Medium client.
type Client interface {
	ListPosts() ([]*Post, error)
}

// A Post contains metadata about a Medium post.
type Post struct {
	Title    string
	Subtitle string
	Link     string

	created int
}

// NewClient creates a caching Medium client that retrieves information
// for the user specified by username.  Data for subsequent calls is
// cached until the expiration period elapses.
func NewClient(username string, expire time.Duration) Client {
	return newClient(baseURL, username, expire)
}

// newClient is the internal constructor for a Client.
func newClient(apiURL *url.URL, username string, expire time.Duration) Client {
	return &cachingClient{
		cache: memocache.New(expire),
		client: &client{
			client:   &http.Client{},
			apiURL:   apiURL,
			username: username,
		},
	}
}

// A cachingClient is a caching Medium client.
type cachingClient struct {
	cache  memocache.Cache
	client Client
}

// ListPosts implements Client.
func (c *cachingClient) ListPosts() ([]*Post, error) {
	posts, err := c.cache.Get(func() (memocache.Object, error) {
		return c.client.ListPosts()
	})
	if err != nil {
		return nil, err
	}

	return posts.([]*Post), nil
}

var _ Client = &client{}

// A client is a basic Medium client that can retrieve metadata
// about posts.
type client struct {
	client   *http.Client
	apiURL   *url.URL
	username string
}

// postMetadata is the structure returned by Medium containing post metadata.
type postMetadata struct {
	Payload struct {
		References struct {
			Post map[string]rawPost `json:"Post"`
		} `json:"references"`
	} `json:"payload"`
}

// rawPost is the raw structure of a Post.
type rawPost struct {
	CreatedAt  int            `json:"createdAt"`
	Title      string         `json:"title"`
	UniqueSlug string         `json:"uniqueSlug"`
	Content    rawPostContent `json:"content"`
}

// rawPostContent is an object within a rawPost containing more metadata.
type rawPostContent struct {
	Subtitle string `json:"subtitle"`
}

// ListPosts implements Client.
func (c *client) ListPosts() ([]*Post, error) {
	req, err := c.newRequest(
		http.MethodGet,
		fmt.Sprintf("/@%s/latest", c.username),
	)
	if err != nil {
		return nil, err
	}

	var md postMetadata
	if _, err := c.do(req, &md); err != nil {
		return nil, err
	}

	posts := make([]*Post, 0, len(md.Payload.References.Post))
	for _, v := range md.Payload.References.Post {
		// Make a copy of baseURL for modification.
		link := *baseURL
		link.Path = fmt.Sprintf("/@%s/%s", c.username, v.UniqueSlug)

		posts = append(posts, &Post{
			Title:    v.Title,
			Subtitle: v.Content.Subtitle,
			Link:     link.String(),
			created:  v.CreatedAt,
		})
	}

	// Sort posts newest to oldest.
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].created > posts[j].created
	})

	return posts, nil
}

// newRequest creates a new HTTP request, using the specified HTTP method and
// API endpoint. Additionally, it accepts a struct which can be marshaled to
// a JSON body.
func (c *client) newRequest(method string, endpoint string) (*http.Request, error) {
	rel, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	u := c.apiURL.ResolveReference(rel)

	req, err := http.NewRequest(method, u.String(), nil)
	if err != nil {
		return nil, err
	}

	// Must add Accept header to get JSON back.
	req.Header.Add("Accept", applicationJSON)
	req.Header.Add("User-Agent", userAgent)

	return req, nil
}

// do performs an HTTP request using req and unmarshals the result onto v, if
// v is not nil.
func (c *client) do(req *http.Request, v interface{}) (*http.Response, error) {
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if err := checkResponse(res); err != nil {
		return res, err
	}

	// If no second parameter was passed, do not attempt to handle response.
	if v == nil {
		return res, nil
	}

	// Medium's JSON has a prefix to prevent JSON hijacking.
	// Remove it before parsing.
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	b = bytes.TrimPrefix(b, jsonHijackingPrefix)

	return res, json.Unmarshal(b, v)
}

// checkResponse checks for correct content type in a response and for non-200
// HTTP status codes, and returns any errors encountered.
func checkResponse(res *http.Response) error {
	if cType := res.Header.Get("Content-Type"); cType != applicationJSONUTF8 {
		return fmt.Errorf("expected %q content type, but received %q", applicationJSONUTF8, cType)
	}

	// Check for 200-range status code.
	if c := res.StatusCode; 200 <= c && c <= 299 {
		return nil
	}

	return fmt.Errorf("unexpected HTTP status code: %d", res.StatusCode)
}
