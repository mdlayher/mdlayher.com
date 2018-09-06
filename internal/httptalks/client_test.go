package httptalks

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

const talksPath = "https://raw.githubusercontent.com/mdlayher/talks/master/talks.json"

func Test_newClientListTalks(t *testing.T) {
	// Expect only once call to HTTP server because of cache.
	var calls int
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++

		// Verify user agent while we're at it.
		if want, got := r.UserAgent(), "github.com/mdlayher/mdlayher.com/internal/httptalks"; want != got {
			t.Fatalf("unexpected user agent:\n- want: %q\n-  got: %q", want, got)
		}

		if calls > 1 {
			t.Fatalf("too many calls to server: %d", calls)
		}

		_, _ = io.WriteString(w, `
			[
				{
					"Title":       "hello",
					"SlidesLink":  "https://foo.com/slides",
					"Description": "first"
				},
				{
					"Title":       "world",
					"SlidesLink":  "https://bar.com/slides",
					"VideoLink":   "https://bar.com/video",
					"Description": "second"
				},
				{
					"Title":       "goodbye",
					"SlidesLink":  "https://baz.com/slides",
					"BlogLink":    "https://baz.com/blog",
					"Description": "third",
					"AudioLink":   "https://baz.com/audio"
				}
			]
			`)
	}))
	defer s.Close()

	// Cache should expire long after this test completes.
	c := NewClient(s.URL, 1*time.Hour)

	var (
		got []*Talk
		err error
	)

	for i := 0; i < 5; i++ {
		got, err = c.ListTalks(context.Background())
		if err != nil {
			t.Fatalf("error listing talks: %v", err)
		}
	}

	want := []*Talk{
		{
			Title:       "hello",
			Description: "first",
			SlidesLink:  "https://foo.com/slides",
		},
		{
			Title:       "world",
			Description: "second",
			SlidesLink:  "https://bar.com/slides",
			VideoLink:   "https://bar.com/video",
		},
		{
			Title:       "goodbye",
			Description: "third",
			AudioLink:   "https://baz.com/audio",
			BlogLink:    "https://baz.com/blog",
			SlidesLink:  "https://baz.com/slides",
		},
	}

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected talks:\n- want: %v\n-  got: %v", want, got)
	}
}

func TestClientListTalksIntegration(t *testing.T) {
	c := NewClient(talksPath, 1*time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	talks, err := c.ListTalks(ctx)
	if err != nil {
		t.Fatalf("failed to list talks: %v", err)
	}

	for _, talk := range talks {
		t.Log(talk.Title)
	}

	// Should be at least 5 talks.
	if l := len(talks); l < 5 {
		t.Fatalf("expected 5+ talks, but found: %d", l)
	}
}
