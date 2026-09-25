package yaml

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	yaml "go.yaml.in/yaml/v3"
)

func TestMergeFiles(t *testing.T) {
	testCases := []struct {
		name       string
		fsFiles    map[string]string
		inputPaths []string
		outputPath string
		assertions func(t *testing.T, outputPath string, err error)
	}{
		{
			name: "output path is empty",
			assertions: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, "output path must not be empty")
			},
		},
		{
			name:       "no input files",
			outputPath: "merged.yaml",
			assertions: func(t *testing.T, outputPath string, err error) {
				require.NoError(t, err)
				fileBytes, err := os.ReadFile(outputPath)
				require.NoError(t, err)
				fmt.Println(string(fileBytes))
				require.Empty(t, fileBytes)
			},
		},
		{
			name:       "input file does not exist",
			inputPaths: []string{"non-existent.yaml"},
			outputPath: "merged.yaml",
			assertions: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, "error reading input file")
			},
		},
		{
			name: "input file does not contain valid YAML",
			fsFiles: map[string]string{
				"invalid.yaml": ":",
			},
			inputPaths: []string{"invalid.yaml"},
			outputPath: "merged.yaml",
			assertions: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, "error reading input file")
			},
		},
		{
			name: "successful merge",
			// Note: This is not an exhaustive test of all merge behavior. We presume
			// kyaml to be well-tested. We DO, however, make a point of verifying that
			// list merge behavior is consistent with Helm (full replacement).
			fsFiles: map[string]string{
				"empty.yaml": "", // Implementation should tolerate this
				"base.yaml": `
character:
  name: Vader
  ability: Force
  affiliation: Empire
  saberColor: red
vehicles:
- Star Destroyer`,
				"overlay.yaml": `
character:
  name: Luke # Alters character.name from the base
  # ability: Force  # Should be inherited from the base without alteration
  affiliation: Rebel Alliance # Alters character.affiliation from the base
  saberColor: blue # Alters character.saberColor from the base
  siblings: # A completely new field that the base does not have
  - Leia
vehicles: # Completely replaces vehicles list from the base
- Millennium Falcon`,
			},
			inputPaths: []string{
				"empty.yaml",
				"base.yaml",
				"overlay.yaml",
			},
			outputPath: "merged.yaml",
			assertions: func(t *testing.T, outputPath string, err error) {
				require.NoError(t, err)
				fileBytes, err := os.ReadFile(outputPath)
				require.NoError(t, err)
				require.YAMLEq(
					t,
					`
character:
  name: Luke
  ability: Force
  affiliation: Rebel Alliance
  saberColor: blue
  siblings:
  - Leia
vehicles:
- Millennium Falcon`,
					string(fileBytes),
				)
			},
		},
		{
			// This reproduces the exact scenario from
			// https://github.com/akuity/kargo/issues/6740: aliases, the `<<`
			// merge key, and explicit nulls must all survive a merge.
			name: "aliases, merge keys, and explicit nulls are preserved",
			fsFiles: map[string]string{
				"overlay.yaml": `someFlag: false`,
				"values.yaml": `
db:
  name: &db-name mydb
pgbouncer:
  readinessDbName: *db-name
  nodeSelector:
defaults: &defaults
  image: my-image
other:
  <<: *defaults`,
			},
			inputPaths: []string{
				"overlay.yaml",
				"values.yaml",
			},
			outputPath: "merged.yaml",
			assertions: func(t *testing.T, outputPath string, err error) {
				require.NoError(t, err)
				fileBytes, err := os.ReadFile(outputPath)
				require.NoError(t, err)

				var merged map[string]any
				require.NoError(t, yaml.Unmarshal(fileBytes, &merged))

				require.Equal(t, false, merged["someFlag"])

				pgbouncer, ok := merged["pgbouncer"].(map[string]any)
				require.True(t, ok, "pgbouncer should be a map, got %#v", merged["pgbouncer"])
				require.Equal(
					t, "mydb", pgbouncer["readinessDbName"],
					"aliased value should have been preserved",
				)
				nodeSelector, exists := pgbouncer["nodeSelector"]
				require.True(t, exists, "explicit null key should not be dropped")
				require.Nil(t, nodeSelector)

				other, ok := merged["other"].(map[string]any)
				require.True(t, ok, "other should be a map, got %#v", merged["other"])
				require.Equal(
					t, "my-image", other["image"],
					"merge key (<<) value should have been preserved",
				)
			},
		},
		{
			name: "explicit null overrides a prior non-null value without deleting the key",
			fsFiles: map[string]string{
				"base.yaml":     `foo: bar`,
				"override.yaml": `foo:`,
			},
			inputPaths: []string{
				"base.yaml",
				"override.yaml",
			},
			outputPath: "merged.yaml",
			assertions: func(t *testing.T, outputPath string, err error) {
				require.NoError(t, err)
				fileBytes, err := os.ReadFile(outputPath)
				require.NoError(t, err)

				var merged map[string]any
				require.NoError(t, yaml.Unmarshal(fileBytes, &merged))

				foo, exists := merged["foo"]
				require.True(t, exists, "explicit null override should not delete the key")
				require.Nil(t, foo)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			workDir := t.TempDir()
			for filePath, content := range testCase.fsFiles {
				err := os.WriteFile(
					path.Join(workDir, filePath),
					[]byte(content),
					0o600,
				)
				require.NoError(t, err)
			}
			for i, filePath := range testCase.inputPaths {
				testCase.inputPaths[i] = filepath.Join(workDir, filePath)
			}
			if testCase.outputPath != "" {
				testCase.outputPath = filepath.Join(workDir, testCase.outputPath)
			}
			err := MergeFiles(testCase.inputPaths, testCase.outputPath)
			testCase.assertions(t, testCase.outputPath, err)
		})
	}
}
