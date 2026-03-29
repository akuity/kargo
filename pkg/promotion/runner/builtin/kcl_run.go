package builtin

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"
	"k8s.io/apimachinery/pkg/util/yaml"
	kcl "kcl-lang.io/kcl-go"
	kclsettings "kcl-lang.io/kcl-go/pkg/settings"
	gpyrpc "kcl-lang.io/kcl-go/pkg/spec/gpyrpc"
	libyaml "sigs.k8s.io/yaml"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	configbuiltin "github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const stepKindKCLRun = "kcl-run"

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name:  stepKindKCLRun,
			Value: newKCLRunner,
		},
	)
}

type kclRunner struct {
	schemaLoader gojsonschema.JSONLoader
}

func newKCLRunner(promotion.StepRunnerCapabilities) promotion.StepRunner {
	return &kclRunner{schemaLoader: getConfigSchemaLoader(stepKindKCLRun)}
}

func (k *kclRunner) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	cfg, err := k.convert(stepCtx.Config)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &promotion.TerminalError{Err: err}
	}
	return k.run(ctx, stepCtx, cfg)
}

func (k *kclRunner) convert(cfg promotion.Config) (configbuiltin.KCLRunConfig, error) {
	return validateAndConvert[configbuiltin.KCLRunConfig](k.schemaLoader, cfg, stepKindKCLRun)
}

func (k *kclRunner) run(
	_ context.Context,
	stepCtx *promotion.StepContext,
	cfg configbuiltin.KCLRunConfig,
) (promotion.StepResult, error) {
	absPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.Path)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to join path %q: %w", cfg.Path, err)
	}

	absOutPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.OutPath)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to join path %q: %w", cfg.OutPath, err)
	}

	pathList, options, err := k.prepareRun(absPath, stepCtx.WorkDir, cfg)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	result, err := kcl.RunFiles(pathList, options...)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to run KCL: %w", err)
	}

	if err = k.writeOutput(result.GetRawYamlResult(), absOutPath, cfg.OutputFormat); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to write KCL output: %w", err)
	}

	return promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, nil
}

func (k *kclRunner) prepareRun(
	absPath string,
	workDir string,
	cfg configbuiltin.KCLRunConfig,
) ([]string, []kcl.Option, error) {
	if isKCLSettingsFile(absPath) {
		settingsOption, settingsDir, err := k.loadSettingsOption(absPath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load KCL settings file %q: %w", cfg.Path, err)
		}

		options := []kcl.Option{
			settingsOption,
			kcl.WithWorkDir(settingsDir),
		}
		options = append(options, k.argumentOptions(cfg)...)

		depOptions, err := k.dependencyOptions(settingsDir, workDir)
		if err != nil {
			return nil, nil, err
		}
		options = append(options, depOptions...)
		return nil, options, nil
	}

	options := []kcl.Option{kcl.WithWorkDir(workDir)}
	options = append(options, k.argumentOptions(cfg)...)

	depOptions, err := k.dependencyOptions(absPath, workDir)
	if err != nil {
		return nil, nil, err
	}
	options = append(options, depOptions...)

	return []string{absPath}, options, nil
}

func isKCLSettingsFile(path string) bool {
	base := filepath.Base(path)
	return base == "kcl.yaml" || base == "kcl.yml"
}

func (k *kclRunner) argumentOptions(cfg configbuiltin.KCLRunConfig) []kcl.Option {
	if len(cfg.Arguments) == 0 {
		return nil
	}

	args := make([]string, 0, len(cfg.Arguments))
	for _, argument := range cfg.Arguments {
		args = append(args, fmt.Sprintf("%s=%s", argument.Name, argument.Value))
	}
	return []kcl.Option{kcl.WithOptions(args...)}
}

func (k *kclRunner) loadSettingsOption(
	settingsPath string,
) (kcl.Option, string, error) {
	settingsFile, err := kclsettings.LoadFile(settingsPath, nil)
	if err != nil {
		return kcl.Option{}, "", err
	}

	settingsDir := filepath.Dir(settingsPath)
	normalizeSettingsFiles(settingsDir, settingsFile.Config.InputFile)
	normalizeSettingsFiles(settingsDir, settingsFile.Config.InputFiles)
	normalizeSettingsPackageMaps(settingsDir, settingsFile.Config.PackageMaps)

	args := settingsFile.To_ExecProgramArgs()
	args.WorkDir = settingsDir

	return optionFromExecProgramArgs(args), settingsDir, nil
}

func normalizeSettingsFiles(baseDir string, files []string) {
	for i, file := range files {
		switch {
		case strings.Contains(file, "${PWD}"):
			files[i] = strings.ReplaceAll(file, "${PWD}", baseDir)
		case filepath.IsAbs(file):
			continue
		case strings.HasPrefix(file, "${KCL_MOD}"):
			continue
		default:
			files[i] = filepath.Join(baseDir, file)
		}
	}
}

func normalizeSettingsPackageMaps(baseDir string, packageMaps map[string]string) {
	for name, pkgPath := range packageMaps {
		switch {
		case strings.Contains(pkgPath, "${PWD}"):
			packageMaps[name] = strings.ReplaceAll(pkgPath, "${PWD}", baseDir)
		case filepath.IsAbs(pkgPath):
			continue
		case strings.HasPrefix(pkgPath, "${"):
			continue
		default:
			packageMaps[name] = filepath.Join(baseDir, pkgPath)
		}
	}
}

func optionFromExecProgramArgs(args *gpyrpc.ExecProgramArgs) kcl.Option {
	option := kcl.NewOption()
	option.ExecProgramArgs = args
	return *option
}

func (k *kclRunner) dependencyOptions(
	path string,
	workDir string,
) ([]kcl.Option, error) {
	pkgRoot, ok := findKCLPackageRoot(path, workDir)
	if !ok {
		return nil, nil
	}

	deps, err := kcl.UpdateDependencies(&kcl.UpdateDependenciesArgs{ManifestPath: pkgRoot})
	if err != nil {
		return nil, fmt.Errorf("failed to update KCL dependencies: %w", err)
	}
	if len(deps.ExternalPkgs) == 0 {
		return nil, nil
	}

	return []kcl.Option{optionFromExecProgramArgs(&gpyrpc.ExecProgramArgs{
		ExternalPkgs: deps.ExternalPkgs,
	})}, nil
}

func findKCLPackageRoot(path string, workDir string) (string, bool) {
	start := path
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		start = filepath.Dir(path)
	}

	workDir = filepath.Clean(workDir)
	for current := filepath.Clean(start); ; current = filepath.Dir(current) {
		manifestPath := filepath.Join(current, "kcl.mod")
		if info, err := os.Stat(manifestPath); err == nil && !info.IsDir() {
			return current, true
		}

		if current == workDir || current == filepath.Dir(current) {
			break
		}
	}

	return "", false
}

func (k *kclRunner) writeOutput(
	manifest string,
	outPath string,
	outputFmt *configbuiltin.OutputFormat,
) error {
	if pathLooksLikeFile(outPath) {
		if err := os.MkdirAll(filepath.Dir(outPath), 0o700); err != nil {
			return err
		}
		return os.WriteFile(outPath, []byte(strings.TrimSpace(manifest)+"\n"), 0o600)
	}

	if err := os.MkdirAll(outPath, 0o700); err != nil {
		return err
	}

	format := configbuiltin.Kargo
	if outputFmt != nil {
		format = *outputFmt
	}

	return writeManifestDirectory(outPath, manifest, format)
}

func pathLooksLikeFile(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".yaml" || ext == ".yml"
}

func writeManifestDirectory(
	outPath string,
	manifest string,
	format configbuiltin.OutputFormat,
) error {
	if strings.TrimSpace(manifest) == "" {
		return nil
	}

	reader := yaml.NewYAMLReader(bufio.NewReader(strings.NewReader(manifest)))
	var fallbackIndex int

	for {
		document, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read YAML document: %w", err)
		}

		resource := bytes.TrimSpace(document)
		if len(resource) == 0 {
			continue
		}

		resources, err := splitManifestResources(resource)
		if err != nil {
			return err
		}
		for _, resource := range resources {
			fileName := resourceFileName(resource, format)
			if fileName == "" {
				fileName = fmt.Sprintf("resource-%d.yaml", fallbackIndex)
				fallbackIndex++
			}

			if err = os.WriteFile(filepath.Join(outPath, fileName), resource, 0o600); err != nil {
				return fmt.Errorf("failed to write resource to file %q: %w", fileName, err)
			}
		}
	}

	return nil
}

func splitManifestResources(document []byte) ([][]byte, error) {
	var value any
	if err := libyaml.Unmarshal(document, &value); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML document: %w", err)
	}

	resources, err := collectManifestResources(value)
	if err != nil {
		return nil, err
	}
	if len(resources) == 0 {
		return [][]byte{document}, nil
	}
	return resources, nil
}

func collectManifestResources(value any) ([][]byte, error) {
	switch typed := value.(type) {
	case map[string]any:
		if isResourceObject(typed) {
			resource, err := libyaml.Marshal(typed)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal YAML resource: %w", err)
			}
			return [][]byte{bytes.TrimSpace(resource)}, nil
		}

		resources := make([][]byte, 0)
		for _, nested := range typed {
			nestedResources, err := collectManifestResources(nested)
			if err != nil {
				return nil, err
			}
			resources = append(resources, nestedResources...)
		}
		return resources, nil
	case []any:
		resources := make([][]byte, 0)
		for _, nested := range typed {
			nestedResources, err := collectManifestResources(nested)
			if err != nil {
				return nil, err
			}
			resources = append(resources, nestedResources...)
		}
		return resources, nil
	default:
		return nil, nil
	}
}

func isResourceObject(value map[string]any) bool {
	metadata, ok := value["metadata"].(map[string]any)
	if !ok {
		return false
	}
	_, hasAPIVersion := value["apiVersion"]
	_, hasKind := value["kind"]
	_, hasName := metadata["name"]
	return hasAPIVersion && hasKind && hasName
}

func resourceFileName(resource []byte, format configbuiltin.OutputFormat) string {
	group, version, kind, namespace, name := extractYAMLObjectMetadata(resource)
	if kind == "" || name == "" {
		return ""
	}

	switch format {
	case configbuiltin.Kustomize:
		parts := make([]string, 0, 4)
		if namespace != "" {
			parts = append(parts, namespace)
		}
		if group != "" {
			parts = append(parts, strings.ReplaceAll(group, ".", "_"))
		}
		if version != "" {
			parts = append(parts, version)
		}
		parts = append(parts, kind, name)
		return strings.ToLower(strings.Join(parts, "_") + ".yaml")
	case configbuiltin.Kargo:
		fallthrough
	default:
		fileName := kind
		if namespace != "" {
			fileName = namespace + "-" + fileName
		}
		fileName += "-" + name
		return strings.ToLower(fileName + ".yaml")
	}
}

func extractYAMLObjectMetadata(resource []byte) (group, version, kind, namespace, name string) {
	var metaObj struct {
		APIVersion string `json:"apiVersion,omitempty"`
		Kind       string `json:"kind,omitempty"`
		Metadata   struct {
			Name      string `json:"name,omitempty"`
			Namespace string `json:"namespace,omitempty"`
		} `json:"metadata,omitempty"`
	}

	if err := libyaml.Unmarshal(resource, &metaObj); err != nil {
		return "", "", "", "", ""
	}

	version = metaObj.APIVersion
	if parts := strings.Split(metaObj.APIVersion, "/"); len(parts) > 1 {
		group = strings.Join(parts[:len(parts)-1], "/")
		version = parts[len(parts)-1]
	}

	return group, version, metaObj.Kind, metaObj.Metadata.Namespace, metaObj.Metadata.Name
}
