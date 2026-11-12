package metrics

import (
	"fmt"
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigFromEnv(t *testing.T) {
	testCases := []struct {
		name    string
		disable string
		assert  func(*testing.T, Config, error)
	}{
		{
			name: "unset",
			assert: func(t *testing.T, cfg Config, err error) {
				require.NoError(t, err)
				require.True(t, cfg.Enabled(FamilyRESTClient))
				require.True(t, cfg.Enabled(FamilyLifecycle))
			},
		},
		{
			name:    "rest-client disabled",
			disable: "rest-client",
			assert: func(t *testing.T, cfg Config, err error) {
				require.NoError(t, err)
				require.False(t, cfg.Enabled(FamilyRESTClient))
			},
		},
		{
			name:    "case and spaces ignored",
			disable: " Rest-Client ,",
			assert: func(t *testing.T, cfg Config, err error) {
				require.NoError(t, err)
				require.False(t, cfg.Enabled(FamilyRESTClient))
			},
		},
		{
			name:    "lifecycle disabled",
			disable: "lifecycle",
			assert: func(t *testing.T, cfg Config, err error) {
				require.NoError(t, err)
				require.False(t, cfg.Enabled(FamilyLifecycle))
				require.True(t, cfg.Enabled(FamilyRESTClient))
			},
		},
		{
			name:    "all",
			disable: "all",
			assert: func(t *testing.T, cfg Config, err error) {
				require.NoError(t, err)
				require.False(t, cfg.Enabled(FamilyRESTClient))
				require.False(t, cfg.Enabled(FamilyLifecycle))
			},
		},
		{
			name:    "ALL",
			disable: "ALL",
			assert: func(t *testing.T, cfg Config, err error) {
				require.NoError(t, err)
				require.False(t, cfg.Enabled(FamilyRESTClient))
			},
		},
		{
			// Redundant, but valid.
			name:    "all and a family",
			disable: " all ,rest-client",
			assert: func(t *testing.T, cfg Config, err error) {
				require.NoError(t, err)
				require.False(t, cfg.Enabled(FamilyRESTClient))
			},
		},
		{
			// A boolean is not a family.
			name:    "true",
			disable: "true",
			assert: func(t *testing.T, _ Config, err error) {
				require.ErrorContains(t, err, `unknown metrics family "true" in METRICS_DISABLE`)
			},
		},
		{
			name:    "unknown family",
			disable: "rest-client,rest-clients",
			assert: func(t *testing.T, _ Config, err error) {
				require.ErrorContains(t, err, `unknown metrics family "rest-clients" in METRICS_DISABLE`)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Not parallel: manipulates the environment.
			t.Setenv("METRICS_DISABLE", testCase.disable)
			cfg, err := ConfigFromEnv()
			testCase.assert(t, cfg, err)
		})
	}
}

// resetFamilies restores the registered families when t ends.
func resetFamilies(t *testing.T) {
	t.Helper()
	familiesMu.Lock()
	saved := slices.Clone(families)
	familiesMu.Unlock()
	t.Cleanup(func() {
		familiesMu.Lock()
		families = saved
		familiesMu.Unlock()
	})
}

func TestRegisterFamily(t *testing.T) {
	const familyTest Family = "test-family"
	testCases := []struct {
		name    string
		disable string
		assert  func(*testing.T, Config, error)
	}{
		{
			name: "unset",
			assert: func(t *testing.T, cfg Config, err error) {
				require.NoError(t, err)
				require.True(t, cfg.Enabled(familyTest))
				require.True(t, cfg.Enabled(FamilyRESTClient))
			},
		},
		{
			name:    "disabled by name",
			disable: "test-family",
			assert: func(t *testing.T, cfg Config, err error) {
				require.NoError(t, err)
				require.False(t, cfg.Enabled(familyTest))
				require.True(t, cfg.Enabled(FamilyRESTClient))
			},
		},
		{
			name:    "built-in family unaffected",
			disable: "rest-client",
			assert: func(t *testing.T, cfg Config, err error) {
				require.NoError(t, err)
				require.True(t, cfg.Enabled(familyTest))
				require.False(t, cfg.Enabled(FamilyRESTClient))
			},
		},
		{
			name:    "all",
			disable: "all",
			assert: func(t *testing.T, cfg Config, err error) {
				require.NoError(t, err)
				require.False(t, cfg.Enabled(familyTest))
				require.False(t, cfg.Enabled(FamilyRESTClient))
			},
		},
		{
			name:    "unknown family lists registered families",
			disable: "nope",
			assert: func(t *testing.T, _ Config, err error) {
				require.ErrorContains(t, err, "known families: rest-client")
				require.ErrorContains(t, err, "test-family")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Not parallel: manipulates the environment and package state.
			resetFamilies(t)
			RegisterFamily(familyTest)
			t.Setenv("METRICS_DISABLE", testCase.disable)
			cfg, err := ConfigFromEnv()
			testCase.assert(t, cfg, err)
		})
	}
}

func TestRegisterFamily_duplicate(t *testing.T) {
	resetFamilies(t)
	before := len(families)
	RegisterFamily("test-family")
	RegisterFamily("test-family")
	RegisterFamily(FamilyRESTClient)
	require.Len(t, families, before+1)
}

func TestRegisterFamily_invalid(t *testing.T) {
	testCases := []struct {
		name   string
		family Family
	}{
		{name: "empty", family: ""},
		{name: "all", family: "all"},
		{name: "ALL", family: "ALL"},
		{name: "upper case", family: "Test-Family"},
		{name: "underscore", family: "test_family"},
		{name: "space", family: "test family"},
		{name: "comma", family: "test,family"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			resetFamilies(t)
			require.Panics(t, func() { RegisterFamily(testCase.family) })
		})
	}
}

func TestRegisterFamily_concurrent(t *testing.T) {
	resetFamilies(t)
	var wg sync.WaitGroup
	for i := range 10 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			RegisterFamily(Family(fmt.Sprintf("test-family-%d", i)))
		}()
		go func() {
			defer wg.Done()
			_, _ = ConfigFromEnv()
		}()
	}
	wg.Wait()
	for i := range 10 {
		require.Contains(t, families, Family(fmt.Sprintf("test-family-%d", i)))
	}
}
