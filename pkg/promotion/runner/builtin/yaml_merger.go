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
	"github.com/akuity/kargo/internal/yaml"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const stepKindYAMLMerger = "yaml-merge"

func init() {
	promotion.RegisterStepRunner(
		stepKindYAMLMerger,
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
func newYAMLMerger() promotion.StepRunner {
	r := &yamlMerger{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the promotion.StepRunner interface.
func (y *yamlMerger) Name() string {
	return "yaml-merge"
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
	return validateAndConvert[builtin.YAMLMergeConfig](y.schemaLoader, cfg, y.Name())
}

// validate validates yamlMerger configuration against a JSON schema.
func (y *yamlMerger) validate(cfg promotion.Config) error {
	return validate(y.schemaLoader, gojsonschema.NewGoLoader(cfg), y.Name())
}

func (y *yamlMerger) run(
	_ context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.YAMLMergeConfig,
) (promotion.StepResult, error) {

	result := promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}
	failure := promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}

	// sanity check
	if len(cfg.InFiles) == 0 || cfg.OutFile == "" {
		return failure, fmt.Errorf("inFiles and OutFile must not be empty")
	}

	// Secure join the input paths to prevent path traversal attacks.
	filePaths := []string{}
	for _, path := range cfg.InFiles {
		inFile, err := securejoin.SecureJoin(stepCtx.WorkDir, path)
		if err != nil {
			return failure, fmt.Errorf("could not secure join input file %q: %w", path, err)
		}

		// only add existing files
		_, err = os.Stat(inFile)
		if err != nil {
			if cfg.IgnoreMissingFiles {
				continue
			}
			return failure, fmt.Errorf("input file not found:  %s", inFile)

		}
		filePaths = append(filePaths, inFile)

	}

	// Secure join the output path to prevent path traversal attacks.
	outFile, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.OutFile)
	if err != nil {
		return failure, fmt.Errorf("could not secure join outFile %q: %w", cfg.OutFile, err)
	}

	// ensure output path fully exist
	if err = os.MkdirAll(filepath.Dir(outFile), 0o700); err != nil {
		return failure, fmt.Errorf("error creating directory structure %s: %w", filepath.Dir(outFile), err)
	}

	// Merge files
	err = yaml.MergeYAMLFiles(filePaths, outFile)
	if err != nil {
		return failure, fmt.Errorf("could not merge YAML files: %w", err)
	}

	// Add a commit message fragment to the step's output.
	if commitMsg := y.generateCommitMessage(cfg.OutFile, filePaths); commitMsg != "" {
		result.Output = map[string]any{
			"commitMessage": commitMsg,
		}
	}
	return result, nil
}

func (y *yamlMerger) generateCommitMessage(path string, fileList []string) string {
	if len(fileList) <= 1 {
		return "no YAML files merged"
	}

	var commitMsg strings.Builder
	_, _ = commitMsg.WriteString(fmt.Sprintf("Merged YAML files to %s\n", path))
	for _, file := range fileList {
		_, _ = commitMsg.WriteString(fmt.Sprintf("\n- %s", file))
	}

	return commitMsg.String()
}
