package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/google/go-github/v42/github"
)

func Test_clientListRepositories(t *testing.T) {
	// This test covers the actual GitHub client interactions, verifying that
	// any parameters we pass or hard-code are set the way we want.
	const username = "mdlayher"

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify user agent while we're at it.
		if want, got := "github.com/mdlayher/mdlayher.com/internal/github", r.UserAgent(); want != got {
			panicf("unexpected user agent:\n- want: %q\n-  got: %q", want, got)
		}

		if want, got := r.URL.Path, "/users/"+username+"/repos"; want != got {
			panicf("unexpected URL path:\n- want: %q\n-  got: %q", want, got)
		}

		q := r.URL.Query()

		if want, got := "15", q.Get("per_page"); want != got {
			panicf("unexpected per_page parameter:\n- want: %q\n-  got: %q", want, got)
		}
		if want, got := "pushed", q.Get("sort"); want != got {
			panicf("unexpected sort parameter:\n- want: %q\n-  got: %q", want, got)
		}

		v := []struct {
			Name        string `json:"name"`
			HTMLURL     string `json:"html_url"`
			Description string `json:"description"`
			Archived    bool   `json:"archived"`
			Fork        bool   `json:"fork"`
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
			{
				Name:    "archived",
				HTMLURL: "https://github.com/mdlayher/archived",
				// Should not be displayed because repo is archived.
				Archived: true,
			},
			{
				Name:    "fork",
				HTMLURL: "https://github.com/mdlayher/fork",
				// Should not be displayed because repo is a fork.
				Fork: true,
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

	c := newClient(ghc, username)

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

func panicf(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}
