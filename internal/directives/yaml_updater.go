package directives

import (
	"context"
	"fmt"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	intyaml "github.com/akuity/kargo/internal/yaml"
)

func init() {
	builtins.RegisterPromotionStepRunner(newYAMLUpdater(), nil)
}

// yamlUpdater is an implementation of the PromotionStepRunner interface that
// updates the values of specified keys in a YAML file.
type yamlUpdater struct {
	schemaLoader gojsonschema.JSONLoader
}

// newYAMLUpdater returns an implementation of the PromotionStepRunner interface
// that updates the values of specified keys in a YAML file.
func newYAMLUpdater() PromotionStepRunner {
	r := &yamlUpdater{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the PromotionStepRunner interface.
func (y *yamlUpdater) Name() string {
	return "yaml-update"
}

// RunPromotionStep implements the PromotionStepRunner interface.
func (y *yamlUpdater) RunPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
) (PromotionStepResult, error) {
	failure := PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}

	if err := y.validate(stepCtx.Config); err != nil {
		return failure, err
	}

	// Convert the configuration into a typed struct
	cfg, err := ConfigToStruct[YAMLUpdateConfig](stepCtx.Config)
	if err != nil {
		return failure, fmt.Errorf("could not convert config into %s config: %w", y.Name(), err)
	}

	return y.runPromotionStep(ctx, stepCtx, cfg)
}

// validate validates yamlImageUpdater configuration against a JSON schema.
func (y *yamlUpdater) validate(cfg Config) error {
	return validate(y.schemaLoader, gojsonschema.NewGoLoader(cfg), y.Name())
}

func (y *yamlUpdater) runPromotionStep(
	_ context.Context,
	stepCtx *PromotionStepContext,
	cfg YAMLUpdateConfig,
) (PromotionStepResult, error) {
	updates := make([]intyaml.Update, len(cfg.Updates))
	for i, update := range cfg.Updates {
		updates[i] = intyaml.Update{
			Key:   update.Key,
			Value: update.Value,
		}
	}

	result := PromotionStepResult{Status: kargoapi.PromotionPhaseSucceeded}
	if len(updates) > 0 {
		if err := y.updateFile(stepCtx.WorkDir, cfg.Path, updates); err != nil {
			return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
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

func (y *yamlUpdater) updateFile(workDir string, path string, updates []intyaml.Update) error {
	absValuesFile, err := securejoin.SecureJoin(workDir, path)
	if err != nil {
		return fmt.Errorf("error joining path %q: %w", path, err)
	}
	if err := intyaml.SetStringsInFile(absValuesFile, updates); err != nil {
		return fmt.Errorf("error updating image references in values file %q: %w", path, err)
	}
	return nil
}

func (y *yamlUpdater) generateCommitMessage(path string, updates []YAMLUpdate) string {
	if len(updates) == 0 {
		return ""
	}

	var commitMsg strings.Builder
	_, _ = commitMsg.WriteString(fmt.Sprintf("Updated %s\n", path))
	for _, update := range updates {
		_, _ = commitMsg.WriteString(fmt.Sprintf("\n- %s: %q", update.Key, update.Value))
	}

	return commitMsg.String()
}
