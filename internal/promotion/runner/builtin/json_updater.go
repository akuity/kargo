package builtin

import (
	"context"
	"fmt"
	"os"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/tidwall/sjson"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

// jsonUpdater is an implementation of the promotion.StepRunner interface that
// updates the values of specified keys in a JSON file.
type jsonUpdater struct {
	schemaLoader gojsonschema.JSONLoader
}

// newJSONUpdater returns an implementation of the promotion.StepRunner interface
// that updates the values of specified keys in a JSON file.
func newJSONUpdater() promotion.StepRunner {
	r := &jsonUpdater{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the promotion.StepRunner interface.
func (j *jsonUpdater) Name() string {
	return "json-update"
}

// Run implements the promotion.StepRunner interface.
func (j *jsonUpdater) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	failure := promotion.StepResult{Status: kargoapi.PromotionPhaseErrored}

	if err := j.validate(stepCtx.Config); err != nil {
		return failure, err
	}

	cfg, err := promotion.ConfigToStruct[builtin.JSONUpdateConfig](stepCtx.Config)
	if err != nil {
		return failure, fmt.Errorf("could not convert config into %s config: %w", j.Name(), err)
	}

	return j.run(ctx, stepCtx, cfg)
}

// validate validates jsonUpdater configuration against a JSON schema.
func (j *jsonUpdater) validate(cfg promotion.Config) error {
	return validate(j.schemaLoader, gojsonschema.NewGoLoader(cfg), j.Name())
}

func (j *jsonUpdater) run(
	_ context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.JSONUpdateConfig,
) (promotion.StepResult, error) {
	result := promotion.StepResult{Status: kargoapi.PromotionPhaseSucceeded}

	if len(cfg.Updates) > 0 {
		if err := j.updateFile(stepCtx.WorkDir, cfg.Path, cfg.Updates); err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionPhaseErrored},
				fmt.Errorf("JSON file update failed: %w", err)
		}

		if commitMsg := j.generateCommitMessage(cfg.Path, cfg.Updates); commitMsg != "" {
			result.Output = map[string]any{
				"commitMessage": commitMsg,
			}
		}
	}
	return result, nil
}

func (j *jsonUpdater) updateFile(workDir string, path string, updates []builtin.JSONUpdate) error {
	absFilePath, err := securejoin.SecureJoin(workDir, path)
	if err != nil {
		return fmt.Errorf("error joining path %q: %w", path, err)
	}

	fileContent, err := os.ReadFile(absFilePath)
	if err != nil {
		return fmt.Errorf("error reading JSON file %q: %w", absFilePath, err)
	}

	for _, update := range updates {
		if !isValidScalar(update.Value) {
			return fmt.Errorf("value for key %q is not a scalar type", update.Key)
		}
		updatedContent, setErr := sjson.Set(string(fileContent), update.Key, update.Value)
		if setErr != nil {
			return fmt.Errorf("error setting key %q in JSON file: %w", update.Key, setErr)
		}
		fileContent = []byte(updatedContent)
	}

	err = os.WriteFile(absFilePath, fileContent, 0600)
	if err != nil {
		return fmt.Errorf("error writing updated JSON file %q: %w", absFilePath, err)
	}

	return nil
}

func isValidScalar(value any) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64,
		string, bool:
		return true
	default:
		return false
	}
}

func (j *jsonUpdater) generateCommitMessage(path string, updates []builtin.JSONUpdate) string {
	if len(updates) == 0 {
		return ""
	}

	var commitMsg strings.Builder
	_, _ = commitMsg.WriteString(fmt.Sprintf("Updated %s\n", path))

	for _, update := range updates {
		switch v := update.Value.(type) {
		case string:
			_, _ = commitMsg.WriteString(fmt.Sprintf("\n- %s: %q", update.Key, v))
		default:
			_, _ = commitMsg.WriteString(fmt.Sprintf("\n- %s: %v", update.Key, v))
		}
	}

	return commitMsg.String()
}
