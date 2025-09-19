package yaml

import (
	"os"
	"path"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestSetValuesInBytes(t *testing.T) {
	testCases := []struct {
		name       string
		inBytes    []byte
		updates    []Update
		assertions func(*testing.T, []byte, error)
	}{
		{
			name: "invalid YAML",
			// Note: This YAML is invalid because one line is indented with a tab
			inBytes: []byte(`
characters:
- name: Anakin
	affiliation: Light side
`),
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.ErrorContains(t, err, "error unmarshaling input")
				require.Nil(t, bytes)
			},
		},
		{
			name: "no extra quotes around true number",
			inBytes: []byte(`
characters:
- name: Arthur Dent
  answer: idk
`),
			updates: []Update{
				{
					Key:   "characters.0.answer",
					Value: 42,
				},
			},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					[]byte(`
characters:
- name: Arthur Dent
  answer: 42
`),
					bytes,
				)
			},
		},
		{
			name: "extra quotes around string containing a valid number",
			inBytes: []byte(`
characters:
- name: Arthur Dent
  answer: idk
`),
			updates: []Update{
				{
					Key:   "characters.0.answer",
					Value: "42",
				},
			},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					[]byte(`
characters:
- name: Arthur Dent
  answer: "42"
`),
					bytes,
				)
			},
		},
		{
			name: "success",
			inBytes: []byte(`
characters:
- name: Anakin
  affiliation: Light side
`),
			updates: []Update{
				{
					Key:   "characters.0.affiliation",
					Value: "Dark side",
				},
			},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					[]byte(`
characters:
- name: Anakin
  affiliation: Dark side
`),
					bytes,
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			b, err := SetValuesInBytes(testCase.inBytes, testCase.updates)
			testCase.assertions(t, b, err)
		})
	}
}

func TestFindScalarNode(t *testing.T) {
	yamlBytes := []byte(`
characters:
  rebels:
  - name: Skywalker
`)
	testCases := []struct {
		name       string
		keyPath    string
		assertions func(t *testing.T, line, col int, err error)
	}{
		{
			name:    "node not found",
			keyPath: "characters.imperials",
			assertions: func(t *testing.T, _, _ int, err error) {
				require.ErrorContains(t, err, "key path not found")
			},
		},
		{
			name: "node not found due to error parsing int",
			// Really, this is a special case of a key that doesn't address a node,
			// because there is alpha input where numeric input would be expected.
			keyPath: "characters.rebels.first.name",
			assertions: func(t *testing.T, _, _ int, err error) {
				require.ErrorContains(t, err, "strconv.Atoi")
			},
		},
		{
			name:    "node found, but isn't a scalar node",
			keyPath: "characters.rebels",
			assertions: func(t *testing.T, _, _ int, err error) {
				require.ErrorContains(t, err, "key path does not address a scalar node")
			},
		},
		{
			name:    "success",
			keyPath: "characters.rebels.0.name",
			assertions: func(t *testing.T, line, col int, err error) {
				require.NoError(t, err)
				require.Equal(t, 3, line)
				require.Equal(t, 10, col)
			},
		},
	}
	doc := &yaml.Node{}
	err := yaml.Unmarshal(yamlBytes, doc)
	require.NoError(t, err)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			line, col, err := findScalarNode(doc, strings.Split(testCase.keyPath, "."))
			testCase.assertions(t, line, col, err)
		})
	}
}

func TestMergeFiles(t *testing.T) {
	tests := []struct {
		name       string
		inputPaths map[string]string
		outputPath string
		assertions func(*testing.T, string, error)
	}{
		{
			name:       "successful run with modified outputs",
			outputPath: "modified.yaml",
			inputPaths: map[string]string{
				"base.yaml": `
app:
  version: "1.0.0"
features:
  newFeature: false`,
				"overrides.yaml": `
app:
  version: "2.0.0"`,
			},
			assertions: func(t *testing.T, workDir string, err error) {
				assert.NoError(t, err)
				content, err := os.ReadFile(path.Join(workDir, "modified.yaml"))
				require.NoError(t, err)
				assert.Contains(t, string(content), `  version: "2.0.0"`)
				assert.Contains(t, string(content), `  newFeature: false`)
			},
		},
		{
			name:       "successful run with modified outputs using 2 patch files",
			outputPath: "modified.yaml",
			inputPaths: map[string]string{
				"base.yaml": `
app:
  version: "1.0.0"
features:
  newFeature: false
`,
				"overrides.yaml": `
app:
  version: "2.0.0"
`,
				"overrides_2.yaml": `
app:
 version: "4.0.0"
features:
  more: "cakes"
added:
  - 12
  - 32
`,
			},
			assertions: func(t *testing.T, workDir string, err error) {
				assert.NoError(t, err)
				content, err := os.ReadFile(path.Join(workDir, "modified.yaml"))
				require.NoError(t, err)
				assert.Contains(t, string(content), `  version: "4.0.0"`)
				assert.Contains(t, string(content), `  newFeature: false`)
				assert.Contains(t, string(content), `  more: "cakes"`)
			},
		},
		{
			name:       "no extra quotes around true number",
			outputPath: "modified.yaml",
			inputPaths: map[string]string{
				"base.yaml": `
app:
  version: "1.0.0"
features:
  newVersion: unknown
`,
				"overrides.yaml": `
app:
  version: "2.0.0"
features:
    newVersion: 42
`,
			},
			assertions: func(t *testing.T, workDir string, err error) {
				assert.NoError(t, err)
				content, err := os.ReadFile(path.Join(workDir, "modified.yaml"))
				require.NoError(t, err)
				assert.Contains(t, string(content), `  version: "2.0.0"`)
				assert.Contains(t, string(content), `  newVersion: 42`)
			},
		},
		{
			name:       "extra quotes around string containing a valid number",
			outputPath: "modified.yaml",
			inputPaths: map[string]string{
				"base.yaml": `
app:
  version: "1.0.0"
features:
  newVersion: unknown
`,
				"overrides.yaml": `
app:
  version: "2.0.0"
features:
    newVersion: "42"
`,
			},
			assertions: func(t *testing.T, workDir string, err error) {
				assert.NoError(t, err)
				content, err := os.ReadFile(path.Join(workDir, "modified.yaml"))
				require.NoError(t, err)
				assert.Contains(t, string(content), `  version: "2.0.0"`)
				assert.Contains(t, string(content), `  newVersion: "42"`)
			},
		},
		{
			name:       "success with single YAML file",
			outputPath: "modified.yaml",
			inputPaths: map[string]string{
				"base.yaml": `
app:
  version: "1.0.0"
features:
  newVersion: unknown
`,
			},
			assertions: func(t *testing.T, workDir string, err error) {
				assert.NoError(t, err)
				content, err := os.ReadFile(path.Join(workDir, "modified.yaml"))
				require.NoError(t, err)
				assert.Contains(t, string(content), `  version: "1.0.0"`)
				assert.Contains(t, string(content), `  newVersion: unknown`)
			},
		},
		{
			name:       "success with dict and object",
			outputPath: "modified.yaml",
			inputPaths: map[string]string{
				"base.yaml": `
characters:
- name: Anakin
  affiliation: Light side
- name: Jabba
  affiliation: Toads
weapon:
  kind: lightsabre
  color: green
`,
				"overrides.yaml": `
characters:
- name: Anakin
  affiliation: Dark side
weapon:
  color: red
episode: 3
`,
			},
			assertions: func(t *testing.T, workDir string, err error) {
				assert.NoError(t, err)
				content, err := os.ReadFile(path.Join(workDir, "modified.yaml"))
				require.NoError(t, err)
				require.Equal(
					t,
					string(`characters:
- name: Anakin
  affiliation: Dark side
weapon:
  kind: lightsabre
  color: red
episode: 3
`),
					string(content),
				)
			},
		},

		{
			name:       "no InFiles",
			outputPath: "modified.yaml",
			inputPaths: map[string]string{},
			assertions: func(t *testing.T, _ string, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "inFiles and OutFile must not be empty")
			},
		},
		{
			name:       "no OutFile",
			outputPath: "",
			inputPaths: map[string]string{
				"base.yaml": `
app:
  version: "1.0.0"
features:
  newFeature: false
`,
			},
			assertions: func(t *testing.T, _ string, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "inFiles and OutFile must not be empty")
			},
		},
		{
			name:       "no Infiles and no OutFile",
			outputPath: "",
			inputPaths: map[string]string{},
			assertions: func(t *testing.T, _ string, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "inFiles and OutFile must not be empty")
			},
		},
		{
			name:       "failed to read InFiles file",
			outputPath: "modified.yaml",
			inputPaths: map[string]string{
				"no_file.yaml": ``,
			},
			assertions: func(t *testing.T, _ string, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "error parsing first input file")
			},
		},
		{
			name:       "failed to read first InFiles content",
			outputPath: "modified.yaml",
			inputPaths: map[string]string{
				"base.yaml": `
this is:
not a yaml file
`,
			},
			assertions: func(t *testing.T, _ string, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "error parsing first input file")
			},
		},
		{
			name:       "failed to read second InFiles content",
			outputPath: "modified.yaml",
			inputPaths: map[string]string{
				"base.yaml": `
app:
  version: "1.0.0"
features:
  newFeature: false
`,
				"second.yaml": `
this is:
not a yaml file
`,
			},
			assertions: func(t *testing.T, _ string, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "error parsing input file")
			},
		},
		{
			name:       "outputs folder does not exist",
			outputPath: "no_folder/modified.yaml",
			inputPaths: map[string]string{
				"base.yaml": `
app:
  version: "1.0.0"
features:
  newFeature: false
`,
				"overrides.yaml": `
app:
  version: "2.0.0"
`,
			},
			assertions: func(t *testing.T, _ string, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "error writing the merged file to")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()

			// if outputPath is not empty, set it to the right place
			var workOutFile string
			if tt.outputPath != "" && tt.outputPath != "no_file.yaml" {
				workOutFile = path.Join(workDir, tt.outputPath)
			}

			workInFiles := []string{}
			for p, c := range tt.inputPaths {
				workInFiles = append(workInFiles, path.Join(workDir, p))

				// do not add the file if it is supposed to be in a non-existing folder
				if path.Dir(p) == "no_folder" {
					continue
				}
				require.NoError(t, os.MkdirAll(path.Join(workDir, path.Dir(p)), 0o700))

				// create the file if is is supposed to exist
				if p != "no_file.yaml" {
					require.NoError(t, os.WriteFile(path.Join(workDir, p), []byte(c), 0o600))
				}

			}

			sort.Strings(workInFiles)
			err := MergeFiles(workInFiles, workOutFile)
			tt.assertions(t, workDir, err)
		})
	}
}
