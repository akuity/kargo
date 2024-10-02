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
	"helm.sh/helm/v3/pkg/release"
)

func Test_helmTemplateRunner_runPromotionStep(t *testing.T) {
	tests := []struct {
		name       string
		files      map[string]string
		cfg        HelmTemplateConfig
		assertions func(*testing.T, string, PromotionStepResult, error)
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
			assertions: func(t *testing.T, workDir string, result PromotionStepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, PromotionStepResult{Status: PromotionStatusSuccess}, result)

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
			assertions: func(t *testing.T, workDir string, result PromotionStepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, PromotionStepResult{Status: PromotionStatusSuccess}, result)

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
			name: "successful run with output directory",
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
				OutPath:     "output/",
				ReleaseName: "test-release",
				Namespace:   "test-namespace",
			},
			assertions: func(t *testing.T, workDir string, result PromotionStepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, PromotionStepResult{Status: PromotionStatusSuccess}, result)

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
			name: "missing values file",
			cfg: HelmTemplateConfig{
				ValuesFiles: []string{"non-existent.yaml"},
			},
			assertions: func(t *testing.T, workDir string, result PromotionStepResult, err error) {
				require.ErrorContains(t, err, "failed to compose values")
				assert.Equal(t, PromotionStepResult{Status: PromotionStatusFailure}, result)

				require.NoFileExists(t, filepath.Join(workDir, "output.yaml"))
			},
		},
		{
			name: "invalid chart",
			cfg: HelmTemplateConfig{
				Path: "./",
			},
			assertions: func(t *testing.T, workDir string, result PromotionStepResult, err error) {
				require.ErrorContains(t, err, "failed to load chart")
				assert.Equal(t, PromotionStepResult{Status: PromotionStatusFailure}, result)

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
			assertions: func(t *testing.T, workDir string, result PromotionStepResult, err error) {
				require.ErrorContains(t, err, "missing chart dependencies")
				assert.Equal(t, PromotionStepResult{Status: PromotionStatusFailure}, result)

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
			assertions: func(t *testing.T, workDir string, result PromotionStepResult, err error) {
				require.ErrorContains(t, err, "failed to initialize Helm action config")
				assert.Equal(t, PromotionStepResult{Status: PromotionStatusFailure}, result)

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
			assertions: func(t *testing.T, workDir string, result PromotionStepResult, err error) {
				require.ErrorContains(t, err, "failed to render chart")
				assert.Equal(t, PromotionStepResult{Status: PromotionStatusFailure}, result)

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
			assertions: func(t *testing.T, workDir string, result PromotionStepResult, err error) {
				require.ErrorContains(t, err, "failed to write rendered chart")
				assert.Equal(t, PromotionStepResult{Status: PromotionStatusFailure}, result)

				require.NoFileExists(t, filepath.Join(workDir, "output.yaml"))
			},
		},
	}

	runner := &helmTemplateRunner{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()
			for p, c := range tt.files {
				require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(workDir, p)), 0o700))
				require.NoError(t, os.WriteFile(filepath.Join(workDir, p), []byte(c), 0o600))
			}
			stepCtx := &PromotionStepContext{
				WorkDir: workDir,
				Project: "test-project",
			}
			result, err := runner.runPromotionStep(context.Background(), stepCtx, tt.cfg)
			tt.assertions(t, workDir, result, err)
		})
	}
}

func Test_helmTemplateRunner_composeValues(t *testing.T) {
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

	runner := &helmTemplateRunner{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for p, c := range tt.valuesContents {
				require.NoError(t, os.WriteFile(filepath.Join(tt.workDir, p), []byte(c), 0o600))
			}
			result, err := runner.composeValues(tt.workDir, tt.valuesFiles)
			tt.assertions(t, result, err)
		})
	}
}

func Test_helmTemplateRunner_newInstallAction(t *testing.T) {
	tests := []struct {
		name       string
		cfg        HelmTemplateConfig
		project    string
		absOutPath string
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
			name: "output directory",
			cfg: HelmTemplateConfig{
				OutPath: "output/",
			},
			absOutPath: "/tmp/output",
			assertions: func(t *testing.T, client *action.Install, err error) {
				require.NoError(t, err)
				assert.Equal(t, "/tmp/output", client.OutputDir)
			},
		},
		{
			name: "output file (YAML)",
			cfg: HelmTemplateConfig{
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
			cfg: HelmTemplateConfig{
				OutPath: "output.yml",
			},
			absOutPath: "/tmp/output.yml",
			assertions: func(t *testing.T, client *action.Install, err error) {
				require.NoError(t, err)
				assert.Empty(t, client.OutputDir)
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

	runner := &helmTemplateRunner{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := runner.newInstallAction(tt.cfg, tt.project, tt.absOutPath)
			tt.assertions(t, client, err)
		})
	}
}

func Test_helmTemplateRunner_loadChart(t *testing.T) {
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
		cfg        HelmTemplateConfig
		rls        *release.Release
		setup      func(*testing.T) (outPath string)
		assertions func(*testing.T, string, error)
	}{
		{
			name: "successful write to file",
			cfg: HelmTemplateConfig{
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
			cfg: HelmTemplateConfig{
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
			cfg: HelmTemplateConfig{
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
			name: "skip test hooks",
			cfg: HelmTemplateConfig{
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
			cfg: HelmTemplateConfig{
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
			cfg: HelmTemplateConfig{
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
			cfg: HelmTemplateConfig{
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
