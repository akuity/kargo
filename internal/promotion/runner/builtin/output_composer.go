package builtin

import (
	"context"
	"maps"

	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	pkgPromotion "github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

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
func newOutputComposer() pkgPromotion.StepRunner {
	r := &outputComposer{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the promotion.StepRunner interface.
func (c *outputComposer) Name() string {
	return "compose-output"
}

// Run implements the promotion.StepRunner interface.
func (c *outputComposer) Run(
	_ context.Context,
	stepCtx *pkgPromotion.StepContext,
) (pkgPromotion.StepResult, error) {
	cfg, err := c.convert(stepCtx.Config)
	if err != nil {
		return pkgPromotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &pkgPromotion.TerminalError{Err: err}
	}
	return c.run(cfg)
}

func (c *outputComposer) convert(cfg pkgPromotion.Config) (builtin.ComposeOutput, error) {
	return validateAndConvert[builtin.ComposeOutput](c.schemaLoader, cfg, c.Name())
}

func (c *outputComposer) run(
	cfg builtin.ComposeOutput,
) (pkgPromotion.StepResult, error) {
	return pkgPromotion.StepResult{
		Status: kargoapi.PromotionStepStatusSucceeded,
		Output: maps.Clone(cfg),
	}, nil
}
