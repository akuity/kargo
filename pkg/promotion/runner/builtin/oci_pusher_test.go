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
			name:   "imageRef is not specified",
			config: promotion.Config{},
			expectedProblems: []string{
				"(root): imageRef is required",
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
				"imageRef": "registry.example.com/image:tag",
				"destRef":  "registry.example.com/image:newtag",
			},
		},
		{
			name: "valid config with OCI protocol",
			config: promotion.Config{
				"imageRef": "oci://registry.example.com/chart:1.0.0",
				"destRef":  "oci://registry.example.com/chart:2.0.0",
			},
		},
		{
			name: "valid config with all optional fields",
			config: promotion.Config{
				"imageRef":              "registry.example.com/image:tag",
				"destRef":               "registry.example.com/image:newtag",
				"insecureSkipTLSVerify": true,
				"annotations": map[string]any{
					"org.opencontainers.image.source": "https://github.com/example/repo",
				},
			},
		},
	}

	r := newOCIPusher(promotion.StepRunnerCapabilities{})
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
				ImageRef: srcImageRef,
				DestRef:  fmt.Sprintf("%s/test/image:v2.0.0", regHost),
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
				ImageRef: fmt.Sprintf("%s/test/image@%s", regHost, srcDigest.String()),
				DestRef:  fmt.Sprintf("%s/test/image:pinned", regHost),
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
				ImageRef: srcIndexRef,
				DestRef:  fmt.Sprintf("%s/test/multiarch:v2.0.0", regHost),
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
				ImageRef: srcImageRef,
				DestRef:  fmt.Sprintf("%s/test/image:annotated", regHost),
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
			name: "push index with annotations",
			cfg: builtin.OCIPushConfig{
				ImageRef: srcIndexRef,
				DestRef:  fmt.Sprintf("%s/test/multiarch:annotated", regHost),
				Annotations: map[string]string{
					"io.kargo.test": "true",
				},
			},
			assertions: func(t *testing.T, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, string(kargoapi.PromotionStepStatusSucceeded), string(result.Status))

				// Verify annotations on the pushed index.
				dstRef, parseErr := name.ParseReference(
					fmt.Sprintf("%s/test/multiarch:annotated", regHost),
				)
				require.NoError(t, parseErr)
				idx, getErr := remote.Index(dstRef)
				require.NoError(t, getErr)
				manifest, mErr := idx.IndexManifest()
				require.NoError(t, mErr)
				assert.Equal(t, "true", manifest.Annotations["io.kargo.test"])
			},
		},
		{
			name: "cross-repo push",
			cfg: builtin.OCIPushConfig{
				ImageRef: srcImageRef,
				DestRef:  fmt.Sprintf("%s/other/repo:latest", regHost),
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
				ImageRef: fmt.Sprintf("%s/nonexistent/image:v1.0.0", regHost),
				DestRef:  fmt.Sprintf("%s/test/image:copy", regHost),
			},
			assertions: func(t *testing.T, _ promotion.StepResult, err error) {
				assert.ErrorContains(t, err, "failed to get source artifact")
			},
		},
		{
			name: "invalid source reference",
			cfg: builtin.OCIPushConfig{
				ImageRef: "invalid::ref",
				DestRef:  fmt.Sprintf("%s/test/image:copy", regHost),
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
				ImageRef: srcImageRef,
				DestRef:  "invalid::ref",
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
				credsDB:      &credentials.FakeDB{},
				schemaLoader: getConfigSchemaLoader(stepKindOCIPush),
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

	runner := &ociPusher{}
	dstRef, err := name.ParseReference("localhost:5000/test:tag")
	require.NoError(t, err)

	_, err = runner.push(desc, dstRef, nil, nil)
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
				ImageRef: "registry.example.com/image:tag",
				DestRef:  "registry.example.com/image:newtag",
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
				credsDB:      tt.credsDB,
				schemaLoader: getConfigSchemaLoader(stepKindOCIPush),
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
		credsDB:      &credentials.FakeDB{},
		schemaLoader: getConfigSchemaLoader(stepKindOCIPush),
	}

	// Push without specifying annotations.
	result, err := runner.run(context.Background(), &promotion.StepContext{
		Project: "fake-project",
	}, builtin.OCIPushConfig{
		ImageRef: fmt.Sprintf("%s/test/annotated:v1", regHost),
		DestRef:  fmt.Sprintf("%s/test/annotated:v2", regHost),
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
		credsDB:      &credentials.FakeDB{},
		schemaLoader: getConfigSchemaLoader(stepKindOCIPush),
	}

	result, err := runner.run(context.Background(), &promotion.StepContext{
		Project: "fake-project",
	}, builtin.OCIPushConfig{
		ImageRef: fmt.Sprintf("%s/test/oci:v1", regHost),
		DestRef:  fmt.Sprintf("%s/test/oci:v2", regHost),
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
