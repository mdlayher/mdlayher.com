package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v42/github"
)

// This test covers the actual GitHub client interactions, verifying that
// any parameters we pass or hard-code are set the way we want.
const username = "mdlayher"

func Test_clientListRepositories(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify user agent while we're at it.
		if want, got := "github.com/mdlayher/mdlayher.com/internal/github", r.UserAgent(); want != got {
			panicf("unexpected user agent:\n- want: %q\n-  got: %q", want, got)
		}

		if r.URL.Path == "/users/"+username+"/repos" {
			handleRepos(w, r)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/repos/"+username) {
			handleReleases(w, r)
			return
		}

		http.NotFound(w, r)
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
			Tag:         "v1.0.0",
		},
		{
			Name: "world",
			Link: "https://github.com/mdlayher/world",
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("unexpected repos (-want +got):\n%s", diff)
	}
}

func handleRepos(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	if diff := cmp.Diff("30", q.Get("per_page")); diff != "" {
		panicf("unexpected per_page parameter (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("pushed", q.Get("sort")); diff != "" {
		panicf("unexpected sort parameter (-want +got):\n%s", diff)
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
}

func handleReleases(w http.ResponseWriter, r *http.Request) {
	// Pop end elements off the URL to get the repo name and verify path.
	repo := path.Base(path.Dir(r.URL.Path))
	log.Println(repo)

	if diff := cmp.Diff(r.URL.Path, fmt.Sprintf("/repos/%s/%s/releases", username, repo)); diff != "" {
		panicf("unexpected releases path (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff("1", r.URL.Query().Get("per_page")); diff != "" {
		panicf("unexpected per_page parameter (-want +got):\n%s", diff)
	}

	// Only have a tag for repo "hello".
	var tag string
	if repo == "hello" {
		tag = "v1.0.0"
	}

	v := []struct {
		TagName string `json:"tag_name"`
	}{{TagName: tag}}

	_ = json.NewEncoder(w).Encode(v)
}

func panicf(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}
