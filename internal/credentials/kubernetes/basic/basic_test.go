package basic

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/akuity/kargo/internal/credentials"
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
			name:     "ssh private key only",
			credType: credentials.TypeGit,
			repoURL:  "https://github.com/example/repository.git",
			data: map[string][]byte{
				sshPrivateKey: []byte("key-data"),
			},
			expected: true,
		},
		{
			name:     "all credentials",
			credType: credentials.TypeGit,
			repoURL:  "https://github.com/example/repository.git",
			data: map[string][]byte{
				usernameKey:   []byte("user"),
				passwordKey:   []byte("pass"),
				sshPrivateKey: []byte("key-data"),
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
		{
			name:     "nil ssh private key value",
			credType: credentials.TypeGit,
			repoURL:  "https://github.com/example/repository.git",
			data: map[string][]byte{
				sshPrivateKey: nil,
			},
			expected: false,
		},
	}

	provider := &CredentialProvider{}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := provider.Supports(test.credType, test.repoURL, test.data)
			assert.Equal(t, test.expected, result)
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
				assert.Empty(t, creds.SSHPrivateKey, "SSHPrivateKey should be empty")
			},
		},
		{
			name:     "ssh private key only",
			credType: credentials.TypeGit,
			repoURL:  "https://github.com/example/repository.git",
			data: map[string][]byte{
				sshPrivateKey: []byte("key-data"),
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Empty(t, creds.Username, "Username should be empty")
				assert.Empty(t, creds.Password, "Password should be empty")
				assert.Equal(t, "key-data", creds.SSHPrivateKey)
			},
		},
		{
			name:     "all credentials",
			credType: credentials.TypeGit,
			repoURL:  "https://github.com/example/repository.git",
			data: map[string][]byte{
				usernameKey:   []byte("user"),
				passwordKey:   []byte("pass"),
				sshPrivateKey: []byte("key-data"),
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, "user", creds.Username)
				assert.Equal(t, "pass", creds.Password)
				assert.Equal(t, "key-data", creds.SSHPrivateKey)
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
		{
			name:     "empty ssh private key string",
			credType: credentials.TypeGit,
			repoURL:  "https://github.com/example/repository.git",
			data: map[string][]byte{
				sshPrivateKey: []byte(""),
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
			creds, err := provider.GetCredentials(context.Background(), "", test.credType, test.repoURL, test.data)
			test.assertions(t, creds, err)
		})
	}
}
