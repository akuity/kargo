package namer

import (
	"errors"
	"math/rand/v2"
	"slices"
)

// Namer generates random, human-friendly names.
type Namer interface {
	// Name returns a randomly generated name of the form
	// <descriptor>-<noun>.
	Name() string
}

type namer struct {
	descriptors []string
	nouns       []string
	intN        func(int) int
}

// NewDefault returns a Namer that composes names from the default descriptor
// and noun lists.
func NewDefault() Namer {
	return &namer{
		descriptors: defaultDescriptors,
		nouns:       defaultNouns,
		intN:        rand.IntN,
	}
}

// New returns a Namer that composes names from the given descriptors and
// nouns. Both lists must be non-empty. The Namer keeps its own copies, so
// later changes to the lists passed in do not affect it.
func New(descriptors, nouns []string) (Namer, error) {
	if len(descriptors) == 0 {
		return nil, errors.New("descriptors list must not be empty")
	}
	if len(nouns) == 0 {
		return nil, errors.New("nouns list must not be empty")
	}
	return &namer{
		descriptors: slices.Clone(descriptors),
		nouns:       slices.Clone(nouns),
		intN:        rand.IntN,
	}, nil
}

// Name implements Namer.
func (n *namer) Name() string {
	return n.descriptors[n.intN(len(n.descriptors))] +
		"-" + n.nouns[n.intN(len(n.nouns))]
}
