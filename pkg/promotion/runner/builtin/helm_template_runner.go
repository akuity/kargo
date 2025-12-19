package builtin

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
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
	"k8s.io/apimachinery/pkg/util/yaml"
	libyaml "sigs.k8s.io/yaml"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/helm"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const stepKindHelmTemplate = "helm-template"

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name: stepKindHelmTemplate,
			Metadata: promotion.StepRunnerMetadata{
				RequiredCapabilities: []promotion.StepRunnerCapability{
					promotion.StepCapabilityAccessCredentials,
				},
			},
			Value: newHelmTemplateRunner,
		},
	)
}

// outLayoutIsFlat returns true if the output layout is "flat".
func outLayoutIsFlat(cfg builtin.HelmTemplateConfig) bool {
	return cfg.OutLayout != nil && *cfg.OutLayout == builtin.Flat
}

// outLayoutIsHelm returns true if the output layout is "helm" or not specified
// (default).
func outLayoutIsHelm(cfg builtin.HelmTemplateConfig) bool {
	if cfg.OutLayout == nil {
		return true
	}
	return cfg.OutLayout != nil && *cfg.OutLayout == builtin.Helm
}

// outPathIsFile returns true if the output path contains a YAML extension.
// When true, all rendered manifests will be written to a single file.
// Otherwise, the output path is considered to target a directory where the
// rendered manifests will be written according to the specified outLayout.
func outPathIsFile(cfg builtin.HelmTemplateConfig) bool {
	ext := filepath.Ext(cfg.OutPath)
	return ext == ".yaml" || ext == ".yml"
}

// helmTemplateRunner is an implementation of the promotion.StepRunner interface
// that renders a Helm chart.
type helmTemplateRunner struct {
	schemaLoader gojsonschema.JSONLoader
	credsDB      credentials.Database
}

// newHelmTemplateRunner returns an implementation of the promotion.StepRunner
// interface that renders a Helm chart.
func newHelmTemplateRunner(caps promotion.StepRunnerCapabilities) promotion.StepRunner {
	return &helmTemplateRunner{
		credsDB:      caps.CredsDB,
		schemaLoader: getConfigSchemaLoader(stepKindHelmTemplate),
	}
}

// Run implements the promotion.StepRunner interface.
func (h *helmTemplateRunner) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	cfg, err := h.convert(stepCtx.Config)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &promotion.TerminalError{Err: err}
	}
	return h.run(ctx, stepCtx, cfg)
}

func (h *helmTemplateRunner) convert(cfg promotion.Config) (builtin.HelmTemplateConfig, error) {
	return validateAndConvert[builtin.HelmTemplateConfig](h.schemaLoader, cfg, stepKindHelmTemplate)
}

func (h *helmTemplateRunner) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.HelmTemplateConfig,
) (promotion.StepResult, error) {
	composedValues, err := h.composeValues(stepCtx.WorkDir, cfg)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to compose values: %w", err)
	}

	if cfg.BuildDependencies {
		if err = h.buildDependencies(ctx, stepCtx, cfg.Path); err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf("failed to build chart dependencies: %w", err)
		}
	}

	chartRequested, err := h.loadChart(stepCtx.WorkDir, cfg.Path)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to load chart from %q: %w", cfg.Path, err)
	}

	absOutPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.OutPath)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to join path %q: %w", cfg.OutPath, err)
	}

	if err = h.checkDependencies(chartRequested); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("missing chart dependencies: %w", err)
	}

	install, err := h.newInstallAction(cfg, stepCtx.Project, absOutPath)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to initialize Helm action config: %w", err)
	}

	rls, err := install.RunWithContext(ctx, chartRequested, composedValues)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to render chart: %w", err)
	}

	if err = h.writeOutput(cfg, rls, absOutPath); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to write rendered chart: %w", err)
	}
	return promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, nil

}

// buildDependencies builds the dependencies for the given chart
func (h *helmTemplateRunner) buildDependencies(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	relPath string,
) error {
	manager, err := helm.NewEphemeralDependencyManager(h.credsDB, stepCtx.Project, stepCtx.WorkDir)
	if err != nil {
		return fmt.Errorf("failed to create Helm dependency manager: %w", err)
	}
	return manager.Build(ctx, relPath)
}

// composeValues composes the values from the given values files and set values.
// It merges the value files in the order they are provided. Set values are merged
// into last as overrides
func (h *helmTemplateRunner) composeValues(
	workDir string,
	cfg builtin.HelmTemplateConfig,
) (map[string]any, error) {
	valueOpts := &values.Options{}
	for _, p := range cfg.ValuesFiles {
		absValuesPath, err := securejoin.SecureJoin(workDir, p)
		if err != nil {
			return nil, fmt.Errorf("failed to join path %q: %w", p, err)
		}
		if cfg.IgnoreMissingValueFiles {
			if _, err := os.Stat(absValuesPath); os.IsNotExist(err) {
				continue
			}
		}
		valueOpts.ValueFiles = append(valueOpts.ValueFiles, absValuesPath)
	}

	// Process setValues, separating forceString values from regular values
	for _, setValue := range cfg.SetValues {
		if setValue.Literal {
			// When literal is true, use --set-literal behavior.
			valueOpts.LiteralValues = append(
				valueOpts.LiteralValues,
				fmt.Sprintf("%s=%s", setValue.Key, setValue.Value),
			)
			continue
		}

		// Default behavior uses --set which allows type inference.
		valueOpts.Values = append(valueOpts.Values, fmt.Sprintf("%s=%s", setValue.Key, setValue.Value))
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

	// If the output path is a directory AND the output layout is "helm" or not
	// specified, set the output directory to the output path.
	if !outPathIsFile(cfg) && outLayoutIsHelm(cfg) {
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

// writeOutput writes the rendered manifest to the output path based on the
// configured layout.
func (h *helmTemplateRunner) writeOutput(cfg builtin.HelmTemplateConfig, rls *release.Release, outPath string) error {
	var (
		manifests       bytes.Buffer
		outPathIsFile   = outPathIsFile(cfg)
		outLayoutIsFlat = outLayoutIsFlat(cfg)
	)

	// Handle the main manifest resources based on the output layout.
	switch {
	case outPathIsFile:
		_, _ = fmt.Fprintln(&manifests, strings.TrimSpace(rls.Manifest))
	case outLayoutIsFlat:
		// Flat layout: write the main manifest resources to individual files.
		if err := h.writeManifestFlat(outPath, rls.Manifest); err != nil {
			return fmt.Errorf("failed to write rendered manifest: %w", err)
		}
	}

	if !cfg.DisableHooks {
		for _, hook := range rls.Hooks {
			if cfg.SkipTests && isTestHook(hook) {
				continue
			}

			if outPathIsFile {
				_, _ = fmt.Fprintf(&manifests, "---\n# Source: %s\n%s\n", hook.Path, hook.Manifest)
				continue
			}

			// Handle hooks based on the output layout.
			switch {
			case outLayoutIsFlat:
				// Flat layout: write hook manifest resources to individual
				// files.
				if err := h.writeHookFlat(outPath, hook); err != nil {
					return fmt.Errorf("failed to write hook %q: %w", hook.Path, err)
				}
			case outLayoutIsHelm(cfg):
				// Helm layout: use Helm's file writing logic.
				exists := true
				outDir := outPath
				if cfg.UseReleaseName {
					outDir = filepath.Join(outDir, cfg.ReleaseName)
				}
				if _, err := os.Stat(filepath.Join(outDir, hook.Path)); err != nil {
					if !os.IsNotExist(err) {
						return fmt.Errorf("failed to check if file %q exists: %w", hook.Path, err)
					}
					exists = false
				}

				if err := writeToHelmFile(outDir, hook.Path, hook.Manifest, exists); err != nil {
					return fmt.Errorf("failed to write hook %q: %w", hook.Path, err)
				}
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

// writeManifestFlat writes the main manifest resources to individual files in
// the output directory.
func (h *helmTemplateRunner) writeManifestFlat(outPath, manifest string) error {
	if manifest == "" {
		return nil
	}

	// Ensure the output directory exists.
	if err := os.MkdirAll(outPath, 0o700); err != nil {
		return fmt.Errorf("failed to create output directory %q: %w", outPath, err)
	}

	// Write the manifest resources to individual files.
	return h.writeResourcesFlat(outPath, manifest)
}

// writeHookFlat writes a hook's resources to individual files in the output
// directory.
func (h *helmTemplateRunner) writeHookFlat(outPath string, hook *release.Hook) error {
	// Ensure the output directory exists.
	if err := os.MkdirAll(outPath, 0o700); err != nil {
		return fmt.Errorf("failed to create output directory %q: %w", outPath, err)
	}

	// Write the hook's manifest resources to individual files.
	return h.writeResourcesFlat(outPath, hook.Manifest)
}

// writeResourcesFlat reads YAML documents from a manifest and writes each
// resource to its own file.
func (h *helmTemplateRunner) writeResourcesFlat(outPath, manifest string) error {
	if strings.TrimSpace(manifest) == "" {
		return nil
	}

	reader := yaml.NewYAMLReader(bufio.NewReader(strings.NewReader(manifest)))

	var i int

	for {
		document, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read YAML document: %w", err)
		}

		// Skip empty documents.
		resource := bytes.TrimSpace(document)
		if resource == nil {
			continue
		}

		// Generate a filename for the resource.
		fileName := h.generateResourceFilename(resource)
		if fileName == "" {
			fileName = fmt.Sprintf("resource-%d.yaml", i)
		}

		// Write the resource to the output directory.
		if err = os.WriteFile(filepath.Join(outPath, fileName), resource, 0o600); err != nil {
			return fmt.Errorf("failed to write resource to file %q: %w", fileName, err)
		}
	}

	return nil
}

// generateResourceFilename generates a descriptive filename based on the
// Kubernetes resource metadata in the format of [group-]kind-namespace-name.yaml.
func (h *helmTemplateRunner) generateResourceFilename(resource []byte) string {
	group, kind, namespace, name := extractObjectMetadata(resource)

	if kind == "" || name == "" {
		return ""
	}

	fileName := kind
	if group != "" {
		fileName = strings.ReplaceAll(group, ".", "_") + "-" + fileName
	}
	if namespace != "" {
		fileName += "-" + namespace
	}
	fileName += "-" + name

	return fmt.Sprintf("%s.yaml", strings.ToLower(fileName))
}

// extractObjectMetadata extracts the group, kind, namespace, and name from the
// metadata of a Kubernetes YAML resource.
func extractObjectMetadata(resource []byte) (group, kind, namespace, name string) {
	var metaObj struct {
		APIVersion string `json:"apiVersion,omitempty"`
		Kind       string `json:"kind,omitempty"`
		Metadata   struct {
			Name      string `json:"name,omitempty"`
			Namespace string `json:"namespace,omitempty"`
		} `json:"metadata,omitempty"`
	}

	if err := libyaml.Unmarshal(resource, &metaObj); err != nil {
		return "", "", "", ""
	}

	if parts := strings.Split(metaObj.APIVersion, "/"); len(parts) > 1 {
		group = strings.Join(parts[:len(parts)-1], "/")
	}

	return group, metaObj.Kind, metaObj.Metadata.Namespace, metaObj.Metadata.Name
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

	if _, err = fmt.Fprintf(f, "---\n# Source: %s\n%s\n", name, data); err != nil {
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
