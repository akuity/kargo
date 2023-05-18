package config

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestMustAtoi(t *testing.T) {
	t.Parallel()
	testSets := map[string]struct {
		Input     string
		Expected  int
		MustPanic bool
	}{
		"valid integer": {
			Input:    "123",
			Expected: 123,
		},
		"invalid integer": {
			Input:     "the integer",
			MustPanic: true,
		},
	}
	for name, ts := range testSets {
		t.Run(name, func(t *testing.T) {
			if ts.MustPanic {
				require.Panics(t, func() {
					_ = MustAtoi(ts.Input)
				})
			} else {
				require.Equal(t, ts.Expected, MustAtoi(ts.Input))
			}
		})
	}
}

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
		Expected  time.Duration
		MustPanic bool
	}{
		"valid duration": {
			Input:    "30s",
			Expected: 30 * time.Second,
		},
		"invalid duration": {
			Input:     "life, the universe, and everything",
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

func TestMustParseLogLevel(t *testing.T) {
	t.Parallel()
	testSets := map[string]struct {
		Input     string
		Expected  logrus.Level
		MustPanic bool
	}{
		"lower case": {
			Input:    "warn",
			Expected: logrus.WarnLevel,
		},
		"upper case": {
			Input:    "DEBUG",
			Expected: logrus.DebugLevel,
		},
		"mixed case": {
			Input:    "Info",
			Expected: logrus.InfoLevel,
		},
		"invalid level": {
			Input:     "Bogus",
			MustPanic: true,
		},
	}
	for name, ts := range testSets {
		t.Run(name, func(t *testing.T) {
			if ts.MustPanic {
				require.Panics(t, func() {
					_ = MustParseLogLevel(ts.Input)
				})
			} else {
				require.Equal(t, ts.Expected, MustParseLogLevel(ts.Input))
			}
		})
	}
}
