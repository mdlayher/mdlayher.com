package memocache

import (
	"reflect"
	"testing"
	"time"
)

const (
	foo = "foo"
	bar = "bar"
)

// cacheDate is an arbitrary date in the past for testing the
// expiringCache.
var cacheDate = time.Date(2017, time.July, 6, 0, 0, 0, 0, time.UTC)

func Test_expiringCacheGet(t *testing.T) {
	tests := []struct {
		name   string
		init   Object
		clear  bool
		expire time.Duration
		last   time.Time
		get    func() (Object, error)
		want   Object
	}{
		{
			name:   "empty cache",
			expire: 1 * time.Second,
			get: func() (Object, error) {
				return foo, nil
			},
			want: foo,
		},
		{
			name: "cache has foo, but expired",
			init: foo,
			// Cache expired one second after time in past.
			expire: 1 * time.Second,
			last:   cacheDate,
			get: func() (Object, error) {
				return bar, nil
			},
			want: bar,
		},
		{
			name: "cache has foo, but not expired",
			init: foo,
			// Cache expires 24 hours after now.
			expire: 24 * time.Hour,
			last:   time.Now(),
			get: func() (Object, error) {
				return bar, nil
			},
			want: foo,
		},
		{
			name:  "cache has foo, is cleared, but not expired",
			init:  foo,
			clear: true,
			// Cache expires 24 hours after now.
			expire: 24 * time.Hour,
			last:   time.Now(),
			get: func() (Object, error) {
				return bar, nil
			},
			want: bar,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &expiringCache{
				expire: tt.expire,
				last:   tt.last,
				c:      &cache{},
			}

			c.c.o = tt.init

			if tt.clear {
				c.Clear()
			}

			obj, err := c.Get(tt.get)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.want, obj; !reflect.DeepEqual(want, got) {
				t.Fatalf("unexpected object:\n- want: %v\n-  got: %v", want, got)
			}
		})
	}

}
func Test_cacheClearGet(t *testing.T) {
	tests := []struct {
		name  string
		init  Object
		clear bool
		get   func() (Object, error)
		want  Object
	}{
		{
			name: "empty cache",
			get: func() (Object, error) {
				return foo, nil
			},
			want: foo,
		},
		{
			name: "cache has foo",
			init: foo,
			get: func() (Object, error) {
				return bar, nil
			},
			want: foo,
		},
		{
			name:  "cache has foo but is cleared",
			init:  foo,
			clear: true,
			get: func() (Object, error) {
				return bar, nil
			},
			want: bar,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &cache{}
			c.o = tt.init

			if tt.clear {
				c.Clear()
			}

			obj, err := c.Get(tt.get)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.want, obj; !reflect.DeepEqual(want, got) {
				t.Fatalf("unexpected object:\n- want: %v\n-  got: %v", want, got)
			}
		})
	}
}
