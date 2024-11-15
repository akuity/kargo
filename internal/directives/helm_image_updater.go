package directives

import (
	"context"
	"fmt"
	"slices"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/freight"
	libYAML "github.com/akuity/kargo/internal/yaml"
)

func init() {
	builtins.RegisterPromotionStepRunner(
		newHelmImageUpdater(),
		&StepRunnerPermissions{
			AllowKargoClient: true,
		},
	)
}

// helmImageUpdater is an implementation of the PromotionStepRunner interface
// that updates image references in a Helm values file.
type helmImageUpdater struct {
	schemaLoader gojsonschema.JSONLoader
}

// newHelmImageUpdater returns an implementation of the PromotionStepRunner
// interface that updates image references in a Helm values file.
func newHelmImageUpdater() PromotionStepRunner {
	r := &helmImageUpdater{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the PromotionStepRunner HealthCheckStepRunner interface.
func (h *helmImageUpdater) Name() string {
	return "helm-update-image"
}

// RunPromotionStep implements the PromotionStepRunner interface.
func (h *helmImageUpdater) RunPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
) (PromotionStepResult, error) {
	failure := PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}

	if err := h.validate(stepCtx.Config); err != nil {
		return failure, err
	}

	// Convert the configuration into a typed struct
	cfg, err := ConfigToStruct[HelmUpdateImageConfig](stepCtx.Config)
	if err != nil {
		return failure, fmt.Errorf("could not convert config into %s config: %w", h.Name(), err)
	}

	return h.runPromotionStep(ctx, stepCtx, cfg)
}

// validate validates helmImageUpdater configuration against a JSON schema.
func (h *helmImageUpdater) validate(cfg Config) error {
	return validate(h.schemaLoader, gojsonschema.NewGoLoader(cfg), h.Name())
}

func (h *helmImageUpdater) runPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
	cfg HelmUpdateImageConfig,
) (PromotionStepResult, error) {
	updates, err := h.generateImageUpdates(ctx, stepCtx, cfg)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("failed to generate image updates: %w", err)
	}

	result := PromotionStepResult{Status: kargoapi.PromotionPhaseSucceeded}
	if len(updates) > 0 {
		if err = h.updateValuesFile(stepCtx.WorkDir, cfg.Path, updates); err != nil {
			return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
				fmt.Errorf("values file update failed: %w", err)
		}

		if commitMsg := h.generateCommitMessage(cfg.Path, updates); commitMsg != "" {
			result.Output = map[string]any{
				"commitMessage": commitMsg,
			}
		}
	}
	return result, nil
}

func (h *helmImageUpdater) generateImageUpdates(
	ctx context.Context,
	stepCtx *PromotionStepContext,
	cfg HelmUpdateImageConfig,
) (map[string]string, error) {
	updates := make(map[string]string, len(cfg.Images))
	for _, image := range cfg.Images {
		switch image.Value {
		case ImageAndTag, Tag, ImageAndDigest, Digest:
			// TODO(krancour): Remove this for v1.2.0
			desiredOrigin := h.getDesiredOrigin(image.FromOrigin)
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
			updates[image.Key] = h.getValue(targetImage, image.Value)
		default:
			updates[image.Key] = image.Value
		}
	}
	return updates, nil
}

func (h *helmImageUpdater) getDesiredOrigin(fromOrigin *ChartFromOrigin) *kargoapi.FreightOrigin {
	if fromOrigin == nil {
		return nil
	}
	return &kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKind(fromOrigin.Kind),
		Name: fromOrigin.Name,
	}
}

// TODO(krancour): Remove this for v1.2.0
func (h *helmImageUpdater) getValue(image *kargoapi.Image, value string) string {
	switch value {
	case ImageAndTag:
		return fmt.Sprintf("%s:%s", image.RepoURL, image.Tag)
	case Tag:
		return image.Tag
	case ImageAndDigest:
		return fmt.Sprintf("%s@%s", image.RepoURL, image.Digest)
	case Digest:
		return image.Digest
	default:
		return value
	}
}

func (h *helmImageUpdater) updateValuesFile(workDir string, path string, changes map[string]string) error {
	absValuesFile, err := securejoin.SecureJoin(workDir, path)
	if err != nil {
		return fmt.Errorf("error joining path %q: %w", path, err)
	}
	if err := libYAML.SetStringsInFile(absValuesFile, changes); err != nil {
		return fmt.Errorf("error updating image references in values file %q: %w", path, err)
	}
	return nil
}

func (h *helmImageUpdater) generateCommitMessage(path string, updates map[string]string) string {
	if len(updates) == 0 {
		return ""
	}

	var commitMsg strings.Builder
	_, _ = commitMsg.WriteString(fmt.Sprintf("Updated %s\n", path))
	keys := make([]string, 0, len(updates))
	for key := range updates {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		_, _ = commitMsg.WriteString(fmt.Sprintf("\n- %s: %q", key, updates[key]))
	}

	return commitMsg.String()
}
