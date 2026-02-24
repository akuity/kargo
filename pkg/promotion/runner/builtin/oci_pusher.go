package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"

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

	if len(annotations) > 0 {
		img = &annotatedImage{Image: img, annotations: annotations}
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

	if len(annotations) > 0 {
		idx = &annotatedIndex{base: idx, annotations: annotations}
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

// annotatedIndex wraps a v1.ImageIndex to add annotations to its index
// manifest. See annotatedImage for rationale. The base field is unexported
// because v1.ImageIndex has an ImageIndex() method that would collide with
// an embedded field of the same name.
type annotatedIndex struct {
	base        v1.ImageIndex
	annotations map[string]string
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
	if m.Annotations == nil {
		m.Annotations = map[string]string{}
	}
	maps.Copy(m.Annotations, a.annotations)
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
	return a.base.Image(h)
}

func (a *annotatedIndex) ImageIndex(h v1.Hash) (v1.ImageIndex, error) {
	return a.base.ImageIndex(h)
}
