package git

import (
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRepoCache(t *testing.T) {
	t.Run("defaults to temp dir", func(t *testing.T) {
		cache := NewRepoCache(nil)
		require.NotNil(t, cache)
		assert.Equal(t, os.TempDir(), cache.baseDir)
		assert.NotNil(t, cache.entries)
	})

	t.Run("uses provided base dir", func(t *testing.T) {
		dir := t.TempDir()
		cache := NewRepoCache(&RepoCacheOptions{BaseDir: dir})
		require.NotNil(t, cache)
		assert.Equal(t, dir, cache.baseDir)
	})
}

func TestRepoCacheGetOrCreateEntry(t *testing.T) {
	cache := NewRepoCache(nil)

	entry1 := cache.getOrCreateEntry("https://example.com/repo1.git")
	require.NotNil(t, entry1)

	// Same URL should return same entry
	entry2 := cache.getOrCreateEntry("https://example.com/repo1.git")
	assert.Same(t, entry1, entry2)

	// Different URL should return different entry
	entry3 := cache.getOrCreateEntry("https://example.com/repo2.git")
	assert.NotSame(t, entry1, entry3)
}

func TestRepoCacheGetOrCreateEntryConcurrent(t *testing.T) {
	cache := NewRepoCache(nil)
	const url = "https://example.com/repo.git"

	var wg sync.WaitGroup
	entries := make([]*repoCacheEntry, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			entries[idx] = cache.getOrCreateEntry(url)
		}(i)
	}
	wg.Wait()

	// All goroutines should get the same entry
	for i := 1; i < len(entries); i++ {
		assert.Same(t, entries[0], entries[i])
	}
}

func TestRepoCacheCloseEmpty(t *testing.T) {
	cache := NewRepoCache(nil)
	err := cache.Close()
	assert.NoError(t, err)
	assert.Empty(t, cache.entries)
}
