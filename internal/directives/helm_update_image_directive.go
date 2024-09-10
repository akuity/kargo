package directives

import (
	"context"
	"fmt"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/freight"
	libYAML "github.com/akuity/kargo/internal/yaml"
)

func init() {
	// Register the helm-update-image directive with the builtins registry.
	builtins.RegisterDirective(newHelmUpdateImageDirective(), &DirectivePermissions{
		AllowKargoClient: true,
	})
}

type helmUpdateImageDirective struct {
	schemaLoader gojsonschema.JSONLoader
}

// newHelmUpdateImageDirective creates a new helm-update-image directive.
func newHelmUpdateImageDirective() Directive {
	d := &helmUpdateImageDirective{}
	d.schemaLoader = getConfigSchemaLoader(d.Name())
	return d
}

// Name implements the Directive interface.
func (d *helmUpdateImageDirective) Name() string {
	return "helm-update-image"
}

// Run implements the Directive interface.
func (d *helmUpdateImageDirective) Run(ctx context.Context, stepCtx *StepContext) (Result, error) {
	failure := Result{Status: StatusFailure}

	// Validate the configuration against the JSON Schema
	if err := validate(
		d.schemaLoader,
		gojsonschema.NewGoLoader(stepCtx.Config),
		d.Name(),
	); err != nil {
		return failure, err
	}

	// Convert the configuration into a typed struct
	cfg, err := configToStruct[HelmUpdateImageConfig](stepCtx.Config)
	if err != nil {
		return failure, fmt.Errorf("could not convert config into %s config: %w", d.Name(), err)
	}

	return d.run(ctx, stepCtx, cfg)
}

func (d *helmUpdateImageDirective) run(
	ctx context.Context,
	stepCtx *StepContext,
	cfg HelmUpdateImageConfig,
) (Result, error) {
	changes, err := d.generateImageUpdates(ctx, stepCtx, cfg)
	if err != nil {
		return Result{Status: StatusFailure}, fmt.Errorf("failed to generate image updates: %w", err)
	}

	if len(changes) > 0 {
		if err := d.updateValuesFile(stepCtx.WorkDir, cfg.Path, changes); err != nil {
			return Result{Status: StatusFailure}, fmt.Errorf("values file update failed: %w", err)
		}
	}

	return Result{Status: StatusSuccess}, nil
}

func (d *helmUpdateImageDirective) generateImageUpdates(
	ctx context.Context,
	stepCtx *StepContext,
	cfg HelmUpdateImageConfig,
) (map[string]string, error) {
	changes := make(map[string]string, len(cfg.Images))
	for _, image := range cfg.Images {
		desiredOrigin := d.getDesiredOrigin(image.FromOrigin)

		targetImage, err := freight.FindImage(
			ctx,
			stepCtx.KargoClient,
			stepCtx.Project,
			stepCtx.FreightRequests,
			desiredOrigin,
			stepCtx.Freight.References(),
			image.Image,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to find image %s: %w", image.Image, err)
		}

		if targetImage == nil {
			continue
		}

		value, err := d.getImageValue(targetImage, image.Value)
		if err != nil {
			return nil, err
		}

		changes[image.Key] = value
	}
	return changes, nil
}

func (d *helmUpdateImageDirective) getDesiredOrigin(fromOrigin *ChartFromOrigin) *kargoapi.FreightOrigin {
	if fromOrigin == nil {
		return nil
	}
	return &kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKind(fromOrigin.Kind),
		Name: fromOrigin.Name,
	}
}

func (d *helmUpdateImageDirective) getImageValue(image *kargoapi.Image, valueType Value) (string, error) {
	switch valueType {
	case ImageAndTag:
		return fmt.Sprintf("%s:%s", image.RepoURL, image.Tag), nil
	case Tag:
		return image.Tag, nil
	case ImageAndDigest:
		return fmt.Sprintf("%s@%s", image.RepoURL, image.Digest), nil
	case Digest:
		return image.Digest, nil
	default:
		return "", fmt.Errorf("unknown image value type %q", valueType)
	}
}

func (d *helmUpdateImageDirective) updateValuesFile(workDir, path string, changes map[string]string) error {
	absValuesFile, err := securejoin.SecureJoin(workDir, path)
	if err != nil {
		return fmt.Errorf("error joining path %q: %w", path, err)
	}
	if err := libYAML.SetStringsInFile(absValuesFile, changes); err != nil {
		return fmt.Errorf("error updating image references in values file %q: %w", path, err)
	}
	return nil
}
