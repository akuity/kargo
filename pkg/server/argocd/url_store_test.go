package argocd

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewURLStore(t *testing.T) {
	store := NewURLStore()
	require.NotNil(t, store)

	shards := store.GetShards()
	assert.Empty(t, shards)
}

func TestURLStore_SetStaticShards(t *testing.T) {
	store := NewURLStore()

	staticShards := map[string]string{
		"":           "https://argocd.example.com",
		"production": "https://argocd-prod.example.com",
	}
	store.SetStaticShards(staticShards, "argocd")

	shards := store.GetShards()
	require.Len(t, shards, 2)

	assert.Equal(t, "https://argocd.example.com", shards[""].Url)
	assert.Equal(t, "argocd", shards[""].Namespace)

	assert.Equal(t, "https://argocd-prod.example.com", shards["production"].Url)
	assert.Equal(t, "argocd", shards["production"].Namespace)
}

func TestURLStore_UpdateDynamicShard(t *testing.T) {
	store := NewURLStore()
	store.SetStaticShards(nil, "argocd")

	store.UpdateDynamicShard("staging", "https://argocd-staging.example.com")

	shards := store.GetShards()
	require.Len(t, shards, 1)
	assert.Equal(t, "https://argocd-staging.example.com", shards["staging"].Url)
	assert.Equal(t, "argocd", shards["staging"].Namespace)
}

func TestURLStore_DynamicOverridesStatic(t *testing.T) {
	store := NewURLStore()

	// Set static shards
	staticShards := map[string]string{
		"production": "https://argocd-old.example.com",
	}
	store.SetStaticShards(staticShards, "argocd")

	// Verify static value
	shards := store.GetShards()
	assert.Equal(t, "https://argocd-old.example.com", shards["production"].Url)

	// Add dynamic shard with same name - should override
	store.UpdateDynamicShard("production", "https://argocd-new.example.com")

	shards = store.GetShards()
	require.Len(t, shards, 1)
	assert.Equal(t, "https://argocd-new.example.com", shards["production"].Url)
}

func TestURLStore_DeleteDynamicShard_RestoresStatic(t *testing.T) {
	store := NewURLStore()

	// Set static shards
	staticShards := map[string]string{
		"production": "https://argocd-static.example.com",
	}
	store.SetStaticShards(staticShards, "argocd")

	// Add dynamic override
	store.UpdateDynamicShard("production", "https://argocd-dynamic.example.com")

	// Verify dynamic override is active
	shards := store.GetShards()
	assert.Equal(t, "https://argocd-dynamic.example.com", shards["production"].Url)

	// Delete dynamic shard
	store.DeleteDynamicShard("production")

	// Verify static shard is restored
	shards = store.GetShards()
	require.Len(t, shards, 1)
	assert.Equal(t, "https://argocd-static.example.com", shards["production"].Url)
}

func TestURLStore_DeleteDynamicShard_NoStatic(t *testing.T) {
	store := NewURLStore()
	store.SetStaticShards(nil, "argocd")

	// Add dynamic shard
	store.UpdateDynamicShard("staging", "https://argocd-staging.example.com")

	// Verify it exists
	shards := store.GetShards()
	require.Len(t, shards, 1)

	// Delete it
	store.DeleteDynamicShard("staging")

	// Verify it's gone
	shards = store.GetShards()
	assert.Empty(t, shards)
}

func TestURLStore_MergeStaticAndDynamic(t *testing.T) {
	store := NewURLStore()

	// Set static shards
	staticShards := map[string]string{
		"":           "https://argocd-default.example.com",
		"production": "https://argocd-prod.example.com",
	}
	store.SetStaticShards(staticShards, "argocd")

	// Add dynamic shards (one override, one new)
	store.UpdateDynamicShard("production", "https://argocd-prod-new.example.com")
	store.UpdateDynamicShard("staging", "https://argocd-staging.example.com")

	shards := store.GetShards()
	require.Len(t, shards, 3)

	// Default shard from static
	assert.Equal(t, "https://argocd-default.example.com", shards[""].Url)
	// Production overridden by dynamic
	assert.Equal(t, "https://argocd-prod-new.example.com", shards["production"].Url)
	// Staging from dynamic only
	assert.Equal(t, "https://argocd-staging.example.com", shards["staging"].Url)
}

func TestURLStore_ConcurrentAccess(_ *testing.T) {
	store := NewURLStore()
	store.SetStaticShards(map[string]string{
		"default": "https://argocd.example.com",
	}, "argocd")

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				shards := store.GetShards()
				// Just access to trigger potential race conditions
				_ = len(shards)
			}
		}()
	}

	// Concurrent writers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				store.UpdateDynamicShard("dynamic", "https://test.example.com")
				store.DeleteDynamicShard("dynamic")
			}
		}()
	}

	wg.Wait()
	// If we get here without race detector complaints, the test passes
}

func TestURLStore_EmptyShardName(t *testing.T) {
	store := NewURLStore()
	store.SetStaticShards(nil, "argocd")

	// Empty string is a valid shard name (for single/default ArgoCD)
	store.UpdateDynamicShard("", "https://argocd.example.com")

	shards := store.GetShards()
	require.Len(t, shards, 1)
	assert.Equal(t, "https://argocd.example.com", shards[""].Url)
}
