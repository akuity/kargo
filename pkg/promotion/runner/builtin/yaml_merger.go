package builtin

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

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
	// Read the input files, skipping any that are missing if configured to.
	inputPaths := make([]string, 0, len(cfg.InFiles))
	inputs := make([][]byte, 0, len(cfg.InFiles))
	for _, inputPath := range cfg.InFiles {
		relInputPath, err := promotion.WorkspacePath(inputPath)
		if err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
		}
		input, err := stepCtx.Root.ReadFile(relInputPath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				if cfg.IgnoreMissingFiles {
					continue
				}
				return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
					fmt.Errorf("input file %q not found: %w", inputPath, err)
			}
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf("unexpected error with file %q: %w", inputPath, err)
		}
		inputPaths = append(inputPaths, inputPath)
		inputs = append(inputs, input)
	}

	outFile, err := promotion.WorkspacePath(cfg.OutFile)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	merged, err := yaml.Merge(inputs)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error merging YAML files: %w", err)
	}
	if err = stepCtx.Root.MkdirAll(filepath.Dir(outFile), 0o700); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error creating directory for %q: %w", cfg.OutFile, err)
	}
	if err = stepCtx.Root.WriteFile(outFile, merged, 0o600); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error writing merged YAML to %q: %w", cfg.OutFile, err)
	}

	// Generate commit message with relative paths. This does not include any
	// that were ignored.
	result := promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}
	if msg := y.generateCommitMessage(inputPaths, cfg.OutFile); msg != "" {
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
		_, _ = fmt.Fprintf(&msg, "Wrote %s to %s", inPaths[0], outPath)
	} else {
		_, _ = fmt.Fprintf(&msg, "Merged %d YAML files to %s", len(inPaths), outPath)
		for _, path := range inPaths {
			_, _ = fmt.Fprintf(&msg, "\n- %s", path)
		}
	}
	return msg.String()
}
