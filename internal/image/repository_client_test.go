package image

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

func TestNewRepository(t *testing.T) {
	client, err := newRepositoryClient("debian", false, nil)
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NotNil(t, client.registry)
	require.NotEmpty(t, client.repoURL)
	require.NotNil(t, client.repoRef)
	// Make sure default behaviors are set
	require.NotNil(t, client.getImageByTagFn)
	require.NotNil(t, client.getImageByDigestFn)
	require.NotNil(t, client.getImageFromRemoteDescFn)
	require.NotNil(t, client.getImageFromV1ImageIndexFn)
	require.NotNil(t, client.getImageFromV1ImageFn)
	require.NotNil(t, client.remoteListFn)
	require.NotNil(t, client.remoteGetFn)
}

func TestGetImageByTag(t *testing.T) {
	const testRepoURL = "fake-url"
	const testTag = "fake-tag"

	testRepoRef, err := name.ParseReference(testRepoURL)
	require.NoError(t, err)

	testImage := Image{
		Tag:       testTag,
		CreatedAt: ptr.To(time.Now().UTC()),
	}

	testCases := []struct {
		name       string
		client     *repositoryClient
		assertions func(*testing.T, *Image, error)
	}{
		{
			name: "error getting descriptor by tag",
			client: &repositoryClient{
				repoRef: testRepoRef,
				remoteGetFn: func(
					name.Reference,
					...remote.Option,
				) (*remote.Descriptor, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *Image, err error) {
				require.ErrorContains(t, err, "error getting image descriptor for tag")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error getting image from descriptor",
			client: &repositoryClient{
				repoRef: testRepoRef,
				remoteGetFn: func(
					name.Reference,
					...remote.Option,
				) (*remote.Descriptor, error) {
					return &remote.Descriptor{}, nil
				},
				getImageFromRemoteDescFn: func(
					context.Context,
					*remote.Descriptor,
					*platformConstraint,
				) (*Image, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *Image, err error) {
				require.ErrorContains(t, err, "error getting image from descriptor for tag")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "success",
			client: &repositoryClient{
				repoRef: testRepoRef,
				remoteGetFn: func(
					name.Reference,
					...remote.Option,
				) (*remote.Descriptor, error) {
					return &remote.Descriptor{}, nil
				},
				getImageFromRemoteDescFn: func(
					context.Context,
					*remote.Descriptor,
					*platformConstraint,
				) (*Image, error) {
					return &testImage, nil
				},
			},
			assertions: func(t *testing.T, img *Image, err error) {
				require.NoError(t, err)
				require.Equal(t, testImage, *img)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			img, err := testCase.client.getImageByTag(
				context.Background(),
				testTag,
				nil,
			)
			testCase.assertions(t, img, err)
		})
	}
}

func TestGetImageByDigest(t *testing.T) {
	const testRepoURL = "fake-url"
	const testDigest = "fake-digest"

	testRepoRef, err := name.ParseReference(testRepoURL)
	require.NoError(t, err)

	testImage := Image{
		Digest:    testDigest,
		CreatedAt: ptr.To(time.Now().UTC()),
	}

	testRegistry := &registry{
		imageCache: cache.New(0, 0),
	}
	testRegistry.imageCache.Set(
		testImage.Digest,
		testImage,
		cache.DefaultExpiration,
	)

	testCases := []struct {
		name       string
		client     *repositoryClient
		assertions func(*testing.T, *Image, error)
	}{
		{
			name: "cache hit",
			client: &repositoryClient{
				registry: testRegistry,
			},
			assertions: func(t *testing.T, img *Image, err error) {
				require.NoError(t, err)
				require.Equal(t, testImage, *img)
			},
		},
		{
			name: "error getting descriptor by digest",
			client: &repositoryClient{
				repoRef: testRepoRef,
				registry: &registry{
					imageCache: cache.New(30*time.Minute, time.Hour),
				},
				remoteGetFn: func(
					name.Reference, ...remote.Option,
				) (*remote.Descriptor, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *Image, err error) {
				require.ErrorContains(t, err, "error getting image descriptor for digest")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error getting image from descriptor",
			client: &repositoryClient{
				repoRef: testRepoRef,
				registry: &registry{
					imageCache: cache.New(30*time.Minute, time.Hour),
				},
				remoteGetFn: func(
					name.Reference, ...remote.Option,
				) (*remote.Descriptor, error) {
					return &remote.Descriptor{}, nil
				},
				getImageFromRemoteDescFn: func(
					context.Context, *remote.Descriptor, *platformConstraint,
				) (*Image, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *Image, err error) {
				require.ErrorContains(t, err, "error getting image from descriptor for digest")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "success",
			client: &repositoryClient{
				repoRef: testRepoRef,
				registry: &registry{
					imageCache: cache.New(30*time.Minute, time.Hour),
				},
				remoteGetFn: func(
					name.Reference, ...remote.Option,
				) (*remote.Descriptor, error) {
					return &remote.Descriptor{}, nil
				},
				getImageFromRemoteDescFn: func(
					context.Context,
					*remote.Descriptor,
					*platformConstraint,
				) (*Image, error) {
					return &testImage, nil
				},
			},
			assertions: func(t *testing.T, img *Image, err error) {
				require.NoError(t, err)
				require.Equal(t, testImage, *img)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			img, err := testCase.client.getImageByDigest(
				context.Background(),
				testDigest,
				nil,
			)
			testCase.assertions(t, img, err)
		})
	}
}

func TestGetImageFromRemoteDesc(t *testing.T) {
	testImage := Image{
		CreatedAt: ptr.To(time.Now().UTC()),
	}

	mediaTypes := []types.MediaType{
		types.OCIImageIndex,
		types.DockerManifestList,
		types.OCIManifestSchema1,
		types.DockerManifestSchema2,
	}

	testClient := &repositoryClient{
		getImageFromV1ImageIndexFn: func(
			context.Context, string, v1.ImageIndex, *platformConstraint,
		) (*Image, error) {
			return &testImage, nil
		},
		getImageFromV1ImageFn: func(
			string, v1.Image, *platformConstraint,
		) (*Image, error) {
			return &testImage, nil
		},
	}

	for _, mediaType := range mediaTypes {
		t.Run(string(mediaType), func(t *testing.T) {
			img, err := testClient.getImageFromRemoteDesc(
				context.Background(),
				&remote.Descriptor{
					Descriptor: v1.Descriptor{
						MediaType: mediaType,
					},
				},
				nil,
			)
			require.NoError(t, err)
			require.Equal(t, testImage, *img)
		})
	}
}

func TestImageFromV1ImageIndex(t *testing.T) {
	const testDigest = "fake-digest"

	testImage := Image{
		Digest:    testDigest,
		CreatedAt: ptr.To(time.Now().UTC()),
	}

	testCases := []struct {
		name       string
		idx        v1.ImageIndex
		platform   *platformConstraint
		client     *repositoryClient
		assertions func(*testing.T, *Image, error)
	}{
		{
			name: "empty list or index not supported",
			idx: &mockImageIndex{
				indexManifest: &v1.IndexManifest{
					Manifests: []v1.Descriptor{{}},
				},
			},
			client: &repositoryClient{},
			assertions: func(t *testing.T, _ *Image, err error) {
				require.ErrorContains(t, err, "empty V2 manifest list or OCI index is not supported")
			},
		},
		{
			name: "no refs match platform constraint",
			idx: &mockImageIndex{
				indexManifest: &v1.IndexManifest{
					Manifests: []v1.Descriptor{{
						Platform: &v1.Platform{
							OS:           "linux",
							Architecture: "amd64",
						},
					}},
				},
			},
			platform: &platformConstraint{
				os:   "linux",
				arch: "arm64",
			},
			client: &repositoryClient{},
			assertions: func(t *testing.T, img *Image, err error) {
				require.NoError(t, err)
				require.Nil(t, img)
			},
		},
		{
			name: "multiples refs match platform constraint",
			idx: &mockImageIndex{
				indexManifest: &v1.IndexManifest{
					Manifests: []v1.Descriptor{
						{
							Platform: &v1.Platform{
								OS:           "linux",
								Architecture: "amd64",
							},
						},
						{
							Platform: &v1.Platform{
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
				require.ErrorContains(t, err, "expected only one reference to match platform")
			},
		},
		{
			name: "with platform constraint, error getting image by digest",
			idx: &mockImageIndex{
				indexManifest: &v1.IndexManifest{
					Manifests: []v1.Descriptor{{
						Platform: &v1.Platform{
							OS:           "linux",
							Architecture: "amd64",
						},
					}},
				},
			},
			platform: &platformConstraint{
				os:   "linux",
				arch: "amd64",
			},
			client: &repositoryClient{
				getImageByDigestFn: func(
					context.Context, string, *platformConstraint,
				) (*Image, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *Image, err error) {
				require.ErrorContains(t, err, "error getting image with digest")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "with platform constraint, image found but doesn't match platform",
			idx: &mockImageIndex{
				indexManifest: &v1.IndexManifest{
					Manifests: []v1.Descriptor{{
						Platform: &v1.Platform{
							OS:           "linux",
							Architecture: "amd64",
						}},
					},
				},
			},
			platform: &platformConstraint{
				os:   "linux",
				arch: "amd64",
			},
			client: &repositoryClient{
				getImageByDigestFn: func(
					context.Context, string, *platformConstraint,
				) (*Image, error) {
					return nil, nil
				},
			},
			assertions: func(t *testing.T, _ *Image, err error) {
				require.ErrorContains(t, err, "expected manifest for digest")
				require.ErrorContains(t, err, "to match platform")
			},
		},
		{
			name: "with platform constraint, success",
			idx: &mockImageIndex{
				indexManifest: &v1.IndexManifest{
					Manifests: []v1.Descriptor{{
						Platform: &v1.Platform{
							OS:           "linux",
							Architecture: "amd64",
						}},
					},
				},
			},
			platform: &platformConstraint{
				os:   "linux",
				arch: "amd64",
			},
			client: &repositoryClient{
				getImageByDigestFn: func(
					context.Context, string, *platformConstraint,
				) (*Image, error) {
					return &testImage, nil
				},
			},
			assertions: func(t *testing.T, img *Image, err error) {
				require.NoError(t, err)
				require.Equal(t, testImage, *img)
			},
		},
		{
			name: "without platform constraint, error getting image by digest",
			idx: &mockImageIndex{
				indexManifest: &v1.IndexManifest{
					Manifests: []v1.Descriptor{{
						Platform: &v1.Platform{
							OS:           "linux",
							Architecture: "amd64",
						},
					}},
				},
			},
			client: &repositoryClient{
				getImageByDigestFn: func(
					context.Context, string, *platformConstraint,
				) (*Image, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *Image, err error) {
				require.ErrorContains(t, err, "error getting image with digest")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "without platform constraint, no image found",
			idx: &mockImageIndex{
				indexManifest: &v1.IndexManifest{
					Manifests: []v1.Descriptor{{
						Platform: &v1.Platform{
							OS:           "linux",
							Architecture: "amd64",
						},
					}},
				},
			},
			client: &repositoryClient{
				getImageByDigestFn: func(
					context.Context, string, *platformConstraint,
				) (*Image, error) {
					return nil, nil
				},
			},
			assertions: func(t *testing.T, _ *Image, err error) {
				require.ErrorContains(t, err, "found no image with digest")
			},
		},
		{
			name: "without platform constraint, success",
			idx: &mockImageIndex{
				indexManifest: &v1.IndexManifest{
					Manifests: []v1.Descriptor{{
						Platform: &v1.Platform{
							OS:           "linux",
							Architecture: "amd64",
						},
					}},
				},
			},
			client: &repositoryClient{
				getImageByDigestFn: func(
					context.Context, string, *platformConstraint,
				) (*Image, error) {
					return &testImage, nil
				},
			},
			assertions: func(t *testing.T, img *Image, err error) {
				require.NoError(t, err)
				require.Equal(t, testImage, *img)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			image, err := testCase.client.getImageFromV1ImageIndex(
				context.Background(),
				testDigest,
				testCase.idx,
				testCase.platform,
			)
			testCase.assertions(t, image, err)
		})
	}
}

func TestGetImageFromV1Image(t *testing.T) {
	const testDigest = "fake-digest"

	testCases := []struct {
		name       string
		img        v1.Image
		platform   *platformConstraint
		client     *repositoryClient
		assertions func(*testing.T, *Image, error)
	}{
		{
			name: "no platform constraint",
			img: &mockImage{
				configFile: &v1.ConfigFile{},
			},
			client: &repositoryClient{},
			assertions: func(t *testing.T, img *Image, err error) {
				require.NoError(t, err)
				require.NotNil(t, img)
				require.NotEmpty(t, img.Digest)
				require.NotNil(t, img.CreatedAt)
			},
		},
		{
			name: "does not match platform constraint",
			img: &mockImage{
				configFile: &v1.ConfigFile{
					OS:           "linux",
					Architecture: "amd64",
				},
			},
			platform: &platformConstraint{
				os:   "linux",
				arch: "arm64",
			},
			client: &repositoryClient{},
			assertions: func(t *testing.T, img *Image, err error) {
				require.NoError(t, err)
				require.Nil(t, img)
			},
		},
		{
			name: "matches platform constraint",
			img: &mockImage{
				configFile: &v1.ConfigFile{
					OS:           "linux",
					Architecture: "amd64",
				},
			},
			platform: &platformConstraint{
				os:   "linux",
				arch: "amd64",
			},
			client: &repositoryClient{},
			assertions: func(t *testing.T, img *Image, err error) {
				require.NoError(t, err)
				require.NotNil(t, img)
				require.NotEmpty(t, img.Digest)
				require.NotNil(t, img.CreatedAt)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			image, err := testCase.client.getImageFromV1Image(
				testDigest,
				testCase.img,
				testCase.platform,
			)
			testCase.assertions(t, image, err)
		})
	}
}

type mockImageIndex struct {
	indexManifest *v1.IndexManifest
}

func (m *mockImageIndex) MediaType() (types.MediaType, error) {
	return "", errNotImplemented
}

func (m *mockImageIndex) Digest() (v1.Hash, error) {
	return v1.Hash{}, errNotImplemented
}

func (m *mockImageIndex) Size() (int64, error) {
	return 0, errNotImplemented
}

func (m *mockImageIndex) IndexManifest() (*v1.IndexManifest, error) {
	return m.indexManifest, nil
}

func (m *mockImageIndex) RawManifest() ([]byte, error) {
	return nil, errNotImplemented
}

func (m *mockImageIndex) Image(v1.Hash) (v1.Image, error) {
	return nil, errNotImplemented
}

func (m *mockImageIndex) ImageIndex(v1.Hash) (v1.ImageIndex, error) {
	return nil, errNotImplemented
}

type mockImage struct {
	configFile *v1.ConfigFile
}

func (m *mockImage) Layers() ([]v1.Layer, error) {
	return nil, errNotImplemented
}

func (m *mockImage) MediaType() (types.MediaType, error) {
	return "", errNotImplemented
}

func (m *mockImage) Size() (int64, error) {
	return 0, errNotImplemented
}

func (m *mockImage) ConfigName() (v1.Hash, error) {
	return v1.Hash{}, errNotImplemented
}

func (m *mockImage) ConfigFile() (*v1.ConfigFile, error) {
	return m.configFile, nil
}

func (m *mockImage) RawConfigFile() ([]byte, error) {
	return nil, errNotImplemented
}

func (m *mockImage) Digest() (v1.Hash, error) {
	return v1.Hash{}, errNotImplemented
}

func (m *mockImage) Manifest() (*v1.Manifest, error) {
	return nil, errNotImplemented
}

func (m *mockImage) RawManifest() ([]byte, error) {
	return nil, errNotImplemented
}

func (m *mockImage) LayerByDigest(v1.Hash) (v1.Layer, error) {
	return nil, errNotImplemented
}

func (m *mockImage) LayerByDiffID(v1.Hash) (v1.Layer, error) {
	return nil, errNotImplemented
}

var errNotImplemented = errors.New("not implemented")
