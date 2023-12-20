package gitprovider

import (
	"fmt"
)

// ProviderRegistration holds details on how to instantiate the correct git provider
// based on parameters (i.e. repo URL). It allows programs to selectively register
// GitProviderService implementations by anonymously importing implementation packages.
type ProviderRegistration struct {
	// Predicate is a function which should return true if the given repoURL is appropriate
	// for the provider to handle (e.g. github.com is the domain name)
	Predicate func(repoURL string) bool
	// NewService instantiates the git provider
	NewService func() (GitProviderService, error)
}

var (
	// registeredProviders is a mapping between provider name and provider registration
	registeredProviders = map[string]ProviderRegistration{}
)

// NewGitProviderServiceFromName returns a git provider service by it's registered name
func NewGitProviderServiceFromName(name string) (GitProviderService, error) {
	if reg, ok := registeredProviders[name]; ok {
		return reg.NewService()
	}
	return nil, fmt.Errorf("No registered providers with name %q", name)
}

// NewGitProviderServiceFromURL iterates all registered providers and instantiates the
// appropriate GitProvider service implementation (GitHub, GitLab, BitBucket)
// based on inference of the repo URL.
func NewGitProviderServiceFromURL(repoURL string) (GitProviderService, error) {
	for _, reg := range registeredProviders {
		if reg.Predicate(repoURL) {
			return reg.NewService()
		}
	}
	return nil, fmt.Errorf("No registered providers for %s", repoURL)
}

// RegisterProvider is called by provider implementation packages to register themselves as
// a git provider.
func RegisterProvider(name string, reg ProviderRegistration) {
	if _, alreadyRegistered := registeredProviders[name]; alreadyRegistered {
		panic(fmt.Sprintf("Provider %q already registered", name))
	}
	registeredProviders[name] = reg
}
