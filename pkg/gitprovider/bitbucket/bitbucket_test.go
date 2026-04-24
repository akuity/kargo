package bitbucket

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/gitprovider"
	"github.com/akuity/kargo/pkg/gitprovider/bitbucket/cloud"
)

func TestNewProvider(t *testing.T) {
	t.Run("cloud URL returns cloud provider", func(t *testing.T) {
		p, err := NewProvider("https://bitbucket.org/owner/repo", &gitprovider.Options{Token: "token"})
		require.NoError(t, err)
		require.NotNil(t, p)
	})

	t.Run("non-cloud URL returns datacenter provider", func(t *testing.T) {
		p, err := NewProvider("https://bitbucket.example.com/projects/PROJ/repos/repo", nil)
		require.NoError(t, err)
		require.NotNil(t, p)
	})

	t.Run("invalid URL returns error", func(t *testing.T) {
		p, err := NewProvider("://invalid-url", nil)
		require.Error(t, err)
		require.Nil(t, p)
	})
}

func Test_registration(t *testing.T) {
	t.Run("predicate matches bitbucket.org URL", func(t *testing.T) {
		assert.True(t, registration.Predicate("https://bitbucket.org/owner/repo"))
	})

	t.Run("predicate doesn't match other URLs", func(t *testing.T) {
		assert.False(t, registration.Predicate("https://github.com/owner/repo"))
	})

	t.Run("predicate doesn't match self-hosted URLs", func(t *testing.T) {
		assert.False(t, registration.Predicate("https://bitbucket.example.com/projects/PROJ/repos/repo"))
	})

	t.Run("predicate handles invalid URLs", func(t *testing.T) {
		assert.False(t, registration.Predicate("://invalid-url"))
	})

	t.Run("NewProvider factory works for cloud URL", func(t *testing.T) {
		p, err := registration.NewProvider("https://bitbucket.org/owner/repo", nil)
		assert.NoError(t, err)
		assert.NotNil(t, p)
	})
}

func Test_cloudHostConst(t *testing.T) {
	assert.Equal(t, "bitbucket.org", cloud.Host)
}
