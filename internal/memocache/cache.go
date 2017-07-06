package memocache

import (
	"sync"
	"time"
)

// A Cache is a concurrency-safe, memoizing cache for a single value.
type Cache interface {
	// Clear removes an Object from a Cache.
	Clear()

	// Get returns a single Object from a Cache.
	// If the Object is not already cached, fn is invoked
	// to retrieve and cache the Object.
	Get(fn func() (Object, error)) (Object, error)
}

// An Object is an arbitrary type which can be cached.
type Object interface{}

// New creates a memoizing cache with values that expire after a
// period of time has elapsed.
func New(expire time.Duration) Cache {
	return &expiringCache{
		expire: expire,
		c:      &cache{},
	}
}

// An expiringCache is a memoizing cache for a single value that will
// expire after a certain period of time has elapsed.
type expiringCache struct {
	mu     sync.RWMutex
	expire time.Duration
	last   time.Time

	c *cache
}

// Clear implements Cache.
func (c *expiringCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.last = time.Time{}
	c.c.Clear()
}

// Get implements Cache.
func (c *expiringCache) Get(fn func() (Object, error)) (Object, error) {
	c.mu.RLock()
	if time.Since(c.last) < c.expire {
		defer c.mu.RUnlock()

		return c.c.Get(fn)
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	c.last = time.Now()
	c.c.Clear()

	return c.c.Get(fn)
}

var _ Cache = &cache{}

// A cache is a memoizing cache for a single value.
// Its values never expire.
type cache struct {
	mu sync.RWMutex
	o  Object
}

// Clear implements Cache.
func (c *cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.o = nil
}

// Get implements Cache.
func (c *cache) Get(fn func() (Object, error)) (Object, error) {
	c.mu.RLock()
	if c.o != nil {
		defer c.mu.RUnlock()

		return c.o, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	o, err := fn()
	if err != nil {
		return nil, err
	}

	c.o = o

	return o, nil
}
