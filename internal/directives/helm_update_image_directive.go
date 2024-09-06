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
	return &helmUpdateImageDirective{
		schemaLoader: getConfigSchemaLoader("helm-update-image"),
	}
}

// Name implements the Directive interface.
func (d *helmUpdateImageDirective) Name() string {
	return "helm-update-image"
}

// Run implements the Directive interface.
func (d *helmUpdateImageDirective) Run(ctx context.Context, stepCtx *StepContext) (Result, error) {
	// Validate the configuration against the JSON Schema
	if err := validate(
		d.schemaLoader,
		gojsonschema.NewGoLoader(stepCtx.Config),
		d.Name(),
	); err != nil {
		return ResultFailure, err
	}

	// Convert the configuration into a typed struct
	cfg, err := configToStruct[HelmUpdateImageConfig](stepCtx.Config)
	if err != nil {
		return ResultFailure, fmt.Errorf("could not convert config into %s config: %w", d.Name(), err)
	}

	return d.run(ctx, stepCtx, cfg)
}

func (d *helmUpdateImageDirective) run(
	ctx context.Context,
	stepCtx *StepContext,
	cfg HelmUpdateImageConfig,
) (Result, error) {
	changes := make(map[string]string, len(cfg.Images))
	for _, image := range cfg.Images {
		var desiredOrigin *kargoapi.FreightOrigin
		if image.FromOrigin != nil {
			desiredOrigin = &kargoapi.FreightOrigin{
				Kind: kargoapi.FreightOriginKind(image.FromOrigin.Kind),
				Name: image.FromOrigin.Name,
			}
		}

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
			return ResultFailure, err
		}

		if targetImage == nil {
			continue
		}

		switch image.Value {
		case ImageAndTag:
			changes[image.Key] = fmt.Sprintf("%s:%s", targetImage.RepoURL, targetImage.Tag)
		case Tag:
			changes[image.Key] = targetImage.Tag
		case ImageAndDigest:
			changes[image.Key] = fmt.Sprintf("%s@%s", targetImage.RepoURL, targetImage.Digest)
		case Digest:
			changes[image.Key] = targetImage.Digest
		}
	}

	if len(changes) > 0 {
		absValuesFile, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.Path)
		if err != nil {
			return ResultFailure, fmt.Errorf("error joining path %q: %w", cfg.Path, err)
		}
		if err = libYAML.SetStringsInFile(absValuesFile, changes); err != nil {
			return ResultFailure, fmt.Errorf("error updating image references in values file %q: %w", cfg.Path, err)
		}
	}

	return ResultSuccess, nil
}
