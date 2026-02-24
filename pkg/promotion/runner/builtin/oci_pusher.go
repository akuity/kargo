package builtin

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
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

// push pushes the described artifact to the destination reference, optionally
// applying scoped annotations to the manifest.
func (p *ociPusher) push(
	desc *remote.Descriptor,
	dstRef name.Reference,
	annotations map[string]string,
	dstOpts []remote.Option,
) (v1.Hash, error) {
	scopes := parseAnnotationScopes(annotations)

	switch {
	case desc.MediaType.IsImage():
		img, err := desc.Image()
		if err != nil {
			return v1.Hash{}, fmt.Errorf("failed to resolve source image: %w", err)
		}
		annotated, err := Annotations(img, nil, scopes.manifest)
		if err != nil {
			return v1.Hash{}, fmt.Errorf("failed to annotate image: %w", err)
		}
		img = annotated.(v1.Image) //nolint:forcetypeassert
		if err = remote.Write(dstRef, img, dstOpts...); err != nil {
			return v1.Hash{}, fmt.Errorf("failed to push image to %q: %w", dstRef.String(), err)
		}
		return img.Digest()

	case desc.MediaType.IsIndex():
		idx, err := desc.ImageIndex()
		if err != nil {
			return v1.Hash{}, fmt.Errorf("failed to resolve source image index: %w", err)
		}
		annotated, err := Annotations(idx, scopes.index, scopes.manifest)
		if err != nil {
			return v1.Hash{}, fmt.Errorf("failed to annotate index: %w", err)
		}
		idx = annotated.(v1.ImageIndex) //nolint:forcetypeassert
		if err = remote.WriteIndex(dstRef, idx, dstOpts...); err != nil {
			return v1.Hash{}, fmt.Errorf("failed to push image index to %q: %w", dstRef.String(), err)
		}
		return idx.Digest()

	default:
		return v1.Hash{}, &promotion.TerminalError{
			Err: fmt.Errorf("unsupported media type %q", desc.MediaType),
		}
	}
}
