package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/promotion"
	builtin "github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const stepKindOCIPush = "oci-push"

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name: stepKindOCIPush,
			Metadata: promotion.StepRunnerMetadata{
				RequiredCapabilities: []promotion.StepRunnerCapability{
					promotion.StepCapabilityAccessCredentials,
				},
			},
			Value: newOCIPusher,
		},
	)
}

// ociPusher is an implementation of the promotion.StepRunner interface that
// copies/retags OCI artifacts (container images and Helm charts) between
// registries.
type ociPusher struct {
	schemaLoader gojsonschema.JSONLoader
	credsDB      credentials.Database
}

// newOCIPusher returns an implementation of the promotion.StepRunner interface
// that pushes OCI artifacts to a registry. It uses the provided credentials
// database to authenticate with source and destination registries.
func newOCIPusher(caps promotion.StepRunnerCapabilities) promotion.StepRunner {
	return &ociPusher{
		credsDB:      caps.CredsDB,
		schemaLoader: getConfigSchemaLoader(stepKindOCIPush),
	}
}

// Run implements the promotion.StepRunner interface.
func (p *ociPusher) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	cfg, err := p.convert(stepCtx.Config)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &promotion.TerminalError{Err: err}
	}
	return p.run(ctx, stepCtx, cfg)
}

// convert validates the ociPusher configuration against a JSON schema and
// converts it into a builtin.OCIPushConfig struct.
func (p *ociPusher) convert(cfg promotion.Config) (builtin.OCIPushConfig, error) {
	return validateAndConvert[builtin.OCIPushConfig](p.schemaLoader, cfg, stepKindOCIPush)
}

// run executes the ociPusher step with the provided configuration.
func (p *ociPusher) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.OCIPushConfig,
) (promotion.StepResult, error) {
	srcRef, srcCredType, err := parseOCIReference(cfg.ImageRef)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed},
			&promotion.TerminalError{
				Err: fmt.Errorf("failed to parse source reference %q: %w", cfg.ImageRef, err),
			}
	}

	dstRef, dstCredType, err := parseOCIReference(cfg.DestRef)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed},
			&promotion.TerminalError{
				Err: fmt.Errorf("failed to parse destination reference %q: %w", cfg.DestRef, err),
			}
	}

	srcOpts, err := buildOCIRemoteOptions(
		ctx, p.credsDB, stepCtx.Project, srcRef, srcCredType, cfg.InsecureSkipTLSVerify,
	)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	dstOpts, err := buildOCIRemoteOptions(
		ctx, p.credsDB, stepCtx.Project, dstRef, dstCredType, cfg.InsecureSkipTLSVerify,
	)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	desc, err := remote.Get(srcRef, srcOpts...)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to get source artifact %q: %w", cfg.ImageRef, err)
	}

	digest, err := p.push(desc, dstRef, cfg.Annotations, dstOpts)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	// Extract tag from destination reference if available.
	var tag string
	if t, ok := dstRef.(name.Tag); ok {
		tag = t.TagStr()
	}

	return promotion.StepResult{
		Status: kargoapi.PromotionStepStatusSucceeded,
		Output: map[string]any{
			"image":  dstRef.String(),
			"digest": digest.String(),
			"tag":    tag,
		},
	}, nil
}

// push pushes the described artifact to the destination reference, optionally
// applying annotations to the manifest.
func (p *ociPusher) push(
	desc *remote.Descriptor,
	dstRef name.Reference,
	annotations map[string]string,
	dstOpts []remote.Option,
) (v1.Hash, error) {
	switch {
	case desc.MediaType.IsIndex():
		return p.pushIndex(desc, dstRef, annotations, dstOpts)
	case desc.MediaType.IsImage():
		return p.pushImage(desc, dstRef, annotations, dstOpts)
	default:
		return v1.Hash{}, &promotion.TerminalError{
			Err: fmt.Errorf("unsupported media type %q", desc.MediaType),
		}
	}
}

// annotationScopes holds annotations separated by their target scope.
// Keys prefixed with "index:" target the image index manifest, keys prefixed
// with "manifest:" or unprefixed target image manifests.
type annotationScopes struct {
	index    map[string]string // applied to the image index manifest
	manifest map[string]string // applied to each image manifest
}

// parseAnnotationScopes splits annotation keys by their scope prefix.
// Keys prefixed with "index:" are routed to the index manifest, keys prefixed
// with "manifest:" or unprefixed are routed to image manifests.
func parseAnnotationScopes(annotations map[string]string) annotationScopes {
	scopes := annotationScopes{
		index:    make(map[string]string),
		manifest: make(map[string]string),
	}
	for k, v := range annotations {
		switch {
		case strings.HasPrefix(k, "index:"):
			scopes.index[strings.TrimPrefix(k, "index:")] = v
		case strings.HasPrefix(k, "manifest:"):
			scopes.manifest[strings.TrimPrefix(k, "manifest:")] = v
		default:
			scopes.manifest[k] = v
		}
	}
	return scopes
}

// pushImage pushes a single image to the destination.
func (p *ociPusher) pushImage(
	desc *remote.Descriptor,
	dstRef name.Reference,
	annotations map[string]string,
	dstOpts []remote.Option,
) (v1.Hash, error) {
	img, err := desc.Image()
	if err != nil {
		return v1.Hash{}, fmt.Errorf("failed to resolve source image: %w", err)
	}

	scopes := parseAnnotationScopes(annotations)
	if len(scopes.manifest) > 0 {
		img = &annotatedImage{Image: img, annotations: scopes.manifest}
	}

	if err = remote.Write(dstRef, img, dstOpts...); err != nil {
		return v1.Hash{}, fmt.Errorf("failed to push image to %q: %w", dstRef.String(), err)
	}

	digest, err := img.Digest()
	if err != nil {
		return v1.Hash{}, fmt.Errorf("failed to get digest of pushed image: %w", err)
	}

	return digest, nil
}

// pushIndex pushes an image index (multi-arch manifest) to the destination.
func (p *ociPusher) pushIndex(
	desc *remote.Descriptor,
	dstRef name.Reference,
	annotations map[string]string,
	dstOpts []remote.Option,
) (v1.Hash, error) {
	idx, err := desc.ImageIndex()
	if err != nil {
		return v1.Hash{}, fmt.Errorf("failed to resolve source image index: %w", err)
	}

	scopes := parseAnnotationScopes(annotations)
	if len(scopes.index) > 0 || len(scopes.manifest) > 0 {
		idx, err = newAnnotatedIndex(idx, scopes.index, scopes.manifest)
		if err != nil {
			return v1.Hash{}, fmt.Errorf("failed to prepare annotated index: %w", err)
		}
	}

	if err = remote.WriteIndex(dstRef, idx, dstOpts...); err != nil {
		return v1.Hash{}, fmt.Errorf("failed to push image index to %q: %w", dstRef.String(), err)
	}

	digest, err := idx.Digest()
	if err != nil {
		return v1.Hash{}, fmt.Errorf("failed to get digest of pushed image index: %w", err)
	}

	return digest, nil
}

// annotatedImage wraps a v1.Image to add annotations to its manifest without
// using mutate.Annotations. We avoid mutate.Annotations because its internal
// Layers() implementation enumerates layers via ConfigFile().RootFS.DiffIDs,
// which is empty for non-Docker OCI artifacts (e.g. Helm charts). This causes
// layers to be omitted from the push, leading to MANIFEST_BLOB_UNKNOWN errors
// on cross-repository pushes. This wrapper delegates Layers() and all other
// methods to the base image, overriding only the manifest to include
// annotations.
type annotatedImage struct {
	v1.Image
	annotations map[string]string
}

func (a *annotatedImage) Manifest() (*v1.Manifest, error) {
	m, err := a.Image.Manifest()
	if err != nil {
		return nil, err
	}
	m = m.DeepCopy()
	if m.Annotations == nil {
		m.Annotations = map[string]string{}
	}
	maps.Copy(m.Annotations, a.annotations)
	return m, nil
}

func (a *annotatedImage) RawManifest() ([]byte, error) {
	m, err := a.Manifest()
	if err != nil {
		return nil, err
	}
	return json.Marshal(m)
}

func (a *annotatedImage) Digest() (v1.Hash, error) {
	return partial.Digest(a)
}

func (a *annotatedImage) Size() (int64, error) {
	return partial.Size(a)
}

// annotatedChild holds the precomputed digest and size of an annotated child
// image, used to rewrite index manifest descriptors.
type annotatedChild struct {
	digest v1.Hash
	size   int64
}

// annotatedIndex wraps a v1.ImageIndex to add scoped annotations. Index-scoped
// annotations are applied to the index manifest; manifest-scoped annotations
// are applied to each child image manifest via the Image() method. When
// manifest annotations are present, child descriptors in the index manifest are
// rewritten with new digests/sizes to stay consistent with the annotated
// content. See annotatedImage for rationale on avoiding mutate.Annotations.
// The base field is unexported because v1.ImageIndex has an ImageIndex() method
// that would collide with an embedded field of the same name.
type annotatedIndex struct {
	base                v1.ImageIndex
	indexAnnotations    map[string]string
	manifestAnnotations map[string]string
	// childMap maps original child digest → annotated digest/size.
	// Populated by newAnnotatedIndex when manifestAnnotations is non-empty.
	childMap map[v1.Hash]annotatedChild
}

// newAnnotatedIndex creates an annotatedIndex wrapper. When manifest
// annotations are provided, it eagerly computes the annotated digest and size
// for each child image so that IndexManifest() and Image() stay consistent.
func newAnnotatedIndex(
	base v1.ImageIndex,
	indexAnnotations, manifestAnnotations map[string]string,
) (*annotatedIndex, error) {
	ai := &annotatedIndex{
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
			wrapped := &annotatedImage{
				Image: img, annotations: manifestAnnotations,
			}
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
			ai.childMap[desc.Digest] = annotatedChild{digest: d, size: s}
		}
	}
	return ai, nil
}

func (a *annotatedIndex) MediaType() (types.MediaType, error) {
	return a.base.MediaType()
}

func (a *annotatedIndex) IndexManifest() (*v1.IndexManifest, error) {
	m, err := a.base.IndexManifest()
	if err != nil {
		return nil, err
	}
	m = m.DeepCopy()
	if len(a.indexAnnotations) > 0 {
		if m.Annotations == nil {
			m.Annotations = map[string]string{}
		}
		maps.Copy(m.Annotations, a.indexAnnotations)
	}
	// Rewrite child descriptors with annotated digests/sizes.
	for i, desc := range m.Manifests {
		if child, ok := a.childMap[desc.Digest]; ok {
			m.Manifests[i].Digest = child.digest
			m.Manifests[i].Size = child.size
		}
	}
	return m, nil
}

func (a *annotatedIndex) RawManifest() ([]byte, error) {
	m, err := a.IndexManifest()
	if err != nil {
		return nil, err
	}
	return json.Marshal(m)
}

func (a *annotatedIndex) Digest() (v1.Hash, error) {
	return partial.Digest(a)
}

func (a *annotatedIndex) Size() (int64, error) {
	return partial.Size(a)
}

func (a *annotatedIndex) Image(h v1.Hash) (v1.Image, error) {
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
		return &annotatedImage{Image: img, annotations: a.manifestAnnotations}, nil
	}
	return img, nil
}

func (a *annotatedIndex) ImageIndex(h v1.Hash) (v1.ImageIndex, error) {
	return a.base.ImageIndex(h)
}
