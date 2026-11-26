package image

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestNewNewestBuildSelector(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.ImageSubscription
		assertions func(*testing.T, Selector, error)
	}{
		{
			name: "error building tag based selector",
			sub:  kargoapi.ImageSubscription{}, // No RepoURL
			assertions: func(t *testing.T, _ Selector, err error) {
				require.ErrorContains(t, err, "error building tag based selector")
			},
		},
		{
			name: "success",
			sub: kargoapi.ImageSubscription{
				RepoURL:    "example/image",
				Constraint: "latest",
			},
			assertions: func(t *testing.T, s Selector, err error) {
				require.NoError(t, err)
				l, ok := s.(*newestBuildSelector)
				require.True(t, ok)
				require.NotNil(t, l.tagBasedSelector)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			s, err := newNewestBuildSelector(testCase.sub, nil)
			testCase.assertions(t, s, err)
		})
	}
}

func Test_newestBuildSelector_sort(t *testing.T) {
	timePtr := func(t time.Time) *time.Time {
		return &t
	}

	now := time.Now().UTC()

	images := []image{
		{CreatedAt: &now},
		{CreatedAt: timePtr(now.Add(time.Hour))},
		{CreatedAt: timePtr(now.Add(5 * time.Hour))},
		{CreatedAt: timePtr(now.Add(24 * time.Hour))},
		{CreatedAt: timePtr(now.Add(8 * time.Hour))},
		{CreatedAt: timePtr(now.Add(2 * time.Hour))},
		{CreatedAt: timePtr(now.Add(7 * time.Hour))},
		{CreatedAt: timePtr(now.Add(3 * time.Hour))},
	}

	(&newestBuildSelector{}).sort(images)

	require.Equal(
		t,
		[]image{
			{CreatedAt: timePtr(now.Add(24 * time.Hour))},
			{CreatedAt: timePtr(now.Add(8 * time.Hour))},
			{CreatedAt: timePtr(now.Add(7 * time.Hour))},
			{CreatedAt: timePtr(now.Add(5 * time.Hour))},
			{CreatedAt: timePtr(now.Add(3 * time.Hour))},
			{CreatedAt: timePtr(now.Add(2 * time.Hour))},
			{CreatedAt: timePtr(now.Add(time.Hour))},
			{CreatedAt: &now},
		},
		images,
	)
}

func Test_newestBuildSelector_Select(t *testing.T) {
	older := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := older.Add(time.Hour)
	s, err := newNewestBuildSelector(
		kargoapi.ImageSubscription{
			RepoURL:          "example/image",
			ExpressionFilter: `annotations["approved"] == "true"`,
		},
		nil,
	)
	require.NoError(t, err)
	n, ok := s.(*newestBuildSelector)
	require.True(t, ok)
	withFakeImages(
		n.repoClient,
		image{
			Tag:         "approved",
			CreatedAt:   &older,
			Annotations: map[string]string{"approved": "true"},
		},
		image{Tag: "not-approved", CreatedAt: &newer},
	)
	imgs, err := s.Select(t.Context())
	require.NoError(t, err)
	require.Len(t, imgs, 1)
	require.Equal(t, "approved", imgs[0].Tag)
}

// Test_newestBuildSelector_Select_artifactOnlyIndex is a regression test for
// https://github.com/akuity/kargo/issues/8170. A tag whose descriptor is an OCI
// index containing only artifact (non-image) descriptors must be ignored, not
// allowed to fail discovery of every other tag in the repository.
func Test_newestBuildSelector_Select_artifactOnlyIndex(t *testing.T) {
	const (
		validTag    = "v1.0.0"
		artifactTag = "attestation-index"
	)
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	validDigest := v1.Hash{
		Algorithm: "sha256",
		Hex:       strings.Repeat("a", 64),
	}
	artifactDigest := v1.Hash{
		Algorithm: "sha256",
		Hex:       strings.Repeat("b", 64),
	}

	s, err := newNewestBuildSelector(
		kargoapi.ImageSubscription{RepoURL: "example/issue-8170"},
		nil,
	)
	require.NoError(t, err)
	n, ok := s.(*newestBuildSelector)
	require.True(t, ok)

	n.repoClient.remoteListFn = func(name.Repository, ...remote.Option) ([]string, error) {
		return []string{validTag, artifactTag}, nil
	}
	n.repoClient.remoteGetFn = func(
		ref name.Reference,
		_ ...remote.Option,
	) (*remote.Descriptor, error) {
		desc := &remote.Descriptor{
			Descriptor: v1.Descriptor{
				Digest:      validDigest,
				Annotations: map[string]string{"tag": ref.Identifier()},
			},
		}
		if ref.Identifier() == artifactTag {
			desc.MediaType = types.OCIImageIndex
			desc.Digest = artifactDigest
		}
		return desc, nil
	}

	// Serve the runnable tag from the fake image, but run the artifact tag
	// through the real descriptor/index handling under test.
	realGetImageFromRemoteDesc := n.repoClient.getImageFromRemoteDesc
	n.repoClient.getImageFromRemoteDescFn = func(
		ctx context.Context,
		desc *remote.Descriptor,
	) (*image, error) {
		if desc.Annotations["tag"] == artifactTag {
			return realGetImageFromRemoteDesc(ctx, desc)
		}
		return &image{
			Tag:       validTag,
			Digest:    validDigest.String(),
			CreatedAt: &createdAt,
			Platforms: []platform{{OS: "linux", Arch: "amd64"}},
		}, nil
	}

	// The index contains one descriptor with no platform and one with the
	// "unknown" platform used by artifact/attestation manifests. Neither is a
	// runnable image.
	artifactOnlyIndex := &mockImageIndex{
		indexManifest: &v1.IndexManifest{
			Manifests: []v1.Descriptor{
				{
					MediaType: types.OCIManifestSchema1,
					Digest:    artifactDigest,
				},
				{
					MediaType: types.OCIManifestSchema1,
					Digest:    artifactDigest,
					Platform: &v1.Platform{
						OS:           unknown,
						Architecture: unknown,
					},
				},
			},
		},
	}
	n.repoClient.getImageFromV1ImageIndexFn = func(
		ctx context.Context,
		digest string,
		_ v1.ImageIndex,
	) (*image, error) {
		return n.repoClient.getImageFromV1ImageIndex(ctx, digest, artifactOnlyIndex)
	}

	imgs, err := s.Select(t.Context())
	require.NoError(t, err)
	require.Len(t, imgs, 1)
	require.Equal(t, validTag, imgs[0].Tag)
}
