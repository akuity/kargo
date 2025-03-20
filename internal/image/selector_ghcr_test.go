//go:build ghcr
// +build ghcr

package image

import (
	"context"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/internal/logging"
)

// All test cases in this file are integration tests that rely on ghcr.io. You
// may get rate-limited executing these tests, so they're disabled by default.

func TestSelectImageGHCR(t *testing.T) {
	const kargoRepo = "ghcr.io/akuity/kargo"
	const platform = "linux/amd64"

	ctx := logging.ContextWithLogger(
		context.Background(),
		logging.NewLogger(logging.TraceLevel),
	)

	t.Run("digest strategy", func(t *testing.T) {
		const constraint = "v0.1.0"
		s, err := NewSelector(
			kargoRepo,
			SelectionStrategyDigest,
			&SelectorOptions{
				Constraint:     constraint,
				DiscoveryLimit: 1,
			},
		)
		require.NoError(t, err)

		images, err := s.Select(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, images)

		image := images[0]
		require.Equal(t, constraint, image.Tag)
		require.NotEmpty(t, image.Digest)
		require.NotNil(t, image.CreatedAt)
	})

	t.Run("digest strategy with platform constraint", func(t *testing.T) {
		const constraint = "v0.1.0"
		s, err := NewSelector(
			kargoRepo,
			SelectionStrategyDigest,
			&SelectorOptions{
				Constraint:     constraint,
				Platform:       platform,
				DiscoveryLimit: 1,
			},
		)
		require.NoError(t, err)

		images, err := s.Select(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, images)

		image := images[0]
		require.Equal(t, constraint, image.Tag)
		require.NotEmpty(t, image.Digest)
		require.NotNil(t, image.CreatedAt)
	})

	t.Run("lexical strategy", func(t *testing.T) {
		s, err := NewSelector(
			kargoRepo,
			SelectionStrategyLexical,
			&SelectorOptions{
				DiscoveryLimit: 1,
			},
		)
		require.NoError(t, err)

		images, err := s.Select(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, images)

		image := images[0]
		require.NotEmpty(t, image.Digest)
		require.NotNil(t, image.CreatedAt)
	})

	t.Run("lexical strategy with platform constraint", func(t *testing.T) {
		s, err := NewSelector(
			kargoRepo,
			SelectionStrategyLexical,
			&SelectorOptions{
				Platform:       platform,
				DiscoveryLimit: 1,
			},
		)
		require.NoError(t, err)

		images, err := s.Select(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, images)

		image := images[0]
		require.NotEmpty(t, image.Digest)
		require.NotNil(t, image.CreatedAt)
	})

	t.Run("newest build strategy", func(t *testing.T) {
		s, err := NewSelector(
			kargoRepo,
			SelectionStrategyNewestBuild,
			&SelectorOptions{
				AllowRegex:     `^v0.1.0-rc.2\d$`,
				DiscoveryLimit: 1,
			},
		)
		require.NoError(t, err)

		images, err := s.Select(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, images)

		image := images[0]
		require.Equal(t, "v0.1.0-rc.24", image.Tag)
		require.NotEmpty(t, image.Digest)
		require.NotNil(t, image.CreatedAt)
	})

	t.Run("newest build strategy with platform constraint", func(t *testing.T) {
		s, err := NewSelector(
			kargoRepo,
			SelectionStrategyNewestBuild,
			&SelectorOptions{
				AllowRegex:     `^v0.1.0-rc.2\d$`,
				Platform:       platform,
				DiscoveryLimit: 1,
			},
		)
		require.NoError(t, err)

		images, err := s.Select(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, images)

		image := images[0]
		require.Equal(t, "v0.1.0-rc.24", image.Tag)
		require.NotEmpty(t, image.Digest)
		require.NotNil(t, image.CreatedAt)
	})

	t.Run("semver strategy", func(t *testing.T) {
		s, err := NewSelector(
			kargoRepo,
			SelectionStrategySemVer,
			&SelectorOptions{
				DiscoveryLimit: 1,
			},
		)
		require.NoError(t, err)

		images, err := s.Select(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, images)

		image := images[0]
		semVer, err := semver.NewVersion(image.Tag)
		require.NoError(t, err)
		minimum := semver.MustParse("0.1.0")
		require.True(t, semVer.GreaterThan(minimum) || semVer.Equal(minimum))
		require.True(t, semVer.LessThan(semver.MustParse("2.0.0")))
		require.NotEmpty(t, image.Digest)
		require.NotNil(t, image.CreatedAt)
	})

	t.Run("semver strategy with platform constraint", func(t *testing.T) {
		s, err := NewSelector(
			kargoRepo,
			SelectionStrategySemVer,
			&SelectorOptions{
				Platform:       platform,
				DiscoveryLimit: 1,
			},
		)
		require.NoError(t, err)

		images, err := s.Select(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, images)

		image := images[0]
		semVer, err := semver.NewVersion(image.Tag)
		require.NoError(t, err)
		minimum := semver.MustParse("0.1.0")
		require.True(t, semVer.GreaterThan(minimum) || semVer.Equal(minimum))
		require.True(t, semVer.LessThan(semver.MustParse("2.0.0")))
		require.NotEmpty(t, image.Digest)
		require.NotNil(t, image.CreatedAt)
	})

	t.Run("tolerance for non-image references", func(t *testing.T) {
		// Image lists or indices may contain non-image references. These could be
		// to things like attestations. This test verifies that we ignore such
		// references to avoid parsing errors.
		const tag = "unknown"
		s, err := NewSelector(
			"ghcr.io/akuity/kargo-test",
			SelectionStrategyDigest,
			&SelectorOptions{
				Constraint:     tag,
				DiscoveryLimit: 1,
			},
		)
		require.NoError(t, err)

		images, err := s.Select(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, images)

		image := images[0]
		require.Equal(t, tag, image.Tag)
		require.NotEmpty(t, image.Digest)
		require.NotNil(t, image.CreatedAt)
	})

	t.Run("nothing found", func(t *testing.T) {
		s, err := NewSelector(
			kargoRepo,
			SelectionStrategyDigest,
			&SelectorOptions{
				Constraint: "v0.1.0",
				// Nothing will match this
				Platform:       "linux/made-up-arch",
				DiscoveryLimit: 1,
			},
		)
		require.NoError(t, err)

		images, err := s.Select(ctx)
		require.NoError(t, err)
		require.Empty(t, images)
	})
}
