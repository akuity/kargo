package os

import (
	"testing"

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

func TestGetBoolFromEnvVar(t *testing.T) {
	const testEnvVarName = "ENABLED"
	testCases := []struct {
		name       string
		setup      func()
		assertions func()
	}{
		{
			name: "env var does not exist",
			assertions: func() {
				require.True(t, MustGetEnvAsBool(testEnvVarName, true))
			},
		},
		{
			name: "env var value not parsable as bool",
			setup: func() {
				t.Setenv(testEnvVarName, "not really")
			},
			assertions: func() {
				require.Panics(t, func() {
					MustGetEnvAsBool(testEnvVarName, false)
				})
			},
		},
		{
			name: "env var exists",
			setup: func() {
				t.Setenv(testEnvVarName, "true")
			},
			assertions: func() {
				require.True(t, MustGetEnvAsBool(testEnvVarName, false))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.setup != nil {
				testCase.setup()
			}
			testCase.assertions()
		})
	}
}
