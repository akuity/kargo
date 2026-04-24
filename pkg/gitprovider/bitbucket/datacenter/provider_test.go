package datacenter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_registration(t *testing.T) {
	t.Run("predicate matches self-hosted bitbucket hostname", func(t *testing.T) {
		assert.True(t, registration.Predicate("https://bitbucket.example.com/projects/PROJ/repos/repo"))
	})

	t.Run("predicate matches subdomain of bitbucket", func(t *testing.T) {
		assert.True(t, registration.Predicate("https://git.bitbucket.corp.io/projects/PROJ/repos/repo"))
	})

	t.Run("predicate does not match bitbucket.org (Cloud)", func(t *testing.T) {
		assert.False(t, registration.Predicate("https://bitbucket.org/owner/repo"))
	})

	t.Run("predicate does not match other providers", func(t *testing.T) {
		assert.False(t, registration.Predicate("https://github.com/owner/repo"))
	})

	t.Run("predicate handles invalid URLs", func(t *testing.T) {
		assert.False(t, registration.Predicate("://invalid-url"))
	})

	t.Run("NewProvider factory works", func(t *testing.T) {
		p, err := registration.NewProvider("https://bitbucket.example.com/projects/PROJ/repos/repo", nil)
		assert.NoError(t, err)
		assert.NotNil(t, p)
	})
}
