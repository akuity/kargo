package image

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"go.uber.org/ratelimit"
)

// dockerRegistry is registry configuration for Docker Hub, whose API endpoint
// cannot be inferred from an image prefix because its API endpoint is at
// https://registry-1.docker.io despite Docker Hub images either lacking a
// prefix entirely or beginning with docker.io
var dockerRegistry = &registry{
	name:             "Docker Hub",
	imagePrefix:      "docker.io",
	apiAddress:       "https://registry-1.docker.io",
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
	apiAddress       string
	defaultNamespace string
	imageCache       *cache.Cache
	rateLimiter      ratelimit.Limiter
}

// newRegistry initializes and returns a new registry.
func newRegistry(imagePrefix string) *registry {
	return &registry{
		name:        imagePrefix,
		imagePrefix: imagePrefix,
		apiAddress:  fmt.Sprintf("https://%s", imagePrefix),
		imageCache: cache.New(
			30*time.Minute, // Default ttl for each entry
			time.Hour,      // Cleanup interval
		),
		// TODO: Make this configurable.
		rateLimiter: ratelimit.New(20),
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

// normalizeImageName returns a normalized image name that accounts for the fact
// that some registries have a default namespace that is used when the image
// name doesn't specify one. For example on Docker Hub, "debian" officially
// equates to "library/debian".
func (r *registry) normalizeImageName(image string) string {
	if len(strings.Split(image, "/")) == 1 && r.defaultNamespace != "" {
		return fmt.Sprintf("%s/%s", r.defaultNamespace, image)
	}
	return image
}
