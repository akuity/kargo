package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	promotion.RegisterStepRunner(
		stepKindYAMLMerge,
		promotion.StepRunnerRegistration{Factory: newYAMLMerger},
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
	// Validate and collect input file paths
	filePaths := make([]string, 0, len(cfg.InFiles))
	for _, path := range cfg.InFiles {
		inFile, err := securejoin.SecureJoin(stepCtx.WorkDir, path)
		if err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf("invalid input file path %q: %w", path, err)
		}

		if _, err = os.Stat(inFile); err != nil {
			if os.IsNotExist(err) {
				if cfg.IgnoreMissingFiles {
					continue
				}
				return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
					fmt.Errorf("input file %q not found: %w", path, err)
			}
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf("unexpected error with file %q: %w", path, err)
		}
		filePaths = append(filePaths, inFile)
	}

	// Validate output path
	outFile, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.OutFile)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("invalid output file path %q: %w", cfg.OutFile, err)
	}

	// Ensure output directory exists
	if err = os.MkdirAll(filepath.Dir(outFile), 0o700); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error ensuring existence of directory %q: %w", filepath.Dir(outFile), err)
	}

	if len(filePaths) == 0 {
		if !cfg.IgnoreMissingFiles {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf("no input files found to merge")
		}
		// Create an empty output file when ignoreMissingFiles is true
		if err = os.WriteFile(outFile, []byte{}, 0o600); err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf("error creating empty output file: %w", err)
		}
	} else {
		// Merge YAML files
		if err = yaml.MergeFiles(filePaths, outFile); err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf("error merging YAML files: %w", err)
		}
	}

	// Generate commit message with relative paths
	result := promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}
	if commitMsg := y.generateCommitMessage(cfg.OutFile, cfg.InFiles); commitMsg != "" {
		result.Output = map[string]any{
			"commitMessage": commitMsg,
		}
	}
	return result, nil
}

func (y *yamlMerger) generateCommitMessage(outPath string, inFiles []string) string {
	if len(inFiles) == 0 {
		return ""
	}

	var commitMsg strings.Builder
	if len(inFiles) == 1 {
		_, _ = commitMsg.WriteString(fmt.Sprintf("Merged %s to %s", inFiles[0], outPath))
	} else {
		_, _ = commitMsg.WriteString(fmt.Sprintf("Merged %d YAML files to %s", len(inFiles), outPath))
		for _, file := range inFiles {
			_, _ = commitMsg.WriteString(fmt.Sprintf("\n- %s", file))
		}
	}

	return commitMsg.String()
}
