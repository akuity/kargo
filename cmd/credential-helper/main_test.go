package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_answer(t *testing.T) {
	const (
		testUsername = "user"
		testPassword = "pass"
	)
	testCases := []struct {
		name     string
		prompt   string
		username string
		password string
		assert   func(*testing.T, string, error)
	}{
		{
			name:     "username prompt",
			prompt:   "Username for 'https://example.com': ",
			username: testUsername,
			password: testPassword,
			assert: func(t *testing.T, res string, err error) {
				require.NoError(t, err)
				require.Equal(t, testUsername, res)
			},
		},
		{
			name:     "username prompt, empty username",
			prompt:   "Username for 'https://example.com': ",
			password: testPassword,
			assert: func(t *testing.T, res string, err error) {
				require.NoError(t, err)
				require.Empty(t, res)
			},
		},
		{
			name:     "password prompt",
			prompt:   "Password for 'https://user@example.com': ",
			username: testUsername,
			password: testPassword,
			assert: func(t *testing.T, res string, err error) {
				require.NoError(t, err)
				require.Equal(t, testPassword, res)
			},
		},
		{
			name:     "password prompt, no password",
			prompt:   "Password for 'https://user@example.com': ",
			username: testUsername,
			assert: func(t *testing.T, _ string, err error) {
				require.EqualError(t, err, "GIT_PASSWORD must be set")
			},
		},
		{
			name:     "no prompt",
			username: testUsername,
			password: testPassword,
			assert: func(t *testing.T, res string, err error) {
				require.NoError(t, err)
				require.Equal(t, testPassword, res)
			},
		},
		{
			name:     "any other prompt",
			prompt:   "Certificate Password for 'cert:///x': ",
			username: testUsername,
			password: testPassword,
			assert: func(t *testing.T, res string, err error) {
				require.NoError(t, err)
				require.Equal(t, testPassword, res)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			res, err := answer(
				testCase.prompt,
				testCase.username,
				testCase.password,
			)
			testCase.assert(t, res, err)
		})
	}
}
