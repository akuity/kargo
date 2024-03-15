package image

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/distribution/distribution/v3"
	"github.com/distribution/distribution/v3/manifest/manifestlist"
	"github.com/distribution/distribution/v3/manifest/ocischema"
	"github.com/distribution/distribution/v3/manifest/schema1" // nolint: staticcheck
	"github.com/distribution/distribution/v3/manifest/schema2"
	"github.com/distribution/distribution/v3/registry/client/auth/challenge"
	"github.com/opencontainers/go-digest"
	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/require"
)

func TestNewRepository(t *testing.T) {
	getChallengeManagerBackup := getChallengeManager
	getChallengeManager = func(
		string,
		http.RoundTripper,
	) (challenge.Manager, error) {
		return challenge.NewSimpleManager(), nil
	}
	defer func() {
		getChallengeManager = getChallengeManagerBackup
	}()

	client, err := newRepositoryClient("debian", false, nil)
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NotNil(t, client.registry)
	require.NotEmpty(t, client.image)
	require.NotNil(t, client.repo)
	// Make sure default behaviors are set
	require.NotNil(t, client.getImageByTagFn)
	require.NotNil(t, client.getImageByDigestFn)
	require.NotNil(t, client.getManifestByTagFn)
	require.NotNil(t, client.getManifestByDigestFn)
	require.NotNil(t, client.extractImageFromManifestFn)
	require.NotNil(t, client.extractImageFromV1ManifestFn)
	require.NotNil(t, client.extractImageFromV2ManifestFn)
	require.NotNil(t, client.extractImageFromOCIManifestFn)
	require.NotNil(t, client.extractImageFromCollectionFn)
	require.NotNil(t, client.getBlobFn)
}

func TestGetImageByTag(t *testing.T) {
	timePtr := func(t time.Time) *time.Time {
		return &t
	}

	testImage := Image{
		CreatedAt: timePtr(time.Now().UTC()),
	}
	testRegistry := &registry{}

	testCases := []struct {
		name       string
		tag        string
		client     *repositoryClient
		assertions func(*testing.T, *Image, error)
	}{
		{
			name: "error getting manifest for tag",
			tag:  "fake-tag",
			client: &repositoryClient{
				registry: testRegistry,
				getManifestByTagFn: func(
					context.Context,
					string,
				) (distribution.Manifest, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *Image, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error retrieving manifest")
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name: "error extracting image from manifest",
			tag:  "fake-tag",
			client: &repositoryClient{
				registry: testRegistry,
				getManifestByTagFn: func(
					context.Context,
					string,
				) (distribution.Manifest, error) {
					return &ocischema.DeserializedManifest{}, nil
				},
				extractImageFromManifestFn: func(
					context.Context,
					distribution.Manifest,
					*platformConstraint,
				) (*Image, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *Image, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error extracting image from manifest",
				)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name: "success",
			tag:  "fake-tag",
			client: &repositoryClient{
				image:    "fake-image",
				registry: testRegistry,
				getManifestByTagFn: func(
					context.Context,
					string,
				) (distribution.Manifest, error) {
					return &ocischema.DeserializedManifest{}, nil
				},
				extractImageFromManifestFn: func(
					context.Context,
					distribution.Manifest,
					*platformConstraint,
				) (*Image, error) {
					return &testImage, nil
				},
			},
			assertions: func(t *testing.T, image *Image, err error) {
				require.NoError(t, err)
				require.Equal(t, testImage, *image)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			image, err := testCase.client.getImageByTag(
				context.Background(),
				testCase.tag,
				nil,
			)
			testCase.assertions(t, image, err)
		})
	}
}

func TestGetImageByDigest(t *testing.T) {
	timePtr := func(t time.Time) *time.Time {
		return &t
	}

	testImage := Image{
		CreatedAt: timePtr(time.Now().UTC()),
	}
	testRegistry := &registry{
		imageCache: cache.New(0, 0),
	}
	const testCachedDigest = "fake-cached-digest"
	testRegistry.imageCache.Set(
		testCachedDigest,
		testImage,
		cache.DefaultExpiration,
	)

	testCases := []struct {
		name       string
		digest     digest.Digest
		client     *repositoryClient
		assertions func(*testing.T, *Image, error)
	}{
		{
			name:   "cache hit",
			digest: testCachedDigest,
			client: &repositoryClient{
				registry: testRegistry,
			},
			assertions: func(t *testing.T, image *Image, err error) {
				require.NoError(t, err)
				require.Equal(t, testImage, *image)
			},
		},
		{
			name:   "error getting manifest for digest",
			digest: "fake-digest",
			client: &repositoryClient{
				registry: testRegistry,
				getManifestByDigestFn: func(
					context.Context,
					digest.Digest,
				) (distribution.Manifest, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *Image, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error retrieving manifest")
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name:   "error extracting image from manifest",
			digest: "fake-digest",
			client: &repositoryClient{
				registry: testRegistry,
				getManifestByDigestFn: func(
					context.Context,
					digest.Digest,
				) (distribution.Manifest, error) {
					return &ocischema.DeserializedManifest{}, nil
				},
				extractImageFromManifestFn: func(
					context.Context,
					distribution.Manifest,
					*platformConstraint,
				) (*Image, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *Image, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error extracting image from manifest",
				)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name:   "success",
			digest: "fake-tag",
			client: &repositoryClient{
				image:    "fake-image",
				registry: testRegistry,
				getManifestByDigestFn: func(
					context.Context,
					digest.Digest,
				) (distribution.Manifest, error) {
					return &ocischema.DeserializedManifest{}, nil
				},
				extractImageFromManifestFn: func(
					context.Context,
					distribution.Manifest,
					*platformConstraint,
				) (*Image, error) {
					return &testImage, nil
				},
			},
			assertions: func(t *testing.T, image *Image, err error) {
				require.NoError(t, err)
				require.Equal(t, testImage, *image)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			image, err := testCase.client.getImageByDigest(
				context.Background(),
				testCase.digest,
				nil,
			)
			testCase.assertions(t, image, err)
		})
	}
}

func TestExtractImageFromManifest(t *testing.T) {
	timePtr := func(t time.Time) *time.Time {
		return &t
	}

	testImage := Image{
		CreatedAt: timePtr(time.Now().UTC()),
	}

	testCases := []struct {
		name     string
		manifest distribution.Manifest
		client   *repositoryClient
	}{
		{
			name:     "V1 manifest",
			manifest: &schema1.SignedManifest{}, // nolint: staticcheck
			client: &repositoryClient{
				extractImageFromV1ManifestFn: func(
					*schema1.SignedManifest, // nolint: staticcheck
					*platformConstraint,
				) (*Image, error) {
					return &testImage, nil
				},
			},
		},
		{
			name:     "V2 manifest",
			manifest: &schema2.DeserializedManifest{},
			client: &repositoryClient{
				extractImageFromV2ManifestFn: func(
					context.Context,
					*schema2.DeserializedManifest,
					*platformConstraint,
				) (*Image, error) {
					return &testImage, nil
				},
			},
		},
		{
			name:     "OCI manifest",
			manifest: &ocischema.DeserializedManifest{},
			client: &repositoryClient{
				extractImageFromOCIManifestFn: func(
					context.Context,
					*ocischema.DeserializedManifest,
					*platformConstraint,
				) (*Image, error) {
					return &testImage, nil
				},
			},
		},
		{
			name:     "manifest list",
			manifest: &manifestlist.DeserializedManifestList{},
			client: &repositoryClient{
				extractImageFromCollectionFn: func(
					context.Context,
					distribution.Manifest,
					*platformConstraint,
				) (*Image, error) {
					return &testImage, nil
				},
			},
		},
		{
			name:     "image index",
			manifest: &ocischema.DeserializedImageIndex{},
			client: &repositoryClient{
				extractImageFromCollectionFn: func(
					context.Context,
					distribution.Manifest,
					*platformConstraint,
				) (*Image, error) {
					return &testImage, nil
				},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			image, err := testCase.client.extractImageFromManifest(
				context.Background(),
				testCase.manifest,
				nil,
			)
			require.NoError(t, err)
			require.NotNil(t, image)
			require.Equal(t, testImage, *image)
		})
	}
}

func TestExtractImageFromV1Manifest(t *testing.T) {
	testTime := time.Now().UTC()
	testTimeStr := testTime.Format(time.RFC3339Nano)
	testCases := []struct {
		name       string
		platform   *platformConstraint
		manifest   *schema1.SignedManifest // nolint: staticcheck
		client     *repositoryClient
		assertions func(*testing.T, *Image, error)
	}{
		{
			name:     "manifest has no history",
			manifest: &schema1.SignedManifest{}, // nolint: staticcheck
			client:   &repositoryClient{},
			assertions: func(t *testing.T, _ *Image, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"no history information found in V1 manifest",
				)
			},
		},
		{
			name: "error umarshaling blob",
			// nolint: staticcheck
			manifest: &schema1.SignedManifest{
				Manifest: schema1.Manifest{
					History: []schema1.History{
						{
							V1Compatibility: "junk",
						},
					},
				},
			},
			assertions: func(t *testing.T, _ *Image, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error unmarshaling V1 manifest")
			},
		},
		{
			name: "platform does not match",
			platform: &platformConstraint{
				os:   "linux",
				arch: "amd64",
			},
			// nolint: staticcheck
			manifest: &schema1.SignedManifest{
				Manifest: schema1.Manifest{
					History: []schema1.History{
						{
							V1Compatibility: `{"os": "linux", "architecture": "arm64"}`,
						},
					},
				},
			},
			assertions: func(t *testing.T, image *Image, err error) {
				require.NoError(t, err)
				require.Nil(t, image)
			},
		},
		{
			name: "error parsing timestamp",
			// nolint: staticcheck
			manifest: &schema1.SignedManifest{
				Manifest: schema1.Manifest{
					History: []schema1.History{
						{
							V1Compatibility: `{"created": "junk"}`,
						},
					},
				},
			},
			assertions: func(t *testing.T, _ *Image, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error parsing createdAt timestamp")
			},
		},
		{
			name: "success",
			// nolint: staticcheck
			manifest: &schema1.SignedManifest{
				Manifest: schema1.Manifest{
					History: []schema1.History{
						{
							V1Compatibility: `{"os": "linux", "architecture": "amd64", "created": "` + testTimeStr + `"}`,
						},
					},
				},
			},
			client: &repositoryClient{},
			assertions: func(t *testing.T, image *Image, err error) {
				require.NoError(t, err)
				require.NotNil(t, image)
				require.NotNil(t, image.CreatedAt)
				require.Equal(t, testTime, *image.CreatedAt)
				require.NotNil(t, image.Digest)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			image, err := testCase.client.extractImageFromV1Manifest(
				testCase.manifest,
				testCase.platform,
			)
			testCase.assertions(t, image, err)
		})
	}
}

func TestExtractImageFromV2Manifest(t *testing.T) {
	testTime := time.Now().UTC()
	testTimeStr := testTime.Format(time.RFC3339Nano)
	testCases := []struct {
		name       string
		platform   *platformConstraint
		client     *repositoryClient
		assertions func(*testing.T, *Image, error)
	}{
		{
			name: "error fetching blob",
			client: &repositoryClient{
				getBlobFn: func(context.Context, digest.Digest) ([]byte, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *Image, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error fetching blob")
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name: "error unmarshaling blob",
			client: &repositoryClient{
				getBlobFn: func(context.Context, digest.Digest) ([]byte, error) {
					return []byte("junk"), nil
				},
			},
			assertions: func(t *testing.T, _ *Image, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error unmarshaling blob")
			},
		},
		{
			name: "platform does not match",
			platform: &platformConstraint{
				os:   "linux",
				arch: "amd64",
			},
			client: &repositoryClient{
				getBlobFn: func(context.Context, digest.Digest) ([]byte, error) {
					return []byte(`{"os": "linux", "architecture": "arm64"}`), nil
				},
			},
			assertions: func(t *testing.T, image *Image, err error) {
				require.NoError(t, err)
				require.Nil(t, image)
			},
		},
		{
			name: "error parsing timestamp",
			client: &repositoryClient{
				getBlobFn: func(context.Context, digest.Digest) ([]byte, error) {
					return []byte(`{"created": "junk"}`), nil
				},
			},
			assertions: func(t *testing.T, _ *Image, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error parsing createdAt timestamp")
			},
		},
		{
			name: "success",
			client: &repositoryClient{
				getBlobFn: func(context.Context, digest.Digest) ([]byte, error) {
					return []byte(
						`{"os": "linux", "architecture": "amd64", "created": "` + testTimeStr + `"}`,
					), nil
				},
			},
			assertions: func(t *testing.T, image *Image, err error) {
				require.NoError(t, err)
				require.NotNil(t, image)
				require.NotNil(t, image.CreatedAt)
				require.Equal(t, testTime, *image.CreatedAt)
				require.NotNil(t, image.Digest)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			image, err := testCase.client.extractImageFromV2Manifest(
				context.Background(),
				&schema2.DeserializedManifest{},
				testCase.platform,
			)
			testCase.assertions(t, image, err)
		})
	}
}

func TestExtractImageFromOCIManifest(t *testing.T) {
	testTime := time.Now().UTC()
	testTimeStr := testTime.Format(time.RFC3339Nano)
	testCases := []struct {
		name       string
		platform   *platformConstraint
		client     *repositoryClient
		assertions func(*testing.T, *Image, error)
	}{
		{
			name: "error fetching blob",
			client: &repositoryClient{
				getBlobFn: func(context.Context, digest.Digest) ([]byte, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *Image, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error fetching blob")
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name: "error unmarshaling blob",
			client: &repositoryClient{
				getBlobFn: func(context.Context, digest.Digest) ([]byte, error) {
					return []byte("junk"), nil
				},
			},
			assertions: func(t *testing.T, _ *Image, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error unmarshaling blob")
			},
		},
		{
			name: "doesn't look like an image",
			client: &repositoryClient{
				getBlobFn: func(context.Context, digest.Digest) ([]byte, error) {
					return []byte(`{"os": "", "architecture": ""}`), nil
				},
			},
			assertions: func(t *testing.T, image *Image, err error) {
				require.NoError(t, err)
				require.Nil(t, image)
			},
		},
		{
			name: "platform does not match",
			platform: &platformConstraint{
				os:   "linux",
				arch: "amd64",
			},
			client: &repositoryClient{
				getBlobFn: func(context.Context, digest.Digest) ([]byte, error) {
					return []byte(`{"os": "linux", "architecture": "arm64"}`), nil
				},
			},
			assertions: func(t *testing.T, image *Image, err error) {
				require.NoError(t, err)
				require.Nil(t, image)
			},
		},
		{
			name: "error parsing timestamp",
			client: &repositoryClient{
				getBlobFn: func(context.Context, digest.Digest) ([]byte, error) {
					return []byte(`{"os": "linux", "architecture": "arm64", "created": "junk"}`), nil
				},
			},
			assertions: func(t *testing.T, _ *Image, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error parsing createdAt timestamp")
			},
		},
		{
			name: "success",
			client: &repositoryClient{
				getBlobFn: func(context.Context, digest.Digest) ([]byte, error) {
					return []byte(
						`{"os": "linux", "architecture": "amd64", "created": "` + testTimeStr + `"}`,
					), nil
				},
			},
			assertions: func(t *testing.T, image *Image, err error) {
				require.NoError(t, err)
				require.NotNil(t, image)
				require.NotNil(t, image.CreatedAt)
				require.Equal(t, testTime, *image.CreatedAt)
				require.NotNil(t, image.Digest)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			image, err := testCase.client.extractImageFromOCIManifest(
				context.Background(),
				&ocischema.DeserializedManifest{},
				testCase.platform,
			)
			testCase.assertions(t, image, err)
		})
	}
}

func TestExtractImageFromCollection(t *testing.T) {
	timePtr := func(t time.Time) *time.Time {
		return &t
	}

	testNow := time.Now().UTC()

	testCases := []struct {
		name       string
		collection distribution.Manifest
		platform   *platformConstraint
		client     *repositoryClient
		assertions func(*testing.T, *Image, error)
	}{
		{
			name:       "empty V2 manifest list or OCI index",
			collection: &manifestlist.DeserializedManifestList{},
			client:     &repositoryClient{},
			assertions: func(t *testing.T, _ *Image, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "empty V2 manifest list or OCI index")
			},
		},
		{
			name: "with platform constraint -- no refs matched",
			collection: &manifestlist.DeserializedManifestList{
				ManifestList: manifestlist.ManifestList{
					Manifests: []manifestlist.ManifestDescriptor{
						{
							Platform: manifestlist.PlatformSpec{
								OS:           "linux",
								Architecture: "arm64",
							},
						},
					},
				},
			},
			platform: &platformConstraint{
				os:   "linux",
				arch: "amd64",
			},
			client: &repositoryClient{},
			assertions: func(t *testing.T, image *Image, err error) {
				require.NoError(t, err)
				require.Nil(t, image)
			},
		},
		{
			name: "with platform constraint -- too many refs matched",
			collection: &manifestlist.DeserializedManifestList{
				ManifestList: manifestlist.ManifestList{
					Manifests: []manifestlist.ManifestDescriptor{
						{
							Platform: manifestlist.PlatformSpec{
								OS:           "linux",
								Architecture: "amd64",
							},
						},
						{
							Platform: manifestlist.PlatformSpec{
								OS:           "linux",
								Architecture: "amd64",
							},
						},
					},
				},
			},
			platform: &platformConstraint{
				os:   "linux",
				arch: "amd64",
			},
			client: &repositoryClient{},
			assertions: func(t *testing.T, _ *Image, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"expected only one reference to match platform",
				)
			},
		},
		{
			name: "with platform constraint -- error getting image by digest",
			collection: &manifestlist.DeserializedManifestList{
				ManifestList: manifestlist.ManifestList{
					Manifests: []manifestlist.ManifestDescriptor{
						{
							Platform: manifestlist.PlatformSpec{
								OS:           "linux",
								Architecture: "amd64",
							},
						},
					},
				},
			},
			platform: &platformConstraint{
				os:   "linux",
				arch: "amd64",
			},
			client: &repositoryClient{
				getImageByDigestFn: func(
					context.Context,
					digest.Digest,
					*platformConstraint,
				) (*Image, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *Image, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error getting image from manifest",
				)
			},
		},
		{
			name: "with platform constraint -- no image found",
			collection: &manifestlist.DeserializedManifestList{
				ManifestList: manifestlist.ManifestList{
					Manifests: []manifestlist.ManifestDescriptor{
						{
							Platform: manifestlist.PlatformSpec{
								OS:           "linux",
								Architecture: "amd64",
							},
						},
					},
				},
			},
			platform: &platformConstraint{
				os:   "linux",
				arch: "amd64",
			},
			client: &repositoryClient{
				getImageByDigestFn: func(
					context.Context,
					digest.Digest,
					*platformConstraint,
				) (*Image, error) {
					return nil, nil
				},
			},
			assertions: func(t *testing.T, _ *Image, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "expected manifest for digest")
				require.Contains(t, err.Error(), "to match platform")
			},
		},
		{
			name: "with platform constraint -- success",
			collection: &manifestlist.DeserializedManifestList{
				ManifestList: manifestlist.ManifestList{
					Manifests: []manifestlist.ManifestDescriptor{
						{
							Platform: manifestlist.PlatformSpec{
								OS:           "linux",
								Architecture: "amd64",
							},
						},
					},
				},
			},
			platform: &platformConstraint{
				os:   "linux",
				arch: "amd64",
			},
			client: &repositoryClient{
				getImageByDigestFn: func(
					context.Context,
					digest.Digest,
					*platformConstraint,
				) (*Image, error) {
					return &Image{
						CreatedAt: timePtr(testNow),
					}, nil
				},
			},
			assertions: func(t *testing.T, image *Image, err error) {
				require.NoError(t, err)
				require.NotNil(t, image)
				require.NotNil(t, image.CreatedAt)
				require.Equal(t, testNow, *image.CreatedAt)
			},
		},
		{
			name: "without platform constraint -- error getting image by digest",
			collection: &manifestlist.DeserializedManifestList{
				ManifestList: manifestlist.ManifestList{
					Manifests: []manifestlist.ManifestDescriptor{
						{
							Platform: manifestlist.PlatformSpec{
								OS:           "linux",
								Architecture: "amd64",
							},
						},
					},
				},
			},
			client: &repositoryClient{
				getImageByDigestFn: func(
					context.Context,
					digest.Digest,
					*platformConstraint,
				) (*Image, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *Image, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error getting image from manifest",
				)
			},
		},
		{
			name: "without platform constraint -- no image found",
			collection: &manifestlist.DeserializedManifestList{
				ManifestList: manifestlist.ManifestList{
					Manifests: []manifestlist.ManifestDescriptor{
						{
							Platform: manifestlist.PlatformSpec{
								OS:           "linux",
								Architecture: "amd64",
							},
						},
					},
				},
			},
			client: &repositoryClient{
				getImageByDigestFn: func(
					context.Context,
					digest.Digest,
					*platformConstraint,
				) (*Image, error) {
					return nil, nil
				},
			},
			assertions: func(t *testing.T, _ *Image, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "found no image for manifest")
			},
		},
		{
			name: "without platform constraint -- success",
			collection: &manifestlist.DeserializedManifestList{
				ManifestList: manifestlist.ManifestList{
					Manifests: []manifestlist.ManifestDescriptor{
						{
							Platform: manifestlist.PlatformSpec{
								OS:           "linux",
								Architecture: "amd64",
							},
						},
					},
				},
			},
			client: &repositoryClient{
				getImageByDigestFn: func(
					context.Context,
					digest.Digest,
					*platformConstraint,
				) (*Image, error) {
					return &Image{
						CreatedAt: &testNow,
					}, nil
				},
			},
			assertions: func(t *testing.T, image *Image, err error) {
				require.NoError(t, err)
				require.NotNil(t, image)
				require.NotNil(t, image.CreatedAt)
				require.Equal(t, testNow, *image.CreatedAt)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			image, err := testCase.client.extractImageFromCollection(
				context.Background(),
				testCase.collection,
				testCase.platform,
			)
			testCase.assertions(t, image, err)
		})
	}
}
