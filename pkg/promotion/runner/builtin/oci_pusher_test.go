package builtin

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/promotion"
	builtin "github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_ociPusher_validate(t *testing.T) {
	tests := []validationTestCase{
		{
			name:   "srcRef is not specified",
			config: promotion.Config{},
			expectedProblems: []string{
				"(root): srcRef is required",
			},
		},
		{
			name:   "destRef is not specified",
			config: promotion.Config{},
			expectedProblems: []string{
				"(root): destRef is required",
			},
		},
		{
			name: "valid minimal config",
			config: promotion.Config{
				"srcRef":  "registry.example.com/image:tag",
				"destRef": "registry.example.com/image:newtag",
			},
		},
		{
			name: "valid config with OCI protocol",
			config: promotion.Config{
				"srcRef":  "oci://registry.example.com/chart:1.0.0",
				"destRef": "oci://registry.example.com/chart:2.0.0",
			},
		},
		{
			name: "valid config with non-standard port",
			config: promotion.Config{
				"srcRef":  "an.internal.registry.com:5050/myrepo/myimage:latest",
				"destRef": "an.internal.registry.com:5050/myrepo/myimage:newtag",
			},
		},
		{
			name: "valid config with OCI protocol and non-standard port",
			config: promotion.Config{
				"srcRef":  "oci://registry.example.com:5050/chart:1.0.0",
				"destRef": "oci://registry.example.com:5050/chart:2.0.0",
			},
		},
		{
			name: "valid config with all optional fields",
			config: promotion.Config{
				"srcRef":                "registry.example.com/image:tag",
				"destRef":               "registry.example.com/image:newtag",
				"insecureSkipTLSVerify": true,
				"annotations": map[string]any{
					"org.opencontainers.image.source": "https://github.com/example/repo",
				},
			},
		},
	}

	r := newOCIPusher(promotion.StepRunnerCapabilities{}, ociPusherConfig{
		MaxArtifactSize: int64(1 << 30),
	})
	runner, ok := r.(*ociPusher)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

func Test_ociPusher_run(t *testing.T) {
	// Start an in-memory registry for testing.
	regHandler := registry.New()
	srv := httptest.NewServer(regHandler)
	t.Cleanup(srv.Close)

	// Strip the "http://" prefix to get a valid registry host.
	regHost := srv.Listener.Addr().String()

	// Push a test image to the registry for use as a source.
	srcImageRef := fmt.Sprintf("%s/test/image:v1.0.0", regHost)
	srcRef, err := name.ParseReference(srcImageRef)
	require.NoError(t, err)

	testImg, err := random.Image(256, 1)
	require.NoError(t, err)
	require.NoError(t, remote.Write(srcRef, testImg))

	srcDigest, err := testImg.Digest()
	require.NoError(t, err)

	// Push a test image index to the registry.
	srcIndexRef := fmt.Sprintf("%s/test/multiarch:v1.0.0", regHost)
	idxRef, err := name.ParseReference(srcIndexRef)
	require.NoError(t, err)

	testIdx, err := random.Index(256, 1, 2) // 2 platform images
	require.NoError(t, err)
	require.NoError(t, remote.WriteIndex(idxRef, testIdx))

	srcIdxDigest, err := testIdx.Digest()
	require.NoError(t, err)

	tests := []struct {
		name       string
		cfg        builtin.OCIPushConfig
		assertions func(*testing.T, promotion.StepResult, error)
	}{
		{
			name: "push single image to new tag",
			cfg: builtin.OCIPushConfig{
				SrcRef:  srcImageRef,
				DestRef: fmt.Sprintf("%s/test/image:v2.0.0", regHost),
			},
			assertions: func(t *testing.T, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, string(kargoapi.PromotionStepStatusSucceeded), string(result.Status))
				assert.Equal(t,
					fmt.Sprintf("%s/test/image:v2.0.0", regHost),
					result.Output["image"],
				)
				assert.Equal(t, srcDigest.String(), result.Output["digest"])
				assert.Equal(t, "v2.0.0", result.Output["tag"])

				// Verify the image is retrievable at the destination.
				dstRef, parseErr := name.ParseReference(
					fmt.Sprintf("%s/test/image:v2.0.0", regHost),
				)
				require.NoError(t, parseErr)
				desc, getErr := remote.Get(dstRef)
				require.NoError(t, getErr)
				assert.Equal(t, srcDigest, desc.Digest)
			},
		},
		{
			name: "push image by digest",
			cfg: builtin.OCIPushConfig{
				SrcRef:  fmt.Sprintf("%s/test/image@%s", regHost, srcDigest.String()),
				DestRef: fmt.Sprintf("%s/test/image:pinned", regHost),
			},
			assertions: func(t *testing.T, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, string(kargoapi.PromotionStepStatusSucceeded), string(result.Status))
				assert.Equal(t, srcDigest.String(), result.Output["digest"])
				assert.Equal(t, "pinned", result.Output["tag"])
			},
		},
		{
			name: "push image index",
			cfg: builtin.OCIPushConfig{
				SrcRef:  srcIndexRef,
				DestRef: fmt.Sprintf("%s/test/multiarch:v2.0.0", regHost),
			},
			assertions: func(t *testing.T, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, string(kargoapi.PromotionStepStatusSucceeded), string(result.Status))
				assert.Equal(t, srcIdxDigest.String(), result.Output["digest"])
				assert.Equal(t, "v2.0.0", result.Output["tag"])

				// Verify the index is retrievable at the destination.
				dstRef, parseErr := name.ParseReference(
					fmt.Sprintf("%s/test/multiarch:v2.0.0", regHost),
				)
				require.NoError(t, parseErr)
				desc, getErr := remote.Get(dstRef)
				require.NoError(t, getErr)
				assert.True(t, desc.MediaType.IsIndex())
			},
		},
		{
			name: "push with annotations",
			cfg: builtin.OCIPushConfig{
				SrcRef:  srcImageRef,
				DestRef: fmt.Sprintf("%s/test/image:annotated", regHost),
				Annotations: map[string]string{
					"org.opencontainers.image.source": "https://github.com/example",
				},
			},
			assertions: func(t *testing.T, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, string(kargoapi.PromotionStepStatusSucceeded), string(result.Status))

				// Verify annotations on the pushed manifest.
				dstRef, parseErr := name.ParseReference(
					fmt.Sprintf("%s/test/image:annotated", regHost),
				)
				require.NoError(t, parseErr)
				img, getErr := remote.Image(dstRef)
				require.NoError(t, getErr)
				manifest, mErr := img.Manifest()
				require.NoError(t, mErr)
				assert.Equal(t,
					"https://github.com/example",
					manifest.Annotations["org.opencontainers.image.source"],
				)
			},
		},
		{
			name: "push index with unprefixed annotations goes to child manifests",
			cfg: builtin.OCIPushConfig{
				SrcRef:  srcIndexRef,
				DestRef: fmt.Sprintf("%s/test/multiarch:annotated", regHost),
				Annotations: map[string]string{
					"io.kargo.test": "true",
				},
			},
			assertions: func(t *testing.T, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, string(kargoapi.PromotionStepStatusSucceeded), string(result.Status))

				dstRef, parseErr := name.ParseReference(
					fmt.Sprintf("%s/test/multiarch:annotated", regHost),
				)
				require.NoError(t, parseErr)
				idx, getErr := remote.Index(dstRef)
				require.NoError(t, getErr)

				// Unprefixed annotations should NOT be on the index.
				idxManifest, mErr := idx.IndexManifest()
				require.NoError(t, mErr)
				assert.Empty(t, idxManifest.Annotations["io.kargo.test"])

				// Unprefixed annotations should be on each child manifest.
				for _, desc := range idxManifest.Manifests {
					img, imgErr := idx.Image(desc.Digest)
					require.NoError(t, imgErr)
					m, manifestErr := img.Manifest()
					require.NoError(t, manifestErr)
					assert.Equal(t, "true", m.Annotations["io.kargo.test"])
				}
			},
		},
		{
			name: "push index with scoped annotations",
			cfg: builtin.OCIPushConfig{
				SrcRef:  srcIndexRef,
				DestRef: fmt.Sprintf("%s/test/multiarch:scoped", regHost),
				Annotations: map[string]string{
					"index:io.kargo.index-only":       "idx",
					"manifest:io.kargo.manifest-only": "mfst",
					"io.kargo.default":                "both",
				},
			},
			assertions: func(t *testing.T, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, string(kargoapi.PromotionStepStatusSucceeded), string(result.Status))

				dstRef, parseErr := name.ParseReference(
					fmt.Sprintf("%s/test/multiarch:scoped", regHost),
				)
				require.NoError(t, parseErr)
				idx, getErr := remote.Index(dstRef)
				require.NoError(t, getErr)

				// Index annotations: only "index:" prefixed.
				idxManifest, mErr := idx.IndexManifest()
				require.NoError(t, mErr)
				assert.Equal(t, "idx", idxManifest.Annotations["io.kargo.index-only"])
				assert.Empty(t, idxManifest.Annotations["io.kargo.manifest-only"])
				assert.Empty(t, idxManifest.Annotations["io.kargo.default"])

				// Child manifest annotations: "manifest:" prefixed + unprefixed.
				for _, desc := range idxManifest.Manifests {
					img, imgErr := idx.Image(desc.Digest)
					require.NoError(t, imgErr)
					m, manifestErr := img.Manifest()
					require.NoError(t, manifestErr)
					assert.Equal(t, "mfst", m.Annotations["io.kargo.manifest-only"])
					assert.Equal(t, "both", m.Annotations["io.kargo.default"])
					assert.Empty(t, m.Annotations["io.kargo.index-only"])
				}
			},
		},
		{
			name: "cross-repo push",
			cfg: builtin.OCIPushConfig{
				SrcRef:  srcImageRef,
				DestRef: fmt.Sprintf("%s/other/repo:latest", regHost),
			},
			assertions: func(t *testing.T, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, string(kargoapi.PromotionStepStatusSucceeded), string(result.Status))

				// Verify the image is at the new repo.
				dstRef, parseErr := name.ParseReference(
					fmt.Sprintf("%s/other/repo:latest", regHost),
				)
				require.NoError(t, parseErr)
				desc, getErr := remote.Get(dstRef)
				require.NoError(t, getErr)
				assert.Equal(t, srcDigest, desc.Digest)
			},
		},
		{
			name: "source not found",
			cfg: builtin.OCIPushConfig{
				SrcRef:  fmt.Sprintf("%s/nonexistent/image:v1.0.0", regHost),
				DestRef: fmt.Sprintf("%s/test/image:copy", regHost),
			},
			assertions: func(t *testing.T, _ promotion.StepResult, err error) {
				assert.ErrorContains(t, err, "failed to get source artifact")
			},
		},
		{
			name: "invalid source reference",
			cfg: builtin.OCIPushConfig{
				SrcRef:  "invalid::ref",
				DestRef: fmt.Sprintf("%s/test/image:copy", regHost),
			},
			assertions: func(t *testing.T, _ promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.ErrorContains(t, err, "failed to parse source reference")
				var termErr *promotion.TerminalError
				assert.ErrorAs(t, err, &termErr)
			},
		},
		{
			name: "invalid destination reference",
			cfg: builtin.OCIPushConfig{
				SrcRef:  srcImageRef,
				DestRef: "invalid::ref",
			},
			assertions: func(t *testing.T, _ promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.ErrorContains(t, err, "failed to parse destination reference")
				var termErr *promotion.TerminalError
				assert.ErrorAs(t, err, &termErr)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &ociPusher{
				credsDB:         &credentials.FakeDB{},
				schemaLoader:    getConfigSchemaLoader(stepKindOCIPush),
				maxArtifactSize: int64(1 << 30),
			}

			stepCtx := &promotion.StepContext{
				Project: "fake-project",
			}

			result, err := runner.run(context.Background(), stepCtx, tt.cfg)
			tt.assertions(t, result, err)
		})
	}
}

func Test_ociPusher_push_unsupportedMediaType(t *testing.T) {
	// Create a descriptor with an unsupported media type.
	desc := &remote.Descriptor{
		Descriptor: v1.Descriptor{
			MediaType: types.MediaType("application/vnd.unsupported"),
		},
	}

	runner := &ociPusher{maxArtifactSize: int64(1 << 30)}
	srcRef, err := name.ParseReference("localhost:5000/src:tag")
	require.NoError(t, err)
	dstRef, err := name.ParseReference("localhost:5000/test:tag")
	require.NoError(t, err)

	_, err = runner.push(desc, srcRef, dstRef, nil, nil)
	assert.ErrorContains(t, err, "unsupported media type")
	var termErr *promotion.TerminalError
	assert.ErrorAs(t, err, &termErr)
}

func Test_ociPusher_run_credentialError(t *testing.T) {
	tests := []struct {
		name    string
		cfg     builtin.OCIPushConfig
		credsDB credentials.Database
		errMsg  string
	}{
		{
			name: "source credential error",
			cfg: builtin.OCIPushConfig{
				SrcRef:  "registry.example.com/image:tag",
				DestRef: "registry.example.com/image:newtag",
			},
			credsDB: &credentials.FakeDB{
				GetFn: func(
					_ context.Context, _ string, _ credentials.Type, repoURL string,
				) (*credentials.Credentials, error) {
					if repoURL == "registry.example.com/image" {
						return nil, fmt.Errorf("source cred error")
					}
					return nil, nil
				},
			},
			errMsg: "error obtaining credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &ociPusher{
				credsDB:         tt.credsDB,
				schemaLoader:    getConfigSchemaLoader(stepKindOCIPush),
				maxArtifactSize: int64(1 << 30),
			}

			stepCtx := &promotion.StepContext{
				Project: "fake-project",
			}

			_, err := runner.run(context.Background(), stepCtx, tt.cfg)
			assert.ErrorContains(t, err, tt.errMsg)
		})
	}
}

// Test that annotations don't mutate the source image when none are provided.
func Test_ociPusher_run_noAnnotationsMutation(t *testing.T) {
	regHandler := registry.New()
	srv := httptest.NewServer(regHandler)
	t.Cleanup(srv.Close)
	regHost := srv.Listener.Addr().String()

	// Create an OCI image with existing annotations.
	srcImg, err := random.Image(256, 1)
	require.NoError(t, err)
	annotated, ok := mutate.Annotations(srcImg, map[string]string{
		"existing": "annotation",
	}).(v1.Image)
	require.True(t, ok)
	srcImg = annotated

	srcRef, err := name.ParseReference(fmt.Sprintf("%s/test/annotated:v1", regHost))
	require.NoError(t, err)
	require.NoError(t, remote.Write(srcRef, srcImg))

	runner := &ociPusher{
		credsDB:         &credentials.FakeDB{},
		schemaLoader:    getConfigSchemaLoader(stepKindOCIPush),
		maxArtifactSize: int64(1 << 30),
	}

	// Push without specifying annotations.
	result, err := runner.run(context.Background(), &promotion.StepContext{
		Project: "fake-project",
	}, builtin.OCIPushConfig{
		SrcRef:  fmt.Sprintf("%s/test/annotated:v1", regHost),
		DestRef: fmt.Sprintf("%s/test/annotated:v2", regHost),
	})
	require.NoError(t, err)
	assert.Equal(t, string(kargoapi.PromotionStepStatusSucceeded), string(result.Status))

	// Verify the existing annotation is preserved and no extra ones added.
	dstRef, err := name.ParseReference(fmt.Sprintf("%s/test/annotated:v2", regHost))
	require.NoError(t, err)
	dstImg, err := remote.Image(dstRef)
	require.NoError(t, err)
	manifest, err := dstImg.Manifest()
	require.NoError(t, err)
	assert.Equal(t, "annotation", manifest.Annotations["existing"])
}

func Test_parseAnnotationScopes(t *testing.T) {
	tests := []struct {
		name         string
		annotations  map[string]string
		wantIndex    map[string]string
		wantManifest map[string]string
	}{
		{
			name:         "nil annotations",
			annotations:  nil,
			wantIndex:    map[string]string{},
			wantManifest: map[string]string{},
		},
		{
			name: "unprefixed go to manifest",
			annotations: map[string]string{
				"foo": "bar",
			},
			wantIndex:    map[string]string{},
			wantManifest: map[string]string{"foo": "bar"},
		},
		{
			name: "index prefix",
			annotations: map[string]string{
				"index:foo": "bar",
			},
			wantIndex:    map[string]string{"foo": "bar"},
			wantManifest: map[string]string{},
		},
		{
			name: "manifest prefix",
			annotations: map[string]string{
				"manifest:foo": "bar",
			},
			wantIndex:    map[string]string{},
			wantManifest: map[string]string{"foo": "bar"},
		},
		{
			name: "mixed scopes",
			annotations: map[string]string{
				"index:a":    "1",
				"manifest:b": "2",
				"c":          "3",
			},
			wantIndex:    map[string]string{"a": "1"},
			wantManifest: map[string]string{"b": "2", "c": "3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ociPusher{}
			scopes := p.parseAnnotationScopes(tt.annotations)
			assert.Equal(t, tt.wantIndex, scopes.index)
			assert.Equal(t, tt.wantManifest, scopes.manifest)
		})
	}
}

func Test_ociPusher_run_scopedAnnotationsOnImage(t *testing.T) {
	regHandler := registry.New()
	srv := httptest.NewServer(regHandler)
	t.Cleanup(srv.Close)
	regHost := srv.Listener.Addr().String()

	srcImg, err := random.Image(256, 1)
	require.NoError(t, err)
	srcRef, err := name.ParseReference(fmt.Sprintf("%s/test/scoped:v1", regHost))
	require.NoError(t, err)
	require.NoError(t, remote.Write(srcRef, srcImg))

	runner := &ociPusher{
		credsDB:         &credentials.FakeDB{},
		schemaLoader:    getConfigSchemaLoader(stepKindOCIPush),
		maxArtifactSize: int64(1 << 30),
	}

	// Push with mixed scoped annotations. "index:" should be ignored for images.
	result, err := runner.run(context.Background(), &promotion.StepContext{
		Project: "fake-project",
	}, builtin.OCIPushConfig{
		SrcRef:  fmt.Sprintf("%s/test/scoped:v1", regHost),
		DestRef: fmt.Sprintf("%s/test/scoped:v2", regHost),
		Annotations: map[string]string{
			"index:ignored.key": "ignored",
			"manifest:explicit": "yes",
			"unprefixed":        "also-yes",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, string(kargoapi.PromotionStepStatusSucceeded), string(result.Status))

	dstRef, err := name.ParseReference(fmt.Sprintf("%s/test/scoped:v2", regHost))
	require.NoError(t, err)
	dstImg, err := remote.Image(dstRef)
	require.NoError(t, err)
	manifest, err := dstImg.Manifest()
	require.NoError(t, err)

	// manifest: and unprefixed should appear on the image manifest.
	assert.Equal(t, "yes", manifest.Annotations["explicit"])
	assert.Equal(t, "also-yes", manifest.Annotations["unprefixed"])
	// index: should NOT appear.
	assert.Empty(t, manifest.Annotations["ignored.key"])
}

// Test OCI image with an OCI manifest (not Docker) to ensure annotations work.
func Test_ociPusher_run_ociManifestAnnotations(t *testing.T) {
	regHandler := registry.New()
	srv := httptest.NewServer(regHandler)
	t.Cleanup(srv.Close)
	regHost := srv.Listener.Addr().String()

	// Create an OCI-format image (empty.Image is OCI by default).
	srcImg := empty.Image
	srcRef, err := name.ParseReference(fmt.Sprintf("%s/test/oci:v1", regHost))
	require.NoError(t, err)
	require.NoError(t, remote.Write(srcRef, srcImg))

	runner := &ociPusher{
		credsDB:         &credentials.FakeDB{},
		schemaLoader:    getConfigSchemaLoader(stepKindOCIPush),
		maxArtifactSize: int64(1 << 30),
	}

	result, err := runner.run(context.Background(), &promotion.StepContext{
		Project: "fake-project",
	}, builtin.OCIPushConfig{
		SrcRef:  fmt.Sprintf("%s/test/oci:v1", regHost),
		DestRef: fmt.Sprintf("%s/test/oci:v2", regHost),
		Annotations: map[string]string{
			"test.key": "test.value",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, string(kargoapi.PromotionStepStatusSucceeded), string(result.Status))

	dstRef, err := name.ParseReference(fmt.Sprintf("%s/test/oci:v2", regHost))
	require.NoError(t, err)
	dstImg, err := remote.Image(dstRef)
	require.NoError(t, err)
	manifest, err := dstImg.Manifest()
	require.NoError(t, err)
	assert.Equal(t, "test.value", manifest.Annotations["test.key"])
}

func Test_imageSize(t *testing.T) {
	img, err := random.Image(256, 3)
	require.NoError(t, err)

	p := &ociPusher{}
	sz, err := p.imageSize(img)
	require.NoError(t, err)
	assert.Greater(t, sz, int64(0))

	// Verify it matches the sum of config + layers from the manifest.
	m, err := img.Manifest()
	require.NoError(t, err)
	var expected int64
	expected += m.Config.Size
	for _, l := range m.Layers {
		expected += l.Size
	}
	assert.Equal(t, expected, sz)
}

func Test_indexSize(t *testing.T) {
	idx, err := random.Index(256, 2, 3) // 3 platform images, 2 layers each
	require.NoError(t, err)

	p := &ociPusher{}
	sz, err := p.indexSize(idx)
	require.NoError(t, err)
	assert.Greater(t, sz, int64(0))

	// Verify it equals the sum of imageSize for each child.
	im, err := idx.IndexManifest()
	require.NoError(t, err)
	var expected int64
	for _, desc := range im.Manifests {
		child, imgErr := idx.Image(desc.Digest)
		require.NoError(t, imgErr)
		childSz, szErr := p.imageSize(child)
		require.NoError(t, szErr)
		expected += childSz
	}
	assert.Equal(t, expected, sz)
}

func Test_ociPusher_push_sizeLimitExceeded(t *testing.T) {
	regHandler := registry.New()
	srv := httptest.NewServer(regHandler)
	t.Cleanup(srv.Close)
	regHost := srv.Listener.Addr().String()

	// Push a test image (will exceed our tiny limit).
	srcImg, err := random.Image(256, 1)
	require.NoError(t, err)
	srcRef, err := name.ParseReference(fmt.Sprintf("%s/test/big:v1", regHost))
	require.NoError(t, err)
	require.NoError(t, remote.Write(srcRef, srcImg))

	// Push a test index.
	srcIdx, err := random.Index(256, 1, 2)
	require.NoError(t, err)
	idxRef, err := name.ParseReference(fmt.Sprintf("%s/test/bigidx:v1", regHost))
	require.NoError(t, err)
	require.NoError(t, remote.WriteIndex(idxRef, srcIdx))

	tests := []struct {
		name   string
		srcRef string
	}{
		{
			name:   "image exceeds size limit",
			srcRef: fmt.Sprintf("%s/test/big:v1", regHost),
		},
		{
			name:   "index exceeds size limit",
			srcRef: fmt.Sprintf("%s/test/bigidx:v1", regHost),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &ociPusher{
				credsDB:         &credentials.FakeDB{},
				schemaLoader:    getConfigSchemaLoader(stepKindOCIPush),
				maxArtifactSize: 100, // tiny limit to trigger the error
			}

			result, err := runner.run(context.Background(), &promotion.StepContext{
				Project: "fake-project",
			}, builtin.OCIPushConfig{
				SrcRef:  tt.srcRef,
				DestRef: fmt.Sprintf("%s/test/dst:v1", regHost),
			})
			assert.Equal(t, string(kargoapi.PromotionStepStatusErrored), string(result.Status))
			assert.ErrorContains(t, err, "exceeds maximum allowed size of")
			var termErr *promotion.TerminalError
			assert.ErrorAs(t, err, &termErr)
		})
	}
}

func Test_ociPusher_push_sizeLimitZero(t *testing.T) {
	regHandler := registry.New()
	srv := httptest.NewServer(regHandler)
	t.Cleanup(srv.Close)
	regHost := srv.Listener.Addr().String()

	srcImg, err := random.Image(256, 1)
	require.NoError(t, err)
	srcRef, err := name.ParseReference(fmt.Sprintf("%s/test/img:v1", regHost))
	require.NoError(t, err)
	require.NoError(t, remote.Write(srcRef, srcImg))

	runner := &ociPusher{
		credsDB:         &credentials.FakeDB{},
		schemaLoader:    getConfigSchemaLoader(stepKindOCIPush),
		maxArtifactSize: 0, // blocks all cross-repo pushes
	}
	stepCtx := &promotion.StepContext{Project: "fake-project"}

	t.Run("cross-repo push is blocked", func(t *testing.T) {
		result, err := runner.run(context.Background(), stepCtx, builtin.OCIPushConfig{
			SrcRef:  fmt.Sprintf("%s/test/img:v1", regHost),
			DestRef: fmt.Sprintf("%s/other/repo:v1", regHost),
		})
		assert.Equal(t, string(kargoapi.PromotionStepStatusErrored), string(result.Status))
		assert.ErrorContains(t, err, "cross-repository push is disabled")
	})

	t.Run("same-repo retag succeeds", func(t *testing.T) {
		result, err := runner.run(context.Background(), stepCtx, builtin.OCIPushConfig{
			SrcRef:  fmt.Sprintf("%s/test/img:v1", regHost),
			DestRef: fmt.Sprintf("%s/test/img:v2", regHost),
		})
		require.NoError(t, err)
		assert.Equal(t, string(kargoapi.PromotionStepStatusSucceeded), string(result.Status))
	})
}

func Test_ociPusher_push_sizeLimitDisabled(t *testing.T) {
	regHandler := registry.New()
	srv := httptest.NewServer(regHandler)
	t.Cleanup(srv.Close)
	regHost := srv.Listener.Addr().String()

	srcImg, err := random.Image(256, 1)
	require.NoError(t, err)
	srcRef, err := name.ParseReference(fmt.Sprintf("%s/test/img:v1", regHost))
	require.NoError(t, err)
	require.NoError(t, remote.Write(srcRef, srcImg))

	runner := &ociPusher{
		credsDB:         &credentials.FakeDB{},
		schemaLoader:    getConfigSchemaLoader(stepKindOCIPush),
		maxArtifactSize: -1, // unlimited
	}

	result, err := runner.run(context.Background(), &promotion.StepContext{
		Project: "fake-project",
	}, builtin.OCIPushConfig{
		SrcRef:  fmt.Sprintf("%s/test/img:v1", regHost),
		DestRef: fmt.Sprintf("%s/other/repo:v1", regHost),
	})
	require.NoError(t, err)
	assert.Equal(t, string(kargoapi.PromotionStepStatusSucceeded), string(result.Status))
}

func Test_ociPusherConfig(t *testing.T) {
	tests := []struct {
		name     string
		envValue string // empty means unset
		expected int64
	}{
		{
			name:     "unset returns default 1 GiB",
			envValue: "",
			expected: 1 << 30,
		},
		{
			name:     "zero blocks cross-repo pushes",
			envValue: "0",
			expected: 0,
		},
		{
			name:     "negative one disables limit",
			envValue: "-1",
			expected: -1,
		},
		{
			name:     "custom value",
			envValue: "536870912",
			expected: 536870912,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv("MAX_OCI_PUSH_ARTIFACT_SIZE", tt.envValue)
			}
			cfg := ociPusherConfig{}
			envconfig.MustProcess("", &cfg)
			assert.Equal(t, tt.expected, cfg.MaxArtifactSize)
		})
	}
}
