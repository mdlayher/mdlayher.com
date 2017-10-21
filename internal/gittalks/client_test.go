package gittalks

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"reflect"
	"testing"
	"time"
)

const talksRepo = "https://github.com/mdlayher/talks"

func Test_newClientListTalks(t *testing.T) {
	tc := &testCloner{
		buf: []byte(`
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
	}
]			
`),
	}

	// Cache should expire long after this test completes.
	c := newClient(tc, talksRepo, 1*time.Hour)

	var (
		got []*Talk
		err error
	)

	for i := 0; i < 5; i++ {
		got, err = c.ListTalks(context.Background())
		if err != nil {
			t.Fatalf("error listing talks: %v", err)
		}

		if tc.calls > 1 {
			t.Fatalf("too many calls to cloner: %v", tc.calls)
		}
	}

	want := []*Talk{
		{
			Title:       "hello",
			SlidesLink:  "https://foo.com/slides",
			Description: "first",
		},
		{
			Title:       "world",
			SlidesLink:  "https://bar.com/slides",
			VideoLink:   "https://bar.com/video",
			Description: "second",
		},
	}

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected talks:\n- want: %v\n-  got: %v", want, got)
	}
}

func TestClientListTalksIntegration(t *testing.T) {
	c := NewClient(talksRepo, 1*time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

var _ cloner = &gitCloner{}

type testCloner struct {
	buf   []byte
	calls int
}

func (tc *testCloner) Clone(_ context.Context, _ string) (io.ReadCloser, error) {
	tc.calls++
	return ioutil.NopCloser(bytes.NewReader(tc.buf)), nil
}
