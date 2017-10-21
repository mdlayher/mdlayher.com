package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-github/github"
)

func Test_newClientListRepositories(t *testing.T) {
	// Expect only once call to HTTP server because of cache.
	var calls int
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++

		// Verify user agent while we're at it.
		if want, got := r.UserAgent(), "github.com/mdlayher/mdlayher.com/internal/github"; want != got {
			t.Fatalf("unexpected user agent:\n- want: %q\n-  got: %q", want, got)
		}

		if calls > 1 {
			t.Fatalf("too many calls to GitHub API: %d", calls)
		}
	}))
	defer s.Close()

	u, err := url.Parse(s.URL)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}

	// New go-github insists that base URL should have a trailing slash.
	u.Path = "/"

	ghc := github.NewClient(nil)
	ghc.BaseURL = u

	// Cache should expire long after this test completes.
	c := newClient(ghc, "mdlayher", 1*time.Hour)

	for i := 0; i < 5; i++ {
		if _, err := c.ListRepositories(context.Background()); err != nil {
			t.Fatalf("error listing repositories: %v", err)
		}
	}
}

func Test_clientListRepositories(t *testing.T) {
	// This test covers the actual GitHub client interactions, verifying that
	// any parameters we pass or hard-code are set the way we want.
	const username = "mdlayher"

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if want, got := r.URL.Path, "/users/"+username+"/repos"; want != got {
			t.Fatalf("unexpected URL path:\n- want: %q\n-  got: %q", want, got)
		}

		q := r.URL.Query()

		if want, got := "10", q.Get("per_page"); want != got {
			t.Fatalf("unexpected per_page parameter:\n- want: %q\n-  got: %q", want, got)
		}
		if want, got := "pushed", q.Get("sort"); want != got {
			t.Fatalf("unexpected sort parameter:\n- want: %q\n-  got: %q", want, got)
		}

		v := []struct {
			Name        string `json:"name"`
			HTMLURL     string `json:"html_url"`
			Description string `json:"description"`
		}{
			{
				Name:        "hello",
				HTMLURL:     "https://github.com/mdlayher/hello",
				Description: "first",
			},
			{
				Name:    "world",
				HTMLURL: "https://github.com/mdlayher/world",
				// No Description to confirm the client will not panic.
			},
		}

		_ = json.NewEncoder(w).Encode(v)
	}))
	defer s.Close()

	u, err := url.Parse(s.URL)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}

	// New go-github insists that base URL should have a trailing slash.
	u.Path = "/"

	ghc := github.NewClient(nil)
	ghc.BaseURL = u

	c := &client{
		c:        ghc,
		username: username,
	}

	got, err := c.ListRepositories(context.Background())
	if err != nil {
		t.Fatalf("failed to list repos: %v", err)
	}

	want := []*Repository{
		{
			Name:        "hello",
			Link:        "https://github.com/mdlayher/hello",
			Description: "first",
		},
		{
			Name: "world",
			Link: "https://github.com/mdlayher/world",
		},
	}

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected repos:\n- want: %v\n-  got: %v", want, got)
	}
}
