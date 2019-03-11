package medium

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

func Test_clientListPosts(t *testing.T) {
	// This test covers the actual Medium client interactions, verifying that
	// any parameters we pass or hard-code are set the way we want.
	const username = "mdlayher"

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if want, got := r.URL.Path, "/@"+username+"/latest"; want != got {
			panicf("unexpected URL path:\n- want: %q\n-  got: %q", want, got)
		}

		if want, got := r.Header.Get("Accept"), applicationJSON; want != got {
			panicf("unexpected Accept header:\n- want: %q\n-  got: %q", want, got)
		}

		if want, got := r.UserAgent(), userAgent; want != got {
			panicf("unexpected User-Agent header:\n- want: %q\n-  got: %q", want, got)
		}

		var md postMetadata
		md.Payload.References.Post = map[string]rawPost{
			"deadbeef": {
				CreatedAt:  10,
				Title:      "foo",
				UniqueSlug: "foo-deadbeef",
				Content: rawPostContent{
					Subtitle: "dead beef dead",
				},
			},
			"foobar": {
				CreatedAt:  20,
				Title:      "bar",
				UniqueSlug: "bar-foobar",
				Content: rawPostContent{
					Subtitle: "foo bar baz",
				},
			},
		}

		// Ensure anti-JSON hijacking measures are dealt with.
		w.Header().Set("Content-Type", applicationJSONUTF8)
		_, _ = w.Write(jsonHijackingPrefix)

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

	got, err := c.ListPosts(context.Background())
	if err != nil {
		t.Fatalf("failed to list posts: %v", err)
	}

	want := []*Post{
		{
			Title:    "bar",
			Subtitle: "foo bar baz",
			Link:     "https://medium.com/@" + username + "/bar-foobar",
			created:  20,
		},
		{
			Title:    "foo",
			Subtitle: "dead beef dead",
			Link:     "https://medium.com/@" + username + "/foo-deadbeef",
			created:  10,
		},
	}

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected posts:\n- want: %v\n-  got: %v", want, got)
	}
}

func panicf(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}
