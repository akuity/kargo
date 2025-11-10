package component

import "context"

// Predicate is a constraint for functions that evaluate whether a given input
// of type A matches certain criteria. Predicates return true if the input
// matches, false otherwise, and may return an error if evaluation fails.
type Predicate[A any] interface {
	~func(context.Context, A) (bool, error)
}

// PredicateBasedRegistration associates a predicate function with a value and
// optional metadata. The value and metadata can be anything, including a
// function (often a factory function).
//
// Type parameters:
// - PA: the input type for the predicate
// - P: the predicate function type (must satisfy Predicate[PA])
// - V: the type stored in the registry
// - MV: the type of metadata stored in the registry
type PredicateBasedRegistration[PA any, P Predicate[PA], V, MD any] struct {
	Predicate P
	Value     V
	Metadata  MD
}

// PredicateBasedRegistry provides methods for registering and retrieving values
// and metadata based on predicate evaluation.
//
// Type parameters match those of Registration.
type PredicateBasedRegistry[PA any, P Predicate[PA], V, MD any] interface {
	// Register adds a new registration to the registry.
	Register(PredicateBasedRegistration[PA, P, V, MD]) error
	// MustRegister adds a new registration to the registry and panics if any
	// error is encountered.
	MustRegister(PredicateBasedRegistration[PA, P, V, MD])
	// Get searches for a matching registration by evaluating each registration's
	// predicate against the provided input. Returns the first matching
	// registration, or a zero value if no match is found, as well as a boolean
	// indicating whether a match was found, which callers may use that to
	// distinguish between a true zero return value and a zero value resulting
	// from no match having been found.
	Get(context.Context, PA) (
		PredicateBasedRegistration[PA, P, V, MD],
		bool,
		error,
	)
}

// NewPredicateBasedRegistry returns a default implementation of the
// PredicateBasedRegistry interface. Optional initial registrations may be
// provided when calling this function.
func NewPredicateBasedRegistry[PA any, P Predicate[PA], V, MD any](
	registrations ...PredicateBasedRegistration[PA, P, V, MD],
) (PredicateBasedRegistry[PA, P, V, MD], error) {
	return newListBasedRegistry(registrations...)
}

// MustNewPredicateBasedRegistry returns a default implementation of the
// PredicateBasedRegistry interface. Optional initial registrations may be
// provided when calling this function. This function panics if any error is
// encountered.
func MustNewPredicateBasedRegistry[PA any, P Predicate[PA], V, MD any](
	registrations ...PredicateBasedRegistration[PA, P, V, MD],
) PredicateBasedRegistry[PA, P, V, MD] {
	r, err := NewPredicateBasedRegistry(registrations...)
	if err != nil {
		panic(err)
	}
	return r
}
