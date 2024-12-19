package directives

import (
	"context"
	"fmt"
	"maps"

	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func init() {
	builtins.RegisterPromotionStepRunner(newOutputComposer(), nil)
}

// outputComposer is an implementation of the PromotionStepRunner interface
// that allows composing outputs from previous steps into new outputs.
//
// It works based on the PromotionStepContext.Config field allowing to an
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
//	  prURL: ${{ vars.repoURL }}/pull/${{ outputs['open-pr'].prNumber }}
//	  mergeCommit: ${{ outputs['wait-for-pr'].commit }}
//
// This would create a new output named `custom-outputs` with the keys
// `prURL` and `mergeCommit`, which could be used in subsequent steps
// using e.g. `${{ outputs.custom-outputs.prURL }}`.
type outputComposer struct {
	schemaLoader gojsonschema.JSONLoader
}

// newOutputComposer returns an implementation of the PromotionStepRunner
// interface that composes output from previous steps into new output.
func newOutputComposer() PromotionStepRunner {
	r := &outputComposer{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the PromotionStepRunner interface.
func (c *outputComposer) Name() string {
	return "compose-output"
}

// RunPromotionStep implements the PromotionStepRunner interface.
func (c *outputComposer) RunPromotionStep(
	_ context.Context,
	stepCtx *PromotionStepContext,
) (PromotionStepResult, error) {
	// Validate the configuration against the JSON Schema.
	if err := validate(c.schemaLoader, gojsonschema.NewGoLoader(stepCtx.Config), c.Name()); err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, err
	}

	// Convert the configuration into a typed object.
	cfg, err := ConfigToStruct[ComposeOutput](stepCtx.Config)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("could not convert config into %s config: %w", c.Name(), err)
	}

	return c.runPromotionStep(cfg)
}

func (c *outputComposer) runPromotionStep(
	cfg ComposeOutput,
) (PromotionStepResult, error) {
	return PromotionStepResult{
		Status: kargoapi.PromotionPhaseSucceeded,
		Output: maps.Clone(cfg),
	}, nil
}
