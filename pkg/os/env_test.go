package os

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetEnv(t *testing.T) {
	testSets := map[string]struct {
		Envs     map[string]string
		Key      string
		Default  string
		Expected string
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
	}
	for name, ts := range testSets {
		t.Run(name, func(t *testing.T) {
			for k, v := range ts.Envs {
				t.Setenv(k, v)
			}
			require.Equal(t, ts.Expected, GetEnv(ts.Key, ts.Default))
		})
	}
}
