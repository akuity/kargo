//go:build dockerhub
// +build dockerhub

package image

import (
	"context"
	"testing"

	"github.com/Masterminds/semver"
	log "github.com/sirupsen/logrus"
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

func TestSelectTagDockerHub(t *testing.T) {
	const debianRepo = "debian"
	const platform = "linux/amd64"

	ctx := context.Background()
	logger := logging.LoggerFromContext(ctx)
	logger.Logger.SetLevel(log.TraceLevel)
	ctx = logging.ContextWithLogger(ctx, logger)

	t.Run("digest strategy miss", func(t *testing.T) {
		s, err := NewTagSelector(
			debianRepo,
			TagSelectionStrategyDigest,
			&TagSelectorOptions{
				Constraint: "fake-constraint",
				Creds:      getDockerHubCreds(),
			},
		)
		require.NoError(t, err)

		tag, err := s.SelectTag(ctx)
		require.NoError(t, err)
		require.Nil(t, tag)

	})

	t.Run("digest strategy success", func(t *testing.T) {
		s, err := NewTagSelector(
			debianRepo,
			TagSelectionStrategyDigest,
			&TagSelectorOptions{
				Constraint: "bookworm",
				Creds:      getDockerHubCreds(),
			},
		)
		require.NoError(t, err)

		tag, err := s.SelectTag(ctx)
		require.NoError(t, err)

		require.NotNil(t, tag)
		require.Equal(t, "bookworm", tag.Name)
		require.NotEmpty(t, tag.Digest)
		require.NotNil(t, tag.CreatedAt)
	})

	t.Run("digest strategy miss with platform constraint", func(t *testing.T) {
		s, err := NewTagSelector(
			debianRepo,
			TagSelectionStrategyDigest,
			&TagSelectorOptions{
				Constraint: "bookworm",
				Platform:   "linux/made-up-arch",
				Creds:      getDockerHubCreds(),
			},
		)
		require.NoError(t, err)

		tag, err := s.SelectTag(ctx)
		require.NoError(t, err)
		require.Nil(t, tag)
	})

	t.Run("digest strategy success with platform constraint", func(t *testing.T) {
		s, err := NewTagSelector(
			debianRepo,
			TagSelectionStrategyDigest,
			&TagSelectorOptions{
				Constraint: "bookworm",
				Platform:   platform,
				Creds:      getDockerHubCreds(),
			},
		)
		require.NoError(t, err)

		tag, err := s.SelectTag(ctx)
		require.NoError(t, err)

		require.NotNil(t, tag)
		require.Equal(t, "bookworm", tag.Name)
		require.NotEmpty(t, tag.Digest)
		require.NotNil(t, tag.CreatedAt)
	})

	t.Run("lexical strategy miss", func(t *testing.T) {
		s, err := NewTagSelector(
			debianRepo,
			TagSelectionStrategyLexical,
			&TagSelectorOptions{
				AllowRegex: "nothing-matches-this",
				Creds:      getDockerHubCreds(),
			},
		)
		require.NoError(t, err)

		tag, err := s.SelectTag(ctx)
		require.NoError(t, err)
		require.Nil(t, tag)
	})

	t.Run("lexical strategy success", func(t *testing.T) {
		s, err := NewTagSelector(
			debianRepo,
			TagSelectionStrategyLexical,
			&TagSelectorOptions{
				Creds: getDockerHubCreds(),
			},
		)
		require.NoError(t, err)

		tag, err := s.SelectTag(ctx)
		require.NoError(t, err)

		require.NotNil(t, tag)
		// So far, this is lexically the last Debian release
		require.Contains(t, tag.Name, "wheezy")
		require.NotEmpty(t, tag.Digest)
		require.NotNil(t, tag.CreatedAt)
	})

	t.Run("lexical strategy miss with platform constraint", func(t *testing.T) {
		s, err := NewTagSelector(
			debianRepo,
			TagSelectionStrategyLexical,
			&TagSelectorOptions{
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

		tag, err := s.SelectTag(ctx)
		require.NoError(t, err)
		require.Nil(t, tag)
	})

	t.Run("lexical strategy success with platform constraint", func(t *testing.T) {
		s, err := NewTagSelector(
			debianRepo,
			TagSelectionStrategyLexical,
			&TagSelectorOptions{
				AllowRegex: "^jessie",
				Platform:   platform,
				Creds:      getDockerHubCreds(),
			},
		)
		require.NoError(t, err)

		tag, err := s.SelectTag(ctx)
		require.NoError(t, err)

		require.NotNil(t, tag)
		require.Contains(t, tag.Name, "jessie")
		require.NotEmpty(t, tag.Digest)
		require.NotNil(t, tag.CreatedAt)
	})

	t.Run("newest build strategy miss", func(t *testing.T) {
		s, err := NewTagSelector(
			debianRepo,
			TagSelectionStrategyNewestBuild,
			&TagSelectorOptions{
				AllowRegex: "nothing-matches-this",
				Creds:      getDockerHubCreds(),
			},
		)
		require.NoError(t, err)

		tag, err := s.SelectTag(ctx)
		require.NoError(t, err)
		require.Nil(t, tag)
	})

	t.Run("newest build strategy success", func(t *testing.T) {
		s, err := NewTagSelector(
			debianRepo,
			TagSelectionStrategyNewestBuild,
			&TagSelectorOptions{
				AllowRegex: `^bookworm-202310\d\d$`,
				Creds:      getDockerHubCreds(),
			},
		)
		require.NoError(t, err)

		tag, err := s.SelectTag(ctx)
		require.NoError(t, err)

		require.NotNil(t, tag)
		require.Contains(t, tag.Name, "bookworm-202310")
		require.NotEmpty(t, tag.Digest)
		require.NotNil(t, tag.CreatedAt)
	})

	t.Run("newest build strategy miss with platform constraint", func(t *testing.T) {
		s, err := NewTagSelector(
			debianRepo,
			TagSelectionStrategyNewestBuild,
			&TagSelectorOptions{
				AllowRegex: `^bookworm-202310\d\d$`,
				Platform:   "linux/made-up-arch",
				Creds:      getDockerHubCreds(),
			},
		)
		require.NoError(t, err)

		tag, err := s.SelectTag(ctx)
		require.NoError(t, err)
		require.Nil(t, tag)
	})

	t.Run("newest build strategy success with platform constraint", func(t *testing.T) {
		s, err := NewTagSelector(
			debianRepo,
			TagSelectionStrategyNewestBuild,
			&TagSelectorOptions{
				AllowRegex: `^bookworm-202310\d\d$`,
				Platform:   platform,
				Creds:      getDockerHubCreds(),
			},
		)
		require.NoError(t, err)

		tag, err := s.SelectTag(ctx)
		require.NoError(t, err)

		require.NotNil(t, tag)
		require.Contains(t, tag.Name, "bookworm-202310")
		require.NotEmpty(t, tag.Digest)
		require.NotNil(t, tag.CreatedAt)
	})

	t.Run("semver strategy miss", func(t *testing.T) {
		s, err := NewTagSelector(
			debianRepo,
			TagSelectionStrategySemVer,
			&TagSelectorOptions{
				Constraint: "^99.0",
				Creds:      getDockerHubCreds(),
			},
		)
		require.NoError(t, err)

		tag, err := s.SelectTag(ctx)
		require.NoError(t, err)
		require.Nil(t, tag)
	})

	t.Run("semver strategy success", func(t *testing.T) {
		s, err := NewTagSelector(
			debianRepo,
			TagSelectionStrategySemVer,
			&TagSelectorOptions{
				Constraint: "^12.0",
				Creds:      getDockerHubCreds(),
			},
		)
		require.NoError(t, err)

		tag, err := s.SelectTag(ctx)
		require.NoError(t, err)

		require.NotNil(t, tag)
		semVer, err := semver.NewVersion(tag.Name)
		require.NoError(t, err)
		min := semver.MustParse("12.0.0")
		require.True(t, semVer.GreaterThan(min) || semVer.Equal(min))
		require.True(t, semVer.LessThan(semver.MustParse("13.0.0")))
		require.NotEmpty(t, tag.Digest)
		require.NotNil(t, tag.CreatedAt)
	})

	t.Run("semver strategy miss with platform constraint", func(t *testing.T) {
		s, err := NewTagSelector(
			debianRepo,
			TagSelectionStrategySemVer,
			&TagSelectorOptions{
				Constraint: "^12.0",
				Platform:   "linux/made-up-arch",
				Creds:      getDockerHubCreds(),
			},
		)
		require.NoError(t, err)

		tag, err := s.SelectTag(ctx)
		require.NoError(t, err)
		require.Nil(t, tag)
	})

	t.Run("semver strategy success with platform constraint", func(t *testing.T) {
		s, err := NewTagSelector(
			debianRepo,
			TagSelectionStrategySemVer,
			&TagSelectorOptions{
				Constraint: "^12.0",
				Platform:   platform,
				Creds:      getDockerHubCreds(),
			},
		)
		require.NoError(t, err)

		tag, err := s.SelectTag(ctx)
		require.NoError(t, err)

		require.NotNil(t, tag)
		semVer, err := semver.NewVersion(tag.Name)
		require.NoError(t, err)
		min := semver.MustParse("12.0.0")
		require.True(t, semVer.GreaterThan(min) || semVer.Equal(min))
		require.True(t, semVer.LessThan(semver.MustParse("13.0.0")))
		require.NotEmpty(t, tag.Digest)
		require.NotNil(t, tag.CreatedAt)
	})
}
