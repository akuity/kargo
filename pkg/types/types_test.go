package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMustParseBool(t *testing.T) {
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

func TestMustParseFloat32(t *testing.T) {
	t.Parallel()
	testSets := map[string]struct {
		Input     string
		Expected  float32
		MustPanic bool
	}{
		"valid float32": {
			Input:    "1.23",
			Expected: 1.23,
		},
		"invalid float32": {
			Input:     "float",
			MustPanic: true,
		},
	}
	for name, ts := range testSets {
		t.Run(name, func(t *testing.T) {
			if ts.MustPanic {
				require.Panics(t, func() {
					_ = MustParseFloat32(ts.Input)
				})
			} else {
				require.Equal(t, ts.Expected, MustParseFloat32(ts.Input))
			}
		})
	}
}

func TestMustParseInt(t *testing.T) {
	t.Parallel()
	testSets := map[string]struct {
		Input     string
		Expected  int
		MustPanic bool
	}{
		"valid int": {
			Input:    "123",
			Expected: 123,
		},
		"invalid int": {
			Input:     "int",
			MustPanic: true,
		},
	}
	for name, ts := range testSets {
		t.Run(name, func(t *testing.T) {
			if ts.MustPanic {
				require.Panics(t, func() {
					_ = MustParseInt(ts.Input)
				})
			} else {
				require.Equal(t, ts.Expected, MustParseInt(ts.Input))
			}
		})
	}
}
