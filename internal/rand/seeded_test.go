package rand

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewSeeded(t *testing.T) {
	rand, ok := NewSeeded().(*seeded)
	require.True(t, ok)
	require.NotNil(t, rand)
	require.NotNil(t, rand.rand)
	require.NotNil(t, rand.mut)
}

func TestIntn(t *testing.T) {
	// We have no visibility into the underlying *mathrand.Rand, so we'll test
	// that it is indeed seeded by comparing the results of Intn to results from
	// the unseeded random number generator in the math/rand package. We use a
	// very large n to reduce the likelihood of a coincidental match.
	const n = 2147483647
	require.NotEqual(t, rand.Intn(n), NewSeeded().Intn(n)) // nolint: gosec
}

func TestFloat64(t *testing.T) {
	// We have no visibility into the underlying *mathrand.Rand, so we'll test
	// that it is indeed seeded by comparing the results of Float64 to results
	// from the unseeded random number generator in the math/rand package.
	require.NotEqual(t, rand.Float64(), NewSeeded().Float64()) // nolint: gosec
}
