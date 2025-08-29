package builtin

import (
	"context"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

// metadataSetter is an implementation of the promotion.StepRunner interface
// that updates metadata on Stage or Freight resources.
type metadataSetter struct {
	kargoClient  client.Client
	schemaLoader gojsonschema.JSONLoader
}

// newMetadataSetter returns an implementation of the promotion.StepRunner
// interface that updates metadata on Stage or Freight resources.
func newMetadataSetter(kargoClient client.Client) promotion.StepRunner {
	r := &metadataSetter{
		kargoClient: kargoClient,
	}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the promotion.StepRunner interface.
func (s *metadataSetter) Name() string {
	return "set-metadata"
}

// Run implements the promotion.StepRunner interface.
func (s *metadataSetter) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	cfg, err := s.convert(stepCtx.Config)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &promotion.TerminalError{Err: err}
	}
	return s.run(ctx, stepCtx, cfg)
}

// convert validates the configuration against a JSON schema and converts it
// into a builtin.SetMetadataConfig struct.
func (s *metadataSetter) convert(cfg promotion.Config) (builtin.SetMetadataConfig, error) {
	return validateAndConvert[builtin.SetMetadataConfig](s.schemaLoader, cfg, s.Name())
}

func (s *metadataSetter) run(
	ctx context.Context,
	_ *promotion.StepContext,
	cfg builtin.SetMetadataConfig,
) (promotion.StepResult, error) {
	for _, update := range cfg.Updates {
		plainValues := make(map[string]any)
		for k, v := range update.Values {
			plainValues[k] = v
		}

		switch update.Kind {
		case "Stage":
			if err := s.updateStageMetadata(
				ctx,
				update.Name,
				plainValues,
			); err != nil {
				return promotion.StepResult{
					Status: kargoapi.PromotionStepStatusErrored,
				}, fmt.Errorf("error updating Stage metadata: %w", err)
			}
		case "Freight":
			if err := s.updateFreightMetadata(
				ctx,
				update.Name,
				plainValues,
			); err != nil {
				return promotion.StepResult{
					Status: kargoapi.PromotionStepStatusErrored,
				}, fmt.Errorf("error updating Freight metadata: %w", err)
			}
		default:
			return promotion.StepResult{
					Status: kargoapi.PromotionStepStatusFailed,
				}, &promotion.TerminalError{
					Err: fmt.Errorf("unsupported kind %q", update.Kind),
				}
		}
	}

	return promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, nil
}

func (s *metadataSetter) updateStageMetadata(
	ctx context.Context,
	name string,
	values map[string]any,
) error {
	stage := &kargoapi.Stage{}
	if err := s.kargoClient.Get(
		ctx,
		types.NamespacedName{Name: name},
		stage,
	); err != nil {
		return err
	}

	// Convert values to apiextensionsv1.JSON
	for key, value := range values {
		if err := stage.Status.UpsertMetadata(key, value); err != nil {
			return fmt.Errorf("failed to upsert metadata for key %q: %w", key, err)
		}
	}

	return s.kargoClient.Status().Update(ctx, stage)
}

func (s *metadataSetter) updateFreightMetadata(
	ctx context.Context,
	name string,
	values map[string]any,
) error {
	freight := &kargoapi.Freight{}
	if err := s.kargoClient.Get(
		ctx,
		types.NamespacedName{Name: name},
		freight,
	); err != nil {
		return err
	}

	// Convert values to apiextensionsv1.JSON
	for key, value := range values {
		if err := freight.Status.UpsertMetadata(key, value); err != nil {
			return fmt.Errorf("failed to upsert metadata for key %q: %w", key, err)
		}
	}

	return s.kargoClient.Status().Update(ctx, freight)
}
