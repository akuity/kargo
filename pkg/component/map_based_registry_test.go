package component

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type (
	namedRegistration  = NameBasedRegistration[string, string]
	namedRegistryIface = NameBasedRegistry[string, string]
	namedRegistry      = mapBasedRegistry[string, string]
)

func TestNewMapBasedRegistry(t *testing.T) {
	testCases := []struct {
		name                 string
		opts                 *NameBasedRegistryOptions
		initialRegistrations []namedRegistration
		assertions           func(*testing.T, namedRegistryIface, error)
	}{
		{
			name: "nil options",
			assertions: func(t *testing.T, registry namedRegistryIface, err error) {
				require.NoError(t, err)
				require.NotNil(t, registry)
				r, ok := registry.(*namedRegistry)
				require.True(t, ok)
				require.False(t, r.opts.AllowOverwriting)
			},
		},
		{
			name: "with options",
			opts: &NameBasedRegistryOptions{AllowOverwriting: true},
			assertions: func(t *testing.T, registry namedRegistryIface, err error) {
				require.NoError(t, err)
				require.NotNil(t, registry)
				r, ok := registry.(*namedRegistry)
				require.True(t, ok)
				require.True(t, r.opts.AllowOverwriting)
			},
		},
		{
			name: "without initial registrations",
			assertions: func(t *testing.T, registry namedRegistryIface, err error) {
				require.NoError(t, err)
				require.NotNil(t, registry)
				r, ok := registry.(*namedRegistry)
				require.True(t, ok)
				require.NotNil(t, r.registrations)
				require.Empty(t, r.registrations)
			},
		},
		{
			name: "with initial registrations",
			initialRegistrations: []namedRegistration{{
				Name:     "test",
				Value:    "value",
				Metadata: "meta",
			}},
			assertions: func(t *testing.T, registry namedRegistryIface, err error) {
				require.NoError(t, err)
				r, ok := registry.(*namedRegistry)
				require.True(t, ok)
				require.Equal(
					t,
					map[string]namedRegistration{
						"test": {
							Name:     "test",
							Value:    "value",
							Metadata: "meta",
						},
					},
					r.registrations,
				)
			},
		},
		{
			name: "initial registration overwrites when not allowed",
			initialRegistrations: []namedRegistration{
				{
					Name:     "test",
					Value:    "original",
					Metadata: "meta",
				},
				{
					Name:     "test", // Should not be reused
					Value:    "new",
					Metadata: "meta",
				},
			},
			assertions: func(t *testing.T, _ namedRegistryIface, err error) {
				require.ErrorContains(t, err, "cannot overwrite registration")
			},
		},
		{
			name: "initial registration overwrites when allowed",
			opts: &NameBasedRegistryOptions{AllowOverwriting: true},
			initialRegistrations: []namedRegistration{
				{
					Name:     "test",
					Value:    "original",
					Metadata: "meta",
				},
				{
					Name:     "test", // Overwriting is allowed
					Value:    "new",
					Metadata: "meta",
				},
			},
			assertions: func(t *testing.T, registry namedRegistryIface, err error) {
				require.NoError(t, err)
				r, ok := registry.(*namedRegistry)
				require.True(t, ok)
				require.Equal(
					t,
					map[string]namedRegistration{
						"test": {
							Name:     "test",
							Value:    "new",
							Metadata: "meta",
						},
					},
					r.registrations,
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			registry, err := newMapBasedRegistry(
				testCase.opts,
				testCase.initialRegistrations...,
			)
			testCase.assertions(t, registry, err)
		})
	}
}

func TestMapBasedRegistry_Register(t *testing.T) {
	testReg := namedRegistration{
		Name:     "test",
		Value:    "value",
		Metadata: "meta",
	}
	testCases := []struct {
		name            string
		registry        *namedRegistry
		newRegistration namedRegistration
		assertions      func(*testing.T, *namedRegistry, error)
	}{
		{
			name: "no name",
			registry: &namedRegistry{
				registrations: map[string]namedRegistration{},
			},
			newRegistration: namedRegistration{},
			assertions: func(t *testing.T, registry *namedRegistry, err error) {
				require.ErrorContains(t, err, "registration name cannot be empty")
				require.Empty(t, registry.registrations)
			},
		},
		{
			name: "basic registration",
			registry: &namedRegistry{
				registrations: map[string]namedRegistration{},
			},
			newRegistration: testReg,
			assertions: func(t *testing.T, registry *namedRegistry, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					testReg,
					registry.registrations[testReg.Name],
				)
			},
		},
		{
			name: "overwriting when not allowed",
			registry: &namedRegistry{
				opts:          NameBasedRegistryOptions{AllowOverwriting: false},
				registrations: map[string]namedRegistration{testReg.Name: testReg},
			},
			newRegistration: testReg,
			assertions: func(t *testing.T, registry *namedRegistry, err error) {
				require.ErrorContains(t, err, "cannot overwrite registration")
				require.Equal(
					t,
					testReg,
					registry.registrations[testReg.Name],
				)
			},
		},
		{
			name: "overwriting when allowed",
			registry: &namedRegistry{
				opts:          NameBasedRegistryOptions{AllowOverwriting: false},
				registrations: map[string]namedRegistration{testReg.Name: testReg},
			},
			newRegistration: testReg,
			assertions: func(t *testing.T, registry *namedRegistry, err error) {
				require.ErrorContains(t, err, "cannot overwrite registration")
				require.Equal(
					t,
					testReg,
					registry.registrations[testReg.Name],
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				testCase.registry,
				testCase.registry.Register(testCase.newRegistration),
			)
		})
	}
}

func TestMapBasedRegistry_Get(t *testing.T) {
	testReg := namedRegistration{
		Name:     "test",
		Value:    "value",
		Metadata: "meta",
	}
	testCases := []struct {
		name       string
		registry   *namedRegistry
		assertions func(*testing.T, namedRegistration, error)
	}{
		{
			name:     "key not found",
			registry: &namedRegistry{registrations: map[string]namedRegistration{}},
			assertions: func(t *testing.T, reg namedRegistration, err error) {
				require.Error(t, err)
				require.ErrorAs(t, err, &NamedRegistrationNotFoundError{})
				require.Empty(t, reg)
			},
		},
		{
			name: "key found",
			registry: &namedRegistry{
				registrations: map[string]namedRegistration{
					testReg.Name: testReg,
				},
			},
			assertions: func(t *testing.T, reg namedRegistration, err error) {
				require.NoError(t, err)
				require.Equal(t, testReg, reg)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			reg, err := testCase.registry.Get(testReg.Name)
			testCase.assertions(t, reg, err)
		})
	}
}

func TestMapBasedRegistry_WithFunctionValues(t *testing.T) {
	// Test with function types to ensure the registry works with factory
	// functions
	type (
		factoryFunc         = func(context.Context, string) (string, error)
		factoryRegistration = NameBasedRegistration[factoryFunc, struct{}]
		factoryRegistry     = mapBasedRegistry[factoryFunc, struct{}]
	)

	registry := &factoryRegistry{
		registrations: map[string]factoryRegistration{
			"test-factory": {
				Name: "test-factory",
				Value: func(_ context.Context, input string) (string, error) {
					return "output-" + input, nil
				},
			},
		},
	}

	reg, err := registry.Get("test-factory")
	require.NoError(t, err)

	// Verify the factory function works
	result, err := reg.Value(context.Background(), "test")
	require.NoError(t, err)
	require.Equal(t, "output-test", result)
}
