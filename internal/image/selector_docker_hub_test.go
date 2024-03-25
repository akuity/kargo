//go:build dockerhub
// +build dockerhub

package image

import (
	"context"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/internal/logging"
)

// All test cases in this file are integration tests that rely on Docker Hub.
// You're very likely to get rate-limited executing these tests, unless you're a
// paying Docker customer, so they're disabled by default.
//
// To use your Docker credentials, set env vars:
// - DOCKER_HUB_USERNAME
// - DOCKER_HUB_USERNAME (personal access token)

func TestSelectImageDockerHub(t *testing.T) {
	const debianRepo = "debian"
	const platform = "linux/amd64"

	ctx := context.Background()
	logger := logging.LoggerFromContext(ctx)
	logging.SetLevel(logging.LevelTrace)
	ctx = logging.ContextWithLogger(ctx, logger)

	t.Run("digest strategy miss", func(t *testing.T) {
		s, err := NewSelector(
			debianRepo,
			SelectionStrategyDigest,
			&SelectorOptions{
				Constraint: "fake-constraint",
				Creds:      getDockerHubCreds(),
			},
		)
		require.NoError(t, err)

		image, err := s.Select(ctx)
		require.NoError(t, err)
		require.Nil(t, image)

	})

	t.Run("digest strategy success", func(t *testing.T) {
		s, err := NewSelector(
			debianRepo,
			SelectionStrategyDigest,
			&SelectorOptions{
				Constraint: "bookworm",
				Creds:      getDockerHubCreds(),
			},
		)
		require.NoError(t, err)

		image, err := s.Select(ctx)
		require.NoError(t, err)

		require.NotNil(t, image)
		require.Equal(t, "bookworm", image.Tag)
		require.NotEmpty(t, image.Digest)
		require.NotNil(t, image.CreatedAt)
	})

	t.Run("digest strategy miss with platform constraint", func(t *testing.T) {
		s, err := NewSelector(
			debianRepo,
			SelectionStrategyDigest,
			&SelectorOptions{
				Constraint: "bookworm",
				Platform:   "linux/made-up-arch",
				Creds:      getDockerHubCreds(),
			},
		)
		require.NoError(t, err)

		image, err := s.Select(ctx)
		require.NoError(t, err)
		require.Nil(t, image)
	})

	t.Run("digest strategy success with platform constraint", func(t *testing.T) {
		s, err := NewSelector(
			debianRepo,
			SelectionStrategyDigest,
			&SelectorOptions{
				Constraint: "bookworm",
				Platform:   platform,
				Creds:      getDockerHubCreds(),
			},
		)
		require.NoError(t, err)

		image, err := s.Select(ctx)
		require.NoError(t, err)

		require.NotNil(t, image)
		require.Equal(t, "bookworm", image.Tag)
		require.NotEmpty(t, image.Digest)
		require.NotNil(t, image.CreatedAt)
	})

	t.Run("lexical strategy miss", func(t *testing.T) {
		s, err := NewSelector(
			debianRepo,
			SelectionStrategyLexical,
			&SelectorOptions{
				AllowRegex: "nothing-matches-this",
				Creds:      getDockerHubCreds(),
			},
		)
		require.NoError(t, err)

		image, err := s.Select(ctx)
		require.NoError(t, err)
		require.Nil(t, image)
	})

	t.Run("lexical strategy success", func(t *testing.T) {
		s, err := NewSelector(
			debianRepo,
			SelectionStrategyLexical,
			&SelectorOptions{
				Creds: getDockerHubCreds(),
			},
		)
		require.NoError(t, err)

		image, err := s.Select(ctx)
		require.NoError(t, err)

		require.NotNil(t, image)
		// So far, this is lexically the last Debian release
		require.Contains(t, image.Tag, "wheezy")
		require.NotEmpty(t, image.Digest)
		require.NotNil(t, image.CreatedAt)
	})

	t.Run("lexical strategy miss with platform constraint", func(t *testing.T) {
		s, err := NewSelector(
			debianRepo,
			SelectionStrategyLexical,
			&SelectorOptions{
				// Note: If we go older than jessie, we don't seem to get the correct
				// digest, but jessie is ancient, so for now I am chalking it up to
				// something having to do with the evolution of the Docker Hub API over
				// time.
				AllowRegex: "^jessie",
				Platform:   "linux/made-up-arch",
				Creds:      getDockerHubCreds(),
			},
		)
		require.NoError(t, err)

		image, err := s.Select(ctx)
		require.NoError(t, err)
		require.Nil(t, image)
	})

	t.Run("lexical strategy success with platform constraint", func(t *testing.T) {
		s, err := NewSelector(
			debianRepo,
			SelectionStrategyLexical,
			&SelectorOptions{
				AllowRegex: "^jessie",
				Platform:   platform,
				Creds:      getDockerHubCreds(),
			},
		)
		require.NoError(t, err)

		image, err := s.Select(ctx)
		require.NoError(t, err)

		require.NotNil(t, image)
		require.Contains(t, image.Tag, "jessie")
		require.NotEmpty(t, image.Digest)
		require.NotNil(t, image.CreatedAt)
	})

	t.Run("newest build strategy miss", func(t *testing.T) {
		s, err := NewSelector(
			debianRepo,
			SelectionStrategyNewestBuild,
			&SelectorOptions{
				AllowRegex: "nothing-matches-this",
				Creds:      getDockerHubCreds(),
			},
		)
		require.NoError(t, err)

		image, err := s.Select(ctx)
		require.NoError(t, err)
		require.Nil(t, image)
	})

	t.Run("newest build strategy success", func(t *testing.T) {
		s, err := NewSelector(
			debianRepo,
			SelectionStrategyNewestBuild,
			&SelectorOptions{
				AllowRegex: `^bookworm-202310\d\d$`,
				Creds:      getDockerHubCreds(),
			},
		)
		require.NoError(t, err)

		image, err := s.Select(ctx)
		require.NoError(t, err)

		require.NotNil(t, image)
		require.Contains(t, image.Tag, "bookworm-202310")
		require.NotEmpty(t, image.Digest)
		require.NotNil(t, image.CreatedAt)
	})

	t.Run("newest build strategy miss with platform constraint", func(t *testing.T) {
		s, err := NewSelector(
			debianRepo,
			SelectionStrategyNewestBuild,
			&SelectorOptions{
				AllowRegex: `^bookworm-202310\d\d$`,
				Platform:   "linux/made-up-arch",
				Creds:      getDockerHubCreds(),
			},
		)
		require.NoError(t, err)

		image, err := s.Select(ctx)
		require.NoError(t, err)
		require.Nil(t, image)
	})

	t.Run("newest build strategy success with platform constraint", func(t *testing.T) {
		s, err := NewSelector(
			debianRepo,
			SelectionStrategyNewestBuild,
			&SelectorOptions{
				AllowRegex: `^bookworm-202310\d\d$`,
				Platform:   platform,
				Creds:      getDockerHubCreds(),
			},
		)
		require.NoError(t, err)

		image, err := s.Select(ctx)
		require.NoError(t, err)

		require.NotNil(t, image)
		require.Contains(t, image.Tag, "bookworm-202310")
		require.NotEmpty(t, image.Digest)
		require.NotNil(t, image.CreatedAt)
	})

	t.Run("semver strategy miss", func(t *testing.T) {
		s, err := NewSelector(
			debianRepo,
			SelectionStrategySemVer,
			&SelectorOptions{
				Constraint: "^99.0",
				Creds:      getDockerHubCreds(),
			},
		)
		require.NoError(t, err)

		image, err := s.Select(ctx)
		require.NoError(t, err)
		require.Nil(t, image)
	})

	t.Run("semver strategy success", func(t *testing.T) {
		s, err := NewSelector(
			debianRepo,
			SelectionStrategySemVer,
			&SelectorOptions{
				Constraint: "^12.0",
				Creds:      getDockerHubCreds(),
			},
		)
		require.NoError(t, err)

		image, err := s.Select(ctx)
		require.NoError(t, err)

		require.NotNil(t, image)
		semVer, err := semver.NewVersion(image.Tag)
		require.NoError(t, err)
		min := semver.MustParse("12.0.0")
		require.True(t, semVer.GreaterThan(min) || semVer.Equal(min))
		require.True(t, semVer.LessThan(semver.MustParse("13.0.0")))
		require.NotEmpty(t, image.Digest)
		require.NotNil(t, image.CreatedAt)
	})

	t.Run("semver strategy miss with platform constraint", func(t *testing.T) {
		s, err := NewSelector(
			debianRepo,
			SelectionStrategySemVer,
			&SelectorOptions{
				Constraint: "^12.0",
				Platform:   "linux/made-up-arch",
				Creds:      getDockerHubCreds(),
			},
		)
		require.NoError(t, err)

		image, err := s.Select(ctx)
		require.NoError(t, err)
		require.Nil(t, image)
	})

	t.Run("semver strategy success with platform constraint", func(t *testing.T) {
		s, err := NewSelector(
			debianRepo,
			SelectionStrategySemVer,
			&SelectorOptions{
				Constraint: "^12.0",
				Platform:   platform,
				Creds:      getDockerHubCreds(),
			},
		)
		require.NoError(t, err)

		image, err := s.Select(ctx)
		require.NoError(t, err)

		require.NotNil(t, image)
		semVer, err := semver.NewVersion(image.Tag)
		require.NoError(t, err)
		min := semver.MustParse("12.0.0")
		require.True(t, semVer.GreaterThan(min) || semVer.Equal(min))
		require.True(t, semVer.LessThan(semver.MustParse("13.0.0")))
		require.NotEmpty(t, image.Digest)
		require.NotNil(t, image.CreatedAt)
	})
}
