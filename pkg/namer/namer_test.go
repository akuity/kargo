package namer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewDefault(t *testing.T) {
	t.Parallel()
	n := NewDefault()
	require.NotNil(t, n)
	parts := strings.Split(n.Name(), "-")
	require.Len(t, parts, 2)
	require.Contains(t, defaultDescriptors, parts[0])
	require.Contains(t, defaultNouns, parts[1])
}

func TestNew(t *testing.T) {
	t.Parallel()
	descriptors := []string{"happy"}
	nouns := []string{"cat"}
	testCases := []struct {
		name        string
		descriptors []string
		nouns       []string
		assert      func(*testing.T, Namer, error)
	}{
		{
			name:  "empty descriptors",
			nouns: []string{"cat"},
			assert: func(t *testing.T, n Namer, err error) {
				require.ErrorContains(t, err, "descriptors list must not be empty")
				require.Nil(t, n)
			},
		},
		{
			name:        "empty nouns",
			descriptors: []string{"happy"},
			assert: func(t *testing.T, n Namer, err error) {
				require.ErrorContains(t, err, "nouns list must not be empty")
				require.Nil(t, n)
			},
		},
		{
			name:        "success",
			descriptors: descriptors,
			nouns:       nouns,
			assert: func(t *testing.T, n Namer, err error) {
				require.NoError(t, err)
				require.Equal(t, "happy-cat", n.Name())
				// The Namer must hold its own copies of the lists.
				descriptors[0] = "grumpy"
				nouns[0] = "dog"
				require.Equal(t, "happy-cat", n.Name())
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			n, err := New(testCase.descriptors, testCase.nouns)
			testCase.assert(t, n, err)
		})
	}
}

func TestName(t *testing.T) {
	t.Parallel()
	// Deterministic picks: always the last element of each list.
	n := &namer{
		descriptors: []string{"happy", "sleepy"},
		nouns:       []string{"cat", "dog"},
		intN:        func(i int) int { return i - 1 },
	}
	require.Equal(t, "sleepy-dog", n.Name())
}
