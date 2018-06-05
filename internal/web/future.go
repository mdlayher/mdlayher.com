package web

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/mdlayher/mdlayher.com/internal/github"
	"github.com/mdlayher/mdlayher.com/internal/httptalks"
	"github.com/mdlayher/mdlayher.com/internal/medium"
)

// errNilClient is a sentinel used to indicate an API client is not configured.
var errNilClient = errors.New("client not configured")

// future executes fn immediately in a goroutine and returns a function which
// can be invoked to receive the results of fn.
func (h *handler) future(name string, fn func() (interface{}, error)) func() (interface{}, error) {
	var (
		start = time.Now()

		v   interface{}
		err error
	)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		v, err = fn()
	}()

	return func() (interface{}, error) {
		defer func() {
			// If the future completed extremely quickly, the caching layer must
			// be in play.  Ignore this observation in metrics for now.
			//
			// TODO(mdlayher): decouple the caching layer and remove the need
			// for this total hack.
			dur := time.Now().Sub(start)
			if dur < 1*time.Millisecond {
				return
			}

			h.requestDurationSeconds.WithLabelValues(name).Observe(dur.Seconds())
		}()

		wg.Wait()
		return v, err
	}
}

// fetchGitHub fetches GitHub repositories in a future.
func (h *handler) fetchGitHub(ctx context.Context) func() ([]*github.Repository, error) {
	fn := h.future("github", func() (interface{}, error) {
		// No client configured.
		if h.ghc == nil {
			return nil, errNilClient
		}

		return h.ghc.ListRepositories(ctx)
	})

	return func() ([]*github.Repository, error) {
		v, err := fn()
		switch err {
		case nil:
		case errNilClient:
			return nil, nil
		default:
			return nil, err
		}

		return v.([]*github.Repository), nil
	}
}

// fetchMedium fetches Medium posts in a future.
func (h *handler) fetchMedium(ctx context.Context) func() ([]*medium.Post, error) {
	fn := h.future("medium", func() (interface{}, error) {
		// No client configured.
		if h.mc == nil {
			return nil, errNilClient
		}

		return h.mc.ListPosts()
	})

	return func() ([]*medium.Post, error) {
		v, err := fn()
		switch err {
		case nil:
		case errNilClient:
			return nil, nil
		default:
			return nil, err
		}

		return v.([]*medium.Post), nil
	}
}

// fetchTalks fetches HTTP talks in a future.
func (h *handler) fetchTalks(ctx context.Context) func() ([]*httptalks.Talk, error) {
	fn := h.future("talks", func() (interface{}, error) {
		// No client configured.
		if h.htc == nil {
			return nil, errNilClient
		}

		return h.htc.ListTalks(ctx)
	})

	return func() ([]*httptalks.Talk, error) {
		v, err := fn()
		switch err {
		case nil:
		case errNilClient:
			return nil, nil
		default:
			return nil, err
		}

		return v.([]*httptalks.Talk), nil
	}
}
