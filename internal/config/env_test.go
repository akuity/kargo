package config

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestMustGetEnv(t *testing.T) {
	testSets := map[string]struct {
		Envs      map[string]string
		Key       string
		Default   string
		Expected  string
		MustPanic bool
	}{
		"return env value": {
			Envs: map[string]string{
				"test": "some value",
			},
			Key:      "test",
			Expected: "some value",
		},
		"return default value if env not exists": {
			Key:      "test",
			Default:  "some value",
			Expected: "some value",
		},
		"empty default value": {
			Key:       "test",
			MustPanic: true,
		},
	}
	for name, ts := range testSets {
		t.Run(name, func(t *testing.T) {
			for k, v := range ts.Envs {
				t.Setenv(k, v)
			}
			if ts.MustPanic {
				require.Panics(t, func() {
					_ = MustGetEnv(ts.Key, ts.Default)
				})
			} else {
				require.Equal(t, ts.Expected, MustGetEnv(ts.Key, ts.Default))
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
