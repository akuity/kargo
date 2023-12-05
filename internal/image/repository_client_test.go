package image

import (
	"context"
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
	"github.com/pkg/errors"
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

	client, err := newRepositoryClient("debian", nil)
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NotNil(t, client.registry)
	require.NotEmpty(t, client.image)
	require.NotNil(t, client.repo)
	// Make sure default behaviors are set
	require.NotNil(t, client.getTagByNameFn)
	require.NotNil(t, client.getTagByDigestFn)
	require.NotNil(t, client.getManifestByTagNameFn)
	require.NotNil(t, client.getManifestByDigestFn)
	require.NotNil(t, client.extractTagFromManifestFn)
	require.NotNil(t, client.extractTagFromV1ManifestFn)
	require.NotNil(t, client.extractTagFromV2ManifestFn)
	require.NotNil(t, client.extractTagFromOCIManifestFn)
	require.NotNil(t, client.extractTagFromCollectionFn)
	require.NotNil(t, client.getBlobFn)
}

func TestGetTagByName(t *testing.T) {
	timePtr := func(t time.Time) *time.Time {
		return &t
	}

	testTag := Tag{
		CreatedAt: timePtr(time.Now().UTC()),
	}
	testRegistry := &registry{}

	testCases := []struct {
		name       string
		tag        string
		client     *repositoryClient
		assertions func(*Tag, error)
	}{
		{
			name: "error getting manifest for tag",
			tag:  "fake-tag",
			client: &repositoryClient{
				registry: testRegistry,
				getManifestByTagNameFn: func(
					context.Context,
					string,
				) (distribution.Manifest, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(_ *Tag, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error retrieving manifest")
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name: "error extracting tag from manifest",
			tag:  "fake-tag",
			client: &repositoryClient{
				registry: testRegistry,
				getManifestByTagNameFn: func(
					context.Context,
					string,
				) (distribution.Manifest, error) {
					return &ocischema.DeserializedManifest{}, nil
				},
				extractTagFromManifestFn: func(
					context.Context,
					distribution.Manifest,
					*platformConstraint,
				) (*Tag, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(_ *Tag, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error extracting tag from manifest",
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
				getManifestByTagNameFn: func(
					context.Context,
					string,
				) (distribution.Manifest, error) {
					return &ocischema.DeserializedManifest{}, nil
				},
				extractTagFromManifestFn: func(
					context.Context,
					distribution.Manifest,
					*platformConstraint,
				) (*Tag, error) {
					return &testTag, nil
				},
			},
			assertions: func(tag *Tag, err error) {
				require.NoError(t, err)
				require.Equal(t, testTag, *tag)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.client.getTagByName(
					context.Background(),
					testCase.tag,
					nil,
				),
			)
		})
	}
}

func TestGetTagByDigest(t *testing.T) {
	timePtr := func(t time.Time) *time.Time {
		return &t
	}

	testTag := Tag{
		CreatedAt: timePtr(time.Now().UTC()),
	}
	testRegistry := &registry{
		tagCache: cache.New(0, 0),
	}
	const testCachedDigest = "fake-cached-digest"
	testRegistry.tagCache.Set(
		testCachedDigest,
		testTag,
		cache.DefaultExpiration,
	)

	testCases := []struct {
		name       string
		digest     digest.Digest
		client     *repositoryClient
		assertions func(*Tag, error)
	}{
		{
			name:   "cache hit",
			digest: testCachedDigest,
			client: &repositoryClient{
				registry: testRegistry,
			},
			assertions: func(tag *Tag, err error) {
				require.NoError(t, err)
				require.Equal(t, testTag, *tag)
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
			assertions: func(_ *Tag, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error retrieving manifest")
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name:   "error extracting tag from manifest",
			digest: "fake-digest",
			client: &repositoryClient{
				registry: testRegistry,
				getManifestByDigestFn: func(
					context.Context,
					digest.Digest,
				) (distribution.Manifest, error) {
					return &ocischema.DeserializedManifest{}, nil
				},
				extractTagFromManifestFn: func(
					context.Context,
					distribution.Manifest,
					*platformConstraint,
				) (*Tag, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(_ *Tag, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error extracting tag from manifest",
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
				extractTagFromManifestFn: func(
					context.Context,
					distribution.Manifest,
					*platformConstraint,
				) (*Tag, error) {
					return &testTag, nil
				},
			},
			assertions: func(tag *Tag, err error) {
				require.NoError(t, err)
				require.Equal(t, testTag, *tag)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.client.getTagByDigest(
					context.Background(),
					testCase.digest,
					nil,
				),
			)
		})
	}
}

func TestExtractTagFromManifest(t *testing.T) {
	timePtr := func(t time.Time) *time.Time {
		return &t
	}

	testTag := Tag{
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
				extractTagFromV1ManifestFn: func(
					*schema1.SignedManifest, // nolint: staticcheck
					*platformConstraint,
				) (*Tag, error) {
					return &testTag, nil
				},
			},
		},
		{
			name:     "V2 manifest",
			manifest: &schema2.DeserializedManifest{},
			client: &repositoryClient{
				extractTagFromV2ManifestFn: func(
					context.Context,
					*schema2.DeserializedManifest,
					*platformConstraint,
				) (*Tag, error) {
					return &testTag, nil
				},
			},
		},
		{
			name:     "OCI manifest",
			manifest: &ocischema.DeserializedManifest{},
			client: &repositoryClient{
				extractTagFromOCIManifestFn: func(
					context.Context,
					*ocischema.DeserializedManifest,
					*platformConstraint,
				) (*Tag, error) {
					return &testTag, nil
				},
			},
		},
		{
			name:     "manifest list",
			manifest: &manifestlist.DeserializedManifestList{},
			client: &repositoryClient{
				extractTagFromCollectionFn: func(
					context.Context,
					distribution.Manifest,
					*platformConstraint,
				) (*Tag, error) {
					return &testTag, nil
				},
			},
		},
		{
			name:     "image index",
			manifest: &ocischema.DeserializedImageIndex{},
			client: &repositoryClient{
				extractTagFromCollectionFn: func(
					context.Context,
					distribution.Manifest,
					*platformConstraint,
				) (*Tag, error) {
					return &testTag, nil
				},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			tag, err := testCase.client.extractTagFromManifest(
				context.Background(),
				testCase.manifest,
				nil,
			)
			require.NoError(t, err)
			require.NotNil(t, tag)
			require.Equal(t, testTag, *tag)
		})
	}
}

func TestExtractTagFromV1Manifest(t *testing.T) {
	testTime := time.Now().UTC()
	testTimeStr := testTime.Format(time.RFC3339Nano)
	testCases := []struct {
		name       string
		platform   *platformConstraint
		manifest   *schema1.SignedManifest // nolint: staticcheck
		client     *repositoryClient
		assertions func(*Tag, error)
	}{
		{
			name:     "manifest has no history",
			manifest: &schema1.SignedManifest{}, // nolint: staticcheck
			client:   &repositoryClient{},
			assertions: func(_ *Tag, err error) {
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
			assertions: func(_ *Tag, err error) {
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
			assertions: func(tag *Tag, err error) {
				require.NoError(t, err)
				require.Nil(t, tag)
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
			assertions: func(_ *Tag, err error) {
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
			assertions: func(tag *Tag, err error) {
				require.NoError(t, err)
				require.NotNil(t, tag)
				require.NotNil(t, tag.CreatedAt)
				require.Equal(t, testTime, *tag.CreatedAt)
				require.NotNil(t, tag.Digest)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(testCase.client.extractTagFromV1Manifest(
				testCase.manifest,
				testCase.platform,
			),
			)
		})
	}
}

func TestExtractTagFromV2Manifest(t *testing.T) {
	testTime := time.Now().UTC()
	testTimeStr := testTime.Format(time.RFC3339Nano)
	testCases := []struct {
		name       string
		platform   *platformConstraint
		client     *repositoryClient
		assertions func(*Tag, error)
	}{
		{
			name: "error fetching blob",
			client: &repositoryClient{
				getBlobFn: func(context.Context, digest.Digest) ([]byte, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(_ *Tag, err error) {
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
			assertions: func(_ *Tag, err error) {
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
			assertions: func(tag *Tag, err error) {
				require.NoError(t, err)
				require.Nil(t, tag)
			},
		},
		{
			name: "error parsing timestamp",
			client: &repositoryClient{
				getBlobFn: func(context.Context, digest.Digest) ([]byte, error) {
					return []byte(`{"created": "junk"}`), nil
				},
			},
			assertions: func(_ *Tag, err error) {
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
			assertions: func(tag *Tag, err error) {
				require.NoError(t, err)
				require.NotNil(t, tag)
				require.NotNil(t, tag.CreatedAt)
				require.Equal(t, testTime, *tag.CreatedAt)
				require.NotNil(t, tag.Digest)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.client.extractTagFromV2Manifest(
					context.Background(),
					&schema2.DeserializedManifest{},
					testCase.platform,
				),
			)
		})
	}
}

func TestExtractTagFromOCIManifest(t *testing.T) {
	testTime := time.Now().UTC()
	testTimeStr := testTime.Format(time.RFC3339Nano)
	testCases := []struct {
		name       string
		platform   *platformConstraint
		client     *repositoryClient
		assertions func(*Tag, error)
	}{
		{
			name: "error fetching blob",
			client: &repositoryClient{
				getBlobFn: func(context.Context, digest.Digest) ([]byte, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(_ *Tag, err error) {
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
			assertions: func(_ *Tag, err error) {
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
			assertions: func(tag *Tag, err error) {
				require.NoError(t, err)
				require.Nil(t, tag)
			},
		},
		{
			name: "error parsing timestamp",
			client: &repositoryClient{
				getBlobFn: func(context.Context, digest.Digest) ([]byte, error) {
					return []byte(`{"created": "junk"}`), nil
				},
			},
			assertions: func(_ *Tag, err error) {
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
			assertions: func(tag *Tag, err error) {
				require.NoError(t, err)
				require.NotNil(t, tag)
				require.NotNil(t, tag.CreatedAt)
				require.Equal(t, testTime, *tag.CreatedAt)
				require.NotNil(t, tag.Digest)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.client.extractTagFromOCIManifest(
					context.Background(),
					&ocischema.DeserializedManifest{},
					testCase.platform,
				),
			)
		})
	}
}

func TestExtractTagFromCollection(t *testing.T) {
	timePtr := func(t time.Time) *time.Time {
		return &t
	}

	testNow := time.Now().UTC()

	testCases := []struct {
		name       string
		collection distribution.Manifest
		platform   *platformConstraint
		client     *repositoryClient
		assertions func(*Tag, error)
	}{
		{
			name:       "empty V2 manifest list or OCI index",
			collection: &manifestlist.DeserializedManifestList{},
			client:     &repositoryClient{},
			assertions: func(_ *Tag, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "empty V2 manifest list or OCI index")
			},
		},
		{
			name: "with platform constraint -- no refs matched",
			collection: &manifestlist.DeserializedManifestList{
				ManifestList: manifestlist.ManifestList{
					Manifests: []manifestlist.ManifestDescriptor{{}},
				},
			},
			platform: &platformConstraint{
				os:   "linux",
				arch: "amd64",
			},
			client: &repositoryClient{},
			assertions: func(tag *Tag, err error) {
				require.NoError(t, err)
				require.Nil(t, tag)
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
			assertions: func(_ *Tag, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"expected only one reference to match platform",
				)
			},
		},
		{
			name: "with platform constraint -- error getting tag by digest",
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
				getTagByDigestFn: func(
					context.Context,
					digest.Digest,
					*platformConstraint,
				) (*Tag, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(_ *Tag, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error getting tag from manifest",
				)
			},
		},
		{
			name: "with platform constraint -- no tag found",
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
				getTagByDigestFn: func(
					context.Context,
					digest.Digest,
					*platformConstraint,
				) (*Tag, error) {
					return nil, nil
				},
			},
			assertions: func(_ *Tag, err error) {
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
				getTagByDigestFn: func(
					context.Context,
					digest.Digest,
					*platformConstraint,
				) (*Tag, error) {
					return &Tag{
						CreatedAt: timePtr(testNow),
					}, nil
				},
			},
			assertions: func(tag *Tag, err error) {
				require.NoError(t, err)
				require.NotNil(t, tag)
				require.NotNil(t, tag.CreatedAt)
				require.Equal(t, testNow, *tag.CreatedAt)
			},
		},
		{
			name: "without platform constraint -- error getting tag by digest",
			collection: &manifestlist.DeserializedManifestList{
				ManifestList: manifestlist.ManifestList{
					Manifests: []manifestlist.ManifestDescriptor{{}},
				},
			},
			client: &repositoryClient{
				getTagByDigestFn: func(
					context.Context,
					digest.Digest,
					*platformConstraint,
				) (*Tag, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(_ *Tag, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error getting tag from manifest",
				)
			},
		},
		{
			name: "without platform constraint -- no tag found",
			collection: &manifestlist.DeserializedManifestList{
				ManifestList: manifestlist.ManifestList{
					Manifests: []manifestlist.ManifestDescriptor{{}},
				},
			},
			client: &repositoryClient{
				getTagByDigestFn: func(
					context.Context,
					digest.Digest,
					*platformConstraint,
				) (*Tag, error) {
					return nil, nil
				},
			},
			assertions: func(_ *Tag, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "found no tag for manifest")
			},
		},
		{
			name: "without platform constraint -- success",
			collection: &manifestlist.DeserializedManifestList{
				ManifestList: manifestlist.ManifestList{
					Manifests: []manifestlist.ManifestDescriptor{{}},
				},
			},
			client: &repositoryClient{
				getTagByDigestFn: func(
					context.Context,
					digest.Digest,
					*platformConstraint,
				) (*Tag, error) {
					return &Tag{
						CreatedAt: &testNow,
					}, nil
				},
			},
			assertions: func(tag *Tag, err error) {
				require.NoError(t, err)
				require.NotNil(t, tag)
				require.NotNil(t, tag.CreatedAt)
				require.Equal(t, testNow, *tag.CreatedAt)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.client.extractTagFromCollection(
					context.Background(),
					testCase.collection,
					testCase.platform,
				),
			)
		})
	}
}
