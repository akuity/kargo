package image

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	"github.com/akuity/kargo/pkg/cache"
)

func TestNewRepositoryClient(t *testing.T) {
	client, err := newRepositoryClient("debian", false, nil, true)
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NotNil(t, client.imageCache)
	require.True(t, client.cacheByTag)
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

func Test_repositoryClient_getImageByTag(t *testing.T) {
	const testRepoURL = "fake-url"
	const testTag = "fake-tag"

	testRepoRef, err := name.ParseReference(testRepoURL)
	require.NoError(t, err)

	testImage := image{
		Tag:       testTag,
		CreatedAt: ptr.To(time.Now().UTC()),
		Platforms: []platform{{OS: "linux", Arch: "amd64"}},
	}

	testCases := []struct {
		name               string
		client             *repositoryClient
		setupCache         func(t *testing.T, c cache.Cache[image])
		platformConstraint *platformConstraint
		assertions         func(*testing.T, *image, error)
	}{
		{
			name: "cache hit with no platform constraint",
			client: &repositoryClient{
				repoURL:    testRepoURL,
				cacheByTag: true,
			},
			setupCache: func(t *testing.T, c cache.Cache[image]) {
				err = c.Set(
					t.Context(),
					fmt.Sprintf("%s:%s", testRepoURL, testImage.Tag),
					testImage,
				)
				require.NoError(t, err)
			},
			assertions: func(t *testing.T, img *image, err error) {
				require.NoError(t, err)
				require.Equal(t, testImage, *img)
			},
		},
		{
			name: "cache hit with unsatisfied platform constraint",
			client: &repositoryClient{
				repoURL:    testRepoURL,
				cacheByTag: true,
			},
			setupCache: func(t *testing.T, c cache.Cache[image]) {
				err = c.Set(
					t.Context(),
					fmt.Sprintf("%s:%s", testRepoURL, testImage.Tag),
					testImage,
				)
				require.NoError(t, err)
			},
			platformConstraint: &platformConstraint{os: "linux", arch: "arm64"},
			assertions: func(t *testing.T, img *image, err error) {
				require.NoError(t, err)
				require.Nil(t, img)
			},
		},
		{
			name: "cache hit with satisfied platform constraint",
			client: &repositoryClient{
				repoURL:    testRepoURL,
				cacheByTag: true,
			},
			setupCache: func(t *testing.T, c cache.Cache[image]) {
				err = c.Set(
					t.Context(),
					fmt.Sprintf("%s:%s", testRepoURL, testImage.Tag),
					testImage,
				)
				require.NoError(t, err)
			},
			platformConstraint: &platformConstraint{os: "linux", arch: "amd64"},
			assertions: func(t *testing.T, img *image, err error) {
				require.NoError(t, err)
				require.Equal(t, testImage, *img)
			},
		},
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
			assertions: func(t *testing.T, _ *image, err error) {
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
				) (*image, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *image, err error) {
				require.ErrorContains(t, err, "error getting image from descriptor for tag")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "success with no platform constraint",
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
				) (*image, error) {
					return &testImage, nil
				},
			},
			assertions: func(t *testing.T, img *image, err error) {
				require.NoError(t, err)
				require.Equal(t, testImage, *img)
			},
		},
		{
			name: "unsatisfied platform constraint",
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
				) (*image, error) {
					return &testImage, nil
				},
			},
			platformConstraint: &platformConstraint{os: "linux", arch: "arm64"},
			assertions: func(t *testing.T, img *image, err error) {
				require.NoError(t, err)
				require.Nil(t, img)
			},
		},
		{
			name: "success with satisfied platform constraint",
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
				) (*image, error) {
					return &testImage, nil
				},
			},
			platformConstraint: &platformConstraint{os: "linux", arch: "amd64"},
			assertions: func(t *testing.T, img *image, err error) {
				require.NoError(t, err)
				require.Equal(t, testImage, *img)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.client.imageCache, err = cache.NewInMemoryCache[image](1)
			require.NoError(t, err)
			if testCase.setupCache != nil {
				testCase.setupCache(t, testCase.client.imageCache)
			}
			img, err := testCase.client.getImageByTag(
				context.Background(),
				testTag,
				testCase.platformConstraint,
			)
			testCase.assertions(t, img, err)
		})
	}
}

func Test_repositoryClient_getImageByDigest(t *testing.T) {
	const testRepoURL = "fake-url"
	const testDigest = "fake-digest"

	testRepoRef, err := name.ParseReference(testRepoURL)
	require.NoError(t, err)

	testImage := image{
		Digest:    testDigest,
		CreatedAt: ptr.To(time.Now().UTC()),
	}

	testImageCache, err := cache.NewInMemoryCache[image](1)
	require.NoError(t, err)
	err = testImageCache.Set(t.Context(), testImage.Digest, testImage)
	require.NoError(t, err)

	testCases := []struct {
		name       string
		client     *repositoryClient
		assertions func(*testing.T, *image, error)
	}{
		{
			name: "cache hit",
			client: &repositoryClient{
				imageCache: testImageCache,
			},
			assertions: func(t *testing.T, img *image, err error) {
				require.NoError(t, err)
				require.Equal(t, testImage, *img)
			},
		},
		{
			name: "error getting descriptor by digest",
			client: &repositoryClient{
				repoRef: testRepoRef,
				remoteGetFn: func(
					name.Reference, ...remote.Option,
				) (*remote.Descriptor, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *image, err error) {
				require.ErrorContains(t, err, "error getting image descriptor for digest")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error getting image from descriptor",
			client: &repositoryClient{
				repoRef: testRepoRef,
				remoteGetFn: func(
					name.Reference, ...remote.Option,
				) (*remote.Descriptor, error) {
					return &remote.Descriptor{}, nil
				},
				getImageFromRemoteDescFn: func(
					context.Context,
					*remote.Descriptor,
				) (*image, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *image, err error) {
				require.ErrorContains(t, err, "error getting image from descriptor for digest")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "success",
			client: &repositoryClient{
				repoRef: testRepoRef,
				remoteGetFn: func(
					name.Reference, ...remote.Option,
				) (*remote.Descriptor, error) {
					return &remote.Descriptor{}, nil
				},
				getImageFromRemoteDescFn: func(
					context.Context,
					*remote.Descriptor,
				) (*image, error) {
					return &testImage, nil
				},
			},
			assertions: func(t *testing.T, img *image, err error) {
				require.NoError(t, err)
				require.Equal(t, testImage, *img)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.client.imageCache == nil {
				testCase.client.imageCache, err = cache.NewInMemoryCache[image](1)
				require.NoError(t, err)
			}
			img, err := testCase.client.getImageByDigest(
				context.Background(),
				testDigest,
			)
			testCase.assertions(t, img, err)
		})
	}
}

func Test_repositoryClient_getImageFromRemoteDesc(t *testing.T) {
	testImage := image{
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
			context.Context, string, v1.ImageIndex,
		) (*image, error) {
			return &testImage, nil
		},
		getImageFromV1ImageFn: func(string, v1.Image) (*image, error) {
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
			)
			require.NoError(t, err)
			require.Equal(t, testImage, *img)
		})
	}

	t.Run("with remote descriptor annotations", func(t *testing.T) {
		imageWithAnnotations := image{
			CreatedAt: ptr.To(time.Now().UTC()),
			Annotations: map[string]string{
				"key.one":   "image-value", // This should override descriptor
				"key.two":   "image-value", // This should override descriptor
				"key.three": "image-value", // This is unique to image
			},
		}

		// Remote descriptor with annotations
		remoteDesc := &remote.Descriptor{
			Descriptor: v1.Descriptor{
				MediaType: types.OCIImageIndex,
				Annotations: map[string]string{
					"key.one":  "descriptor-value", // Should be overridden by image
					"key.two":  "descriptor-value", // Should be overridden by image
					"key.four": "descriptor-value", // Unique to descriptor
				},
			},
		}

		testClientWithAnnotations := &repositoryClient{
			getImageFromV1ImageIndexFn: func(
				context.Context, string, v1.ImageIndex,
			) (*image, error) {
				return &imageWithAnnotations, nil
			},
		}

		img, err := testClientWithAnnotations.getImageFromRemoteDesc(
			context.Background(),
			remoteDesc,
		)
		require.NoError(t, err)

		require.NotNil(t, img)
		require.Equal(t, map[string]string{
			"key.one":   "image-value",
			"key.two":   "image-value",
			"key.three": "image-value",
			"key.four":  "descriptor-value",
		}, img.Annotations)
	})
}

func Test_repositoryClient_getImageFromV1ImageIndex(t *testing.T) {
	const testDigest = "fake-digest"

	testImage := image{
		Digest:    testDigest,
		CreatedAt: ptr.To(time.Now().UTC()),
	}

	testCases := []struct {
		name       string
		idx        v1.ImageIndex
		client     *repositoryClient
		assertions func(*testing.T, *image, error)
	}{
		{
			name: "empty list or index not supported",
			idx: &mockImageIndex{
				indexManifest: &v1.IndexManifest{
					Manifests: []v1.Descriptor{{}},
				},
			},
			client: &repositoryClient{},
			assertions: func(t *testing.T, _ *image, err error) {
				require.ErrorContains(t, err, "empty V2 manifest list or OCI index is not supported")
			},
		},
		{
			name: "error getting image by digest",
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
				getImageByDigestFn: func(context.Context, string) (*image, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *image, err error) {
				require.ErrorContains(t, err, "error getting image with digest")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "no image found",
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
				getImageByDigestFn: func(context.Context, string) (*image, error) {
					return nil, nil
				},
			},
			assertions: func(t *testing.T, _ *image, err error) {
				require.ErrorContains(t, err, "found no image with digest")
			},
		},
		{
			name: "with annotations, success",
			idx: &mockImageIndex{
				indexManifest: &v1.IndexManifest{
					Manifests: []v1.Descriptor{{
						Platform: &v1.Platform{
							OS:           "linux",
							Architecture: "amd64",
						},
					}},
					Annotations: map[string]string{
						"org.opencontainers.image.vendor": "Test Vendor",
					},
				},
			},
			client: &repositoryClient{
				getImageByDigestFn: func(context.Context, string) (*image, error) {
					return &image{
						Digest:    testDigest,
						CreatedAt: testImage.CreatedAt,
						Annotations: map[string]string{
							ociCreatedAnnotation: "2023-01-01T00:00:00Z",
						},
					}, nil
				},
			},
			assertions: func(t *testing.T, img *image, err error) {
				require.NoError(t, err)
				require.NotNil(t, img)
				require.Equal(t, testDigest, img.Digest)
				require.Len(t, img.Annotations, 1)
				require.Equal(t, "Test Vendor", img.Annotations["org.opencontainers.image.vendor"])
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			image, err := testCase.client.getImageFromV1ImageIndex(
				context.Background(),
				testDigest,
				testCase.idx,
			)
			testCase.assertions(t, image, err)
		})
	}
}

func Test_repositoryClient_getImageFromV1Image(t *testing.T) {
	const testDigest = "fake-digest"

	testCases := []struct {
		name       string
		img        v1.Image
		client     *repositoryClient
		assertions func(*testing.T, *image, error)
	}{
		{
			name: "basic case",
			img: &mockImage{
				configFile: &v1.ConfigFile{},
			},
			client: &repositoryClient{},
			assertions: func(t *testing.T, img *image, err error) {
				require.NoError(t, err)
				require.NotNil(t, img)
				require.NotEmpty(t, img.Digest)
				require.NotNil(t, img.CreatedAt)
				require.Nil(t, img.Annotations) // No annotations in manifest
			},
		},
		{
			name: "with annotations",
			img: &mockImage{
				configFile: &v1.ConfigFile{},
				manifest: &v1.Manifest{
					Annotations: map[string]string{
						ociCreatedAnnotation:               "2023-01-01T00:00:00Z",
						"org.opencontainers.image.authors": "Test Author",
					},
				},
			},
			client: &repositoryClient{},
			assertions: func(t *testing.T, img *image, err error) {
				require.NoError(t, err)
				require.NotNil(t, img)
				require.NotEmpty(t, img.Digest)
				require.NotNil(t, img.CreatedAt)
				require.NotNil(t, img.Annotations)
				require.Equal(t, "Test Author", img.Annotations["org.opencontainers.image.authors"])
				require.Equal(t, "2023-01-01T00:00:00Z", img.Annotations[ociCreatedAnnotation])
			},
		},
		{
			name: "created date is taken from label",
			img: &mockImage{
				configFile: &v1.ConfigFile{
					Created: v1.Time{Time: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
					Config: v1.Config{
						Labels: map[string]string{
							ociCreatedAnnotation: "2023-02-01T00:00:00Z",
						},
					},
					OS:           "linux",
					Architecture: "amd64",
				},
				manifest: &v1.Manifest{},
			},
			assertions: func(t *testing.T, img *image, err error) {
				require.NoError(t, err)
				require.NotNil(t, img)
				expectedTime, err := time.Parse(time.RFC3339, "2023-02-01T00:00:00Z")
				require.NoError(t, err)
				require.Equal(t, expectedTime, *img.CreatedAt)
			},
		},
		{
			name: "created date is taken from annotation (higher priority than label)",
			img: &mockImage{
				configFile: &v1.ConfigFile{
					Created: v1.Time{Time: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
					Config: v1.Config{
						Labels: map[string]string{
							ociCreatedAnnotation: "2023-02-01T00:00:00Z",
						},
					},
					OS:           "linux",
					Architecture: "amd64",
				},
				manifest: &v1.Manifest{
					Annotations: map[string]string{
						ociCreatedAnnotation: "2023-03-01T00:00:00Z",
					},
				},
			},
			assertions: func(t *testing.T, img *image, err error) {
				require.NoError(t, err)
				require.NotNil(t, img)
				expectedTime, err := time.Parse(time.RFC3339, "2023-03-01T00:00:00Z")
				require.NoError(t, err)
				require.Equal(t, expectedTime, *img.CreatedAt)
			},
		},
		{
			name: "created date is taken from legacy label schema annotation",
			img: &mockImage{
				configFile: &v1.ConfigFile{
					Created: v1.Time{Time: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
					Config: v1.Config{
						Labels: map[string]string{},
					},
					OS:           "linux",
					Architecture: "amd64",
				},
				manifest: &v1.Manifest{
					Annotations: map[string]string{
						legacyBuildDateAnnotation: "2023-04-01T00:00:00Z",
					},
				},
			},
			assertions: func(t *testing.T, img *image, err error) {
				require.NoError(t, err)
				require.NotNil(t, img)
				expectedTime, err := time.Parse(time.RFC3339, "2023-04-01T00:00:00Z")
				require.NoError(t, err)
				require.Equal(t, expectedTime, *img.CreatedAt)
			},
		},
		{
			name: "created date is taken from legacy label schema label",
			img: &mockImage{
				configFile: &v1.ConfigFile{
					Created: v1.Time{Time: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
					Config: v1.Config{
						Labels: map[string]string{
							legacyBuildDateAnnotation: "2023-05-01T00:00:00Z",
						},
					},
					OS:           "linux",
					Architecture: "amd64",
				},
				manifest: &v1.Manifest{},
			},
			assertions: func(t *testing.T, img *image, err error) {
				require.NoError(t, err)
				require.NotNil(t, img)
				expectedTime, err := time.Parse(time.RFC3339, "2023-05-01T00:00:00Z")
				require.NoError(t, err)
				require.Equal(t, expectedTime, *img.CreatedAt)
			},
		},
		{
			name: "OCI annotation takes priority over legacy annotation",
			img: &mockImage{
				configFile: &v1.ConfigFile{
					Created: v1.Time{Time: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
					Config: v1.Config{
						Labels: map[string]string{},
					},
					OS:           "linux",
					Architecture: "amd64",
				},
				manifest: &v1.Manifest{
					Annotations: map[string]string{
						ociCreatedAnnotation:      "2023-06-01T00:00:00Z",
						legacyBuildDateAnnotation: "2023-07-01T00:00:00Z",
					},
				},
			},
			assertions: func(t *testing.T, img *image, err error) {
				require.NoError(t, err)
				require.NotNil(t, img)
				expectedTime, err := time.Parse(time.RFC3339, "2023-06-01T00:00:00Z")
				require.NoError(t, err)
				require.Equal(t, expectedTime, *img.CreatedAt)
			},
		},
		{
			name: "legacy annotation takes priority over OCI label",
			img: &mockImage{
				configFile: &v1.ConfigFile{
					Created: v1.Time{Time: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
					Config: v1.Config{
						Labels: map[string]string{
							ociCreatedAnnotation: "2023-08-01T00:00:00Z",
						},
					},
					OS:           "linux",
					Architecture: "amd64",
				},
				manifest: &v1.Manifest{
					Annotations: map[string]string{
						legacyBuildDateAnnotation: "2023-09-01T00:00:00Z",
					},
				},
			},
			assertions: func(t *testing.T, img *image, err error) {
				require.NoError(t, err)
				require.NotNil(t, img)
				expectedTime, err := time.Parse(time.RFC3339, "2023-09-01T00:00:00Z")
				require.NoError(t, err)
				require.Equal(t, expectedTime, *img.CreatedAt)
			},
		},
		{
			name: "OCI label takes priority over legacy label",
			img: &mockImage{
				configFile: &v1.ConfigFile{
					Created: v1.Time{Time: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
					Config: v1.Config{
						Labels: map[string]string{
							ociCreatedAnnotation:      "2023-10-01T00:00:00Z",
							legacyBuildDateAnnotation: "2023-11-01T00:00:00Z",
						},
					},
					OS:           "linux",
					Architecture: "amd64",
				},
				manifest: &v1.Manifest{},
			},
			assertions: func(t *testing.T, img *image, err error) {
				require.NoError(t, err)
				require.NotNil(t, img)
				expectedTime, err := time.Parse(time.RFC3339, "2023-10-01T00:00:00Z")
				require.NoError(t, err)
				require.Equal(t, expectedTime, *img.CreatedAt)
			},
		},
		{
			name: "fallback to config.Created when no annotations or labels",
			img: &mockImage{
				configFile: &v1.ConfigFile{
					Created: v1.Time{Time: time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)},
					Config: v1.Config{
						Labels: map[string]string{},
					},
					OS:           "linux",
					Architecture: "amd64",
				},
				manifest: &v1.Manifest{},
			},
			assertions: func(t *testing.T, img *image, err error) {
				require.NoError(t, err)
				require.NotNil(t, img)
				expectedTime := time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)
				require.Equal(t, expectedTime, *img.CreatedAt)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			image, err := testCase.client.getImageFromV1Image(
				testDigest,
				testCase.img,
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
	manifest   *v1.Manifest
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
	if m.manifest == nil {
		return &v1.Manifest{}, nil
	}
	return m.manifest, nil
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
