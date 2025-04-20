package gitprovider

import (
	"fmt"
)

// Registration holds details on how to instantiate a correct implementation of
// Interface based on parameters (i.e. repo URL). It allows programs to
// selectively register implementations by anonymously importing their packages.
type Registration struct {
	// Predicate is a function which should return true if the given repoURL is
	// appropriate for the provider to handle (e.g. github.com is the domain
	// name).
	Predicate func(repoURL string) bool
	// NewProvider instantiates the registered provider implementation.
	NewProvider func(repoURL string, opts *Options) (Interface, error)
}

// registeredProviders is a mapping between provider name and provider registration
var registeredProviders = map[string]Registration{}

// New returns an implementation of Interface suitable for the provided
// repository URL and options. It will return an error if no suitable
// implementation is found.
func New(repoURL string, opts *Options) (Interface, error) {
	if opts == nil {
		opts = &Options{}
	}
	if opts.Name != "" {
		if reg, found := registeredProviders[opts.Name]; found {
			return reg.NewProvider(repoURL, opts)
		}
		return nil, fmt.Errorf("No registered providers with name %q", opts.Name)
	}
	for _, reg := range registeredProviders {
		if reg.Predicate(repoURL) {
			return reg.NewProvider(repoURL, opts)
		}
	}
	return nil, fmt.Errorf("No registered providers for %s", repoURL)
}

// Register is called by provider implementation packages to register themselves
// as a git provider.
func Register(name string, reg Registration) {
	if _, alreadyRegistered := registeredProviders[name]; alreadyRegistered {
		panic(fmt.Sprintf("Provider %q already registered", name))
	}
	registeredProviders[name] = reg
}
