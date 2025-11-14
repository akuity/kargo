//go:build dockerhub

package image

import (
	"context"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

// All test cases in this file are integration tests that rely on Docker Hub.
// You're very likely to get rate-limited executing these tests, unless you're a
// paying Docker customer, so they're disabled by default.
//
// To use your Docker credentials, set env vars:
// - DOCKER_HUB_USERNAME
// - DOCKER_HUB_PASSWORD (personal access token)

func TestSelectImageDockerHub(t *testing.T) {
	const debianRepo = "debian"
	const platform = "linux/amd64"

	logger, err := logging.NewLogger(logging.TraceLevel, logging.DefaultFormat)
	require.NoError(t, err)

	ctx := logging.ContextWithLogger(context.Background(), logger)

	t.Run("digest strategy miss", func(t *testing.T) {
		s, err := NewSelector(
			t.Context(),
			kargoapi.ImageSubscription{
				RepoURL:                debianRepo,
				ImageSelectionStrategy: kargoapi.ImageSelectionStrategyDigest,
				Constraint:             "fake-constraint",
				DiscoveryLimit:         1,
			},
			getDockerHubCreds(),
		)
		require.NoError(t, err)

		image, err := s.Select(ctx)
		require.NoError(t, err)
		require.Empty(t, image)
	})

	t.Run("digest strategy success", func(t *testing.T) {
		s, err := NewSelector(
			t.Context(),
			kargoapi.ImageSubscription{
				RepoURL:                debianRepo,
				ImageSelectionStrategy: kargoapi.ImageSelectionStrategyDigest,
				Constraint:             "bookworm",
				DiscoveryLimit:         1,
			},
			getDockerHubCreds(),
		)
		require.NoError(t, err)

		images, err := s.Select(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, images)
		require.Len(t, images, 1)

		image := images[0]
		require.Equal(t, "bookworm", image.Tag)
		require.NotEmpty(t, image.Digest)
		require.NotNil(t, image.CreatedAt)
	})

	t.Run("digest strategy miss with platform constraint", func(t *testing.T) {
		s, err := NewSelector(
			t.Context(),
			kargoapi.ImageSubscription{
				RepoURL:                debianRepo,
				ImageSelectionStrategy: kargoapi.ImageSelectionStrategyDigest,
				Constraint:             "bookworm",
				Platform:               "linux/made-up-arch",
				DiscoveryLimit:         1,
			},
			getDockerHubCreds(),
		)
		require.NoError(t, err)

		images, err := s.Select(ctx)
		require.NoError(t, err)
		require.Empty(t, images)
	})

	t.Run("digest strategy success with platform constraint", func(t *testing.T) {
		s, err := NewSelector(
			t.Context(),
			kargoapi.ImageSubscription{
				RepoURL:                debianRepo,
				ImageSelectionStrategy: kargoapi.ImageSelectionStrategyDigest,
				Constraint:             "bookworm",
				Platform:               platform,
				DiscoveryLimit:         1,
			},
			getDockerHubCreds(),
		)
		require.NoError(t, err)

		images, err := s.Select(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, images)

		image := images[0]
		require.Equal(t, "bookworm", image.Tag)
		require.NotEmpty(t, image.Digest)
		require.NotNil(t, image.CreatedAt)
	})

	t.Run("lexical strategy miss", func(t *testing.T) {
		s, err := NewSelector(
			t.Context(),
			kargoapi.ImageSubscription{
				RepoURL:                debianRepo,
				ImageSelectionStrategy: kargoapi.ImageSelectionStrategyLexical,
				AllowTags:              "nothing-matches-this",
				DiscoveryLimit:         1,
			},
			getDockerHubCreds(),
		)
		require.NoError(t, err)

		images, err := s.Select(ctx)
		require.NoError(t, err)
		require.Empty(t, images)
	})

	t.Run("lexical strategy success", func(t *testing.T) {
		s, err := NewSelector(
			t.Context(),
			kargoapi.ImageSubscription{
				RepoURL:                debianRepo,
				ImageSelectionStrategy: kargoapi.ImageSelectionStrategyLexical,
				DiscoveryLimit:         1,
			},
			getDockerHubCreds(),
		)
		require.NoError(t, err)

		images, err := s.Select(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, images)

		image := images[0]
		// So far, this is lexically the last Debian release
		require.Contains(t, image.Tag, "wheezy")
		require.NotEmpty(t, image.Digest)
		require.NotNil(t, image.CreatedAt)
	})

	t.Run("lexical strategy miss with platform constraint", func(t *testing.T) {
		s, err := NewSelector(
			t.Context(),
			kargoapi.ImageSubscription{
				RepoURL:                debianRepo,
				ImageSelectionStrategy: kargoapi.ImageSelectionStrategyLexical,
				// Note: If we go older than jessie, we don't seem to get the correct
				// digest, but jessie is ancient, so for now I am chalking it up to
				// something having to do with the evolution of the Docker Hub API over
				// time.
				AllowTags:      "^jessie",
				Platform:       "linux/made-up-arch",
				DiscoveryLimit: 1,
			},
			getDockerHubCreds(),
		)
		require.NoError(t, err)

		images, err := s.Select(ctx)
		require.NoError(t, err)
		require.Empty(t, images)
	})

	t.Run("lexical strategy success with platform constraint", func(t *testing.T) {
		s, err := NewSelector(
			t.Context(),
			kargoapi.ImageSubscription{
				RepoURL:                debianRepo,
				ImageSelectionStrategy: kargoapi.ImageSelectionStrategyLexical,
				AllowTags:              "^jessie",
				Platform:               platform,
				DiscoveryLimit:         1,
			},
			getDockerHubCreds(),
		)
		require.NoError(t, err)

		images, err := s.Select(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, images)

		image := images[0]
		require.Contains(t, image.Tag, "jessie")
		require.NotEmpty(t, image.Digest)
		require.NotNil(t, image.CreatedAt)
	})

	t.Run("newest build strategy miss", func(t *testing.T) {
		s, err := NewSelector(
			t.Context(),
			kargoapi.ImageSubscription{
				RepoURL:                debianRepo,
				ImageSelectionStrategy: kargoapi.ImageSelectionStrategyNewestBuild,
				AllowTags:              "nothing-matches-this",
				DiscoveryLimit:         1,
			},
			getDockerHubCreds(),
		)
		require.NoError(t, err)

		images, err := s.Select(ctx)
		require.NoError(t, err)
		require.Empty(t, images)
	})

	t.Run("newest build strategy success", func(t *testing.T) {
		s, err := NewSelector(
			t.Context(),
			kargoapi.ImageSubscription{
				RepoURL:                debianRepo,
				ImageSelectionStrategy: kargoapi.ImageSelectionStrategyNewestBuild,
				AllowTags:              `^bookworm-202310\d\d$`,
				DiscoveryLimit:         1,
			},
			getDockerHubCreds(),
		)
		require.NoError(t, err)

		images, err := s.Select(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, images)

		image := images[0]
		require.Contains(t, image.Tag, "bookworm-202310")
		require.NotEmpty(t, image.Digest)
		require.NotNil(t, image.CreatedAt)
	})

	t.Run("newest build strategy miss with platform constraint", func(t *testing.T) {
		s, err := NewSelector(
			t.Context(),
			kargoapi.ImageSubscription{
				RepoURL:                debianRepo,
				ImageSelectionStrategy: kargoapi.ImageSelectionStrategyNewestBuild,
				AllowTags:              `^bookworm-202310\d\d$`,
				Platform:               "linux/made-up-arch",
				DiscoveryLimit:         1,
			},
			getDockerHubCreds(),
		)
		require.NoError(t, err)

		images, err := s.Select(ctx)
		require.NoError(t, err)
		require.Nil(t, images)
	})

	t.Run("newest build strategy success with platform constraint", func(t *testing.T) {
		s, err := NewSelector(
			t.Context(),
			kargoapi.ImageSubscription{
				RepoURL:                debianRepo,
				ImageSelectionStrategy: kargoapi.ImageSelectionStrategyNewestBuild,
				AllowTags:              `^bookworm-202310\d\d$`,
				Platform:               platform,
				DiscoveryLimit:         1,
			},
			getDockerHubCreds(),
		)
		require.NoError(t, err)

		images, err := s.Select(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, images)

		image := images[0]
		require.Contains(t, image.Tag, "bookworm-202310")
		require.NotEmpty(t, image.Digest)
		require.NotNil(t, image.CreatedAt)
	})

	t.Run("semver strategy miss", func(t *testing.T) {
		s, err := NewSelector(
			t.Context(),
			kargoapi.ImageSubscription{
				RepoURL:                debianRepo,
				ImageSelectionStrategy: kargoapi.ImageSelectionStrategySemVer,
				Constraint:             "^99.0",
				DiscoveryLimit:         1,
			},
			getDockerHubCreds(),
		)
		require.NoError(t, err)

		images, err := s.Select(ctx)
		require.NoError(t, err)
		require.Empty(t, images)
	})

	t.Run("semver strategy success", func(t *testing.T) {
		s, err := NewSelector(
			t.Context(),
			kargoapi.ImageSubscription{
				RepoURL:                debianRepo,
				ImageSelectionStrategy: kargoapi.ImageSelectionStrategySemVer,
				Constraint:             "^12.0",
				DiscoveryLimit:         1,
			},
			getDockerHubCreds(),
		)
		require.NoError(t, err)

		images, err := s.Select(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, images)

		image := images[0]
		semVer, err := semver.NewVersion(image.Tag)
		require.NoError(t, err)
		minimum := semver.MustParse("12.0.0")
		require.True(t, semVer.GreaterThan(minimum) || semVer.Equal(minimum))
		require.True(t, semVer.LessThan(semver.MustParse("13.0.0")))
		require.NotEmpty(t, image.Digest)
		require.NotNil(t, image.CreatedAt)
	})

	t.Run("semver strategy miss with platform constraint", func(t *testing.T) {
		s, err := NewSelector(
			t.Context(),
			kargoapi.ImageSubscription{
				RepoURL:                debianRepo,
				ImageSelectionStrategy: kargoapi.ImageSelectionStrategySemVer,
				Constraint:             "^12.0",
				Platform:               "linux/made-up-arch",
				DiscoveryLimit:         1,
			},
			getDockerHubCreds(),
		)
		require.NoError(t, err)

		images, err := s.Select(ctx)
		require.NoError(t, err)
		require.Empty(t, images)
	})

	t.Run("semver strategy success with platform constraint", func(t *testing.T) {
		s, err := NewSelector(
			t.Context(),
			kargoapi.ImageSubscription{
				RepoURL:                debianRepo,
				ImageSelectionStrategy: kargoapi.ImageSelectionStrategySemVer,
				Constraint:             "^12.0",
				Platform:               platform,
				DiscoveryLimit:         1,
			},
			getDockerHubCreds(),
		)
		require.NoError(t, err)

		images, err := s.Select(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, images)

		image := images[0]
		semVer, err := semver.NewVersion(image.Tag)
		require.NoError(t, err)
		minimum := semver.MustParse("12.0.0")
		require.True(t, semVer.GreaterThan(minimum) || semVer.Equal(minimum))
		require.True(t, semVer.LessThan(semver.MustParse("13.0.0")))
		require.NotEmpty(t, image.Digest)
		require.NotNil(t, image.CreatedAt)
	})
}
