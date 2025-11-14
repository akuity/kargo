package component

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type (
	predicate             = func(context.Context, string) (bool, error)
	predicateRegistration = PredicateBasedRegistration[
		string,    // Arg to the predicate function
		predicate, // Predicate function
		string,    // Value type
		string,    // Metadata type
	]
	predicateRegistryIface = PredicateBasedRegistry[
		string,    // Arg to the predicate function
		predicate, // Predicate function
		string,    // Value type
		string,    // Metadata type
	]
	predicateRegistry = listBasedRegistry[
		string,    // Arg to the predicate function
		predicate, // Predicate function
		string,    // Value type
		string,    // Metadata type
	]
)

func TestNewListBasedRegistry(t *testing.T) {
	testCases := []struct {
		name                 string
		initialRegistrations []predicateRegistration
		assertions           func(*testing.T, predicateRegistryIface, error)
	}{
		{
			name: "empty registry",
			assertions: func(t *testing.T, reg predicateRegistryIface, err error) {
				require.NoError(t, err)
				r, ok := reg.(*predicateRegistry)
				require.True(t, ok)
				require.NotNil(t, r.registrations)
				require.Empty(t, r.registrations)
			},
		},
		{
			name: "with initial registrations",
			initialRegistrations: []predicateRegistration{{
				Predicate: func(context.Context, string) (bool, error) {
					return false, nil
				},
				Value:    "test",
				Metadata: "meta",
			}},
			assertions: func(t *testing.T, reg predicateRegistryIface, err error) {
				require.NoError(t, err)
				r, ok := reg.(*predicateRegistry)
				require.True(t, ok)
				require.Len(t, r.registrations, 1)
				require.NotNil(t, r.registrations[0].Predicate)
				require.Equal(t, "test", r.registrations[0].Value)
				require.Equal(t, "meta", r.registrations[0].Metadata)
			},
		},
		{
			name: "with an invalid initial registration",
			initialRegistrations: []predicateRegistration{{
				Predicate: nil, // Whoops!
			}},
			assertions: func(t *testing.T, _ predicateRegistryIface, err error) {
				require.ErrorContains(t, err, "registration has nil predicate function")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			registry, err := NewPredicateBasedRegistry(testCase.initialRegistrations...)
			testCase.assertions(t, registry, err)
		})
	}
}

func TestListBasedRegistry_Register(t *testing.T) {
	testCases := []struct {
		name         string
		registration predicateRegistration
		assertions   func(*testing.T, *predicateRegistry, error)
	}{
		{
			name: "missing predicate",
			registration: predicateRegistration{
				Predicate: nil, // Whoops!
			},
			assertions: func(t *testing.T, _ *predicateRegistry, err error) {
				require.ErrorContains(t, err, "registration has nil predicate function")
			},
		},
		{
			name: "success",
			registration: predicateRegistration{
				Predicate: func(context.Context, string) (bool, error) {
					return false, nil
				},
				Value:    "test",
				Metadata: "meta",
			},
			assertions: func(t *testing.T, registry *predicateRegistry, err error) {
				require.NoError(t, err, "foo")
				require.Len(t, registry.registrations, 1)
				require.Len(t, registry.registrations, 1)
				require.NotNil(t, registry.registrations[0].Predicate)
				require.Equal(t, "test", registry.registrations[0].Value)
				require.Equal(t, "meta", registry.registrations[0].Metadata)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			registry := &predicateRegistry{
				registrations: []predicateRegistration{},
			}
			testCase.assertions(t, registry, registry.Register(testCase.registration))
		})
	}
}

func TestListBasedRegistry_Get(t *testing.T) {
	testCases := []struct {
		name       string
		registry   *predicateRegistry
		assertions func(*testing.T, predicateRegistration, error)
	}{
		{
			name: "error evaluating predicate",
			registry: &predicateRegistry{
				registrations: []predicateRegistration{{
					Predicate: func(context.Context, string) (bool, error) {
						return false, errors.New("something went wrong")
					},
				}},
			},
			assertions: func(t *testing.T, reg predicateRegistration, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.Empty(t, reg)
			},
		},
		{
			name:     "no match",
			registry: &predicateRegistry{},
			assertions: func(t *testing.T, reg predicateRegistration, err error) {
				require.Error(t, err)
				require.ErrorAs(t, err, &RegistrationNotFoundError{})
				require.Empty(t, reg)
			},
		},
		{
			name: "match",
			registry: &predicateRegistry{
				registrations: []predicateRegistration{{
					Predicate: func(context.Context, string) (bool, error) {
						return true, nil
					},
					Value:    "output",
					Metadata: "meta",
				}},
			},
			assertions: func(t *testing.T, reg predicateRegistration, err error) {
				require.NoError(t, err)
				require.Equal(t, "output", reg.Value)
				require.Equal(t, "meta", reg.Metadata)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			val, err := testCase.registry.Get(t.Context(), "test")
			testCase.assertions(t, val, err)
		})
	}
}

func TestListBasedRegistry_WithFunctionValues(t *testing.T) {
	// Test with function types to ensure the registry works with factory
	// functions
	type (
		factoryFunc         = func(context.Context, string) (string, error)
		factoryRegistration = PredicateBasedRegistration[
			string,      // Arg to the predicate function
			predicate,   // Predicate function
			factoryFunc, // The value stored in the registry
			struct{},    // This registry uses no metadata
		]
		factoryRegistry = listBasedRegistry[
			string,      // Arg to the predicate function
			predicate,   // Predicate function
			factoryFunc, // The value stored in the registry
			struct{},    // This registry uses no metadata
		]
	)

	registry := &factoryRegistry{
		registrations: []factoryRegistration{{
			Predicate: func(_ context.Context, input string) (bool, error) {
				return input == "match", nil
			},
			Value: func(_ context.Context, input string) (string, error) {
				return "output-" + input, nil
			},
		}},
	}

	// Test matching predicate
	reg, err := registry.Get(context.Background(), "match")
	require.NoError(t, err)

	// Verify the factory function works
	result, err := reg.Value(context.Background(), "test")
	require.NoError(t, err)
	require.Equal(t, "output-test", result)
}
