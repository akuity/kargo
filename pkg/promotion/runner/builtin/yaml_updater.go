package builtin

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/stepconfig"
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
// converts it into a stepconfig.YAMLUpdateConfig struct.
func (y *yamlUpdater) convert(cfg promotion.Config) (stepconfig.YAMLUpdateConfig, error) {
	return validateAndConvert[stepconfig.YAMLUpdateConfig](y.schemaLoader, cfg, stepKindYAMLUpdate)
}

func (y *yamlUpdater) run(
	_ context.Context,
	stepCtx *promotion.StepContext,
	cfg stepconfig.YAMLUpdateConfig,
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
		if err := y.updateFile(stepCtx.Root, cfg.Path, updates); err != nil {
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

func (y *yamlUpdater) updateFile(root *os.Root, path string, updates []yaml.Update) error {
	relPath, err := promotion.WorkspacePath(path)
	if err != nil {
		return err
	}
	inBytes, err := root.ReadFile(relPath)
	if err != nil {
		return fmt.Errorf("error reading values file %q: %w", path, err)
	}
	outBytes, err := yaml.SetValuesInBytes(inBytes, updates)
	if err != nil {
		return fmt.Errorf("error updating image references in values file %q: %w", path, err)
	}
	if err = root.WriteFile(relPath, outBytes, 0o600); err != nil {
		return fmt.Errorf("error writing values file %q: %w", path, err)
	}
	return nil
}

func (y *yamlUpdater) generateCommitMessage(path string, updates []stepconfig.YAMLUpdate) string {
	if len(updates) == 0 {
		return ""
	}

	var commitMsg strings.Builder
	_, _ = fmt.Fprintf(&commitMsg, "Updated %s\n", path)
	for _, update := range updates {
		_, _ = fmt.Fprintf(
			&commitMsg,
			"\n- %s: %v",
			update.Key, yaml.QuoteIfNecessary(update.Value),
		)
	}

	return commitMsg.String()
}
