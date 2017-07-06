package medium

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"
)

func Test_newClientListRepositories(t *testing.T) {
	// Expect only once call to HTTP server because of cache.
	var calls int
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++

		if calls > 1 {
			t.Fatalf("too many calls to GitHub API: %d", calls)
		}

		// Ensure anti-JSON hijacking measures are dealt with.
		w.Header().Set("Content-Type", applicationJSONUTF8)
		_, _ = w.Write(append(jsonHijackingPrefix, "{}"...))
	}))
	defer s.Close()

	u, err := url.Parse(s.URL)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}

	// Cache should expire long after this test completes.
	c := newClient(u, "mdlayher", 1*time.Hour)

	for i := 0; i < 5; i++ {
		if _, err := c.ListPosts(); err != nil {
			t.Fatalf("error listing posts: %v", err)
		}
	}
}

func Test_clientListPosts(t *testing.T) {
	// This test covers the actual Medium client interactions, verifying that
	// any parameters we pass or hard-code are set the way we want.
	const username = "mdlayher"

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if want, got := r.URL.Path, "/@"+username+"/latest"; want != got {
			t.Fatalf("unexpected URL path:\n- want: %q\n-  got: %q", want, got)
		}

		if want, got := r.Header.Get("Accept"), applicationJSON; want != got {
			t.Fatalf("unexpected Accept header:\n- want: %q\n-  got: %q", want, got)
		}

		if want, got := r.UserAgent(), userAgent; want != got {
			t.Fatalf("unexpected User-Agent header:\n- want: %q\n-  got: %q", want, got)
		}

		w.Header().Set("Content-Type", applicationJSONUTF8)

		var md postMetadata
		md.Payload.References.Post = map[string]rawPost{
			"deadbeef": {
				CreatedAt:  10,
				Title:      "foo",
				UniqueSlug: "foo-deadbeef",
			},
			"foobar": {
				CreatedAt:  20,
				Title:      "bar",
				UniqueSlug: "bar-foobar",
			},
		}

		_ = json.NewEncoder(w).Encode(md)
	}))
	defer s.Close()

	u, err := url.Parse(s.URL)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}

	c := &client{
		client:   &http.Client{},
		apiURL:   u,
		username: username,
	}

	got, err := c.ListPosts()
	if err != nil {
		t.Fatalf("failed to list posts: %v", err)
	}

	want := []*Post{
		{
			Title:   "bar",
			Link:    "https://medium.com/@" + username + "/bar-foobar",
			created: 20,
		},
		{
			Title:   "foo",
			Link:    "https://medium.com/@" + username + "/foo-deadbeef",
			created: 10,
		},
	}

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected posts:\n- want: %v\n-  got: %v", want, got)
	}
}
