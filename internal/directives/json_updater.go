package directives

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/tidwall/sjson"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func init() {
	builtins.RegisterPromotionStepRunner(newJSONUpdater(), nil)
}

// jsonUpdater is an implementation of the PromotionStepRunner interface that
// updates the values of specified keys in a JSON file.
type jsonUpdater struct {
	schemaLoader gojsonschema.JSONLoader
}

// newJSONUpdater returns an implementation of the PromotionStepRunner interface
// that updates the values of specified keys in a JSON file.
func newJSONUpdater() PromotionStepRunner {
	r := &jsonUpdater{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the PromotionStepRunner interface.
func (j *jsonUpdater) Name() string {
	return "json-update"
}

// RunPromotionStep implements the PromotionStepRunner interface.
func (j *jsonUpdater) RunPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
) (PromotionStepResult, error) {
	failure := PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}

	if err := j.validate(stepCtx.Config); err != nil {
		return failure, err
	}

	// Convert the configuration into a typed struct
	cfg, err := ConfigToStruct[JSONUpdateConfig](stepCtx.Config)
	if err != nil {
		return failure, fmt.Errorf("could not convert config into %s config: %w", j.Name(), err)
	}

	return j.runPromotionStep(ctx, stepCtx, cfg)
}

// validate validates jsonUpdater configuration against a JSON schema.
func (j *jsonUpdater) validate(cfg Config) error {
	return validate(j.schemaLoader, gojsonschema.NewGoLoader(cfg), j.Name())
}

func (j *jsonUpdater) runPromotionStep(
	_ context.Context,
	stepCtx *PromotionStepContext,
	cfg JSONUpdateConfig,
) (PromotionStepResult, error) {
	updates := make(map[string]any, len(cfg.Updates))
	for _, update := range cfg.Updates {
		var value any
		if update.Value != nil {
			value = update.Value
		}
		updates[update.Key] = value
	}

	result := PromotionStepResult{Status: kargoapi.PromotionPhaseSucceeded}
	if len(updates) > 0 {
		if err := j.updateFile(stepCtx.WorkDir, cfg.Path, updates); err != nil {
			return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
				fmt.Errorf("JSON file update failed: %w", err)
		}

		if commitMsg := j.generateCommitMessage(cfg.Path, updates); commitMsg != "" {
			result.Output = map[string]any{
				"commitMessage": commitMsg,
			}
		}
	}
	return result, nil
}

func (j *jsonUpdater) updateFile(workDir string, path string, changes map[string]any) error {
	absFilePath, err := securejoin.SecureJoin(workDir, path)
	if err != nil {
		return fmt.Errorf("error joining path %q: %w", path, err)
	}

	fileContent, err := os.ReadFile(absFilePath)
	if err != nil {
		return fmt.Errorf("error reading JSON file %q: %w", absFilePath, err)
	}

	for key, value := range changes {
		updatedContent, setErr := sjson.Set(string(fileContent), key, value)
		if setErr != nil {
			return fmt.Errorf("error setting key %q in JSON file: %w", key, setErr)
		}
		fileContent = []byte(updatedContent)
	}

	err = os.WriteFile(absFilePath, fileContent, 0600)
	if err != nil {
		return fmt.Errorf("error writing updated JSON file %q: %w", absFilePath, err)
	}

	return nil
}

func (j *jsonUpdater) generateCommitMessage(path string, updates map[string]any) string {
	if len(updates) == 0 {
		return ""
	}

	var commitMsg strings.Builder
	_, _ = commitMsg.WriteString(fmt.Sprintf("Updated %s\n", path))
	keys := make([]string, 0, len(updates))
	for key := range updates {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	for _, key := range keys {
		value := updates[key]
		switch v := value.(type) {
		case string:
			_, _ = commitMsg.WriteString(fmt.Sprintf("\n- %s: %q", key, v))
		case bool:
			_, _ = commitMsg.WriteString(fmt.Sprintf("\n- %s: \"%v\"", key, v))
		case int, float64:
			_, _ = commitMsg.WriteString(fmt.Sprintf("\n- %s: %v", key, v))
		default:
			_, _ = commitMsg.WriteString(fmt.Sprintf("\n- %s: %v", key, v))
		}
	}

	return commitMsg.String()
}
