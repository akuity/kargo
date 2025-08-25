package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
	kcl "kcl-lang.io/kcl-go"
	"kcl-lang.io/kcl-go/pkg/native"
	"kcl-lang.io/kcl-go/pkg/spec/gpyrpc"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

// kclRunner is an implementation of the promotion.StepRunner interface that
// executes KCL (Kushion Configuration Language) programs.
type kclRunner struct {
	schemaLoader gojsonschema.JSONLoader
}

// newKCLRunner returns an implementation of the promotion.StepRunner interface
// that executes KCL programs.
func newKCLRunner() promotion.StepRunner {
	r := &kclRunner{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the promotion.StepRunner interface.
func (k *kclRunner) Name() string {
	return "kcl-run"
}

// Run implements the promotion.StepRunner interface.
func (k *kclRunner) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	failure := promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}

	if err := validate(
		k.schemaLoader,
		gojsonschema.NewGoLoader(stepCtx.Config),
		k.Name(),
	); err != nil {
		return failure, err
	}

	cfg, err := promotion.ConfigToStruct[builtin.KCLRunConfig](stepCtx.Config)
	if err != nil {
		return failure, fmt.Errorf("could not convert config into %s config: %w", k.Name(), err)
	}

	return k.run(ctx, stepCtx, cfg)
}

func (k *kclRunner) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.KCLRunConfig,
) (promotion.StepResult, error) {
	failure := promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}

	kclFiles, err := k.resolveKCLFiles(stepCtx.WorkDir, cfg.InputPath)
	if err != nil {
		return failure, err
	}

	err = k.resolveDependencies(ctx, stepCtx.WorkDir, cfg)
	if err != nil {
		return failure, err
	}

	externalPkgs, err := k.updateDependencies(ctx, stepCtx.WorkDir)
	if err != nil {
		return failure, err
	}

	opts, err := k.buildKCLOptions(stepCtx.WorkDir, kclFiles, cfg, externalPkgs)
	if err != nil {
		return failure, err
	}

	yamlResult, err := k.executeKCL(ctx, opts, cfg, externalPkgs, kclFiles, stepCtx.WorkDir)
	if err != nil {
		return failure, err
	}

	return k.handleOutput(stepCtx.WorkDir, cfg.OutputPath, yamlResult)
}

func (k *kclRunner) resolveKCLFiles(workDir string, inputPaths []string) ([]string, error) {
	var allKclFiles []string

	for _, inputPath := range inputPaths {
		secureInputPath, err := securejoin.SecureJoin(workDir, inputPath)
		if err != nil {
			return nil, fmt.Errorf("could not secure join inputPath %q: %w", inputPath, err)
		}

		pathInfo, err := os.Stat(secureInputPath)
		if err != nil {
			return nil, fmt.Errorf("input path %q does not exist: %w", inputPath, err)
		}

		if pathInfo.IsDir() {
			kclFiles, err := k.findKCLFiles(secureInputPath)
			if err != nil {
				return nil, fmt.Errorf("could not find KCL files in directory %q: %w", inputPath, err)
			}
			if len(kclFiles) == 0 {
				return nil, fmt.Errorf("no KCL files (*.k) found in directory %q", inputPath)
			}
			allKclFiles = append(allKclFiles, kclFiles...)
		} else {
			allKclFiles = append(allKclFiles, secureInputPath)
		}
	}

	return allKclFiles, nil
}

func (k *kclRunner) resolveValueFiles(workDir string, valueFiles []string) ([]string, error) {
	var resolvedFiles []string

	for _, valueFile := range valueFiles {
		secureValuePath, err := securejoin.SecureJoin(workDir, valueFile)
		if err != nil {
			return nil, fmt.Errorf("could not secure join value file path %q: %w", valueFile, err)
		}

		pathInfo, err := os.Stat(secureValuePath)
		if err != nil {
			return nil, fmt.Errorf("value file %q does not exist: %w", valueFile, err)
		}

		if pathInfo.IsDir() {
			return nil, fmt.Errorf("value file path %q is a directory, expected a file", valueFile)
		}

		// Validate file extension
		ext := filepath.Ext(secureValuePath)
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			return nil, fmt.Errorf("value file %q must have .yaml, .yml, or .json extension", valueFile)
		}

		resolvedFiles = append(resolvedFiles, secureValuePath)
	}

	return resolvedFiles, nil
}

// parseValueFileToOptions parses a YAML or JSON file and converts it to KCL options
func (k *kclRunner) parseValueFileToOptions(filePath string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read value file: %w", err)
	}

	var values map[string]interface{}
	if err := yaml.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("failed to parse YAML/JSON: %w", err)
	}

	var options []string
	for key, value := range values {
		valueStr, err := k.convertValueToString(value)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize value for key %s: %w", key, err)
		}
		options = append(options, fmt.Sprintf("%s=%s", key, valueStr))
	}

	return options, nil
}

// convertValueToString converts a value of any type to its string representation
// suitable for use as a KCL option value
func (k *kclRunner) convertValueToString(value interface{}) (string, error) {
	var valueStr string
	switch v := value.(type) {
	case string:
		valueStr = v
	case int:
		valueStr = fmt.Sprintf("%d", v)
	case int64:
		valueStr = fmt.Sprintf("%d", v)
	case float64:
		valueStr = fmt.Sprintf("%g", v)
	case bool:
		valueStr = fmt.Sprintf("%t", v)
	case nil:
		valueStr = "null"
	default:
		// For complex types, use JSON encoding
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("failed to serialize complex value: %w", err)
		}
		valueStr = string(jsonBytes)
	}
	return valueStr, nil
}

func (k *kclRunner) buildKCLOptions(workDir string, kclFiles []string, cfg builtin.KCLRunConfig, externalPkgs []*gpyrpc.ExternalPkg) ([]kcl.Option, error) {
	var opts []kcl.Option

	opts = append(opts, kcl.WithKFilenames(kclFiles...))
	opts = append(opts, kcl.WithWorkDir(workDir))

	if len(cfg.ValueFiles) > 0 {
		valueFiles, err := k.resolveValueFiles(workDir, cfg.ValueFiles)
		if err != nil {
			return nil, err
		}

		for _, valueFile := range valueFiles {
			options, err := k.parseValueFileToOptions(valueFile)
			if err != nil {
				return nil, fmt.Errorf("error parsing value file %s: %w", valueFile, err)
			}
			if len(options) > 0 {
				opts = append(opts, kcl.WithOptions(options...))
			}
		}
	}

	if len(cfg.Settings) > 0 {
		var keyValuePairs []string
		for key, value := range cfg.Settings {
			keyValuePairs = append(keyValuePairs, fmt.Sprintf("%s=%s", key, value))
		}
		opts = append(opts, kcl.WithOptions(keyValuePairs...))
	}

	if len(cfg.Args) > 0 {
		opts = append(opts, kcl.WithOptions(cfg.Args...))
	}

	return opts, nil
}

func (k *kclRunner) executeKCL(ctx context.Context, opts []kcl.Option, cfg builtin.KCLRunConfig, externalPkgs []*gpyrpc.ExternalPkg, kclFiles []string, workDir string) (string, error) {
	logger := logging.LoggerFromContext(ctx)

	logger.Debug("executing kcl with options", "inputPaths", cfg.InputPath, "args", cfg.Args, "settings", cfg.Settings)

	if len(externalPkgs) > 0 {
		logger.Debug("executing kcl with external packages", "kclFiles", kclFiles, "workDir", workDir, "numExternalPkgs", len(externalPkgs))
		svc := native.NewNativeServiceClient()

		execResult, err := svc.ExecProgram(&gpyrpc.ExecProgram_Args{
			KFilenameList: kclFiles,
			WorkDir:       workDir,
			ExternalPkgs:  externalPkgs,
		})
		if err != nil {
			return "", fmt.Errorf("error executing kcl with external packages: %w", err)
		}

		logger.Debug("kcl executed successfully with external packages", "outputLength", len(execResult.YamlResult))
		return execResult.YamlResult, nil
	}

	result, err := kcl.Run("", opts...)
	if err != nil {
		return "", fmt.Errorf("error executing kcl: %w", err)
	}

	yamlResult := result.GetRawYamlResult()
	logger.Debug("kcl executed successfully", "outputLength", len(yamlResult))

	return yamlResult, nil
}

func (k *kclRunner) handleOutput(workDir, outputPath, yamlResult string) (promotion.StepResult, error) {
	stepResult := promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}

	if outputPath != "" {
		secureOutputPath, err := securejoin.SecureJoin(workDir, outputPath)
		if err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf("could not secure join outputPath %q: %w", outputPath, err)
		}

		if err := os.MkdirAll(filepath.Dir(secureOutputPath), 0755); err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf("could not create output directory: %w", err)
		}

		if err := os.WriteFile(secureOutputPath, []byte(yamlResult), 0644); err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf("could not write output to file %q: %w", outputPath, err)
		}

		stepResult.Output = map[string]any{
			"outputPath": outputPath,
		}
	} else {
		stepResult.Output = map[string]any{
			"output": yamlResult,
		}
	}

	return stepResult, nil
}

func (k *kclRunner) findKCLFiles(dirPath string) ([]string, error) {
	var kclFiles []string

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("could not read directory %q: %w", dirPath, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".k" {
			fullPath := filepath.Join(dirPath, entry.Name())
			kclFiles = append(kclFiles, fullPath)
		}
	}

	return kclFiles, nil
}

func (k *kclRunner) resolveDependencies(ctx context.Context, workDir string, cfg builtin.KCLRunConfig) error {
	kclModPath := filepath.Join(workDir, "kcl.mod")
	if _, err := os.Stat(kclModPath); os.IsNotExist(err) {
		return nil
	}

	logger := logging.LoggerFromContext(ctx)
	logger.Debug("kcl.mod found, dependencies should be resolved via KCL runtime", "workDir", workDir)

	return nil
}

func (k *kclRunner) updateDependencies(ctx context.Context, workDir string) ([]*gpyrpc.ExternalPkg, error) {
	kclModPath := filepath.Join(workDir, "kcl.mod")
	if _, err := os.Stat(kclModPath); os.IsNotExist(err) {
		return nil, nil // No dependencies to update
	}

	logger := logging.LoggerFromContext(ctx)
	logger.Debug("updating kcl dependencies", "workDir", workDir)

	updateArgs := &gpyrpc.UpdateDependencies_Args{
		ManifestPath: workDir,
	}

	result, err := kcl.UpdateDependencies(updateArgs)
	if err != nil {
		return nil, fmt.Errorf("error updating kcl dependencies: %w", err)
	}

	logger.Debug("kcl dependencies updated successfully", "numExternalPkgs", len(result.ExternalPkgs))
	return result.ExternalPkgs, nil
}
