package image

import (
	"strings"
	"sync"
	"time"

	"github.com/akuity/kargo/internal/os"
	"github.com/akuity/kargo/internal/types"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/patrickmn/go-cache"
	"go.uber.org/ratelimit"
)

// dockerRegistry is registry configuration for Docker Hub.
var dockerRegistry = &registry{
	name:             "Docker Hub",
	imagePrefix:      name.DefaultRegistry,
	defaultNamespace: "library",
	imageCache: cache.New(
		30*time.Minute, // Default ttl for each entry
		time.Hour,      // Cleanup interval
	),
	rateLimiter: ratelimit.New(10),
}

var (
	// registries is a map of Registries indexed by image prefix and is pre-loaded
	// with known registries whose settings cannot be inferred from an image's
	// prefix.
	registries = map[string]*registry{
		"":                         dockerRegistry,
		dockerRegistry.imagePrefix: dockerRegistry,
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
}

// check if imagePrefix matches any custom registry hostnames, if so -- apply the custom rate limit.
func checkForCustomRateLimit(imagePrefix string) int {
	defaultRateLimitValue := 20
	hostnames := os.GetEnv("CUSTOM_IMAGE_REGISTRY_HOSTNAMES", "")
	if hostnames != "" {
		hostnameList := strings.SplitSeq(hostnames, ",")
		for hostname := range hostnameList {
			hostname = strings.TrimSpace(hostname)
			if strings.Contains(imagePrefix, hostname) {
				return types.MustParseInt(os.GetEnv("CUSTOM_IMAGE_REGISTRY_RATE_LIMIT", "20"))
			}
		}
	}
	return defaultRateLimitValue
}

// newRegistry initializes and returns a new registry.
func newRegistry(imagePrefix string) *registry {
	rateLimitValue := checkForCustomRateLimit(imagePrefix)
	return &registry{
		name:        imagePrefix,
		imagePrefix: imagePrefix,
		imageCache: cache.New(
			30*time.Minute, // Default ttl for each entry
			time.Hour,      // Cleanup interval
		),
		rateLimiter: ratelimit.New(rateLimitValue),
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
