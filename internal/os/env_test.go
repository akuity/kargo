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
