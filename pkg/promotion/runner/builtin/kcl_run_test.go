package builtin

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	configbuiltin "github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_kclRunner_convert(t *testing.T) {
	tests := []validationTestCase{
		{
			name:   "path not specified",
			config: promotion.Config{},
			expectedProblems: []string{
				"(root): path is required",
				"(root): outPath is required",
			},
		},
		{
			name: "path is empty string",
			config: promotion.Config{
				"path":    "",
				"outPath": "./out.yaml",
			},
			expectedProblems: []string{
				"path: String length must be greater than or equal to 1",
			},
		},
		{
			name: "outPath is empty string",
			config: promotion.Config{
				"path":    "./app/main.k",
				"outPath": "",
			},
			expectedProblems: []string{
				"outPath: String length must be greater than or equal to 1",
			},
		},
		{
			name: "argument name missing",
			config: promotion.Config{
				"path":    "./app/main.k",
				"outPath": "./out.yaml",
				"arguments": []promotion.Config{{
					"value": "demo",
				}},
			},
			expectedProblems: []string{
				"arguments.0: name is required",
			},
		},
		{
			name: "argument name empty",
			config: promotion.Config{
				"path":    "./app/main.k",
				"outPath": "./out.yaml",
				"arguments": []promotion.Config{{
					"name":  "",
					"value": "demo",
				}},
			},
			expectedProblems: []string{
				"arguments.0.name: String length must be greater than or equal to 1",
			},
		},
		{
			name: "invalid outputFormat",
			config: promotion.Config{
				"path":         "./app/main.k",
				"outPath":      "./out",
				"outputFormat": "invalid",
			},
			expectedProblems: []string{
				"outputFormat: outputFormat must be one of the following:",
			},
		},
		{
			name: "valid minimal config",
			config: promotion.Config{
				"path":    "./app/main.k",
				"outPath": "./out.yaml",
			},
		},
		{
			name: "valid config with arguments and output format",
			config: promotion.Config{
				"path":         "./app/main.k",
				"outPath":      "./out",
				"outputFormat": "kustomize",
				"arguments": []promotion.Config{{
					"name":  "appName",
					"value": "demo",
				}},
			},
		},
	}

	r := newKCLRunner(promotion.StepRunnerCapabilities{})
	runner, ok := r.(*kclRunner)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

func Test_kclRunner_run(t *testing.T) {
	tests := []struct {
		name       string
		setupFiles func(*testing.T, string)
		config     configbuiltin.KCLRunConfig
		assertions func(*testing.T, string, promotion.StepResult, error)
	}{
		{
			name: "successful run to file",
			setupFiles: func(t *testing.T, dir string) {
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "app"), 0o700))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "app", "main.k"), []byte(`
app = {
    apiVersion = "apps/v1"
    kind = "Deployment"
    metadata = {
        name = "demo"
    }
}
`), 0o600))
			},
			config: configbuiltin.KCLRunConfig{
				Path:    "./app/main.k",
				OutPath: "./out.yaml",
			},
			assertions: func(t *testing.T, dir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				b, readErr := os.ReadFile(filepath.Join(dir, "out.yaml"))
				require.NoError(t, readErr)
				assert.Contains(t, string(b), "kind: Deployment")
				assert.Contains(t, string(b), "name: demo")
			},
		},
		{
			name: "successful run to directory with kargo format",
			setupFiles: func(t *testing.T, dir string) {
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "app"), 0o700))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "app", "main.k"), []byte(`
items = [
    {
        apiVersion = "apps/v1"
        kind = "Deployment"
        metadata = {
            name = "demo"
        }
    },
    {
        apiVersion = "v1"
        kind = "Service"
        metadata = {
            name = "demo"
            namespace = "prod"
        }
    }
]
`), 0o600))
			},
			config: configbuiltin.KCLRunConfig{
				Path:         "./app/main.k",
				OutPath:      "./out",
				OutputFormat: ptr.To(configbuiltin.Kargo),
			},
			assertions: func(t *testing.T, dir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				assert.FileExists(t, filepath.Join(dir, "out", "deployment-demo.yaml"))
				assert.FileExists(t, filepath.Join(dir, "out", "prod-service-demo.yaml"))
			},
		},
		{
			name: "successful run to directory with kustomize format",
			setupFiles: func(t *testing.T, dir string) {
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "app"), 0o700))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "app", "main.k"), []byte(`
app = {
    apiVersion = "apps/v1"
    kind = "Deployment"
    metadata = {
        name = option("appName")
        namespace = "prod"
    }
}
`), 0o600))
			},
			config: configbuiltin.KCLRunConfig{
				Path:         "./app/main.k",
				OutPath:      "./out",
				OutputFormat: ptr.To(configbuiltin.Kustomize),
				Arguments: []configbuiltin.Argument{{
					Name:  "appName",
					Value: "demo",
				}},
			},
			assertions: func(t *testing.T, dir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				assert.FileExists(t, filepath.Join(dir, "out", "prod_apps_v1_deployment_demo.yaml"))
			},
		},
		{
			name: "successful run from kcl.yaml input path",
			setupFiles: func(t *testing.T, dir string) {
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "app"), 0o700))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "app", "main.k"), []byte(`
app = {
    apiVersion = "v1"
    kind = "ConfigMap"
    metadata = {
        name = option("app-name")
        namespace = option("namespace")
    }
    data = {
        version = option("version")
    }
}
`), 0o600))
				require.NoError(
					t,
					os.WriteFile(
						filepath.Join(dir, "app", "kcl.yaml"),
						[]byte(`kcl_cli_configs:
  file:
    - main.k

kcl_options:
  - key: app-name
    value: demo
  - key: namespace
    value: prod
  - key: version
    value: "v1.2.3"
`),
						0o600,
					),
				)
			},
			config: configbuiltin.KCLRunConfig{
				Path:    "./app/kcl.yaml",
				OutPath: "./out.yaml",
			},
			assertions: func(t *testing.T, dir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				b, readErr := os.ReadFile(filepath.Join(dir, "out.yaml"))
				require.NoError(t, readErr)
				assert.Contains(t, string(b), "kind: ConfigMap")
				assert.Contains(t, string(b), "name: demo")
				assert.Contains(t, string(b), "namespace: prod")
				assert.Contains(t, string(b), "version: v1.2.3")
			},
		},
		{
			name: "successful run with public remote dependency",
			setupFiles: func(t *testing.T, dir string) {
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "app"), 0o700))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "app", "kcl.mod"), []byte(`[package]
name = "remote_dep"
edition = "0.0.1"
version = "0.0.1"

[dependencies]
helloworld = { oci = "oci://ghcr.io/kcl-lang/helloworld", tag = "0.1.0" }
`), 0o600))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "app", "main.k"), []byte(`
import helloworld

message = helloworld.The_first_kcl_program
`), 0o600))
			},
			config: configbuiltin.KCLRunConfig{
				Path:    "./app",
				OutPath: "./out.yaml",
			},
			assertions: func(t *testing.T, dir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				b, readErr := os.ReadFile(filepath.Join(dir, "out.yaml"))
				require.NoError(t, readErr)
				assert.Contains(t, string(b), "Hello World")
			},
		},
		{
			name:       "missing input file",
			setupFiles: func(*testing.T, string) {},
			config: configbuiltin.KCLRunConfig{
				Path:    "./missing/main.k",
				OutPath: "./out.yaml",
			},
			assertions: func(t *testing.T, dir string, result promotion.StepResult, err error) {
				require.Error(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)
				assert.NoFileExists(t, filepath.Join(dir, "out.yaml"))
			},
		},
	}

	runner := &kclRunner{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			tt.setupFiles(t, tempDir)

			stepCtx := &promotion.StepContext{WorkDir: tempDir}
			result, err := runner.run(t.Context(), stepCtx, tt.config)
			tt.assertions(t, tempDir, result, err)
		})
	}
}

func Test_kclRunner_Run(t *testing.T) {
	t.Run("invalid config returns terminal error", func(t *testing.T) {
		runner := newKCLRunner(promotion.StepRunnerCapabilities{})

		result, err := runner.Run(
			t.Context(),
			&promotion.StepContext{
				WorkDir: t.TempDir(),
				Config:  promotion.Config{},
			},
		)

		require.Error(t, err)
		assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed}, result)

		var terminalErr *promotion.TerminalError
		require.True(t, errors.As(err, &terminalErr))
	})

	t.Run("valid config delegates to run", func(t *testing.T) {
		tempDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "app"), 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "app", "main.k"), []byte(`
app = {
    apiVersion = "v1"
    kind = "ConfigMap"
    metadata = {
        name = "demo"
    }
}
`), 0o600))

		runner := newKCLRunner(promotion.StepRunnerCapabilities{})
		result, err := runner.Run(
			t.Context(),
			&promotion.StepContext{
				WorkDir: tempDir,
				Config: promotion.Config{
					"path":    "./app/main.k",
					"outPath": "./out.yaml",
				},
			},
		)

		require.NoError(t, err)
		assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)
		assert.FileExists(t, filepath.Join(tempDir, "out.yaml"))
	})
}

func Test_normalizeSettingsFiles(t *testing.T) {
	baseDir := t.TempDir()
	absPath := filepath.Join(baseDir, "absolute.k")
	files := []string{
		"${PWD}/from-pwd.k",
		absPath,
		"${KCL_MOD}/from-mod.k",
		"relative.k",
	}

	normalizeSettingsFiles(baseDir, files)

	assert.Equal(t, filepath.Join(baseDir, "from-pwd.k"), files[0])
	assert.Equal(t, absPath, files[1])
	assert.Equal(t, "${KCL_MOD}/from-mod.k", files[2])
	assert.Equal(t, filepath.Join(baseDir, "relative.k"), files[3])
}

func Test_normalizeSettingsPackageMaps(t *testing.T) {
	baseDir := t.TempDir()
	absPath := filepath.Join(baseDir, "absolute")
	packageMaps := map[string]string{
		"pwd": "${PWD}/from-pwd",
		"abs": absPath,
		"env": "${KCL_MOD}/from-mod",
		"rel": "relative",
	}

	normalizeSettingsPackageMaps(baseDir, packageMaps)

	assert.Equal(t, filepath.Join(baseDir, "from-pwd"), packageMaps["pwd"])
	assert.Equal(t, absPath, packageMaps["abs"])
	assert.Equal(t, "${KCL_MOD}/from-mod", packageMaps["env"])
	assert.Equal(t, filepath.Join(baseDir, "relative"), packageMaps["rel"])
}

func Test_kclRunner_loadSettingsOption(t *testing.T) {
	t.Run("returns error for invalid settings", func(t *testing.T) {
		runner := &kclRunner{}
		settingsPath := filepath.Join(t.TempDir(), "kcl.yaml")
		require.NoError(t, os.WriteFile(settingsPath, []byte(":"), 0o600))

		_, _, err := runner.loadSettingsOption(settingsPath)
		require.Error(t, err)
	})

	t.Run("normalizes file and package map paths", func(t *testing.T) {
		runner := &kclRunner{}
		baseDir := t.TempDir()
		absFile := filepath.Join(baseDir, "absolute.k")
		settingsPath := filepath.Join(baseDir, "kcl.yaml")

		require.NoError(t, os.WriteFile(settingsPath, []byte(`kcl_cli_configs:
  file:
    - relative.k
    - ${PWD}/pwd.k
    - ${KCL_MOD}/mod.k
  package_maps:
    rel: relative-package
    pwd: ${PWD}/pwd-package
    abs: /tmp/absolute-package

kcl_options:
  - key: app-name
    value: demo
`), 0o600))
		require.NoError(t, os.WriteFile(absFile, []byte(""), 0o600))

		option, settingsDir, err := runner.loadSettingsOption(settingsPath)
		require.NoError(t, err)
		assert.Equal(t, baseDir, settingsDir)
		require.NotNil(t, option.ExecProgramArgs)
		assert.Equal(t, baseDir, option.WorkDir)
		assert.Contains(t, option.KFilenameList, filepath.Join(baseDir, "relative.k"))
		assert.Contains(t, option.KFilenameList, filepath.Join(baseDir, "pwd.k"))
		assert.Contains(t, option.KFilenameList, "${KCL_MOD}/mod.k")
		require.Len(t, option.ExternalPkgs, 3)
	})
}

func Test_kclRunner_dependencyOptions(t *testing.T) {
	runner := &kclRunner{}
	workDir := t.TempDir()

	t.Run("returns nil when no package root exists", func(t *testing.T) {
		options, err := runner.dependencyOptions(filepath.Join(workDir, "missing"), workDir)
		require.NoError(t, err)
		assert.Nil(t, options)
	})

	t.Run("returns nil when package has no external deps", func(t *testing.T) {
		pkgDir := filepath.Join(workDir, "nodeps")
		require.NoError(t, os.MkdirAll(pkgDir, 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "kcl.mod"), []byte(`[package]
name = "nodeps"
edition = "0.0.1"
version = "0.0.1"
`), 0o600))

		options, err := runner.dependencyOptions(pkgDir, workDir)
		require.NoError(t, err)
		assert.Nil(t, options)
	})

	t.Run("returns error for invalid manifest", func(t *testing.T) {
		pkgDir := filepath.Join(workDir, "badmod")
		require.NoError(t, os.MkdirAll(pkgDir, 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "kcl.mod"), []byte("not valid toml"), 0o600))

		_, err := runner.dependencyOptions(pkgDir, workDir)
		require.Error(t, err)
	})
}

func Test_writeManifestDirectory(t *testing.T) {
	t.Run("writes fallback file name for non-resource documents", func(t *testing.T) {
		outDir := t.TempDir()

		err := writeManifestDirectory(outDir, "foo: bar\n", configbuiltin.Kargo)
		require.NoError(t, err)

		b, readErr := os.ReadFile(filepath.Join(outDir, "resource-0.yaml"))
		require.NoError(t, readErr)
		assert.Contains(t, string(b), "foo: bar")
	})

	t.Run("ignores empty manifests", func(t *testing.T) {
		outDir := t.TempDir()
		require.NoError(t, writeManifestDirectory(outDir, " \n ", configbuiltin.Kargo))

		entries, err := os.ReadDir(outDir)
		require.NoError(t, err)
		assert.Empty(t, entries)
	})
}

func Test_safeOutputResourcePath(t *testing.T) {
	outDir := t.TempDir()

	t.Run("accepts plain file name", func(t *testing.T) {
		path, err := safeOutputResourcePath(outDir, "configmap.yaml")
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(outDir, "configmap.yaml"), path)
	})

	t.Run("rejects dot path", func(t *testing.T) {
		_, err := safeOutputResourcePath(outDir, ".")
		require.Error(t, err)
	})

	t.Run("rejects absolute path", func(t *testing.T) {
		_, err := safeOutputResourcePath(outDir, "/tmp/configmap.yaml")
		require.Error(t, err)
	})

	t.Run("rejects nested path", func(t *testing.T) {
		_, err := safeOutputResourcePath(outDir, "nested/configmap.yaml")
		require.Error(t, err)
	})
}

func Test_splitManifestResources(t *testing.T) {
	t.Run("returns original document for non-resource yaml", func(t *testing.T) {
		resources, err := splitManifestResources([]byte("foo: bar\n"))
		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, []byte("foo: bar\n"), resources[0])
	})

	t.Run("returns error for invalid yaml", func(t *testing.T) {
		_, err := splitManifestResources([]byte(": bad"))
		require.Error(t, err)
	})
}

func Test_resourceMetadataHelpers(t *testing.T) {
	assert.Empty(t, resourceFileName([]byte("foo: bar\n"), configbuiltin.Kargo))

	group, version, kind, namespace, name := extractYAMLObjectMetadata([]byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: demo
  namespace: prod
`))
	assert.Equal(t, "apps", group)
	assert.Equal(t, "v1", version)
	assert.Equal(t, "Deployment", kind)
	assert.Equal(t, "prod", namespace)
	assert.Equal(t, "demo", name)

	group, version, kind, namespace, name = extractYAMLObjectMetadata([]byte(":"))
	assert.Empty(t, group)
	assert.Empty(t, version)
	assert.Empty(t, kind)
	assert.Empty(t, namespace)
	assert.Empty(t, name)
}
