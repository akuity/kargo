package mutate

import (
	"encoding/json"
	"fmt"
	"maps"
	"sync"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

// Annotations wraps an OCI artifact to overlay annotations on its manifest.
// The input must be a v1.Image or v1.ImageIndex.
//
// For a v1.Image, only manifestAnnotations are applied; indexAnnotations
// are ignored. For a v1.ImageIndex, indexAnnotations are applied to the
// index manifest and manifestAnnotations to each child image manifest.
//
// This function exists because go-containerregistry's mutate.Annotations has
// a Layers() implementation that enumerates layers via
// ConfigFile().RootFS.DiffIDs, which is empty for non-Docker OCI artifacts
// (e.g. Helm charts). This causes layers to be omitted from the push, leading
// to MANIFEST_BLOB_UNKNOWN errors on cross-repository pushes. The wrappers
// here delegate Layers() and all other methods to the base artifact, overriding
// only the manifest to include annotations.
func Annotations(
	f partial.WithRawManifest,
	indexAnnotations, manifestAnnotations map[string]string,
) (partial.WithRawManifest, error) {
	switch v := f.(type) {
	case v1.Image:
		if len(manifestAnnotations) == 0 {
			return v, nil
		}
		return &image{base: v, annotations: manifestAnnotations}, nil
	case v1.ImageIndex:
		if len(indexAnnotations) == 0 && len(manifestAnnotations) == 0 {
			return v, nil
		}
		return newIndex(v, indexAnnotations, manifestAnnotations)
	default:
		return nil, fmt.Errorf("unsupported type for annotation: %T", f)
	}
}

// image wraps a v1.Image to overlay annotations on its manifest. All methods
// delegate to the base image; only the manifest is modified.
type image struct {
	base        v1.Image
	annotations map[string]string

	computed bool
	manifest *v1.Manifest
	sync.Mutex
}

var _ v1.Image = (*image)(nil)

func (i *image) compute() error {
	i.Lock()
	defer i.Unlock()
	if i.computed {
		return nil
	}
	m, err := i.base.Manifest()
	if err != nil {
		return err
	}
	manifest := m.DeepCopy()
	if manifest.Annotations == nil {
		manifest.Annotations = map[string]string{}
	}
	maps.Copy(manifest.Annotations, i.annotations)
	i.manifest = manifest
	i.computed = true
	return nil
}

func (i *image) MediaType() (types.MediaType, error) {
	return i.base.MediaType()
}

func (i *image) Layers() ([]v1.Layer, error) {
	return i.base.Layers()
}

func (i *image) ConfigName() (v1.Hash, error) {
	return i.base.ConfigName()
}

func (i *image) ConfigFile() (*v1.ConfigFile, error) {
	return i.base.ConfigFile()
}

func (i *image) RawConfigFile() ([]byte, error) {
	return i.base.RawConfigFile()
}

func (i *image) LayerByDigest(h v1.Hash) (v1.Layer, error) {
	return i.base.LayerByDigest(h)
}

func (i *image) LayerByDiffID(h v1.Hash) (v1.Layer, error) {
	return i.base.LayerByDiffID(h)
}

func (i *image) Manifest() (*v1.Manifest, error) {
	if err := i.compute(); err != nil {
		return nil, err
	}
	return i.manifest.DeepCopy(), nil
}

func (i *image) RawManifest() ([]byte, error) {
	if err := i.compute(); err != nil {
		return nil, err
	}
	return json.Marshal(i.manifest)
}

func (i *image) Digest() (v1.Hash, error) {
	if err := i.compute(); err != nil {
		return v1.Hash{}, err
	}
	return partial.Digest(i)
}

func (i *image) Size() (int64, error) {
	if err := i.compute(); err != nil {
		return -1, err
	}
	return partial.Size(i)
}

// annotatedChild holds the precomputed digest and size of an annotated child
// image, used to rewrite index manifest descriptors.
type annotatedChild struct {
	digest v1.Hash
	size   int64
}

// index wraps a v1.ImageIndex to overlay scoped annotations. Index-scoped
// annotations are applied to the index manifest; manifest-scoped annotations
// are applied to each child image manifest via Image(). When manifest
// annotations are present, child descriptors are rewritten with updated
// digests/sizes to stay consistent with the annotated content.
type index struct {
	base                v1.ImageIndex
	indexAnnotations    map[string]string
	manifestAnnotations map[string]string
	// childMap maps original child digest → annotated digest/size.
	// Populated eagerly by newIndex when manifestAnnotations is non-empty.
	childMap map[v1.Hash]annotatedChild

	computed bool
	manifest *v1.IndexManifest
	sync.Mutex
}

var _ v1.ImageIndex = (*index)(nil)

// newIndex creates an index wrapper. When manifest annotations are provided,
// it eagerly computes the annotated digest and size for each child image so
// that IndexManifest() and Image() stay consistent.
func newIndex(
	base v1.ImageIndex,
	indexAnnotations, manifestAnnotations map[string]string,
) (*index, error) {
	idx := &index{
		base:                base,
		indexAnnotations:    indexAnnotations,
		manifestAnnotations: manifestAnnotations,
		childMap:            make(map[v1.Hash]annotatedChild),
	}
	if len(manifestAnnotations) > 0 {
		m, err := base.IndexManifest()
		if err != nil {
			return nil, fmt.Errorf("failed to get index manifest: %w", err)
		}
		for _, desc := range m.Manifests {
			if !desc.MediaType.IsImage() {
				continue
			}
			img, err := base.Image(desc.Digest)
			if err != nil {
				return nil, fmt.Errorf(
					"failed to resolve child image %s: %w", desc.Digest, err,
				)
			}
			wrapped := &image{base: img, annotations: manifestAnnotations}
			d, err := wrapped.Digest()
			if err != nil {
				return nil, fmt.Errorf(
					"failed to compute annotated digest for %s: %w",
					desc.Digest, err,
				)
			}
			s, err := wrapped.Size()
			if err != nil {
				return nil, fmt.Errorf(
					"failed to compute annotated size for %s: %w",
					desc.Digest, err,
				)
			}
			idx.childMap[desc.Digest] = annotatedChild{digest: d, size: s}
		}
	}
	return idx, nil
}

func (a *index) compute() error {
	a.Lock()
	defer a.Unlock()
	if a.computed {
		return nil
	}
	m, err := a.base.IndexManifest()
	if err != nil {
		return err
	}
	manifest := m.DeepCopy()
	if len(a.indexAnnotations) > 0 {
		if manifest.Annotations == nil {
			manifest.Annotations = map[string]string{}
		}
		maps.Copy(manifest.Annotations, a.indexAnnotations)
	}
	// Rewrite child descriptors with annotated digests/sizes.
	for i, desc := range manifest.Manifests {
		if child, ok := a.childMap[desc.Digest]; ok {
			manifest.Manifests[i].Digest = child.digest
			manifest.Manifests[i].Size = child.size
		}
	}
	a.manifest = manifest
	a.computed = true
	return nil
}

func (a *index) MediaType() (types.MediaType, error) {
	return a.base.MediaType()
}

func (a *index) IndexManifest() (*v1.IndexManifest, error) {
	if err := a.compute(); err != nil {
		return nil, err
	}
	return a.manifest.DeepCopy(), nil
}

func (a *index) RawManifest() ([]byte, error) {
	if err := a.compute(); err != nil {
		return nil, err
	}
	return json.Marshal(a.manifest)
}

func (a *index) Digest() (v1.Hash, error) {
	if err := a.compute(); err != nil {
		return v1.Hash{}, err
	}
	return partial.Digest(a)
}

func (a *index) Size() (int64, error) {
	if err := a.compute(); err != nil {
		return -1, err
	}
	return partial.Size(a)
}

func (a *index) Image(h v1.Hash) (v1.Image, error) {
	// Reverse-map: if h is an annotated digest, find the original.
	origHash := h
	for orig, child := range a.childMap {
		if child.digest == h {
			origHash = orig
			break
		}
	}
	img, err := a.base.Image(origHash)
	if err != nil {
		return nil, err
	}
	if len(a.manifestAnnotations) > 0 {
		return &image{base: img, annotations: a.manifestAnnotations}, nil
	}
	return img, nil
}

func (a *index) ImageIndex(h v1.Hash) (v1.ImageIndex, error) {
	return a.base.ImageIndex(h)
}
