package builtin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/utils/ptr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_helmTemplateRunner_convert(t *testing.T) {
	tests := []validationTestCase{
		{
			name:   "outPath not specified",
			config: promotion.Config{},
			expectedProblems: []string{
				"(root): outPath is required",
			},
		},
		{
			name: "outPath is empty string",
			config: promotion.Config{
				"outPath": "",
			},
			expectedProblems: []string{
				"outPath: String length must be greater than or equal to 1",
			},
		},
		{
			name: "path not specified",
			config: promotion.Config{
				"outPath": "/output/path",
			},
			expectedProblems: []string{
				"(root): path is required",
			},
		},
		{
			name: "path is empty string",
			config: promotion.Config{
				"outPath": "/output/path",
				"path":    "",
			},
			expectedProblems: []string{
				"path: String length must be greater than or equal to 1",
			},
		},
		{
			name: "releaseName not specified",
			config: promotion.Config{
				"outPath": "/output/path",
				"path":    "/chart/path",
			},
			expectedProblems: []string{
				"(root): releaseName is required",
			},
		},
		{
			name: "releaseName is empty string",
			config: promotion.Config{
				"outPath":     "/output/path",
				"path":        "/chart/path",
				"releaseName": "",
			},
			expectedProblems: []string{
				"releaseName: String length must be greater than or equal to 1",
			},
		},
		{
			name: "all required fields missing",
			config: promotion.Config{
				"namespace": "default",
			},
			expectedProblems: []string{
				"(root): outPath is required",
				"(root): path is required",
				"(root): releaseName is required",
			},
		},
		{
			name: "valuesFiles contains empty string",
			config: promotion.Config{
				"outPath":     "/output/path",
				"path":        "/chart/path",
				"releaseName": "my-release",
				"valuesFiles": []string{"values.yaml", ""},
			},
			expectedProblems: []string{
				"valuesFiles.1: String length must be greater than or equal to 1",
			},
		},
		{
			name: "setValues key not specified",
			config: promotion.Config{
				"outPath":     "/output/path",
				"path":        "/chart/path",
				"releaseName": "my-release",
				"setValues": []promotion.Config{
					{
						"value": "some-value",
					},
				},
			},
			expectedProblems: []string{
				"setValues.0: key is required",
			},
		},
		{
			name: "setValues key is empty string",
			config: promotion.Config{
				"outPath":     "/output/path",
				"path":        "/chart/path",
				"releaseName": "my-release",
				"setValues": []promotion.Config{
					{
						"key":   "",
						"value": "some-value",
					},
				},
			},
			expectedProblems: []string{
				"setValues.0.key: String length must be greater than or equal to 1",
			},
		},
		{
			name: "setValues value not specified",
			config: promotion.Config{
				"outPath":     "/output/path",
				"path":        "/chart/path",
				"releaseName": "my-release",
				"setValues": []promotion.Config{
					{
						"key": "image.tag",
					},
				},
			},
			expectedProblems: []string{
				"setValues.0: value is required",
			},
		},
		{
			name: "apiVersions contains empty string",
			config: promotion.Config{
				"outPath":     "/output/path",
				"path":        "/chart/path",
				"releaseName": "my-release",
				"apiVersions": []string{"v1", ""},
			},
			expectedProblems: []string{
				"apiVersions.1: String length must be greater than or equal to 1",
			},
		},
		{
			name: "valid minimal config",
			config: promotion.Config{
				"outPath":     "/output/path",
				"path":        "/chart/path",
				"releaseName": "my-release",
			},
			expectedProblems: nil,
		},
		{
			name: "valid config with optional boolean fields",
			config: promotion.Config{
				"outPath":           "/output/path",
				"path":              "/chart/path",
				"releaseName":       "my-release",
				"useReleaseName":    true,
				"buildDependencies": true,
				"includeCRDs":       true,
				"disableHooks":      true,
				"skipTests":         true,
			},
			expectedProblems: nil,
		},
		{
			name: "valid config with namespace and kubeVersion",
			config: promotion.Config{
				"outPath":     "/output/path",
				"path":        "/chart/path",
				"releaseName": "my-release",
				"namespace":   "production",
				"kubeVersion": "1.28.0",
			},
			expectedProblems: nil,
		},
		{
			name: "valid config with valuesFiles",
			config: promotion.Config{
				"outPath":     "/output/path",
				"path":        "/chart/path",
				"releaseName": "my-release",
				"valuesFiles": []string{"values.yaml", "values-prod.yaml"},
			},
			expectedProblems: nil,
		},
		{
			name: "valid config with setValues",
			config: promotion.Config{
				"outPath":     "/output/path",
				"path":        "/chart/path",
				"releaseName": "my-release",
				"setValues": []promotion.Config{
					{
						"key":   "image.tag",
						"value": "v1.2.3",
					},
					{
						"key":   "replicaCount",
						"value": "3",
					},
					{
						"key":   "service.type",
						"value": "",
					}, // Empty value should be valid
				},
			},
			expectedProblems: nil,
		},
		{
			name: "valid config with setValues using literal",
			config: promotion.Config{
				"outPath":     "/output/path",
				"path":        "/chart/path",
				"releaseName": "my-release",
				"setValues": []promotion.Config{
					{
						"key":     "image.tag",
						"value":   "v1.2.3",
						"literal": true,
					},
					{
						"key":   "replicaCount",
						"value": "3",
					},
				},
			},
			expectedProblems: nil,
		},
		{
			name: "valid config with apiVersions",
			config: promotion.Config{
				"outPath":     "/output/path",
				"path":        "/chart/path",
				"releaseName": "my-release",
				"apiVersions": []string{"v1", "apps/v1", "networking.k8s.io/v1"},
			},
		},
		{
			name: "valid kitchen sink",
			config: promotion.Config{
				"outPath":        "/output/manifests",
				"path":           "/path/to/helm/chart",
				"releaseName":    "my-application",
				"useReleaseName": true,
				"namespace":      "production",
				"valuesFiles":    []string{"values.yaml", "values-prod.yaml", "secrets.yaml"},
				"setValues": []promotion.Config{
					{
						"key":   "image.repository",
						"value": "myregistry.com/myapp",
					},
					{
						"key":   "image.tag",
						"value": "v2.1.0",
					},
					{
						"key":   "ingress.enabled",
						"value": "true",
					},
				},
				"buildDependencies": true,
				"includeCRDs":       true,
				"disableHooks":      false,
				"skipTests":         true,
				"kubeVersion":       "1.29.0",
				"apiVersions":       []string{"v1", "apps/v1", "networking.k8s.io/v1", "cert-manager.io/v1"},
			},
			expectedProblems: nil,
		},
	}

	r := newHelmTemplateRunner(promotion.StepRunnerCapabilities{})
	runner, ok := r.(*helmTemplateRunner)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

func Test_helmTemplateRunner_run(t *testing.T) {
	tests := []struct {
		name       string
		files      map[string]string
		cfg        builtin.HelmTemplateConfig
		assertions func(*testing.T, string, promotion.StepResult, error)
	}{
		{
			name: "successful run",
			files: map[string]string{
				"values.yaml": "key: value",
				"charts/test-chart/Chart.yaml": `apiVersion: v1
name: test-chart
version: 0.1.0`,
				"charts/test-chart/templates/test.yaml": `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Release.Name }}-configmap
  namespace: {{ .Release.Namespace }}
data:
  value: {{ .Values.key }}
`,
			},
			cfg: builtin.HelmTemplateConfig{
				Path:        "charts/test-chart",
				ValuesFiles: []string{"values.yaml"},
				OutPath:     "output.yaml",
				ReleaseName: "test-release",
				Namespace:   "test-namespace",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				outPath := filepath.Join(workDir, "output.yaml")
				require.FileExists(t, outPath)
				content, err := os.ReadFile(outPath)
				require.NoError(t, err)
				assert.Equal(t, `---
# Source: test-chart/templates/test.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-release-configmap
  namespace: test-namespace
data:
  value: value
`, string(content))
			},
		},
		{
			name: "successful run with flat layout",
			files: map[string]string{
				"values.yaml": "key: value",
				"charts/test-chart/Chart.yaml": `apiVersion: v1
name: test-chart
version: 0.1.0`,
				"charts/test-chart/templates/configmap.yaml": `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-configmap
  namespace: test-namespace
data:
  value: {{ .Values.key }}
---
apiVersion: v1
kind: Secret
metadata:
  name: test-secret
  namespace: test-namespace
data:
  secret: dGVzdA==
`,
			},
			cfg: builtin.HelmTemplateConfig{
				Path:        "charts/test-chart",
				ValuesFiles: []string{"values.yaml"},
				OutPath:     "output",
				OutLayout:   ptr.To(builtin.Flat),
				ReleaseName: "test-release",
				Namespace:   "test-namespace",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				outDir := filepath.Join(workDir, "output")
				require.DirExists(t, outDir)

				// Check that individual resource files were created
				files, err := os.ReadDir(outDir)
				require.NoError(t, err)
				assert.Len(t, files, 2) // Should have 2 files for ConfigMap and Secret

				// Check ConfigMap file
				configMapFile := filepath.Join(outDir, "configmap-test-namespace-test-configmap.yaml")
				require.FileExists(t, configMapFile)
				content, err := os.ReadFile(configMapFile)
				require.NoError(t, err)
				assert.Contains(t, string(content), "kind: ConfigMap")
				assert.Contains(t, string(content), "name: test-configmap")

				// Check Secret file
				secretFile := filepath.Join(outDir, "secret-test-namespace-test-secret.yaml")
				require.FileExists(t, secretFile)
				content, err = os.ReadFile(secretFile)
				require.NoError(t, err)
				assert.Contains(t, string(content), "kind: Secret")
				assert.Contains(t, string(content), "name: test-secret")
			},
		},
		{
			name: "successful run with multiple values",
			files: map[string]string{
				"values1.yaml": "key1: value1",
				"values2.yaml": "key2: value2",
				"charts/test-chart/Chart.yaml": `apiVersion: v1
name: test-chart
version: 0.1.0`,
				"charts/test-chart/templates/test.yaml": `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Release.Name }}-configmap
  namespace: {{ .Release.Namespace }}
data:
  value1: {{ .Values.key1 }}
  value2: {{ .Values.key2 }}
`,
			},
			cfg: builtin.HelmTemplateConfig{
				Path:        "charts/test-chart",
				ValuesFiles: []string{"values1.yaml", "values2.yaml"},
				OutPath:     "output.yaml",
				ReleaseName: "test-release",
				Namespace:   "test-namespace",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				outPath := filepath.Join(workDir, "output.yaml")
				require.FileExists(t, outPath)
				content, err := os.ReadFile(outPath)
				require.NoError(t, err)
				assert.Equal(t, `---
# Source: test-chart/templates/test.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-release-configmap
  namespace: test-namespace
data:
  value1: value1
  value2: value2
`, string(content))
			},
		},
		{
			name: "successful run with set values",
			files: map[string]string{
				"values.yaml": "key1: value1",
				"charts/test-chart/Chart.yaml": `apiVersion: v1
name: test-chart
version: 0.1.0`,
				"charts/test-chart/templates/test.yaml": `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Release.Name }}-configmap
  namespace: {{ .Release.Namespace }}
data:
  value1: {{ .Values.key1 }}
  value2: {{ .Values.key2 }}
`,
			},
			cfg: builtin.HelmTemplateConfig{
				Path:        "charts/test-chart",
				ValuesFiles: []string{"values.yaml"},
				SetValues:   []builtin.SetValues{{Key: "key2", Value: "value2"}},
				OutPath:     "output.yaml",
				ReleaseName: "test-release",
				Namespace:   "test-namespace",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				outPath := filepath.Join(workDir, "output.yaml")
				require.FileExists(t, outPath)
				content, err := os.ReadFile(outPath)
				require.NoError(t, err)
				assert.Equal(t, `---
# Source: test-chart/templates/test.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-release-configmap
  namespace: test-namespace
data:
  value1: value1
  value2: value2
`, string(content))
			},
		},
		{
			name: "successful run with literal set values",
			files: map[string]string{
				"charts/test-chart/Chart.yaml": `apiVersion: v1
name: test-chart
version: 0.1.0`,
				"charts/test-chart/templates/test.yaml": `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Release.Name }}-configmap
  namespace: {{ .Release.Namespace }}
data:
  literalValue: {{ .Values.literalKey }}
  normalValue: {{ .Values.normalKey }}
`,
			},
			cfg: builtin.HelmTemplateConfig{
				Path: "charts/test-chart",
				SetValues: []builtin.SetValues{
					{Key: "literalKey", Value: "foo,bar,baz", Literal: true},
					{Key: "normalKey", Value: "normal-string", Literal: false},
				},
				OutPath:     "output.yaml",
				ReleaseName: "test-release",
				Namespace:   "test-namespace",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				outPath := filepath.Join(workDir, "output.yaml")
				require.FileExists(t, outPath)
				content, err := os.ReadFile(outPath)
				require.NoError(t, err)
				// Both literal and non-literal values should be present in output
				assert.Contains(t, string(content), "literalValue: foo,bar,baz")
				assert.Contains(t, string(content), "normalValue: normal-string")
			},
		},
		{
			name: "successful run with output directory (helm layout)",
			files: map[string]string{
				"values.yaml": "key: value",
				"charts/test-chart/Chart.yaml": `apiVersion: v1
name: test-chart
version: 0.1.0`,
				"charts/test-chart/templates/test.yaml": `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Release.Name }}-configmap
  namespace: {{ .Release.Namespace }}
data:
  value: {{ .Values.key }}
`,
			},
			cfg: builtin.HelmTemplateConfig{
				Path:        "charts/test-chart",
				ValuesFiles: []string{"values.yaml"},
				OutPath:     "output/",
				ReleaseName: "test-release",
				Namespace:   "test-namespace",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				outPath := filepath.Join(workDir, "output", "test-chart")
				require.DirExists(t, outPath)

				content, err := os.ReadFile(filepath.Join(outPath, "templates/test.yaml"))
				require.NoError(t, err)
				assert.Equal(t, `---
# Source: test-chart/templates/test.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-release-configmap
  namespace: test-namespace
data:
  value: value
`, string(content))
			},
		},
		{
			name: "successful run with UseReleaseName",
			files: map[string]string{
				"values.yaml": "key: value",
				"charts/test-chart/Chart.yaml": `apiVersion: v1
name: test-chart
version: 0.1.0`,
				"charts/test-chart/templates/test.yaml": `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Release.Name }}-configmap
  namespace: {{ .Release.Namespace }}
data:
  value: {{ .Values.key }}
`,
			},
			cfg: builtin.HelmTemplateConfig{
				Path:           "charts/test-chart",
				ValuesFiles:    []string{"values.yaml"},
				OutPath:        "output/",
				ReleaseName:    "test-release",
				UseReleaseName: true,
				Namespace:      "test-namespace",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				outPath := filepath.Join(workDir, "output", "test-release", "test-chart")
				require.DirExists(t, outPath)

				content, err := os.ReadFile(filepath.Join(outPath, "templates/test.yaml"))
				require.NoError(t, err)
				assert.Equal(t, `---
# Source: test-chart/templates/test.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-release-configmap
  namespace: test-namespace
data:
  value: value
`, string(content))
			},
		},
		{
			name: "missing values file",
			cfg: builtin.HelmTemplateConfig{
				ValuesFiles: []string{"non-existent.yaml"},
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				require.ErrorContains(t, err, "failed to compose values")
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)

				require.NoFileExists(t, filepath.Join(workDir, "output.yaml"))
			},
		},
		{
			name: "invalid chart",
			cfg: builtin.HelmTemplateConfig{
				Path: "./",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				require.ErrorContains(t, err, "failed to load chart")
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)

				require.NoFileExists(t, filepath.Join(workDir, "output.yaml"))
			},
		},
		{
			name: "Helm action initialization error",
			files: map[string]string{
				"charts/test-chart/Chart.yaml": `apiVersion: v1
name: test-chart
version: 0.1.0`,
			},
			cfg: builtin.HelmTemplateConfig{
				Path:        "charts/test-chart",
				KubeVersion: "invalid",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				require.ErrorContains(t, err, "failed to initialize Helm action config")
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)

				require.NoFileExists(t, filepath.Join(workDir, "output.yaml"))
			},
		},
		{
			name: "template rendering error",
			files: map[string]string{
				"charts/test-chart/Chart.yaml": `apiVersion: v1
name: test-chart
version: 0.1.0`,
				"charts/test-chart/templates/test.yaml": `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Release.Name }}-configmap
data:
  value: {{ .Values.nonexistent | quote }}}
`,
			},
			cfg: builtin.HelmTemplateConfig{
				Path:    "charts/test-chart",
				OutPath: "output.yaml",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				require.ErrorContains(t, err, "failed to render chart")
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)

				require.NoFileExists(t, filepath.Join(workDir, "output.yaml"))
			},
		},
		{
			name: "invalid output path",
			files: map[string]string{
				"output.yaml/foo": "", // Create "output.yaml" as directory
				"chart/Chart.yaml": `apiVersion: v1
name: test-chart
version: 0.1.0`,
				"chart/templates/test.yaml": `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Release.Name }}-configmap
`,
			},
			cfg: builtin.HelmTemplateConfig{
				Path:    "./chart/",
				OutPath: "output.yaml",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				require.ErrorContains(t, err, "failed to write rendered chart")
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)

				require.NoFileExists(t, filepath.Join(workDir, "output.yaml"))
			},
		},
	}

	r := newHelmTemplateRunner(promotion.StepRunnerCapabilities{
		CredsDB: &credentials.FakeDB{},
	})
	runner, ok := r.(*helmTemplateRunner)

	require.True(t, ok)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()
			for p, c := range tt.files {
				require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(workDir, p)), 0o700))
				require.NoError(t, os.WriteFile(filepath.Join(workDir, p), []byte(c), 0o600))
			}
			stepCtx := &promotion.StepContext{
				WorkDir: workDir,
				Project: "test-project",
			}
			result, err := runner.run(context.Background(), stepCtx, tt.cfg)
			tt.assertions(t, workDir, result, err)
		})
	}
}

func Test_helmTemplateRunner_composeValues(t *testing.T) {
	tests := []struct {
		name           string
		workDir        string
		cfg            builtin.HelmTemplateConfig
		valuesContents map[string]string
		assertions     func(*testing.T, map[string]any, error)
	}{
		{
			name:    "successful composition",
			workDir: t.TempDir(),
			cfg: builtin.HelmTemplateConfig{
				ValuesFiles: []string{"values1.yaml", "values2.yaml"},
				SetValues:   []builtin.SetValues{{Key: "key3", Value: "value3"}},
			},
			valuesContents: map[string]string{
				"values1.yaml": "key1: value1",
				"values2.yaml": "key2: value2",
			},
			assertions: func(t *testing.T, result map[string]any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, map[string]any{
					"key1": "value1",
					"key2": "value2",
					"key3": "value3",
				}, result)
			},
		},
		{
			name:    "successful composition with no files",
			workDir: t.TempDir(),
			cfg: builtin.HelmTemplateConfig{
				ValuesFiles: []string{},
				SetValues:   []builtin.SetValues{{Key: "key3", Value: "value3"}},
			},
			valuesContents: map[string]string{},
			assertions: func(t *testing.T, result map[string]any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, map[string]any{
					"key3": "value3",
				}, result)
			},
		},
		{
			name:    "successful composition with no set values",
			workDir: t.TempDir(),
			cfg: builtin.HelmTemplateConfig{
				ValuesFiles: []string{"values1.yaml", "values2.yaml"},
				SetValues:   []builtin.SetValues{},
			},
			valuesContents: map[string]string{
				"values1.yaml": "key1: value1",
				"values2.yaml": "key2: value2",
			}, assertions: func(t *testing.T, result map[string]any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, map[string]any{
					"key1": "value1",
					"key2": "value2",
				}, result)
			},
		},
		{
			name:    "file not found",
			workDir: t.TempDir(),
			cfg: builtin.HelmTemplateConfig{
				ValuesFiles: []string{"non_existent.yaml"},
				SetValues:   []builtin.SetValues{},
			},
			assertions: func(t *testing.T, result map[string]any, err error) {
				assert.Error(t, err)
				assert.Nil(t, result)
			},
		},
		{
			name:    "ignore missing value files - file missing",
			workDir: t.TempDir(),
			cfg: builtin.HelmTemplateConfig{
				ValuesFiles:             []string{"non_existent.yaml"},
				IgnoreMissingValueFiles: true,
				SetValues:               []builtin.SetValues{{Key: "key1", Value: "value1"}},
			},
			assertions: func(t *testing.T, result map[string]any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, map[string]any{
					"key1": "value1",
				}, result)
			},
		},
		{
			name:    "ignore missing value files - file exists",
			workDir: t.TempDir(),
			cfg: builtin.HelmTemplateConfig{
				ValuesFiles:             []string{"values.yaml"},
				IgnoreMissingValueFiles: true,
			},
			valuesContents: map[string]string{
				"values.yaml": "key1: value1",
			},
			assertions: func(t *testing.T, result map[string]any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, map[string]any{
					"key1": "value1",
				}, result)
			},
		},
		{
			name:    "ignore missing value files - mixed existing and missing",
			workDir: t.TempDir(),
			cfg: builtin.HelmTemplateConfig{
				ValuesFiles:             []string{"values.yaml", "missing.yaml"},
				IgnoreMissingValueFiles: true,
			},
			valuesContents: map[string]string{
				"values.yaml": "key1: value1",
			},
			assertions: func(t *testing.T, result map[string]any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, map[string]any{
					"key1": "value1",
				}, result)
			},
		},
		{
			name:    "composition with literal values",
			workDir: t.TempDir(),
			cfg: builtin.HelmTemplateConfig{
				ValuesFiles: []string{},
				SetValues: []builtin.SetValues{
					{Key: "normalKey", Value: "normalValue", Literal: false},
					{Key: "literalKey", Value: "foo,bar,baz", Literal: true},
				},
			},
			valuesContents: map[string]string{},
			assertions: func(t *testing.T, result map[string]any, err error) {
				assert.NoError(t, err)
				// Both values should be present in the merged result
				assert.Equal(t, "normalValue", result["normalKey"])
				assert.Equal(t, "foo,bar,baz", result["literalKey"])
			},
		},
		{
			name:    "composition with only literal values",
			workDir: t.TempDir(),
			cfg: builtin.HelmTemplateConfig{
				ValuesFiles: []string{},
				SetValues: []builtin.SetValues{
					{Key: "key1", Value: "foo,bar,baz", Literal: true},
					{Key: "key2", Value: "baz,bar,foo", Literal: true},
				},
			},
			valuesContents: map[string]string{},
			assertions: func(t *testing.T, result map[string]any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, map[string]any{
					"key1": "foo,bar,baz",
					"key2": "baz,bar,foo",
				}, result)
			},
		},
	}

	runner := &helmTemplateRunner{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for p, c := range tt.valuesContents {
				require.NoError(t, os.WriteFile(filepath.Join(tt.workDir, p), []byte(c), 0o600))
			}
			result, err := runner.composeValues(tt.workDir, tt.cfg)
			tt.assertions(t, result, err)
		})
	}
}

func Test_helmTemplateRunner_newInstallAction(t *testing.T) {
	tests := []struct {
		name       string
		cfg        builtin.HelmTemplateConfig
		project    string
		absOutPath string
		assertions func(*testing.T, *action.Install, error)
	}{
		{
			name: "default values",
			cfg:  builtin.HelmTemplateConfig{},
			assertions: func(t *testing.T, client *action.Install, err error) {
				require.NoError(t, err)
				assert.True(t, client.DryRun)
				assert.Equal(t, "client", client.DryRunOption)
				assert.True(t, client.Replace)
				assert.True(t, client.ClientOnly)
				assert.Equal(t, "release-name", client.ReleaseName)
				assert.Equal(t, "", client.Namespace)
				assert.False(t, client.IncludeCRDs)
				assert.Empty(t, client.APIVersions)
				assert.Nil(t, client.KubeVersion)
				assert.False(t, client.UseReleaseName)
				assert.False(t, client.DisableHooks)
			},
		},
		{
			name: "custom values",
			cfg: builtin.HelmTemplateConfig{
				ReleaseName:    "custom-release",
				UseReleaseName: true,
				Namespace:      "custom-namespace",
				IncludeCRDs:    true,
				APIVersions:    []string{"v1", "v2"},
				KubeVersion:    "1.20.0",
				DisableHooks:   true,
			},
			assertions: func(t *testing.T, client *action.Install, err error) {
				require.NoError(t, err)
				assert.Equal(t, "custom-release", client.ReleaseName)
				assert.True(t, client.UseReleaseName)
				assert.Equal(t, "custom-namespace", client.Namespace)
				assert.True(t, client.IncludeCRDs)
				assert.Equal(t, chartutil.VersionSet{"v1", "v2"}, client.APIVersions)
				assert.NotNil(t, client.KubeVersion)
				assert.Equal(t, "v1.20.0", client.KubeVersion.String())
				assert.True(t, client.DisableHooks)
			},
		},
		{
			name: "output directory with helm layout",
			cfg: builtin.HelmTemplateConfig{
				OutPath:   "output/",
				OutLayout: ptr.To(builtin.Helm),
			},
			absOutPath: "/tmp/output",
			assertions: func(t *testing.T, client *action.Install, err error) {
				require.NoError(t, err)
				assert.Equal(t, "/tmp/output", client.OutputDir)
			},
		},
		{
			name: "output directory with flat layout",
			cfg: builtin.HelmTemplateConfig{
				OutPath:   "output/",
				OutLayout: ptr.To(builtin.Flat),
			},
			absOutPath: "/tmp/output",
			assertions: func(t *testing.T, client *action.Install, err error) {
				require.NoError(t, err)
				assert.Empty(t, client.OutputDir) // Should not set OutputDir for flat layout
			},
		},
		{
			name: "output file (YAML)",
			cfg: builtin.HelmTemplateConfig{
				OutPath: "output.yaml",
			},
			absOutPath: "/tmp/output.yaml",
			assertions: func(t *testing.T, client *action.Install, err error) {
				require.NoError(t, err)
				assert.Empty(t, client.OutputDir)
			},
		},
		{
			name: "output file (YML)",
			cfg: builtin.HelmTemplateConfig{
				OutPath: "output.yml",
			},
			absOutPath: "/tmp/output.yml",
			assertions: func(t *testing.T, client *action.Install, err error) {
				require.NoError(t, err)
				assert.Empty(t, client.OutputDir)
			},
		},
		{
			name:    "project used as default namespace",
			cfg:     builtin.HelmTemplateConfig{},
			project: "test-project",
			assertions: func(t *testing.T, client *action.Install, err error) {
				require.NoError(t, err)
				assert.Equal(t, "test-project", client.Namespace)
			},
		},
		{
			name: "KubeVersion parsing error",
			cfg: builtin.HelmTemplateConfig{
				KubeVersion: "invalid",
			},
			assertions: func(t *testing.T, client *action.Install, err error) {
				require.ErrorContains(t, err, "failed to parse Kubernetes version")
				require.Nil(t, client)
			},
		},
	}

	runner := &helmTemplateRunner{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := runner.newInstallAction(tt.cfg, tt.project, tt.absOutPath)
			tt.assertions(t, client, err)
		})
	}
}

func Test_helmTemplateRunner_loadChart(t *testing.T) {
	absWorkDir, err := filepath.Abs("../../../helm/testdata/charts")
	require.NoError(t, err)

	tests := []struct {
		name       string
		workDir    string
		path       string
		assertions func(*testing.T, *chart.Chart, error)
	}{
		{
			name:    "successful load",
			workDir: absWorkDir,
			path:    "demo-0.1.0.tgz",
			assertions: func(t *testing.T, c *chart.Chart, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, c)
				assert.Equal(t, "demo", c.Name())
				assert.Equal(t, "0.1.0", c.Metadata.Version)
			},
		},
		{
			name:    "chart not found",
			workDir: t.TempDir(),
			path:    "nonexistent.tgz",
			assertions: func(t *testing.T, c *chart.Chart, err error) {
				assert.Error(t, err)
				assert.Nil(t, c)
			},
		},
	}

	runner := &helmTemplateRunner{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := runner.loadChart(tt.workDir, tt.path)
			tt.assertions(t, c, err)
		})
	}
}

func Test_helmTemplateRunner_checkDependencies(t *testing.T) {
	tests := []struct {
		name       string
		chart      *chart.Chart
		assertions func(*testing.T, error)
	}{
		{
			name: "no dependencies",
			chart: &chart.Chart{
				Metadata: &chart.Metadata{},
			},
			assertions: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "with dependencies",
			chart: &chart.Chart{
				Metadata: &chart.Metadata{
					Dependencies: []*chart.Dependency{
						{Name: "dep1", Version: "1.0.0"},
					},
				},
			},
			assertions: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "found in Chart.yaml, but missing in charts/ directory")
			},
		},
	}

	runner := &helmTemplateRunner{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runner.checkDependencies(tt.chart)
			tt.assertions(t, err)
		})
	}
}

func Test_helmTemplateRunner_writeOutput(t *testing.T) {
	tests := []struct {
		name       string
		cfg        builtin.HelmTemplateConfig
		rls        *release.Release
		setup      func(*testing.T) (outPath string)
		assertions func(*testing.T, string, error)
	}{
		{
			name: "successful write to file",
			cfg: builtin.HelmTemplateConfig{
				OutPath: "output.yaml",
			},
			rls: &release.Release{
				Manifest: "key: value",
			},
			setup: func(t *testing.T) (workDir string) {
				return t.TempDir()
			},
			assertions: func(t *testing.T, workDir string, err error) {
				require.NoError(t, err)

				content, err := os.ReadFile(filepath.Join(workDir, "output.yaml"))
				require.NoError(t, err)
				assert.Equal(t, "key: value\n", string(content))
			},
		},
		{
			name: "write to non-existent directory",
			cfg: builtin.HelmTemplateConfig{
				OutPath: "subdir/output.yaml",
			},
			rls: &release.Release{
				Manifest: "key: value",
			},
			setup: func(t *testing.T) (workDir string) {
				return t.TempDir()
			},
			assertions: func(t *testing.T, workDir string, err error) {
				require.NoError(t, err)

				content, err := os.ReadFile(filepath.Join(workDir, "subdir", "output.yaml"))
				require.NoError(t, err)
				assert.Equal(t, "key: value\n", string(content))
			},
		},
		{
			name: "write hooks to separate files",
			cfg: builtin.HelmTemplateConfig{
				OutPath: "output",
			},
			rls: &release.Release{
				Manifest: "main: manifest",
				Hooks: []*release.Hook{
					{Path: "hook1.yaml", Manifest: "hook1: content"},
					{Path: "hook2.yaml", Manifest: "hook2: content"},
				},
			},
			setup: func(t *testing.T) (outPath string) {
				return t.TempDir()
			},
			assertions: func(t *testing.T, workDir string, err error) {
				require.NoError(t, err)

				hook1Content, err := os.ReadFile(filepath.Join(workDir, "output", "hook1.yaml"))
				require.NoError(t, err)
				assert.Equal(t, "---\n# Source: hook1.yaml\nhook1: content\n", string(hook1Content))

				hook2Content, err := os.ReadFile(filepath.Join(workDir, "output", "hook2.yaml"))
				require.NoError(t, err)
				assert.Equal(t, "---\n# Source: hook2.yaml\nhook2: content\n", string(hook2Content))
			},
		},
		{
			name: "write manifest and hooks to flat layout",
			cfg: builtin.HelmTemplateConfig{
				OutPath:   "output",
				OutLayout: ptr.To(builtin.Flat),
			},
			rls: &release.Release{
				Manifest: `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-configmap
  namespace: test-ns
data:
  key: value
---
apiVersion: v1
kind: Secret
metadata:
  name: test-secret
  namespace: test-ns
data:
  secret: dGVzdA==`,
				Hooks: []*release.Hook{
					{
						Path: "hook1.yaml",
						Manifest: `---
apiVersion: v1
kind: Job
metadata:
  name: test-job
  namespace: test-ns
spec:
  template:
    spec:
      containers:
      - name: test
        image: test`,
					},
				},
			},
			setup: func(t *testing.T) (outPath string) {
				return t.TempDir()
			},
			assertions: func(t *testing.T, workDir string, err error) {
				require.NoError(t, err)

				outDir := filepath.Join(workDir, "output")
				files, err := os.ReadDir(outDir)
				require.NoError(t, err)
				assert.Len(t, files, 3) // ConfigMap, Secret, and Job

				// Check that files are named descriptively
				expectedFiles := []string{
					"configmap-test-ns-test-configmap.yaml",
					"secret-test-ns-test-secret.yaml",
					"job-test-ns-test-job.yaml",
				}

				actualFiles := make([]string, len(files))
				for i, f := range files {
					actualFiles[i] = f.Name()
				}

				for _, expected := range expectedFiles {
					assert.Contains(t, actualFiles, expected)
				}
			},
		},
		{
			name: "skip test hooks",
			cfg: builtin.HelmTemplateConfig{
				OutPath:   "output",
				SkipTests: true,
			},
			rls: &release.Release{
				Manifest: "main: manifest",
				Hooks: []*release.Hook{
					{Path: "hook1.yaml", Manifest: "hook1: content"},
					{Path: "test.yaml", Manifest: "test: content", Events: []release.HookEvent{release.HookTest}},
				},
			},
			setup: func(t *testing.T) (outPath string) {
				return t.TempDir()
			},
			assertions: func(t *testing.T, workDir string, err error) {
				require.NoError(t, err)

				hook1Content, err := os.ReadFile(filepath.Join(workDir, "output", "hook1.yaml"))
				require.NoError(t, err)
				assert.Equal(t, "---\n# Source: hook1.yaml\nhook1: content\n", string(hook1Content))

				assert.NoFileExists(t, filepath.Join(workDir, "output", "test.yaml"))
			},
		},
		{
			name: "write hooks to single file",
			cfg: builtin.HelmTemplateConfig{
				OutPath: "output.yaml",
			},
			rls: &release.Release{
				Manifest: "main: manifest",
				Hooks: []*release.Hook{
					{Path: "hook1.yaml", Manifest: "hook1: content"},
					{Path: "hook2.yaml", Manifest: "hook2: content"},
				},
			},
			setup: func(t *testing.T) (outPath string) {
				return t.TempDir()
			},
			assertions: func(t *testing.T, workDir string, err error) {
				require.NoError(t, err)
				content, err := os.ReadFile(filepath.Join(workDir, "output.yaml"))
				require.NoError(t, err)
				assert.Equal(t, `main: manifest
---
# Source: hook1.yaml
hook1: content
---
# Source: hook2.yaml
hook2: content
`, string(content))
			},
		},
		{
			name: "disable hooks",
			cfg: builtin.HelmTemplateConfig{
				OutPath:      "output",
				DisableHooks: true,
			},
			rls: &release.Release{
				Manifest: "main: manifest",
				Hooks: []*release.Hook{
					{Path: "hook1.yaml", Manifest: "hook1: content"},
				},
			},
			setup: func(t *testing.T) (outPath string) {
				return t.TempDir()
			},
			assertions: func(t *testing.T, workDir string, err error) {
				require.NoError(t, err)
				_, err = os.Stat(filepath.Join(workDir, "output", "hook1.yaml"))
				assert.True(t, os.IsNotExist(err))
			},
		},
		{
			name: "append to existing hook file",
			cfg: builtin.HelmTemplateConfig{
				OutPath: "output",
			},
			rls: &release.Release{
				Manifest: "main: manifest",
				Hooks: []*release.Hook{
					{Path: "hook1.yaml", Manifest: "new content"},
				},
			},
			setup: func(t *testing.T) (workDir string) {
				workDir = t.TempDir()

				require.NoError(t, os.MkdirAll(filepath.Join(workDir, "output"), 0o700))
				require.NoError(t, os.WriteFile(
					filepath.Join(workDir, "output", "hook1.yaml"),
					[]byte("existing content\n"),
					0o600,
				))

				return workDir
			},
			assertions: func(t *testing.T, workDir string, err error) {
				require.NoError(t, err)

				content, err := os.ReadFile(filepath.Join(workDir, "output", "hook1.yaml"))
				require.NoError(t, err)
				assert.Equal(t, "existing content\n---\n# Source: hook1.yaml\nnew content\n", string(content))
			},
		},
		{
			name: "UseReleaseName with helm layout",
			cfg: builtin.HelmTemplateConfig{
				OutPath:        "output",
				ReleaseName:    "my-release",
				UseReleaseName: true,
			},
			rls: &release.Release{
				Manifest: "main: manifest",
				Hooks: []*release.Hook{
					{Path: "hook1.yaml", Manifest: "hook1: content"},
				},
			},
			setup: func(t *testing.T) (outPath string) {
				return t.TempDir()
			},
			assertions: func(t *testing.T, workDir string, err error) {
				require.NoError(t, err)

				hookContent, err := os.ReadFile(filepath.Join(workDir, "output", "my-release", "hook1.yaml"))
				require.NoError(t, err)
				assert.Equal(t, "---\n# Source: hook1.yaml\nhook1: content\n", string(hookContent))
			},
		},
	}

	runner := &helmTemplateRunner{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := tt.setup(t)
			err := runner.writeOutput(tt.cfg, tt.rls, filepath.Join(workDir, tt.cfg.OutPath))
			tt.assertions(t, workDir, err)
		})
	}
}

func Test_generateResourceFilename(t *testing.T) {
	runner := &helmTemplateRunner{}

	tests := []struct {
		name     string
		resource string
		expected string
	}{
		{
			name: "simple ConfigMap",
			resource: `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
  namespace: default
data:
  key: value`,
			expected: "configmap-default-test-config.yaml",
		},
		{
			name: "resource with API group",
			resource: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
  namespace: prod
spec:
  replicas: 3`,
			expected: "apps-deployment-prod-test-deployment.yaml",
		},
		{
			name: "cluster-scoped resource",
			resource: `apiVersion: v1
kind: ClusterRole
metadata:
  name: test-cluster-role
rules: []`,
			expected: "clusterrole-test-cluster-role.yaml",
		},
		{
			name: "resource with complex API group",
			resource: `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: test-netpol
  namespace: kube-system
spec: {}`,
			expected: "networking_k8s_io-networkpolicy-kube-system-test-netpol.yaml",
		},
		{
			name:     "invalid YAML",
			resource: `invalid: yaml: content: [`,
			expected: "",
		},
		{
			name: "missing kind",
			resource: `apiVersion: v1
metadata:
  name: test-resource`,
			expected: "",
		},
		{
			name: "missing name",
			resource: `apiVersion: v1
kind: ConfigMap
metadata:
  namespace: default`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runner.generateResourceFilename([]byte(tt.resource))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_extractObjectMetadata(t *testing.T) {
	tests := []struct {
		name              string
		resource          string
		expectedGroup     string
		expectedKind      string
		expectedNamespace string
		expectedName      string
	}{
		{
			name: "core API resource",
			resource: `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
  namespace: default`,
			expectedGroup:     "",
			expectedKind:      "ConfigMap",
			expectedNamespace: "default",
			expectedName:      "test-config",
		},
		{
			name: "resource with API group",
			resource: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
  namespace: prod`,
			expectedGroup:     "apps",
			expectedKind:      "Deployment",
			expectedNamespace: "prod",
			expectedName:      "test-deployment",
		},
		{
			name: "cluster-scoped resource",
			resource: `apiVersion: v1
kind: ClusterRole
metadata:
  name: test-cluster-role`,
			expectedGroup:     "",
			expectedKind:      "ClusterRole",
			expectedNamespace: "",
			expectedName:      "test-cluster-role",
		},
		{
			name:              "invalid YAML",
			resource:          `invalid: yaml: content: [`,
			expectedGroup:     "",
			expectedKind:      "",
			expectedNamespace: "",
			expectedName:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			group, kind, namespace, name := extractObjectMetadata([]byte(tt.resource))
			assert.Equal(t, tt.expectedGroup, group)
			assert.Equal(t, tt.expectedKind, kind)
			assert.Equal(t, tt.expectedNamespace, namespace)
			assert.Equal(t, tt.expectedName, name)
		})
	}
}

func Test_outLayoutIsFlat(t *testing.T) {
	tests := []struct {
		name     string
		cfg      builtin.HelmTemplateConfig
		expected bool
	}{
		{
			name:     "nil layout (default)",
			cfg:      builtin.HelmTemplateConfig{},
			expected: false,
		},
		{
			name: "flat layout",
			cfg: builtin.HelmTemplateConfig{
				OutLayout: ptr.To(builtin.Flat),
			},
			expected: true,
		},
		{
			name: "helm layout",
			cfg: builtin.HelmTemplateConfig{
				OutLayout: ptr.To(builtin.Helm),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := outLayoutIsFlat(tt.cfg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_outLayoutIsHelm(t *testing.T) {
	tests := []struct {
		name     string
		cfg      builtin.HelmTemplateConfig
		expected bool
	}{
		{
			name:     "nil layout (default)",
			cfg:      builtin.HelmTemplateConfig{},
			expected: true,
		},
		{
			name: "flat layout",
			cfg: builtin.HelmTemplateConfig{
				OutLayout: ptr.To(builtin.Flat),
			},
			expected: false,
		},
		{
			name: "helm layout",
			cfg: builtin.HelmTemplateConfig{
				OutLayout: ptr.To(builtin.Helm),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := outLayoutIsHelm(tt.cfg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_outPathIsFile(t *testing.T) {
	tests := []struct {
		name     string
		cfg      builtin.HelmTemplateConfig
		expected bool
	}{
		{
			name: "YAML file",
			cfg: builtin.HelmTemplateConfig{
				OutPath: "output.yaml",
			},
			expected: true,
		},
		{
			name: "YML file",
			cfg: builtin.HelmTemplateConfig{
				OutPath: "output.yml",
			},
			expected: true,
		},
		{
			name: "directory",
			cfg: builtin.HelmTemplateConfig{
				OutPath: "output/",
			},
			expected: false,
		},
		{
			name: "directory without slash",
			cfg: builtin.HelmTemplateConfig{
				OutPath: "output",
			},
			expected: false,
		},
		{
			name: "other file extension",
			cfg: builtin.HelmTemplateConfig{
				OutPath: "output.txt",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := outPathIsFile(tt.cfg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_defaultValue(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		defValue any
		expected any
	}{
		{
			name:     "string: use value",
			value:    "test",
			defValue: "default",
			expected: "test",
		},
		{
			name:     "string: use default",
			value:    "",
			defValue: "default",
			expected: "default",
		},
		{
			name:     "int: use value",
			value:    42,
			defValue: 0,
			expected: 42,
		},
		{
			name:     "int: use default",
			value:    0,
			defValue: 42,
			expected: 42,
		},
		{
			name:     "slice: use value",
			value:    []string{"a", "b"},
			defValue: []string{},
			expected: []string{"a", "b"},
		},
		{
			name:     "slice: use default",
			value:    []string{},
			defValue: []string{"c", "d"},
			expected: []string{"c", "d"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, defaultValue(tt.value, tt.defValue))
		})
	}
}

func Test_isTestHook(t *testing.T) {
	tests := []struct {
		name     string
		hook     *release.Hook
		expected bool
	}{
		{
			name: "test hook",
			hook: &release.Hook{
				Events: []release.HookEvent{release.HookTest},
			},
			expected: true,
		},
		{
			name: "pre-install hook",
			hook: &release.Hook{
				Events: []release.HookEvent{release.HookPreInstall},
			},
			expected: false,
		},
		{
			name: "multiple events including test",
			hook: &release.Hook{
				Events: []release.HookEvent{release.HookPreInstall, release.HookTest},
			},
			expected: true,
		},
		{
			name: "no events",
			hook: &release.Hook{
				Events: []release.HookEvent{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTestHook(tt.hook)
			assert.Equal(t, tt.expected, result)
		})
	}
}
