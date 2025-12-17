package builtin

import (
	"context"
	"fmt"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
	"github.com/xeipuuv/gojsonschema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const stepKindSetFreightAlias = "set-freight-alias"

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name: stepKindSetFreightAlias,
			Metadata: promotion.StepRunnerMetadata{
				RequiredCapabilities: []promotion.StepRunnerCapability{
					promotion.StepCapabilityAccessControlPlane,
				},
			},
			Value: newSetFreightAlias,
		},
	)
}

// setFreightAlias is an implementation of the promotion.StepRunner interface
// that updates alias on Freight resources.
type setFreightAlias struct {
	kargoClient  client.Client
	schemaLoader gojsonschema.JSONLoader
}

// convert validates the configuration against a JSON schema and converts it
// into a builtin.SetFreightAliasConfig struct.
func (s *setFreightAlias) convert(cfg promotion.Config) (builtin.SetFreightAliasConfig, error) {
	return validateAndConvert[builtin.SetFreightAliasConfig](s.schemaLoader, cfg, stepKindSetFreightAlias)
}

// newSetFreightAlias returns an implementation of the promotion.StepRunner
// interface that updates alias on Freight resources.
func newSetFreightAlias(caps promotion.StepRunnerCapabilities) promotion.StepRunner {
	return &setFreightAlias{
		kargoClient:  caps.KargoClient,
		schemaLoader: getConfigSchemaLoader(stepKindSetFreightAlias),
	}
}

// Run implements the promotion.StepRunner interface.
func (s *setFreightAlias) Run(
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

func (s *setFreightAlias) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.SetFreightAliasConfig,
) (promotion.StepResult, error) {
	// This step intentionally targets a Freight specified explicitly via `freightID`
	// instead of implicitly operating on `stepCtx.TargetFreightRef`.
	//
	// While most promotion steps act on the Freight currently being promoted,
	// updating a Freight alias is a project-scoped mutation rather than a
	// promotion-scoped one. There are valid use cases where a user may want to
	// update the alias of a different Freight in the same project that is not
	// currently being promoted
	freight, err := api.GetFreightByNameOrAlias(ctx, s.kargoClient, stepCtx.Project, cfg.FreightID, "")
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &promotion.TerminalError{Err: fmt.Errorf("get freight: %w", err)}
	}

	if freight == nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &promotion.TerminalError{Err: fmt.Errorf("freight %q not found in project %q", cfg.FreightID, stepCtx.Project)}
	}

	// check if alias is already used by another freight in the project
	freightList := kargoapi.FreightList{}
	if err := s.kargoClient.List(ctx, &freightList, client.InNamespace(stepCtx.Project), client.MatchingLabels{kargoapi.LabelKeyAlias: cfg.NewAlias}); err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &promotion.TerminalError{Err: fmt.Errorf("list freight: %w", err)}
	}

	if len(freightList.Items) > 1 || (len(freightList.Items) == 1 && freightList.Items[0].Name != freight.Name) {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &promotion.TerminalError{Err: fmt.Errorf("alias %q already used by another piece of Freight in proejct %q", cfg.NewAlias, stepCtx.Project)}
	}

	patchBytes := []byte(
		fmt.Sprintf(
			`{"metadata":{"labels":{%q:%q}},"alias":%q}`,
			kargoapi.LabelKeyAlias,
			cfg.NewAlias,
			cfg.NewAlias,
		),
	)
	patch := client.RawPatch(types.MergePatchType, patchBytes)

	if err := s.kargoClient.Patch(ctx, freight, patch); err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &promotion.TerminalError{Err: fmt.Errorf("patch freight alias: %w", err)}
	}

	return promotion.StepResult{
		Status: kargoapi.PromotionStepStatusSucceeded,
		Message: fmt.Sprintf(
			"updated alias of Freight %q from %q to %q",
			freight.Name,
			freight.Alias,
			cfg.NewAlias,
		),
	}, nil
}
