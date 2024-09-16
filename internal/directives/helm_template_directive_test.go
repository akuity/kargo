package directives

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
)

func Test_helmTemplateDirective_run(t *testing.T) {
	tests := []struct {
		name       string
		files      map[string]string
		cfg        HelmTemplateConfig
		assertions func(*testing.T, string, Result, error)
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
			cfg: HelmTemplateConfig{
				Path:        "charts/test-chart",
				ValuesFiles: []string{"values.yaml"},
				OutPath:     "output.yaml",
				ReleaseName: "test-release",
				Namespace:   "test-namespace",
			},
			assertions: func(t *testing.T, workDir string, result Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, Result{Status: StatusSuccess}, result)

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
			cfg: HelmTemplateConfig{
				Path:        "charts/test-chart",
				ValuesFiles: []string{"values1.yaml", "values2.yaml"},
				OutPath:     "output.yaml",
				ReleaseName: "test-release",
				Namespace:   "test-namespace",
			},
			assertions: func(t *testing.T, workDir string, result Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, Result{Status: StatusSuccess}, result)

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
			name: "missing values file",
			cfg: HelmTemplateConfig{
				ValuesFiles: []string{"non-existent.yaml"},
			},
			assertions: func(t *testing.T, workDir string, result Result, err error) {
				require.ErrorContains(t, err, "failed to compose values")
				assert.Equal(t, Result{Status: StatusFailure}, result)

				require.NoFileExists(t, filepath.Join(workDir, "output.yaml"))
			},
		},
		{
			name: "invalid chart",
			cfg: HelmTemplateConfig{
				Path: "./",
			},
			assertions: func(t *testing.T, workDir string, result Result, err error) {
				require.ErrorContains(t, err, "failed to load chart")
				assert.Equal(t, Result{Status: StatusFailure}, result)

				require.NoFileExists(t, filepath.Join(workDir, "output.yaml"))
			},
		},
		{
			name: "missing dependencies",
			files: map[string]string{
				"charts/test-chart/Chart.yaml": `apiVersion: v2
name: test-chart
version: 0.1.0
dependencies:
- name: subchart
  version: 0.1.0
  repository: https://example.com/charts
`,
			},
			cfg: HelmTemplateConfig{
				Path:    "charts/test-chart",
				OutPath: "output.yaml",
			},
			assertions: func(t *testing.T, workDir string, result Result, err error) {
				require.ErrorContains(t, err, "missing chart dependencies")
				assert.Equal(t, Result{Status: StatusFailure}, result)

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
			cfg: HelmTemplateConfig{
				Path:        "charts/test-chart",
				KubeVersion: "invalid",
			},
			assertions: func(t *testing.T, workDir string, result Result, err error) {
				require.ErrorContains(t, err, "failed to initialize Helm action config")
				assert.Equal(t, Result{Status: StatusFailure}, result)

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
			cfg: HelmTemplateConfig{
				Path:    "charts/test-chart",
				OutPath: "output.yaml",
			},
			assertions: func(t *testing.T, workDir string, result Result, err error) {
				require.ErrorContains(t, err, "failed to render chart")
				assert.Equal(t, Result{Status: StatusFailure}, result)

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
			cfg: HelmTemplateConfig{
				Path:    "./chart/",
				OutPath: "output.yaml",
			},
			assertions: func(t *testing.T, workDir string, result Result, err error) {
				require.ErrorContains(t, err, "failed to write rendered chart")
				assert.Equal(t, Result{Status: StatusFailure}, result)

				require.NoFileExists(t, filepath.Join(workDir, "output.yaml"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()
			for p, c := range tt.files {
				require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(workDir, p)), 0o700))
				require.NoError(t, os.WriteFile(filepath.Join(workDir, p), []byte(c), 0o600))
			}

			d := &helmTemplateDirective{}
			stepCtx := &StepContext{
				WorkDir: workDir,
				Project: "test-project",
			}
			result, err := d.run(context.Background(), stepCtx, tt.cfg)
			tt.assertions(t, workDir, result, err)
		})
	}
}

func Test_helmTemplateDirective_composeValues(t *testing.T) {
	tests := []struct {
		name           string
		workDir        string
		valuesFiles    []string
		valuesContents map[string]string
		assertions     func(*testing.T, map[string]any, error)
	}{
		{
			name:        "successful composition",
			workDir:     t.TempDir(),
			valuesFiles: []string{"values1.yaml", "values2.yaml"},
			valuesContents: map[string]string{
				"values1.yaml": "key1: value1",
				"values2.yaml": "key2: value2",
			},
			assertions: func(t *testing.T, result map[string]any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, map[string]any{
					"key1": "value1",
					"key2": "value2",
				}, result)
			},
		},
		{
			name:        "file not found",
			workDir:     t.TempDir(),
			valuesFiles: []string{"non_existent.yaml"},
			assertions: func(t *testing.T, result map[string]any, err error) {
				assert.Error(t, err)
				assert.Nil(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for p, c := range tt.valuesContents {
				require.NoError(t, os.WriteFile(filepath.Join(tt.workDir, p), []byte(c), 0o600))
			}

			d := &helmTemplateDirective{}
			result, err := d.composeValues(tt.workDir, tt.valuesFiles)
			tt.assertions(t, result, err)
		})
	}
}

func Test_helmTemplateDirective_newInstallAction(t *testing.T) {
	tests := []struct {
		name       string
		cfg        HelmTemplateConfig
		project    string
		assertions func(*testing.T, *action.Install, error)
	}{
		{
			name: "default values",
			cfg:  HelmTemplateConfig{},
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
			},
		},
		{
			name: "custom values",
			cfg: HelmTemplateConfig{
				ReleaseName: "custom-release",
				Namespace:   "custom-namespace",
				IncludeCRDs: true,
				APIVersions: []string{"v1", "v2"},
				KubeVersion: "1.20.0",
			},
			assertions: func(t *testing.T, client *action.Install, err error) {
				require.NoError(t, err)
				assert.Equal(t, "custom-release", client.ReleaseName)
				assert.Equal(t, "custom-namespace", client.Namespace)
				assert.True(t, client.IncludeCRDs)
				assert.Equal(t, chartutil.VersionSet{"v1", "v2"}, client.APIVersions)
				assert.NotNil(t, client.KubeVersion)
				assert.Equal(t, "v1.20.0", client.KubeVersion.String())
			},
		},
		{
			name: "KubeVersion parsing error",
			cfg: HelmTemplateConfig{
				KubeVersion: "invalid",
			},
			assertions: func(t *testing.T, client *action.Install, err error) {
				require.ErrorContains(t, err, "failed to parse Kubernetes version")
				require.Nil(t, client)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &helmTemplateDirective{}
			client, err := d.newInstallAction(tt.cfg, tt.project)
			tt.assertions(t, client, err)
		})
	}
}

func TestHelmTemplateDirective_loadChart(t *testing.T) {
	tests := []struct {
		name       string
		workDir    string
		path       string
		assertions func(*testing.T, *chart.Chart, error)
	}{
		{
			name:    "successful load",
			workDir: "testdata/helm/charts",
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &helmTemplateDirective{}
			c, err := d.loadChart(tt.workDir, tt.path)
			tt.assertions(t, c, err)
		})
	}
}

func Test_helmTemplateDirective_checkDependencies(t *testing.T) {
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &helmTemplateDirective{}
			err := d.checkDependencies(tt.chart)
			tt.assertions(t, err)
		})
	}
}

func Test_helmTemplateDirective_writeOutput(t *testing.T) {
	tests := []struct {
		name       string
		workDir    string
		outPath    string
		manifest   string
		setup      func(string)
		assertions func(*testing.T, string, error)
	}{
		{
			name:     "successful write",
			workDir:  t.TempDir(),
			outPath:  "output.yaml",
			manifest: "key: value",
			assertions: func(t *testing.T, workDir string, err error) {
				require.NoError(t, err)
				content, err := os.ReadFile(filepath.Join(workDir, "output.yaml"))
				require.NoError(t, err)
				assert.Equal(t, "key: value", string(content))
			},
		},
		{
			name:     "write to non-existent directory",
			workDir:  t.TempDir(),
			outPath:  "subdir/output.yaml",
			manifest: "key: value",
			assertions: func(t *testing.T, workDir string, err error) {
				require.NoError(t, err)
				content, err := os.ReadFile(filepath.Join(workDir, "subdir", "output.yaml"))
				require.NoError(t, err)
				assert.Equal(t, "key: value", string(content))
			},
		},
		{
			name:     "path traversal outside work directory",
			workDir:  t.TempDir(),
			outPath:  "../../output.yaml",
			manifest: "key: value",
			assertions: func(t *testing.T, workDir string, err error) {
				require.NoError(t, err)
				content, err := os.ReadFile(filepath.Join(workDir, "output.yaml"))
				require.NoError(t, err)
				assert.Equal(t, "key: value", string(content))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(tt.workDir)
			}
			d := &helmTemplateDirective{}
			err := d.writeOutput(tt.workDir, tt.outPath, tt.manifest)
			tt.assertions(t, tt.workDir, err)
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
