package types

import (
	"testing"
	"time"

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

func TestMustParseDuration(t *testing.T) {
	t.Parallel()
	testSets := map[string]struct {
		Input     string
		Expected  *time.Duration
		MustPanic bool
	}{
		"empty duration": {
			Input:    "",
			Expected: nil,
		},
		"valid duration": {
			Input:    "1h",
			Expected: MustParseDuration("1h"),
		},
		"invalid duration": {
			Input:     "duration",
			MustPanic: true,
		},
	}
	for name, ts := range testSets {
		t.Run(name, func(t *testing.T) {
			if ts.MustPanic {
				require.Panics(t, func() {
					_ = MustParseDuration(ts.Input)
				})
			} else {
				require.Equal(t, ts.Expected, MustParseDuration(ts.Input))
			}
		})
	}
}
