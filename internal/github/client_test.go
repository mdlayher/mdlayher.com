package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/google/go-github/github"
)

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
				Name:        "world",
				HTMLURL:     "https://github.com/mdlayher/world",
				Description: "second",
			},
		}

		_ = json.NewEncoder(w).Encode(v)
	}))
	defer s.Close()

	u, err := url.Parse(s.URL)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}

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
			Name:        "world",
			Link:        "https://github.com/mdlayher/world",
			Description: "second",
		},
	}

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected repos:\n- want: %v\n-  got: %v", want, got)
	}
}
