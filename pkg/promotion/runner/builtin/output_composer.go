package builtin

import (
	"context"
	"maps"

	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const stepKindComposeOutput = "compose-output"

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name: stepKindComposeOutput,
			Metadata: promotion.StepRunnerMetadata{
				RequiredCapabilities: []promotion.StepRunnerCapability{
					promotion.StepCapabilityTaskOutputPropagation,
				},
			},
			Value: newOutputComposer,
		},
	)
}

// outputComposer is an implementation of the promotion.StepRunner interface
// that allows composing outputs from previous steps into new outputs.
//
// It works based on the promotion.StepContext.Config field allowing to an
// arbitrary number of key-value pairs to be exported as outputs.
// Because the values are allowed to be expressions and can contain
// references to outputs from previous steps, this allows for remapping
// the outputs of previous steps to new keys, or even combining them
// into new structures.
//
// An example configuration for this step would look like this:
//
//	step: compose-output
//	as: custom-outputs
//	config:
//	  prURL: ${{ vars.repoURL }}/pull/${{ outputs['open-pr'].pr.id }}
//	  mergeCommit: ${{ outputs['wait-for-pr'].commit }}
//
// This would create a new output named `custom-outputs` with the keys
// `prURL` and `mergeCommit`, which could be used in subsequent steps
// using e.g. `${{ outputs.custom-outputs.prURL }}`.
type outputComposer struct {
	schemaLoader gojsonschema.JSONLoader
}

// newOutputComposer returns an implementation of the promotion.StepRunner
// interface that composes output from previous steps into new output.
func newOutputComposer(_ promotion.StepRunnerCapabilities) promotion.StepRunner {
	return &outputComposer{
		schemaLoader: getConfigSchemaLoader(stepKindComposeOutput),
	}
}

// Run implements the promotion.StepRunner interface.
func (c *outputComposer) Run(
	_ context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	cfg, err := c.convert(stepCtx.Config)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &promotion.TerminalError{Err: err}
	}
	return c.run(cfg)
}

func (c *outputComposer) convert(cfg promotion.Config) (builtin.ComposeOutput, error) {
	return validateAndConvert[builtin.ComposeOutput](c.schemaLoader, cfg, stepKindComposeOutput)
}

func (c *outputComposer) run(
	cfg builtin.ComposeOutput,
) (promotion.StepResult, error) {
	return promotion.StepResult{
		Status: kargoapi.PromotionStepStatusSucceeded,
		Output: maps.Clone(cfg),
	}, nil
}
