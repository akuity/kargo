package mutate

import (
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAnnotations_dockerImage verifies that Annotations merges annotations onto
// a Docker-format image manifest, changes the digest, and does not mutate the
// base image.
func TestAnnotations_dockerImage(t *testing.T) {
	base, err := random.Image(256, 2)
	require.NoError(t, err)

	baseManifest, err := base.Manifest()
	require.NoError(t, err)
	baseDigest, err := base.Digest()
	require.NoError(t, err)

	annotations := map[string]string{
		"io.kargo.test": "true",
		"org.example":   "value",
	}

	result, err := Annotations(base, nil, annotations)
	require.NoError(t, err)
	img, ok := result.(v1.Image)
	require.True(t, ok, "expected v1.Image")

	// Annotations appear on the annotated manifest.
	m, err := img.Manifest()
	require.NoError(t, err)
	assert.Equal(t, "true", m.Annotations["io.kargo.test"])
	assert.Equal(t, "value", m.Annotations["org.example"])

	// Digest changes.
	d, err := img.Digest()
	require.NoError(t, err)
	assert.NotEqual(t, baseDigest, d)

	// Digest is stable across calls (compute-once).
	d2, err := img.Digest()
	require.NoError(t, err)
	assert.Equal(t, d, d2)

	// Base manifest is not mutated.
	baseAfter, err := base.Manifest()
	require.NoError(t, err)
	assert.Equal(t, baseManifest.Annotations, baseAfter.Annotations)
}

// TestAnnotations_ociImage verifies that Annotations works on an OCI-format
// image with empty DiffIDs (e.g. Helm charts), a regression case for
// go-containerregistry's mutate.Annotations which would cause
// MANIFEST_BLOB_UNKNOWN errors on cross-repo pushes.
func TestAnnotations_ociImage(t *testing.T) {
	base := empty.Image

	annotations := map[string]string{
		"io.kargo.helm": "chart",
	}

	result, err := Annotations(base, nil, annotations)
	require.NoError(t, err)
	img := result.(v1.Image) //nolint:forcetypeassert

	m, err := img.Manifest()
	require.NoError(t, err)
	assert.Equal(t, "chart", m.Annotations["io.kargo.helm"])

	// Layers should delegate to base and not break.
	layers, err := img.Layers()
	require.NoError(t, err)
	assert.Empty(t, layers)
}

// TestAnnotations_imageLayers verifies that Layers() delegates to the base
// image rather than enumerating via ConfigFile.RootFS.DiffIDs. This ensures
// all layers are preserved during cross-repository pushes.
func TestAnnotations_imageLayers(t *testing.T) {
	base, err := random.Image(256, 3)
	require.NoError(t, err)

	baseLayers, err := base.Layers()
	require.NoError(t, err)
	require.Len(t, baseLayers, 3)

	result, err := Annotations(base, nil, map[string]string{"k": "v"})
	require.NoError(t, err)
	img := result.(v1.Image) //nolint:forcetypeassert

	layers, err := img.Layers()
	require.NoError(t, err)
	require.Len(t, layers, 3)

	// Verify layer digests match the base.
	for i, l := range layers {
		d, dErr := l.Digest()
		require.NoError(t, dErr)
		bd, bdErr := baseLayers[i].Digest()
		require.NoError(t, bdErr)
		assert.Equal(t, bd, d)
	}
}

// TestAnnotations_index verifies that Annotations applies index annotations to
// the index manifest and manifest annotations to each child image, rewriting
// child descriptors with updated digests.
func TestAnnotations_index(t *testing.T) {
	base, err := random.Index(256, 1, 2)
	require.NoError(t, err)

	idxAnns := map[string]string{"io.kargo.idx": "index-val"}
	mfstAnns := map[string]string{"io.kargo.mfst": "manifest-val"}

	result, err := Annotations(base, idxAnns, mfstAnns)
	require.NoError(t, err)
	idx, ok := result.(v1.ImageIndex)
	require.True(t, ok, "expected v1.ImageIndex")

	// Index annotations appear on the index manifest.
	im, err := idx.IndexManifest()
	require.NoError(t, err)
	assert.Equal(t, "index-val", im.Annotations["io.kargo.idx"])
	assert.Empty(t, im.Annotations["io.kargo.mfst"])

	// Child manifests have manifest annotations.
	for _, desc := range im.Manifests {
		img, imgErr := idx.Image(desc.Digest)
		require.NoError(t, imgErr)
		m, mErr := img.Manifest()
		require.NoError(t, mErr)
		assert.Equal(t, "manifest-val", m.Annotations["io.kargo.mfst"])
		assert.Empty(t, m.Annotations["io.kargo.idx"])
	}

	// Digest is stable (compute-once).
	d1, err := idx.Digest()
	require.NoError(t, err)
	d2, err := idx.Digest()
	require.NoError(t, err)
	assert.Equal(t, d1, d2)
}

// TestAnnotations_indexOnly verifies that when only index annotations are
// provided, child image digests remain unchanged.
func TestAnnotations_indexOnly(t *testing.T) {
	base, err := random.Index(256, 1, 2)
	require.NoError(t, err)

	result, err := Annotations(base, map[string]string{"k": "v"}, nil)
	require.NoError(t, err)
	idx := result.(v1.ImageIndex) //nolint:forcetypeassert

	im, err := idx.IndexManifest()
	require.NoError(t, err)
	assert.Equal(t, "v", im.Annotations["k"])

	// Children should not have extra annotations.
	baseIM, err := base.IndexManifest()
	require.NoError(t, err)
	for _, desc := range baseIM.Manifests {
		// Descriptor digests should be unchanged (no manifest annotations applied).
		found := false
		for _, d := range im.Manifests {
			if d.Digest == desc.Digest {
				found = true
				break
			}
		}
		assert.True(t, found, "child digest %s should be unchanged", desc.Digest)
	}
}

// TestAnnotations_indexManifestOnly verifies that when only manifest
// annotations are provided, no index-level annotations are added and child
// images receive the annotations.
func TestAnnotations_indexManifestOnly(t *testing.T) {
	base, err := random.Index(256, 1, 2)
	require.NoError(t, err)

	result, err := Annotations(base, nil, map[string]string{"k": "v"})
	require.NoError(t, err)
	idx := result.(v1.ImageIndex) //nolint:forcetypeassert

	im, err := idx.IndexManifest()
	require.NoError(t, err)

	// No index-level annotations added.
	assert.Empty(t, im.Annotations["k"])

	// Children should have the annotation.
	for _, desc := range im.Manifests {
		img, imgErr := idx.Image(desc.Digest)
		require.NoError(t, imgErr)
		m, mErr := img.Manifest()
		require.NoError(t, mErr)
		assert.Equal(t, "v", m.Annotations["k"])
	}
}

// TestAnnotations_typeCheck verifies that the unified entrypoint returns the
// correct concrete type for images and indexes, and returns an error for
// unsupported types.
func TestAnnotations_typeCheck(t *testing.T) {
	t.Run("image returns v1.Image", func(t *testing.T) {
		img, err := random.Image(256, 1)
		require.NoError(t, err)
		result, err := Annotations(img, nil, map[string]string{"k": "v"})
		require.NoError(t, err)
		_, ok := result.(v1.Image)
		assert.True(t, ok)
	})

	t.Run("index returns v1.ImageIndex", func(t *testing.T) {
		idx, err := random.Index(256, 1, 1)
		require.NoError(t, err)
		result, err := Annotations(idx, map[string]string{"k": "v"}, nil)
		require.NoError(t, err)
		_, ok := result.(v1.ImageIndex)
		assert.True(t, ok)
	})

	t.Run("unsupported type returns error", func(t *testing.T) {
		raw := &fakeRawManifest{}
		_, err := Annotations(raw, nil, map[string]string{"k": "v"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported type")
	})
}

// TestAnnotations_noAnnotations verifies that when both annotation maps are
// empty, the base artifact is returned unchanged (no wrapping).
func TestAnnotations_noAnnotations(t *testing.T) {
	t.Run("image returned unchanged", func(t *testing.T) {
		img, err := random.Image(256, 1)
		require.NoError(t, err)
		result, err := Annotations(img, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, img, result, "should return base image unchanged")
	})

	t.Run("index returned unchanged", func(t *testing.T) {
		idx, err := random.Index(256, 1, 1)
		require.NoError(t, err)
		result, err := Annotations(idx, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, idx, result, "should return base index unchanged")
	})
}

// fakeRawManifest is a type that implements partial.WithRawManifest but is
// neither a v1.Image nor a v1.ImageIndex, used to test the type check error.
type fakeRawManifest struct{}

var _ partial.WithRawManifest = (*fakeRawManifest)(nil)

func (f *fakeRawManifest) RawManifest() ([]byte, error) {
	return []byte(`{}`), nil
}
