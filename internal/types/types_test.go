package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseBool(t *testing.T) {
	t.Parallel()
	testSets := map[string]struct {
		Input     string
		Expected  bool
		MustPanic bool
	}{
		"valid bool (true)": {
			Input:    "true",
			Expected: true,
		},
		"valid bool (false)": {
			Input:    "false",
			Expected: false,
		},
		"invalid bool": {
			Input:     "bool",
			MustPanic: true,
		},
	}
	for name, ts := range testSets {
		t.Run(name, func(t *testing.T) {
			if ts.MustPanic {
				require.Panics(t, func() {
					_ = MustParseBool(ts.Input)
				})
			} else {
				require.Equal(t, ts.Expected, MustParseBool(ts.Input))
			}
		})
	}
}
