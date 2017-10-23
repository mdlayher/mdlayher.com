package web

import (
	"context"
	"errors"
	"sync"

	"github.com/mdlayher/mdlayher.com/internal/github"
	"github.com/mdlayher/mdlayher.com/internal/httptalks"
	"github.com/mdlayher/mdlayher.com/internal/medium"
)

// errNilClient is a sentinel used to indicate an API client is not configured.
var errNilClient = errors.New("client not configured")

// future executes fn immediately in a goroutine and returns a function which
// can be invoked to receive the results of fn.
func future(name string, fn func() (interface{}, error)) func() (interface{}, error) {
	// TODO(mdlayher): instrument with prometheus using name.

	var (
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
		wg.Wait()
		return v, err
	}
}

// fetchGitHub fetches GitHub repositories in a future.
func (h *handler) fetchGitHub(ctx context.Context) func() ([]*github.Repository, error) {
	fn := future("github", func() (interface{}, error) {
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
	fn := future("medium", func() (interface{}, error) {
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
	fn := future("talks", func() (interface{}, error) {
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
