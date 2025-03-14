package builtin

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/release"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

// outPathIsFile returns true if the output path contains a YAML extension.
// Otherwise, the output path is considered to target a directory where the
// rendered manifest will be written to.
func outPathIsFile(cfg builtin.HelmTemplateConfig) bool {
	ext := filepath.Ext(cfg.OutPath)
	return ext == ".yaml" || ext == ".yml"
}

// helmTemplateRunner is an implementation of the promotion.StepRunner interface
// that renders a Helm chart.
type helmTemplateRunner struct {
	schemaLoader gojsonschema.JSONLoader
}

// newHelmTemplateRunner returns an implementation of the promotion.StepRunner
// interface that renders a Helm chart.
func newHelmTemplateRunner() promotion.StepRunner {
	r := &helmTemplateRunner{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the promotion.StepRunner interface.
func (h *helmTemplateRunner) Name() string {
	return "helm-template"
}

// Run implements the promotion.StepRunner interface.
func (h *helmTemplateRunner) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	failure := promotion.StepResult{Status: kargoapi.PromotionPhaseErrored}

	// Validate the configuration against the JSON Schema
	if err := validate(
		h.schemaLoader,
		gojsonschema.NewGoLoader(stepCtx.Config),
		h.Name(),
	); err != nil {
		return failure, err
	}

	// Convert the configuration into a typed struct
	cfg, err := promotion.ConfigToStruct[builtin.HelmTemplateConfig](stepCtx.Config)
	if err != nil {
		return failure, fmt.Errorf("could not convert config into %s config: %w", h.Name(), err)
	}

	return h.run(ctx, stepCtx, cfg)
}

func (h *helmTemplateRunner) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.HelmTemplateConfig,
) (promotion.StepResult, error) {
	composedValues, err := h.composeValues(stepCtx.WorkDir, cfg.ValuesFiles)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("failed to compose values: %w", err)
	}

	chartRequested, err := h.loadChart(stepCtx.WorkDir, cfg.Path)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("failed to load chart from %q: %w", cfg.Path, err)
	}

	if err = h.checkDependencies(chartRequested); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("missing chart dependencies: %w", err)
	}

	absOutPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.OutPath)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("failed to join path %q: %w", cfg.OutPath, err)
	}

	install, err := h.newInstallAction(cfg, stepCtx.Project, absOutPath)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("failed to initialize Helm action config: %w", err)
	}

	rls, err := install.RunWithContext(ctx, chartRequested, composedValues)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("failed to render chart: %w", err)
	}

	if err = h.writeOutput(cfg, rls, absOutPath); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("failed to write rendered chart: %w", err)
	}
	return promotion.StepResult{Status: kargoapi.PromotionPhaseSucceeded}, nil
}

// composeValues composes the values from the given values files. It merges the
// values in the order they are provided.
func (h *helmTemplateRunner) composeValues(workDir string, valuesFiles []string) (map[string]any, error) {
	valueOpts := &values.Options{}
	for _, p := range valuesFiles {
		absValuesPath, err := securejoin.SecureJoin(workDir, p)
		if err != nil {
			return nil, fmt.Errorf("failed to join path %q: %w", p, err)
		}
		valueOpts.ValueFiles = append(valueOpts.ValueFiles, absValuesPath)
	}
	return valueOpts.MergeValues(nil)
}

// newInstallAction creates a new Helm install action with the given
// configuration. It sets the action to dry-run mode and client-only mode,
// meaning that it will not install the chart, but only render the manifest.
func (h *helmTemplateRunner) newInstallAction(
	cfg builtin.HelmTemplateConfig,
	project, absOutPath string,
) (*action.Install, error) {
	client := action.NewInstall(&action.Configuration{})

	client.DryRun = true
	client.DryRunOption = "client"
	client.Replace = true
	client.ClientOnly = true
	client.ReleaseName = defaultValue(cfg.ReleaseName, "release-name")
	client.UseReleaseName = cfg.UseReleaseName
	client.Namespace = defaultValue(cfg.Namespace, project)
	client.IncludeCRDs = cfg.IncludeCRDs
	client.APIVersions = cfg.APIVersions
	client.DisableHooks = cfg.DisableHooks

	// If the output path does not have a YAML extension, it is considered a
	// directory where the manifest will be written to.
	if !outPathIsFile(cfg) {
		client.OutputDir = absOutPath
	}

	if cfg.KubeVersion != "" {
		kubeVersion, err := chartutil.ParseKubeVersion(cfg.KubeVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Kubernetes version %q: %w", cfg.KubeVersion, err)
		}
		client.KubeVersion = kubeVersion
	}

	return client, nil
}

// loadChart loads the chart from the given path.
func (h *helmTemplateRunner) loadChart(workDir, relPath string) (*chart.Chart, error) {
	absChartPath, err := securejoin.SecureJoin(workDir, relPath)
	if err != nil {
		return nil, fmt.Errorf("failed to join relPath %q: %w", relPath, err)
	}
	return loader.Load(absChartPath)
}

// checkDependencies checks if the chart has all its dependencies.
func (h *helmTemplateRunner) checkDependencies(chartRequested *chart.Chart) error {
	if req := chartRequested.Metadata.Dependencies; req != nil {
		if err := action.CheckDependencies(chartRequested, req); err != nil {
			return err
		}
	}
	return nil
}

// writeOutput writes the rendered manifest to the output path.
func (h *helmTemplateRunner) writeOutput(cfg builtin.HelmTemplateConfig, rls *release.Release, outPath string) error {
	var (
		manifests     bytes.Buffer
		outPathIsFile = outPathIsFile(cfg)
	)

	if outPathIsFile {
		_, _ = fmt.Fprintln(&manifests, strings.TrimSpace(rls.Manifest))
	}

	if !cfg.DisableHooks {
		for _, h := range rls.Hooks {
			if cfg.SkipTests && isTestHook(h) {
				continue
			}

			if outPathIsFile {
				_, _ = fmt.Fprintf(&manifests, "---\n# Source: %s\n%s\n", h.Path, h.Manifest)
				continue
			}

			exists := true
			outDir := outPath
			if cfg.UseReleaseName {
				outDir = filepath.Join(outDir, cfg.ReleaseName)
			}
			if _, err := os.Stat(filepath.Join(outDir, h.Path)); err != nil {
				if !os.IsNotExist(err) {
					return fmt.Errorf("failed to check if file %q exists: %w", h.Path, err)
				}
				exists = false
			}

			if err := writeToHelmFile(outPath, h.Path, h.Manifest, exists); err != nil {
				return fmt.Errorf("failed to write hook %q: %w", h.Path, err)
			}
		}
	}

	if !outPathIsFile {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o700); err != nil {
		return fmt.Errorf("failed to create directory %q: %w", cfg.OutPath, err)
	}
	return os.WriteFile(outPath, manifests.Bytes(), 0o600)
}

// defaultValue returns the value if it is not zero or empty, otherwise it
// returns the default value.
func defaultValue[T any](value, defaultValue T) T {
	if v := reflect.ValueOf(value); !v.IsValid() || v.IsZero() || (v.Kind() == reflect.Slice && v.Len() == 0) {
		return defaultValue
	}
	return value
}

// isTestHook returns true if the hook is a test hook.
func isTestHook(h *release.Hook) bool {
	for _, e := range h.Events {
		if e == release.HookTest {
			return true
		}
	}
	return false
}

// The logic below is directly derived from the Helm source code:
// https://github.com/helm/helm/blob/b2286c4caabdfdcf2baaecb42819db9d38c04597/cmd/helm/template.go#L222
// Licensed under the Apache License 2.0.

// writeToHelmFile writes the given data to the output directory with the given
// name. If the appendMode flag is set to true, the data is appended to the file.
func writeToHelmFile(outputDir string, name string, data string, appendMode bool) (err error) {
	outfileName := strings.Join([]string{outputDir, name}, string(filepath.Separator))

	if err = ensureDirectoryForHelmFile(outfileName); err != nil {
		return err
	}

	f, err := createOrOpenHelmFile(outfileName, appendMode)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	if _, err = f.WriteString(fmt.Sprintf("---\n# Source: %s\n%s\n", name, data)); err != nil {
		return err
	}

	return nil
}

// createOrOpenHelmFile creates or opens the file with the given name. If the
// append flag is set to true, the file is opened in append mode.
func createOrOpenHelmFile(filename string, appendMode bool) (*os.File, error) {
	if appendMode {
		return os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0o600)
	}
	return os.Create(filename)
}

// ensureDirectoryForHelmFile ensures that the directory for the given file
// exists. If the directory does not exist, it is created with the default
// permissions.
func ensureDirectoryForHelmFile(file string) error {
	baseDir := path.Dir(file)
	if _, err := os.Stat(baseDir); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.MkdirAll(baseDir, 0o755)
}
