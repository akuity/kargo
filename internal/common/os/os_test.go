package os

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const testEnvVarName = "FOO"

func TestGetEnvVar(t *testing.T) {
	const testDefaultVal = "default"
	testCases := []struct {
		name       string
		setup      func()
		assertions func()
	}{
		{
			name: "env var does not exist",
			assertions: func() {
				require.Equal(
					t,
					testDefaultVal,
					GetEnvVar(testEnvVarName, testDefaultVal),
				)
			},
		},
		{
			name: "env var exists",
			setup: func() {
				t.Setenv(testEnvVarName, "foo")
			},
			assertions: func() {
				require.Equal(t, "foo", GetEnvVar(testEnvVarName, testDefaultVal))
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

func TestGetRequiredEnvVar(t *testing.T) {
	testCases := []struct {
		name       string
		setup      func()
		assertions func()
	}{
		{
			name: "env var does not exist",
			assertions: func() {
				_, err := GetRequiredEnvVar(testEnvVarName)
				require.Error(t, err)
			},
		},
		{
			name: "env var exists",
			setup: func() {
				t.Setenv(testEnvVarName, "foo")
			},
			assertions: func() {
				val, err := GetRequiredEnvVar(testEnvVarName)
				require.NoError(t, err)
				require.Equal(t, "foo", val)
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

func TestGetStringSliceFromEnvVar(t *testing.T) {
	testDefaultVal := []string{"foo", "bar"}
	testCases := []struct {
		name       string
		setup      func()
		assertions func()
	}{
		{
			name: "env var does not exist",
			assertions: func() {
				val := GetStringSliceFromEnvVar(testEnvVarName, testDefaultVal)
				require.Equal(t, testDefaultVal, val)
			},
		},
		{
			name: "env var exists",
			setup: func() {
				t.Setenv(testEnvVarName, "a,b,c")
			},
			assertions: func() {
				val := GetStringSliceFromEnvVar(testEnvVarName, testDefaultVal)
				require.Equal(t, []string{"a", "b", "c"}, val)
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

func TestGetIntFromEnvVar(t *testing.T) {
	const testDefaultVal = 1
	testCases := []struct {
		name       string
		setup      func()
		assertions func()
	}{
		{
			name: "env var does not exist",
			assertions: func() {
				val, err := GetIntFromEnvVar(testEnvVarName, testDefaultVal)
				require.NoError(t, err)
				require.Equal(t, testDefaultVal, val)
			},
		},
		{
			name: "env var value not parsable as int",
			setup: func() {
				t.Setenv(testEnvVarName, "life, the universe, and everything")
			},
			assertions: func() {
				_, err := GetIntFromEnvVar(testEnvVarName, testDefaultVal)
				require.Error(t, err)
				require.Contains(t, err.Error(), "was not parsable as an int")
			},
		},
		{
			name: "env var exists",
			setup: func() {
				t.Setenv(testEnvVarName, "42")
			},
			assertions: func() {
				val, err := GetIntFromEnvVar(testEnvVarName, testDefaultVal)
				require.NoError(t, err)
				require.Equal(t, 42, val)
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

func TestGetRequiredIntFromEnvVar(t *testing.T) {
	testCases := []struct {
		name       string
		setup      func()
		assertions func()
	}{
		{
			name: "env var does not exist",
			assertions: func() {
				_, err := GetRequiredIntFromEnvVar(testEnvVarName)
				require.Error(t, err)

			},
		},
		{
			name: "env var exists but is not parsable as an int",
			setup: func() {
				t.Setenv(testEnvVarName, "foo")
			},
			assertions: func() {
				_, err := GetRequiredIntFromEnvVar(testEnvVarName)
				require.Error(t, err)
				require.Contains(t, err.Error(), "not parsable as an int")
			},
		},
		{
			name: "env var exists and is parsable as an int",
			setup: func() {
				t.Setenv(testEnvVarName, "42")
			},
			assertions: func() {
				val, err := GetRequiredIntFromEnvVar(testEnvVarName)
				require.NoError(t, err)
				require.Equal(t, 42, val)
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

func TestGetBoolFromEnvVar(t *testing.T) {
	testCases := []struct {
		name       string
		setup      func()
		assertions func()
	}{
		{
			name: "env var does not exist",
			assertions: func() {
				val, err := GetBoolFromEnvVar(testEnvVarName, true)
				require.NoError(t, err)
				require.True(t, val)
			},
		},
		{
			name: "env var value not parsable as int",
			setup: func() {
				t.Setenv(testEnvVarName, "not really")
			},
			assertions: func() {
				_, err := GetBoolFromEnvVar(testEnvVarName, false)
				require.Error(t, err)
				require.Contains(t, err.Error(), "was not parsable as a bool")
			},
		},
		{
			name: "env var exists",
			setup: func() {
				t.Setenv(testEnvVarName, "true")
			},
			assertions: func() {
				val, err := GetBoolFromEnvVar(testEnvVarName, false)
				require.NoError(t, err)
				require.Equal(t, true, val)
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

func TestGetDurationFromEnvVar(t *testing.T) {
	const testDefaultVal = time.Minute
	testCases := []struct {
		name       string
		setup      func()
		assertions func()
	}{
		{
			name: "env var does not exist",
			assertions: func() {
				val, err := GetDurationFromEnvVar(testEnvVarName, testDefaultVal)
				require.NoError(t, err)
				require.Equal(t, testDefaultVal, val)
			},
		},
		{
			name: "env var value not parsable as duration",
			setup: func() {
				t.Setenv(testEnvVarName, "life, the universe, and everything")
			},
			assertions: func() {
				_, err := GetDurationFromEnvVar(testEnvVarName, testDefaultVal)
				require.Error(t, err)
				require.Contains(t, err.Error(), "was not parsable as a duration")
			},
		},
		{
			name: "env var exists",
			setup: func() {
				t.Setenv(testEnvVarName, "20s")
			},
			assertions: func() {
				val, err := GetDurationFromEnvVar(testEnvVarName, testDefaultVal)
				require.NoError(t, err)
				require.Equal(t, 20*time.Second, val)
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

func TestGetIPNetSliceFromEnvVar(t *testing.T) {
	_, ipNet1, err := net.ParseCIDR("192.168.1.0/24")
	require.NoError(t, err)
	_, ipNet2, err := net.ParseCIDR("192.168.2.0/24")
	require.NoError(t, err)
	testDefaultVal := []net.IPNet{*ipNet1, *ipNet2}
	testCases := []struct {
		name       string
		setup      func()
		assertions func()
	}{
		{
			name: "env var does not exist",
			assertions: func() {
				val, err := GetIPNetSliceFromEnvVar(testEnvVarName, testDefaultVal)
				require.NoError(t, err)
				require.Equal(t, testDefaultVal, val)
			},
		},
		{
			name: "error parsing env var",
			setup: func() {
				t.Setenv(testEnvVarName, "192.168.1.125") // Not CIDR
			},
			assertions: func() {
				_, err := GetIPNetSliceFromEnvVar(testEnvVarName, testDefaultVal)
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"was not parsable as a slice of CIDR address",
				)
				require.Contains(t, err.Error(), testEnvVarName)
			},
		},
		{
			name: "env var exists and is parsable",
			setup: func() {
				t.Setenv(testEnvVarName, "192.168.3.0/24,192.168.4.0/24")
			},
			assertions: func() {
				val, err := GetIPNetSliceFromEnvVar(testEnvVarName, testDefaultVal)
				require.NoError(t, err)
				require.Len(t, val, 2)
				require.Equal(t, "192.168.3.0/24", val[0].String())
				require.Equal(t, "192.168.4.0/24", val[1].String())
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
