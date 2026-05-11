package builtin

import (
	"context"
	"fmt"

	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const stepKindFail = "fail"

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name:  stepKindFail,
			Value: newFailer,
		},
	)
}

// failer is an implementation of the promotion.StepRunner interface that
// always fails with an optional message.
type failer struct {
	schemaLoader gojsonschema.JSONLoader
}

// newFailer returns an implementation of the promotion.StepRunner interface
// that always fails with an optional message.
func newFailer(_ promotion.StepRunnerCapabilities) promotion.StepRunner {
	return &failer{
		schemaLoader: getConfigSchemaLoader(stepKindFail),
	}
}

// Run implements the promotion.StepRunner interface.
func (f *failer) Run(
	_ context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	cfg, err := f.convert(stepCtx.Config)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &promotion.TerminalError{Err: err}
	}
	return f.run(cfg)
}

func (f *failer) run(
	cfg builtin.FailConfig,
) (promotion.StepResult, error) {
	err := fmt.Errorf("failed")
	if cfg.Message != "" {
		err = fmt.Errorf("failed: %s", cfg.Message)
	}

	return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed},
		&promotion.TerminalError{Err: err}
}

func (f *failer) convert(cfg promotion.Config) (builtin.FailConfig, error) {
	return validateAndConvert[builtin.FailConfig](f.schemaLoader, cfg, stepKindFail)
}
