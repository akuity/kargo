package cache

import (
	"context"
	"fmt"

	lru "github.com/hashicorp/golang-lru/v2"
)

// inMemoryCache is an in-memory implementation of the Cache interface.
type inMemoryCache[V any] struct {
	// cache is a simple, internal cache that utilizes a least recently used key
	// eviction policy.
	cache *lru.Cache[string, V]
}

// NewInMemoryCache returns an in-memory implementation of the Cache interface
// configured with the specified size (maximum number of entries).
func NewInMemoryCache[V any](size int) (Cache[V], error) {
	cache, err := lru.New[string, V](size)
	if err != nil {
		return nil, fmt.Errorf("error initializing cache: %w", err)
	}
	return &inMemoryCache[V]{cache: cache}, nil
}

// Get implements Cache.
func (c *inMemoryCache[V]) Get(_ context.Context, key string) (V, bool, error) {
	val, found := c.cache.Get(key)
	return val, found, nil
}

// Set implements Cache.
func (c *inMemoryCache[V]) Set(_ context.Context, key string, value V) error {
	c.cache.Add(key, value)
	return nil
}
