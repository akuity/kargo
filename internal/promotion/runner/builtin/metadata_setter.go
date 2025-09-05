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
	type metadataUpsertable interface {
		UpsertMetadata(key string, value any) error
	}

	for _, update := range cfg.Updates {
		var obj client.Object
		var upsertable metadataUpsertable

		switch update.Kind {
		case "Stage":
			stage := &kargoapi.Stage{}
			if err := s.kargoClient.Get(
				ctx,
				types.NamespacedName{Name: update.Name},
				stage,
			); err != nil {
				return promotion.StepResult{
					Status: kargoapi.PromotionStepStatusErrored,
				}, fmt.Errorf("error getting Stage: %w", err)
			}
			obj = stage
			upsertable = &stage.Status
		case "Freight":
			freight := &kargoapi.Freight{}
			if err := s.kargoClient.Get(
				ctx,
				types.NamespacedName{Name: update.Name},
				freight,
			); err != nil {
				return promotion.StepResult{
					Status: kargoapi.PromotionStepStatusErrored,
				}, fmt.Errorf("error getting Freight: %w", err)
			}
			obj = freight
			upsertable = &freight.Status
		default:
			return promotion.StepResult{
					Status: kargoapi.PromotionStepStatusFailed,
				}, &promotion.TerminalError{
					Err: fmt.Errorf("unsupported kind %q", update.Kind),
				}
		}

		for k, v := range update.Values {
			if err := upsertable.UpsertMetadata(k, v); err != nil {
				return promotion.StepResult{
					Status: kargoapi.PromotionStepStatusErrored,
				}, fmt.Errorf("failed to upsert metadata for key %q: %w", k, err)
			}
		}

		if err := s.kargoClient.Status().Update(ctx, obj); err != nil {
			return promotion.StepResult{
				Status: kargoapi.PromotionStepStatusErrored,
			}, fmt.Errorf("failed to update status for %s/%s: %w", update.Kind, update.Name, err)
		}
	}

	return promotion.StepResult{
		Status: kargoapi.PromotionStepStatusSucceeded,
	}, nil
}
