package builtin

import (
	"context"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/kubeclient"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const stepKindSetMetadata = "set-metadata"

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name: stepKindSetMetadata,
			Metadata: promotion.StepRunnerMetadata{
				RequiredCapabilities: []promotion.StepRunnerCapability{
					promotion.StepCapabilityAccessControlPlane,
				},
			},
			Value: newMetadataSetter,
		},
	)
}

// metadataSetter is an implementation of the promotion.StepRunner interface
// that updates metadata on Stage or Freight resources.
type metadataSetter struct {
	kargoClient  client.Client
	schemaLoader gojsonschema.JSONLoader
}

// newMetadataSetter returns an implementation of the promotion.StepRunner
// interface that updates metadata on Stage or Freight resources.
func newMetadataSetter(caps promotion.StepRunnerCapabilities) promotion.StepRunner {
	return &metadataSetter{
		kargoClient:  caps.KargoClient,
		schemaLoader: getConfigSchemaLoader(stepKindSetMetadata),
	}
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
	return validateAndConvert[builtin.SetMetadataConfig](s.schemaLoader, cfg, stepKindSetMetadata)
}

func (s *metadataSetter) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.SetMetadataConfig,
) (promotion.StepResult, error) {
	for _, update := range cfg.Updates {
		switch update.Kind {
		case "Stage":
			stage := &kargoapi.Stage{}
			if err := s.kargoClient.Get(
				ctx,
				types.NamespacedName{
					Name:      update.Name,
					Namespace: stepCtx.Project,
				},
				stage,
			); err != nil {
				return promotion.StepResult{
					Status: kargoapi.PromotionStepStatusErrored,
				}, fmt.Errorf("error getting Stage: %w", err)
			}
			newStatus := stage.Status.DeepCopy()
			if err := s.upsertMetadata(newStatus, update.Values); err != nil {
				return promotion.StepResult{
						Status: kargoapi.PromotionStepStatusErrored,
					}, fmt.Errorf(
						"error updating metadata for Stage %q in namespace %q: %w",
						stage.Name, stage.Namespace, err,
					)
			}
			if err := kubeclient.PatchStatus(
				ctx,
				s.kargoClient,
				stage,
				func(status *kargoapi.StageStatus) { *status = *newStatus },
			); err != nil {
				return promotion.StepResult{
						Status: kargoapi.PromotionStepStatusErrored,
					}, fmt.Errorf(
						"error patching status of Stage %q in namespace %q: %w",
						stage.Name, stage.Namespace, err,
					)
			}

		case "Freight":
			freight := &kargoapi.Freight{}
			if err := s.kargoClient.Get(
				ctx,
				types.NamespacedName{
					Name:      update.Name,
					Namespace: stepCtx.Project,
				},
				freight,
			); err != nil {
				return promotion.StepResult{
					Status: kargoapi.PromotionStepStatusErrored,
				}, fmt.Errorf("error getting Freight: %w", err)
			}
			newStatus := freight.Status.DeepCopy()
			if err := s.upsertMetadata(newStatus, update.Values); err != nil {
				return promotion.StepResult{
						Status: kargoapi.PromotionStepStatusErrored,
					}, fmt.Errorf(
						"error updating metadata for Freight %q in namespace %q: %w",
						freight.Name, freight.Namespace, err,
					)
			}
			if err := kubeclient.PatchStatus(
				ctx,
				s.kargoClient,
				freight,
				func(status *kargoapi.FreightStatus) { *status = *newStatus },
			); err != nil {
				return promotion.StepResult{
						Status: kargoapi.PromotionStepStatusErrored,
					}, fmt.Errorf(
						"error patching status of Freight %q in namespace %q: %w",
						freight.Name, freight.Namespace, err,
					)
			}

		default:
			return promotion.StepResult{
					Status: kargoapi.PromotionStepStatusFailed,
				}, &promotion.TerminalError{
					Err: fmt.Errorf("unsupported kind %q", update.Kind),
				}
		}
	}

	return promotion.StepResult{
		Status: kargoapi.PromotionStepStatusSucceeded,
	}, nil
}

type metadataUpsertable interface {
	UpsertMetadata(key string, value any) error
}

func (s *metadataSetter) upsertMetadata(
	u metadataUpsertable,
	metadata map[string]any,
) error {
	for k, v := range metadata {
		if err := u.UpsertMetadata(k, v); err != nil {
			return err
		}
	}
	return nil
}
