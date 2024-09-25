package directives

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli/values"
)

func init() {
	builtins.RegisterPromotionStepRunner(
		newHelmTemplateRunner(),
		&StepRunnerPermissions{
			AllowArgoCDClient:  true,
			AllowCredentialsDB: true,
		},
	)
}

// helmTemplateRunner is an implementation of the PromotionStepRunner interface
// that renders a Helm chart.
type helmTemplateRunner struct {
	schemaLoader gojsonschema.JSONLoader
}

// newHelmTemplateRunner returns an implementation of the PromotionStepRunner
// interface that renders a Helm chart.
func newHelmTemplateRunner() PromotionStepRunner {
	r := &helmTemplateRunner{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the PromotionStepRunner interface.
func (h *helmTemplateRunner) Name() string {
	return "helm-template"
}

// RunPromotionStep implements the PromotionStepRunner interface.
func (h *helmTemplateRunner) RunPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
) (PromotionStepResult, error) {
	failure := PromotionStepResult{Status: PromotionStatusFailure}

	// Validate the configuration against the JSON Schema
	if err := validate(
		h.schemaLoader,
		gojsonschema.NewGoLoader(stepCtx.Config),
		h.Name(),
	); err != nil {
		return failure, err
	}

	// Convert the configuration into a typed struct
	cfg, err := configToStruct[HelmTemplateConfig](stepCtx.Config)
	if err != nil {
		return failure, fmt.Errorf("could not convert config into %s config: %w", h.Name(), err)
	}

	return h.runPromotionStep(ctx, stepCtx, cfg)
}

func (h *helmTemplateRunner) runPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
	cfg HelmTemplateConfig,
) (PromotionStepResult, error) {
	composedValues, err := h.composeValues(stepCtx.WorkDir, cfg.ValuesFiles)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("failed to compose values: %w", err)
	}

	chartRequested, err := h.loadChart(stepCtx.WorkDir, cfg.Path)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("failed to load chart from %q: %w", cfg.Path, err)
	}

	if err = h.checkDependencies(chartRequested); err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("missing chart dependencies: %w", err)
	}

	install, err := h.newInstallAction(cfg, stepCtx.Project)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("failed to initialize Helm action config: %w", err)
	}

	rls, err := install.RunWithContext(ctx, chartRequested, composedValues)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("failed to render chart: %w", err)
	}

	if err = h.writeOutput(stepCtx.WorkDir, cfg.OutPath, rls.Manifest); err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("failed to write rendered chart: %w", err)
	}
	return PromotionStepResult{Status: PromotionStatusSuccess}, nil
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
func (h *helmTemplateRunner) newInstallAction(cfg HelmTemplateConfig, project string) (*action.Install, error) {
	client := action.NewInstall(&action.Configuration{})

	client.DryRun = true
	client.DryRunOption = "client"
	client.Replace = true
	client.ClientOnly = true
	client.ReleaseName = defaultValue(cfg.ReleaseName, "release-name")
	client.Namespace = defaultValue(cfg.Namespace, project)
	client.IncludeCRDs = cfg.IncludeCRDs
	client.APIVersions = cfg.APIVersions

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
func (h *helmTemplateRunner) loadChart(workDir, path string) (*chart.Chart, error) {
	absChartPath, err := securejoin.SecureJoin(workDir, path)
	if err != nil {
		return nil, fmt.Errorf("failed to join path %q: %w", path, err)
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

// writeOutput writes the rendered manifest to the output path. It creates the
// directory if it does not exist.
func (h *helmTemplateRunner) writeOutput(workDir, outPath, manifest string) error {
	absOutPath, err := securejoin.SecureJoin(workDir, outPath)
	if err != nil {
		return fmt.Errorf("failed to join path %q: %w", outPath, err)
	}
	if err = os.MkdirAll(filepath.Dir(absOutPath), 0o700); err != nil {
		return fmt.Errorf("failed to create directory %q: %w", outPath, err)
	}
	return os.WriteFile(absOutPath, []byte(manifest), 0o600)
}

// defaultValue returns the value if it is not zero or empty, otherwise it
// returns the default value.
func defaultValue[T any](value, defaultValue T) T {
	if v := reflect.ValueOf(value); !v.IsValid() || v.IsZero() || (v.Kind() == reflect.Slice && v.Len() == 0) {
		return defaultValue
	}
	return value
}
