package argocd

import (
	"sync"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

// URLStore provides thread-safe access to ArgoCD shard URLs.
// It merges static configuration (from Helm values) with dynamically
// discovered secrets, where dynamic shards override static ones.
type URLStore interface {
	// GetShards returns all ArgoCD shards merged from static and dynamic sources.
	// Dynamic (secret-based) shards override static (Helm) shards with the same name.
	GetShards() map[string]*svcv1alpha1.ArgoCDShard

	// SetStaticShards sets the base configuration from Helm values.
	// This should be called once during initialization.
	SetStaticShards(shards map[string]string, defaultNamespace string)

	// UpdateDynamicShard adds or updates a shard from a discovered secret.
	UpdateDynamicShard(name string, url string)

	// DeleteDynamicShard removes a shard that was loaded from a secret.
	// If a static shard with the same name exists, it becomes active again.
	DeleteDynamicShard(name string)
}

type urlStore struct {
	mu               sync.RWMutex
	staticShards     map[string]string // shard name -> URL
	dynamicShards    map[string]string // shard name -> URL
	defaultNamespace string
}

// NewURLStore creates a new URLStore for managing ArgoCD shard URLs.
func NewURLStore() URLStore {
	return &urlStore{
		staticShards:  make(map[string]string),
		dynamicShards: make(map[string]string),
	}
}

func (s *urlStore) GetShards() map[string]*svcv1alpha1.ArgoCDShard {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*svcv1alpha1.ArgoCDShard, len(s.staticShards)+len(s.dynamicShards))
	for name, url := range s.staticShards {
		result[name] = &svcv1alpha1.ArgoCDShard{
			Url:       url,
			Namespace: s.defaultNamespace,
		}
	}
	for name, url := range s.dynamicShards {
		result[name] = &svcv1alpha1.ArgoCDShard{
			Url:       url,
			Namespace: s.defaultNamespace,
		}
	}

	return result
}

func (s *urlStore) SetStaticShards(shards map[string]string, defaultNamespace string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.defaultNamespace = defaultNamespace
	s.staticShards = make(map[string]string, len(shards))
	for name, url := range shards {
		s.staticShards[name] = url
	}
}

func (s *urlStore) UpdateDynamicShard(name string, url string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.dynamicShards[name] = url
}

func (s *urlStore) DeleteDynamicShard(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.dynamicShards, name)
}
