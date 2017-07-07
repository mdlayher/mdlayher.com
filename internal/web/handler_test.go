package web

import (
	"context"
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mdlayher/mdlayher.com/internal/github"
	"github.com/mdlayher/mdlayher.com/internal/medium"
)

func Test_handlerRedirectSubdomainTLS(t *testing.T) {
	const domain = "mdlayher.com"

	h := NewHandler(StaticContent{
		Domain: domain,
	}, nil, nil)

	tests := []struct {
		prefix   string
		redirect bool
	}{
		{
			prefix:   "",
			redirect: false,
		},
		{
			prefix:   "www.",
			redirect: true,
		},
		{
			prefix:   "www.sub.",
			redirect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.prefix, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.TLS = &tls.ConnectionState{
				// If prefix is not empty, expect to be redirected to
				// base domain without prefix.
				ServerName: tt.prefix + domain,
			}

			h.ServeHTTP(w, r)

			wantCode := http.StatusMovedPermanently
			if !tt.redirect {
				wantCode = http.StatusOK
			}

			if want, got := wantCode, w.Code; want != got {
				t.Fatalf("unexpected HTTP response code:\n- want: %d\n-  got: %d", want, got)
			}

			if !tt.redirect {
				return
			}

			if want, got := "https://"+domain, w.Header().Get("Location"); want != got {
				t.Fatalf("unexpected Location header:\n- want: %q\n-  got: %q", want, got)
			}
		})
	}
}

func Test_handlerServeHTTP(t *testing.T) {
	tests := []struct {
		name   string
		static StaticContent
		ghc    github.Client
		mc     medium.Client
		check  func(t *testing.T, res *http.Response)
	}{
		{
			name: "include HSTS header",
			check: func(t *testing.T, res *http.Response) {
				h := res.Header.Get("Strict-Transport-Security")
				if h == "" {
					t.Fatal("response did not include HSTS header")
				}

				if !strings.Contains(h, "max-age=") || !strings.Contains(h, "; includeSubdomains") {
					t.Fatalf("malformed HSTS header: %q", h)
				}
			},
		},
		{
			name: "head contains static content",
			static: StaticContent{
				Name:   "Matt Layher",
				Domain: "mdlayher.com",
			},
			check: bodyContains(t, []string{
				"<title>Matt Layher</title>",
				`<meta name="description" content="Matt Layher - mdlayher.com" />`,
			}),
		},
		{
			name: "body contains static content",
			static: StaticContent{
				Name:    "Matt Layher",
				Tagline: "mdlayher",
			},
			check: bodyContains(t, []string{
				"<h1>Matt Layher</h1>",
				"<p>mdlayher</p>",
			}),
		},
		{
			name: "body contains static links",
			static: StaticContent{
				Links: []Link{
					{
						Title: "foo",
						Link:  "https://bar.com",
					},
					{
						Title: "bar",
						Link:  "https://baz.com",
					},
				},
			},
			check: bodyContains(t, []string{
				`<li><a href="https://bar.com">foo</a></li>`,
				`<li><a href="https://baz.com">bar</a></li>`,
			}),
		},
		{
			name: "body contains GitHub content",
			ghc: &testGitHubClient{
				repos: []*github.Repository{
					{
						Name:        "foobar",
						Link:        "https://foo.com",
						Description: "foo bar",
					},
					{
						Name:        "barbaz",
						Link:        "https://bar.com",
						Description: "bar baz",
					},
				},
			},
			check: bodyContains(t, []string{
				`<li><a href="https://foo.com">foobar</a></li>`,
				`<ul><li>foo bar</li></ul>`,
				`<li><a href="https://bar.com">barbaz</a></li>`,
				`<ul><li>bar baz</li></ul>`,
			}),
		},
		{
			name: "body contains Medium content",
			mc: &testMediumClient{
				posts: []*medium.Post{
					{
						Title:    "Foo Bar",
						Subtitle: "foo bar baz",
						Link:     "https://foo.com",
					},
					{
						Title:    "Bar Baz",
						Subtitle: "bar baz qux",
						Link:     "https://bar.com",
					},
				},
			},
			check: bodyContains(t, []string{
				`<li><a href="https://foo.com">Foo Bar</a></li>`,
				`<ul><li>foo bar baz</li></ul>`,
				`<li><a href="https://bar.com">Bar Baz</a></li>`,
				`<ul><li>bar baz qux</li></ul>`,
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, testServer(t, tt.static, tt.ghc, tt.mc))
		})
	}
}

// bodyContains checks that an HTTP response body contains each
// string contained in strs.
func bodyContains(t *testing.T, strs []string) func(t *testing.T, res *http.Response) {
	return func(t *testing.T, res *http.Response) {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		defer res.Body.Close()

		bs := string(body)

		for _, s := range strs {
			// Numbers are a bit friendlier than HTML names.
			t.Run("", func(t *testing.T) {
				if !strings.Contains(bs, s) {
					t.Fatalf("expected body to contain %q, but it does not", s)
				}
			})
		}
	}
}

// testServer performs a single HTTP request against a handler
// populated with content and returns the HTTP response.
func testServer(t *testing.T, static StaticContent, ghc github.Client, mc medium.Client) *http.Response {
	s := httptest.NewServer(NewHandler(static, ghc, mc))
	defer s.Close()

	req, err := http.NewRequest(http.MethodGet, s.URL, nil)
	if err != nil {
		t.Fatalf("failed to create HTTP request: %v", err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to perform HTTP request: %v", err)
	}

	return res
}

var _ github.Client = &testGitHubClient{}

// testGitHubClient is a github.Client that returns static content.
type testGitHubClient struct {
	repos []*github.Repository
}

func (c *testGitHubClient) ListRepositories(_ context.Context) ([]*github.Repository, error) {
	return c.repos, nil
}

var _ medium.Client = &testMediumClient{}

// testMediumClient is a medium.Client that returns static content.
type testMediumClient struct {
	posts []*medium.Post
}

func (c *testMediumClient) ListPosts() ([]*medium.Post, error) {
	return c.posts, nil
}
