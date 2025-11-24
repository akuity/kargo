package basic

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/credentials"
)

func TestCredentialProvider_Supports(t *testing.T) {
	tests := []struct {
		name     string
		credType credentials.Type
		repoURL  string
		data     map[string][]byte
		expected bool
	}{
		{
			name:     "empty data",
			credType: credentials.TypeGit,
			repoURL:  "https://github.com/example/repository.git",
			data:     map[string][]byte{},
			expected: false,
		},
		{
			name:     "username only",
			credType: credentials.TypeGit,
			repoURL:  "https://github.com/example/repository.git",
			data: map[string][]byte{
				usernameKey: []byte("user"),
			},
			expected: false,
		},
		{
			name:     "password only",
			credType: credentials.TypeGit,
			repoURL:  "https://github.com/example/repository.git",
			data: map[string][]byte{
				passwordKey: []byte("pass"),
			},
			expected: false,
		},
		{
			name:     "username and password",
			credType: credentials.TypeGit,
			repoURL:  "https://github.com/example/repository.git",
			data: map[string][]byte{
				usernameKey: []byte("user"),
				passwordKey: []byte("pass"),
			},
			expected: true,
		},
		{
			name:     "nil username value",
			credType: credentials.TypeGit,
			repoURL:  "https://github.com/example/repository.git",
			data: map[string][]byte{
				usernameKey: nil,
				passwordKey: []byte("pass"),
			},
			expected: false,
		},
		{
			name:     "nil password value",
			credType: credentials.TypeGit,
			repoURL:  "https://github.com/example/repository.git",
			data: map[string][]byte{
				usernameKey: []byte("user"),
				passwordKey: nil,
			},
			expected: false,
		},
	}

	provider := &CredentialProvider{}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			supports, err := provider.Supports(
				t.Context(),
				credentials.Request{
					Type:    test.credType,
					RepoURL: test.repoURL,
					Data:    test.data,
				},
			)
			require.NoError(t, err)
			require.Equal(t, test.expected, supports)
		})
	}
}

func TestCredentialProvider_GetCredentials(t *testing.T) {
	tests := []struct {
		name       string
		credType   credentials.Type
		repoURL    string
		data       map[string][]byte
		assertions func(t *testing.T, creds *credentials.Credentials, err error)
	}{
		{
			name:     "empty data",
			credType: credentials.TypeGit,
			repoURL:  "https://github.com/example/repository.git",
			data:     map[string][]byte{},
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.Nil(t, creds)
			},
		},
		{
			name:     "username only",
			credType: credentials.TypeGit,
			repoURL:  "https://github.com/example/repository.git",
			data: map[string][]byte{
				usernameKey: []byte("user"),
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.Nil(t, creds)
			},
		},
		{
			name:     "password only",
			credType: credentials.TypeGit,
			repoURL:  "https://github.com/example/repository.git",
			data: map[string][]byte{
				passwordKey: []byte("pass"),
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.Nil(t, creds)
			},
		},
		{
			name:     "username and password",
			credType: credentials.TypeGit,
			repoURL:  "https://github.com/example/repository.git",
			data: map[string][]byte{
				usernameKey: []byte("user"),
				passwordKey: []byte("pass"),
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, "user", creds.Username)
				assert.Equal(t, "pass", creds.Password)
			},
		},
		{
			name:     "empty username and password strings",
			credType: credentials.TypeGit,
			repoURL:  "https://github.com/example/repository.git",
			data: map[string][]byte{
				usernameKey: []byte(""),
				passwordKey: []byte(""),
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.Nil(t, creds)
			},
		},
	}

	provider := &CredentialProvider{}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			creds, err := provider.GetCredentials(
				t.Context(),
				credentials.Request{
					Type:    test.credType,
					RepoURL: test.repoURL,
					Data:    test.data,
				},
			)
			test.assertions(t, creds, err)
		})
	}
}
