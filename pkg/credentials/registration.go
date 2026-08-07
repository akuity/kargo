package credentials

import (
	"github.com/akuity/kargo/pkg/os"
	"github.com/akuity/kargo/pkg/types"
)

// ProvidersEnabled indicates whether credential providers should add themselves
// to DefaultProviderRegistry. Every provider's init() consults this before
// doing anything else.
//
// Kargo's components share a single binary, so a provider's init() runs no
// matter which component was invoked, and deciding whether a provider applies
// is not free: it reads configuration and, in one case, reaches for a cloud
// provider's metadata server. Only components that resolve credentials have any
// use for that, so the rest opt out by leaving this unset. An environment
// variable is the only mechanism available, as init() runs before a component
// has had any chance to say what it is.
func ProvidersEnabled() bool {
	return types.MustParseBool(os.GetEnv("CREDENTIAL_PROVIDERS_ENABLED", "false"))
}
