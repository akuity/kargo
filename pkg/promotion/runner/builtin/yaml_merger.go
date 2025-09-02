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
// updates do a merge of the multiple YAML files.
type yamlMerger struct {
	schemaLoader gojsonschema.JSONLoader
}

// newYAMLMerger returns an implementation of the promotion.StepRunner interface
// that updates the values of specified keys in a YAML file.
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
	failure := promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}

	if err := y.validate(stepCtx.Config); err != nil {
		return failure, err
	}

	// Convert the configuration into a typed struct
	cfg, err := promotion.ConfigToStruct[builtin.YAMLMergeConfig](stepCtx.Config)
	if err != nil {
		return failure, fmt.Errorf("could not convert config into %s config: %w", y.Name(), err)
	}

	return y.run(ctx, stepCtx, cfg)
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

	mergedFiles := []string{} // keep track of files actually merged

	// Secure join the paths to prevent path traversal attacks.
	yamlData := []string{}
	for _, path := range cfg.InPaths {
		inPath, err := securejoin.SecureJoin(stepCtx.WorkDir, path)
		if err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf("could not secure join inPath %q: %w", path, err)
		}

		inBytes, err := os.ReadFile(inPath)
		if err != nil {
			// we skip if file does not exist
			if cfg.IgnoreMissingFiles && os.IsNotExist(err) {
				continue
			}
			return failure, fmt.Errorf(
				"error reading file %q: %w",
				inPath,
				err,
			)
		}

		// we skip if file is empty
		if len(inBytes) == 0 {
			continue
		}

		mergedFiles = append(mergedFiles, path)
		yamlData = append(yamlData, string(inBytes))
	}

	// merge YAML files
	outYAML, err := yaml.MergeYAMLFiles(yamlData)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("could not merge YAML files: %w", err)
	}

	// write yaml file
	outPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.OutPath)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("could not secure join outPath %q: %w", cfg.OutPath, err)
	}

	if err = os.MkdirAll(filepath.Dir(outPath), 0o700); err != nil {
		return failure, fmt.Errorf("error creating directory structure %s: %w", filepath.Dir(outPath), err)
	}
	if err = os.WriteFile(outPath, []byte(outYAML), 0o600); err != nil {
		return failure, fmt.Errorf("error writing to file %s: %w", outPath, err)
	}

	// add commit msg
	if commitMsg := y.generateCommitMessage(cfg.OutPath, mergedFiles); commitMsg != "" {
		result.Output = map[string]any{
			"commitMessage": commitMsg,
		}
	}
	return result, nil
}

func (y *yamlMerger) generateCommitMessage(path string, fileList []string) string {
	if len(path) <= 1 {
		return "no YAML file merged"
	}

	var commitMsg strings.Builder
	_, _ = commitMsg.WriteString(fmt.Sprintf("Merged YAML files to %s\n", path))
	for _, file := range fileList {
		_, _ = commitMsg.WriteString(
			fmt.Sprintf(
				"\n- %s",
				file,
			),
		)
	}

	return commitMsg.String()
}
