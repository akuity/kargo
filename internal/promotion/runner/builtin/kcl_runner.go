package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"
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

func (k *kclRunner) resolveKCLFiles(workDir, inputPath string) ([]string, error) {
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
		return kclFiles, nil
	}

	return []string{secureInputPath}, nil
}

func (k *kclRunner) buildKCLOptions(workDir string, kclFiles []string, cfg builtin.KCLRunConfig, externalPkgs []*gpyrpc.ExternalPkg) ([]kcl.Option, error) {
	var opts []kcl.Option

	opts = append(opts, kcl.WithKFilenames(kclFiles...))
	opts = append(opts, kcl.WithWorkDir(workDir))

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

	logger.Debug("executing kcl with options", "inputPath", cfg.InputPath, "args", cfg.Args, "settings", cfg.Settings)

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
