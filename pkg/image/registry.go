package image

import (
	"sync"

	"github.com/google/go-containerregistry/pkg/name"
	"go.uber.org/ratelimit"

	"github.com/akuity/kargo/pkg/os"
	"github.com/akuity/kargo/pkg/types"
)

var (
	// rateLimit is the rate limit (in requests per second) to apply to all
	// registry interactions on a per-registry basis.
	rateLimit = types.MustParseInt(os.GetEnv("IMAGE_REGISTRY_RATE_LIMIT", "20"))

	// dockerRegistry is registry configuration for Docker Hub.
	dockerRegistry = &registry{
		name:             "Docker Hub",
		imagePrefix:      name.DefaultRegistry,
		defaultNamespace: "library",
		rateLimiter:      ratelimit.New(rateLimit),
	}

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
	rateLimiter      ratelimit.Limiter
}

// newRegistry initializes and returns a new registry.
func newRegistry(imagePrefix string) *registry {
	return &registry{
		name:        imagePrefix,
		imagePrefix: imagePrefix,
		// TODO(krancour): We probably need to make this tunable per registry in
		// the future.
		rateLimiter: ratelimit.New(rateLimit),
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
