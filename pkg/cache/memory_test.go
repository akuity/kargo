package cache

import (
	"context"
	"testing"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/stretchr/testify/require"
)

func TestNewInMemoryCache(t *testing.T) {
	testCases := []struct {
		name      string
		size      int
		expectErr bool
	}{
		{
			name:      "valid size",
			size:      10,
			expectErr: false,
		},
		{
			name:      "zero size",
			size:      0,
			expectErr: true,
		},
		{
			name:      "negative size",
			size:      -1,
			expectErr: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			cache, err := NewInMemoryCache[string](testCase.size)
			if testCase.expectErr {
				require.Error(t, err)
				require.Nil(t, cache)
			} else {
				require.NoError(t, err)
				require.NotNil(t, cache)
				inMemCache, ok := cache.(*inMemoryCache[string])
				require.True(t, ok)
				require.NotNil(t, inMemCache.cache)
			}
		})
	}
}

func TestInMemoryCache_Get(t *testing.T) {
	const testKey = "key"

	type testStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	alice := testStruct{Name: "Alice", Age: 30}

	testCases := []struct {
		name       string
		setup      func(*lru.Cache[string, testStruct])
		assertions func(*testing.T, testStruct, bool, error)
	}{
		{
			name: "key not found",
			assertions: func(t *testing.T, _ testStruct, found bool, err error) {
				require.NoError(t, err)
				require.False(t, found)
			},
		},
		{
			name: "key found",
			setup: func(c *lru.Cache[string, testStruct]) {
				c.Add(testKey, alice)
			},
			assertions: func(t *testing.T, value testStruct, found bool, err error) {
				require.NoError(t, err)
				require.True(t, found)
				require.Equal(t, alice, value)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			internalCache, err := lru.New[string, testStruct](1)
			require.NoError(t, err)
			if testCase.setup != nil {
				testCase.setup(internalCache)
			}
			cache := &inMemoryCache[testStruct]{cache: internalCache}
			value, found, err := cache.Get(context.Background(), testKey)
			require.NoError(t, err)
			testCase.assertions(t, value, found, err)
		})
	}
}

func TestInMemoryCache_Set(t *testing.T) {
	const testKey = "key"

	type testStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	alice := testStruct{Name: "Alice", Age: 30}

	bob := testStruct{Name: "Bob", Age: 25}

	testCases := []struct {
		name       string
		setup      func(*lru.Cache[string, testStruct])
		value      testStruct
		assertions func(*testing.T, *lru.Cache[string, testStruct], error)
	}{
		{
			name:  "initial write",
			value: alice,
			assertions: func(
				t *testing.T,
				c *lru.Cache[string, testStruct],
				err error,
			) {
				require.NoError(t, err)
				value, found := c.Get(testKey)
				require.True(t, found)
				require.Equal(t, alice, value)
			},
		},
		{
			name: "overwrite",
			setup: func(c *lru.Cache[string, testStruct]) {
				c.Add(testKey, alice)
			},
			value: bob,
			assertions: func(
				t *testing.T,
				c *lru.Cache[string, testStruct],
				err error,
			) {
				require.NoError(t, err)
				value, found := c.Get(testKey)
				require.True(t, found)
				require.Equal(t, bob, value)
			},
		},
		{
			name: "lru eviction when cache is full",
			setup: func(c *lru.Cache[string, testStruct]) {
				c.Add("alice-key", alice)
			},
			value: bob,
			assertions: func(
				t *testing.T,
				c *lru.Cache[string, testStruct],
				err error,
			) {
				require.NoError(t, err)
				value, found := c.Get(testKey)
				require.True(t, found)
				require.Equal(t, bob, value)
				// The cache size is 1, so Alice should have been evicted.
				_, found = c.Get("alice-key")
				require.False(t, found)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			internalCache, err := lru.New[string, testStruct](1)
			require.NoError(t, err)
			if testCase.setup != nil {
				testCase.setup(internalCache)
			}
			cache := &inMemoryCache[testStruct]{cache: internalCache}
			err = cache.Set(t.Context(), testKey, testCase.value)
			testCase.assertions(t, internalCache, err)
		})
	}
}
