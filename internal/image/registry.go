package image

import (
	"sync"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/patrickmn/go-cache"
	"go.uber.org/ratelimit"
)

const (
	defaultCacheEntryTTL        = 30 * time.Minute
	defaultCacheCleanupInterval = time.Hour
	defaultTagPageSize          = 1000
)

// dockerRegistry is registry configuration for Docker Hub.
var dockerRegistry = &registry{
	name:        "Docker Hub",
	imagePrefix: name.DefaultRegistry,
	// The default namespace of "library" is the main reason we have
	// registry-specified configuration for Docker Hub.
	defaultNamespace: "library",
	imageCache: cache.New(
		defaultCacheEntryTTL,        // Default ttl for each entry
		defaultCacheCleanupInterval, // Cleanup interval
	),
	rateLimiter: ratelimit.New(10),
	tagPageSize: defaultTagPageSize,
}

// quayRegistry is registry configuration for Quay.io.
var quayRegistry = &registry{
	name:             "Quay.io",
	imagePrefix:      "quay.io",
	defaultNamespace: "",
	imageCache: cache.New(
		defaultCacheEntryTTL,        // Default ttl for each entry
		defaultCacheCleanupInterval, // Cleanup interval
	),
	rateLimiter: ratelimit.New(20),
	// Quay does not like when you ask for more than 100 tags at a time
	tagPageSize: 100,
}

var (
	// registries is a map of Registries indexed by image prefix and is pre-loaded
	// with known registries whose settings cannot be inferred from an image's
	// prefix.
	registries = map[string]*registry{
		"":                         dockerRegistry,
		dockerRegistry.imagePrefix: dockerRegistry,
		quayRegistry.imagePrefix:   quayRegistry,
	}
	// registriesMu is for preventing concurrent access to the registries map.
	registriesMu sync.Mutex
)

// registry holds information on how to access any specific image container
// registry.
type registry struct {
	name             string
	imagePrefix      string
	defaultNamespace string
	imageCache       *cache.Cache
	rateLimiter      ratelimit.Limiter
	tagPageSize      int
}

// newRegistry initializes and returns a new registry.
func newRegistry(imagePrefix string) *registry {
	return &registry{
		name:        imagePrefix,
		imagePrefix: imagePrefix,
		imageCache: cache.New(
			defaultCacheEntryTTL,        // Default ttl for each entry
			defaultCacheCleanupInterval, // Cleanup interval
		),
		// TODO: Make this configurable.
		rateLimiter: ratelimit.New(20),
		tagPageSize: defaultTagPageSize,
	}
}

// getRegistry retrieves the registry associated with the given image prefix. If
// no such registry is found, a new one is initialized and added to the
// registries map.
func getRegistry(imagePrefix string) *registry {
	registriesMu.Lock()
	defer registriesMu.Unlock()
	if registry, ok := registries[imagePrefix]; ok {
		return registry
	}
	registry := newRegistry(imagePrefix)
	registries[registry.imagePrefix] = registry
	return registry
}
