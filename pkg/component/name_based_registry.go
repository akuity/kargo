package component

// NameBasedRegistryOptions represents configuration options for a
// NameBasedRegistry.
type NameBasedRegistryOptions struct {
	// AllowOverwriting indicates whether existing registrations may be
	// overwritten with new ones. When this is false, an attempt to overwrite an
	// existing registration produces an error.
	AllowOverwriting bool
}

// NameBasedRegistration associates a name (key) with a value and optional
// metadata. The value and metadata can be anything, including a function (often
// a factory function).
//
// Type parameters:
// - V: the type stored in the registry
// - MV: the type of metadata stored in the registry
type NameBasedRegistration[V, MD any] struct {
	Name     string
	Value    V
	Metadata MD
}

// NameBasedRegistry provides methods for registering and retrieving values and
// optional metadata by name (key). The value can be anything, including a
// function (often a factory function).
type NameBasedRegistry[V, MD any] interface {
	// Register adds a new registration to the registry.
	Register(NameBasedRegistration[V, MD]) error
	// MustRegister adds a new registration to the registry and panics if any
	// error is encountered.
	MustRegister(NameBasedRegistration[V, MD])
	// Get returns the registration matching the provided name (key) or, if none
	// is found, an empty registration and a NamedRegistrationNotFoundError.
	Get(string) (NameBasedRegistration[V, MD], error)
}

// NewNameBasedRegistry returns a default implementation of the
// NameBasedRegistry interface. Optional initial registrations may be provided
// when calling this function.
func NewNameBasedRegistry[V, MD any](
	opts *NameBasedRegistryOptions,
	registrations ...NameBasedRegistration[V, MD],
) (NameBasedRegistry[V, MD], error) {
	return newMapBasedRegistry(opts, registrations...)
}

// Must NewNameBasedRegistry returns a default implementation of the
// NameBasedRegistry interface. Optional initial registrations may be provided
// when calling this function. This function panics if any error is encountered.
func MustNewNameBasedRegistry[V, MD any](
	opts *NameBasedRegistryOptions,
	registrations ...NameBasedRegistration[V, MD],
) NameBasedRegistry[V, MD] {
	r, err := newMapBasedRegistry(opts, registrations...)
	if err != nil {
		panic(err)
	}
	return r
}
