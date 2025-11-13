package ssh

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
			name:     "ssh private specified",
			credType: credentials.TypeGit,
			repoURL:  "https://github.com/example/repository.git",
			data:     map[string][]byte{sshPrivateKey: []byte("key-data")},
			expected: true,
		},
		{
			name:     "nil ssh private key value",
			credType: credentials.TypeGit,
			repoURL:  "https://github.com/example/repository.git",
			data:     map[string][]byte{sshPrivateKey: nil},
			expected: false,
		},
	}

	p := &CredentialProvider{}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			supports, err := p.Supports(
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
			name:     "ssh private key specified",
			credType: credentials.TypeGit,
			repoURL:  "https://github.com/example/repository.git",
			data:     map[string][]byte{sshPrivateKey: []byte("key-data")},
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, "key-data", creds.SSHPrivateKey)
			},
		},
		{
			name:     "empty ssh private key string",
			credType: credentials.TypeGit,
			repoURL:  "https://github.com/example/repository.git",
			data:     map[string][]byte{sshPrivateKey: []byte("")},
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
