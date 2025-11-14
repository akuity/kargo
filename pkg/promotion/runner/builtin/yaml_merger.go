package builtin

import (
	"context"
	"fmt"
	"os"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
	"github.com/akuity/kargo/pkg/yaml"
)

const stepKindYAMLMerge = "yaml-merge"

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name:  stepKindYAMLMerge,
			Value: newYAMLMerger,
		},
	)
}

// yamlMerger is an implementation of the promotion.StepRunner interface that
// merges multiple YAML files.
type yamlMerger struct {
	schemaLoader gojsonschema.JSONLoader
}

// newYAMLMerger returns an implementation of the promotion.StepRunner interface
// that merges multiple YAML files.
func newYAMLMerger(promotion.StepRunnerCapabilities) promotion.StepRunner {
	return &yamlMerger{schemaLoader: getConfigSchemaLoader(stepKindYAMLMerge)}
}

// Run implements the promotion.StepRunner interface.
func (y *yamlMerger) Run(
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

// convert validates yamlMerger configuration against a JSON schema and
// converts it into a builtin.YAMLMergeConfig struct.
func (y *yamlMerger) convert(cfg promotion.Config) (builtin.YAMLMergeConfig, error) {
	return validateAndConvert[builtin.YAMLMergeConfig](y.schemaLoader, cfg, stepKindYAMLMerge)
}

func (y *yamlMerger) run(
	_ context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.YAMLMergeConfig,
) (promotion.StepResult, error) {
	// Update input paths and check for the existence of files at those locations
	relInputPaths := make([]string, 0, len(cfg.InFiles))
	absInputPaths := make([]string, 0, len(cfg.InFiles))
	for _, relInputPath := range cfg.InFiles {
		absInputPath, err := securejoin.SecureJoin(stepCtx.WorkDir, relInputPath)
		if err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf(
					"error joining path %s with work dir %s: %w",
					relInputPath, stepCtx.WorkDir, err,
				)
		}
		if _, err = os.Stat(absInputPath); err != nil {
			if os.IsNotExist(err) {
				if cfg.IgnoreMissingFiles {
					continue
				}
				return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
					fmt.Errorf("input file %q not found: %w", relInputPath, err)
			}
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf("unexpected error with file %q: %w", relInputPath, err)
		}
		relInputPaths = append(relInputPaths, relInputPath)
		absInputPaths = append(absInputPaths, absInputPath)
	}

	outFile, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.OutFile)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf(
				"error joining path %s with work dir %s: %w",
				cfg.OutFile, stepCtx.WorkDir, err,
			)
	}

	if err = yaml.MergeFiles(absInputPaths, outFile); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error merging YAML files: %w", err)
	}

	// Generate commit message with relative paths. This does not include any
	// that were ignored.
	result := promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}
	if msg := y.generateCommitMessage(relInputPaths, cfg.OutFile); msg != "" {
		result.Output = map[string]any{"commitMessage": msg}
	}
	return result, nil
}

func (y *yamlMerger) generateCommitMessage(
	inPaths []string,
	outPath string,
) string {
	if len(inPaths) == 0 {
		return ""
	}
	var msg strings.Builder
	if len(inPaths) == 1 {
		_, _ = msg.WriteString(fmt.Sprintf("Wrote %s to %s", inPaths[0], outPath))
	} else {
		_, _ = msg.WriteString(
			fmt.Sprintf("Merged %d YAML files to %s", len(inPaths), outPath),
		)
		for _, path := range inPaths {
			_, _ = msg.WriteString(fmt.Sprintf("\n- %s", path))
		}
	}
	return msg.String()
}
