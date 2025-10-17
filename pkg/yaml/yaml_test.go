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
		name        string
		files       map[string]string
		outPath     string
		skipFiles   []string // Files to not create (for testing missing files)
		skipOutDir  bool     // Skip creating output directory
		expectedErr string
		validate    func(*testing.T, string)
	}{
		{
			name:        "error when no input files provided",
			outPath:     "modified.yaml",
			files:       map[string]string{},
			expectedErr: "inFiles and outFile must not be empty",
		},
		{
			name: "error when no output path provided",
			files: map[string]string{
				"base.yaml": `app:
  version: "1.0.0"
`,
			},
			expectedErr: "inFiles and outFile must not be empty",
		},
		{
			name:        "error when both input and output are empty",
			expectedErr: "inFiles and outFile must not be empty",
		},
		{
			name:    "error when input file does not exist",
			outPath: "modified.yaml",
			files: map[string]string{
				"missing.yaml": ``,
			},
			skipFiles:   []string{"missing.yaml"},
			expectedErr: "error parsing first input file",
		},
		{
			name:    "error when first file has invalid YAML",
			outPath: "modified.yaml",
			files: map[string]string{
				"invalid.yaml": `this is:
not a yaml file
`,
			},
			expectedErr: "error parsing first input file",
		},
		{
			name:    "error when output directory does not exist",
			outPath: "missing_dir/output.yaml",
			files: map[string]string{
				"base.yaml": `app:
  version: "1.0.0"
`,
			},
			skipOutDir:  true,
			expectedErr: "error writing the merged file to",
		},
		{
			name:    "merge single file successfully",
			outPath: "modified.yaml",
			files: map[string]string{
				"base.yaml": `app:
  version: "1.0.0"
features:
  newVersion: unknown
`,
			},
			validate: func(t *testing.T, workDir string) {
				content, err := os.ReadFile(path.Join(workDir, "modified.yaml"))
				require.NoError(t, err)
				assert.Contains(t, string(content), `version: "1.0.0"`)
				assert.Contains(t, string(content), `newVersion: unknown`)
			},
		},
		{
			name:    "merge multiple files successfully",
			outPath: "modified.yaml",
			files: map[string]string{
				"base.yaml": `app:
  version: "1.0.0"
features:
  newFeature: false`,
				"overrides.yaml": `app:
  version: "2.0.0"`,
			},
			validate: func(t *testing.T, workDir string) {
				content, err := os.ReadFile(path.Join(workDir, "modified.yaml"))
				require.NoError(t, err)
				assert.Contains(t, string(content), `version: "2.0.0"`)
				assert.Contains(t, string(content), `newFeature: false`)
			},
		},
		{
			name:    "merge three files with cascading overrides",
			outPath: "modified.yaml",
			files: map[string]string{
				"base.yaml": `app:
  version: "1.0.0"
features:
  newFeature: false
`,
				"overrides.yaml": `app:
  version: "2.0.0"
`,
				"overrides_2.yaml": `app:
  version: "4.0.0"
features:
  more: "foo"
added:
  - 12
  - 32
`,
			},
			validate: func(t *testing.T, workDir string) {
				content, err := os.ReadFile(path.Join(workDir, "modified.yaml"))
				require.NoError(t, err)
				assert.Contains(t, string(content), `version: "4.0.0"`)
				assert.Contains(t, string(content), `newFeature: false`)
				assert.Contains(t, string(content), `more: "foo"`)
			},
		},
		{
			name:    "merge complex structures with arrays",
			outPath: "modified.yaml",
			files: map[string]string{
				"base.yaml": `characters:
- name: Something
  affiliation: Light side
- name: Wise
  affiliation: Toads
weapon:
  kind: lightsabre
  color: green
`,
				"overrides.yaml": `characters:
- name: Something
  affiliation: Dark side
weapon:
  color: red
episode: 3
`,
			},
			validate: func(t *testing.T, workDir string) {
				content, err := os.ReadFile(path.Join(workDir, "modified.yaml"))
				require.NoError(t, err)
				expected := `characters:
- name: Something
  affiliation: Dark side
weapon:
  kind: lightsabre
  color: red
episode: 3
`
				assert.Equal(t, expected, string(content))
			},
		},
		{
			name:    "preserve number types without extra quotes",
			outPath: "modified.yaml",
			files: map[string]string{
				"base.yaml": `app:
  version: "1.0.0"
features:
  newVersion: unknown
`,
				"overrides.yaml": `app:
  version: "2.0.0"
features:
  newVersion: 42
`,
			},
			validate: func(t *testing.T, workDir string) {
				content, err := os.ReadFile(path.Join(workDir, "modified.yaml"))
				require.NoError(t, err)
				assert.Contains(t, string(content), `version: "2.0.0"`)
				assert.Contains(t, string(content), `newVersion: 42`)
				assert.NotContains(t, string(content), `newVersion: "42"`)
			},
		},
		{
			name:    "preserve string numbers with quotes",
			outPath: "modified.yaml",
			files: map[string]string{
				"base.yaml": `app:
  version: "1.0.0"
features:
  newVersion: unknown
`,
				"overrides.yaml": `app:
  version: "2.0.0"
features:
  newVersion: "42"
`,
			},
			validate: func(t *testing.T, workDir string) {
				content, err := os.ReadFile(path.Join(workDir, "modified.yaml"))
				require.NoError(t, err)
				assert.Contains(t, string(content), `version: "2.0.0"`)
				assert.Contains(t, string(content), `newVersion: "42"`)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()

			// Build input file paths and create files
			var inPaths []string
			for fileName, content := range tt.files {
				filePath := path.Join(workDir, fileName)
				inPaths = append(inPaths, filePath)

				// Skip creating files that should be missing
				skip := false
				for _, skipFile := range tt.skipFiles {
					if fileName == skipFile {
						skip = true
						break
					}
				}
				if skip {
					continue
				}

				// Create directory and file
				require.NoError(t, os.MkdirAll(path.Dir(filePath), 0o700))
				require.NoError(t, os.WriteFile(filePath, []byte(content), 0o600))
			}

			// Sort for consistent ordering
			sort.Strings(inPaths)

			// Build output path
			var outPath string
			if tt.outPath != "" {
				outPath = path.Join(workDir, tt.outPath)
				// Create output directory unless we're testing missing directory
				if !tt.skipOutDir {
					require.NoError(t, os.MkdirAll(path.Dir(outPath), 0o700))
				}
			}

			err := MergeFiles(inPaths, outPath)

			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, workDir)
				}
			}
		})
	}
}
