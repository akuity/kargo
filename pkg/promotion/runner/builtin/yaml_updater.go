package builtin

import (
	"context"
	"fmt"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
	"github.com/akuity/kargo/pkg/yaml"
)

const stepKindYAMLUpdate = "yaml-update"

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name:  stepKindYAMLUpdate,
			Value: newYAMLUpdater,
		},
	)
}

// yamlUpdater is an implementation of the promotion.StepRunner interface that
// updates the values of specified keys in a YAML file.
type yamlUpdater struct {
	schemaLoader gojsonschema.JSONLoader
}

// newYAMLUpdater returns an implementation of the promotion.StepRunner interface
// that updates the values of specified keys in a YAML file.
func newYAMLUpdater(promotion.StepRunnerCapabilities) promotion.StepRunner {
	return &yamlUpdater{schemaLoader: getConfigSchemaLoader(stepKindYAMLUpdate)}
}

// Run implements the promotion.StepRunner interface.
func (y *yamlUpdater) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	cfg, err := y.convert(stepCtx.Config)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &promotion.TerminalError{Err: err}
	}
	return y.run(ctx, stepCtx, cfg)
}

// convert validates yamlUpdater configuration against a JSON schema and
// converts it into a builtin.YAMLUpdateConfig struct.
func (y *yamlUpdater) convert(cfg promotion.Config) (builtin.YAMLUpdateConfig, error) {
	return validateAndConvert[builtin.YAMLUpdateConfig](y.schemaLoader, cfg, stepKindYAMLUpdate)
}

func (y *yamlUpdater) run(
	_ context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.YAMLUpdateConfig,
) (promotion.StepResult, error) {
	updates := make([]yaml.Update, len(cfg.Updates))
	for i, update := range cfg.Updates {
		updates[i] = yaml.Update{
			Key:   update.Key,
			Value: update.Value,
		}
	}

	result := promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}
	if len(updates) > 0 {
		if err := y.updateFile(stepCtx.WorkDir, cfg.Path, updates); err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf("values file update failed: %w", err)
		}

		if commitMsg := y.generateCommitMessage(cfg.Path, cfg.Updates); commitMsg != "" {
			result.Output = map[string]any{
				"commitMessage": commitMsg,
			}
		}
	}
	return result, nil
}

func (y *yamlUpdater) updateFile(workDir string, path string, updates []yaml.Update) error {
	absValuesFile, err := securejoin.SecureJoin(workDir, path)
	if err != nil {
		return fmt.Errorf("error joining path %q: %w", path, err)
	}
	if err := yaml.SetValuesInFile(absValuesFile, updates); err != nil {
		return fmt.Errorf("error updating image references in values file %q: %w", path, err)
	}
	return nil
}

func (y *yamlUpdater) generateCommitMessage(path string, updates []builtin.YAMLUpdate) string {
	if len(updates) == 0 {
		return ""
	}

	var commitMsg strings.Builder
	_, _ = commitMsg.WriteString(fmt.Sprintf("Updated %s\n", path))
	for _, update := range updates {
		_, _ = commitMsg.WriteString(
			fmt.Sprintf(
				"\n- %s: %v",
				update.Key,
				yaml.QuoteIfNecessary(update.Value),
			),
		)
	}

	return commitMsg.String()
}
