package builtin

import (
	"context"
	"fmt"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	inttoml "github.com/akuity/kargo/pkg/toml"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const stepKindTOMLUpdate = "toml-update"

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name:  stepKindTOMLUpdate,
			Value: newTOMLUpdater,
		},
	)
}

// tomlUpdater is an implementation of the promotion.StepRunner interface that
// updates the values of specified keys in a TOML file.
type tomlUpdater struct {
	schemaLoader gojsonschema.JSONLoader
}

// newTOMLUpdater returns an implementation of the promotion.StepRunner interface
// that updates the values of specified keys in a TOML file.
func newTOMLUpdater(promotion.StepRunnerCapabilities) promotion.StepRunner {
	return &tomlUpdater{schemaLoader: getConfigSchemaLoader(stepKindTOMLUpdate)}
}

// Run implements the promotion.StepRunner interface.
func (t *tomlUpdater) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	cfg, err := t.convert(stepCtx.Config)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &promotion.TerminalError{Err: err}
	}
	return t.run(ctx, stepCtx, cfg)
}

// convert validates tomlUpdater configuration against a JSON schema and
// converts it into a builtin.TOMLUpdateConfig struct.
func (t *tomlUpdater) convert(cfg promotion.Config) (builtin.TOMLUpdateConfig, error) {
	return validateAndConvert[builtin.TOMLUpdateConfig](t.schemaLoader, cfg, stepKindTOMLUpdate)
}

func (t *tomlUpdater) run(
	_ context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.TOMLUpdateConfig,
) (promotion.StepResult, error) {
	updates := make([]inttoml.Update, len(cfg.Updates))
	for i, update := range cfg.Updates {
		updates[i] = inttoml.Update{
			Key:   update.Key,
			Value: update.Value,
		}
	}

	result := promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}
	if len(updates) > 0 {
		if err := t.updateFile(stepCtx.WorkDir, cfg.Path, updates); err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf("TOML file update failed: %w", err)
		}

		if commitMsg := t.generateCommitMessage(cfg.Path, cfg.Updates); commitMsg != "" {
			result.Output = map[string]any{
				"commitMessage": commitMsg,
			}
		}
	}
	return result, nil
}

func (t *tomlUpdater) updateFile(workDir string, path string, updates []inttoml.Update) error {
	absFilePath, err := securejoin.SecureJoin(workDir, path)
	if err != nil {
		return fmt.Errorf("error joining path %q: %w", path, err)
	}
	if err := inttoml.SetValuesInFile(absFilePath, updates); err != nil {
		return fmt.Errorf("error updating TOML file %q: %w", path, err)
	}
	return nil
}

func (t *tomlUpdater) generateCommitMessage(path string, updates []builtin.TomlUpdate) string {
	if len(updates) == 0 {
		return ""
	}

	var commitMsg strings.Builder
	_, _ = fmt.Fprintf(&commitMsg, "Updated %s\n", path)
	for _, update := range updates {
		_, _ = fmt.Fprintf(
			&commitMsg,
			"\n- %s: %s",
			update.Key,
			inttoml.FormatValueString(update.Value),
		)
	}

	return commitMsg.String()
}
