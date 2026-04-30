package component

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNamedRegistrationNotFoundError_Error(t *testing.T) {
	testCases := []struct {
		name       string
		err        NamedRegistrationNotFoundError
		assertions func(*testing.T, string)
	}{
		{
			name: "with name",
			err: NamedRegistrationNotFoundError{
				Name: "test-registration",
			},
			assertions: func(t *testing.T, msg string) {
				require.Equal(t, "registration with name test-registration not found", msg)
			},
		},
		{
			name: "with empty name",
			err: NamedRegistrationNotFoundError{
				Name: "",
			},
			assertions: func(t *testing.T, msg string) {
				require.Equal(t, "registration with name  not found", msg)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(t, testCase.err.Error())
		})
	}
}

func TestRegistrationNotFoundError_Error(t *testing.T) {
	err := RegistrationNotFoundError{}
	require.Equal(t, "no matching registration found", err.Error())
}

func TestIsNotFoundError(t *testing.T) {
	testCases := []struct {
		name       string
		err        error
		assertions func(*testing.T, bool)
	}{
		{
			name: "nil error",
			err:  nil,
			assertions: func(t *testing.T, result bool) {
				require.False(t, result)
			},
		},
		{
			name: "RegistrationNotFoundError",
			err:  RegistrationNotFoundError{},
			assertions: func(t *testing.T, result bool) {
				require.True(t, result)
			},
		},
		{
			name: "NamedRegistrationNotFoundError",
			err: NamedRegistrationNotFoundError{
				Name: "test",
			},
			assertions: func(t *testing.T, result bool) {
				require.True(t, result)
			},
		},
		{
			name: "wrapped RegistrationNotFoundError",
			err:  fmt.Errorf("wrapped: %w", RegistrationNotFoundError{}),
			assertions: func(t *testing.T, result bool) {
				require.True(t, result)
			},
		},
		{
			name: "wrapped NamedRegistrationNotFoundError",
			err: fmt.Errorf("wrapped: %w", NamedRegistrationNotFoundError{
				Name: "test",
			}),
			assertions: func(t *testing.T, result bool) {
				require.True(t, result)
			},
		},
		{
			name: "different error type",
			err:  errors.New("some other error"),
			assertions: func(t *testing.T, result bool) {
				require.False(t, result)
			},
		},
		{
			name: "wrapped different error type",
			err:  fmt.Errorf("wrapped: %w", errors.New("some other error")),
			assertions: func(t *testing.T, result bool) {
				require.False(t, result)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(t, IsNotFoundError(testCase.err))
		})
	}
}
