package web

import (
	"testing"
	"time"
)

func TestHSTSHeader(t *testing.T) {
	tests := []struct {
		elapsed time.Duration
		header  string
	}{
		{
			elapsed: 1 * time.Minute,
			header:  "max-age=300; includeSubdomains",
		},
		{
			elapsed: 5*time.Minute + 1*time.Second,
			header:  "max-age=604800; includeSubdomains",
		},
		{
			elapsed: 7*24*time.Hour + 1*time.Second,
			header:  "max-age=2592000; includeSubdomains",
		},
		{
			elapsed: 30*24*time.Hour + 1*time.Second,
			header:  "max-age=63072000; includeSubdomains; preload",
		},
		{
			elapsed: twoYears + 1*time.Second,
			header:  "max-age=63072000; includeSubdomains; preload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.elapsed.String(), func(t *testing.T) {
			// Simulate elapsed time since hstsStart.
			now := hstsStart.Add(tt.elapsed)

			if want, got := tt.header, HSTSHeader(now); want != got {
				t.Fatalf("unexpected HSTS header value:\n- want: %q\n-  got: %q", want, got)
			}
		})
	}
}
