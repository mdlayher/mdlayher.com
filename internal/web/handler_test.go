package web

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func Test_handlerServeHTTP(t *testing.T) {
	tests := []struct {
		name  string
		c     Content
		check func(t *testing.T, res *http.Response)
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
			name: "head contains content",
			c: Content{
				Name:   "Matt Layher",
				Domain: "mdlayher.com",
			},
			check: bodyContains(t, []string{
				"<title>Matt Layher</title>",
				`<meta name="description" content="Matt Layher - mdlayher.com" />`,
			}),
		},
		{
			name: "body contains content",
			c: Content{
				Name:    "Matt Layher",
				Tagline: "mdlayher",
			},
			check: bodyContains(t, []string{
				"<h1>Matt Layher</h1>",
				"<p>mdlayher</p>",
			}),
		},
		{
			name: "body contains links",
			c: Content{
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, testServer(t, tt.c))
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
// populated with Content and returns the HTTP response.
func testServer(t *testing.T, c Content) *http.Response {
	s := httptest.NewServer(NewHandler(c))
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
