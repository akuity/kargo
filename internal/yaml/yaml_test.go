package yaml

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestRead(t *testing.T) {
	// This test progressively builds the following directory structure:
	// "testRootPath"
	// ├── a
	// │   ├── a.yaml
	// │   └── b.yaml
	// ├── c
	// │   ├── c.yaml
	// │   ├── d.yaml -> d/d.yaml
	// │   ├── e.yaml -> e
	// │   ├── f -> c
	// │   └── g -> "testRootPath"
	// ├── d
	// │   └── d.yaml
	// └── e
	//     └── e.yaml

	testRootPath := t.TempDir()

	var result []byte

	aDir := filepath.Join(testRootPath, "a")
	err := os.Mkdir(aDir, 0755)
	require.NoError(t, err)
	t.Run("empty directory", func(t *testing.T) {
		result, err = Read([]string{aDir})
		require.NoError(t, err)
		require.Empty(t, result)
	})

	aFile := filepath.Join(aDir, "a.yaml")
	aFileBytes := []byte{}
	err = os.WriteFile(aFile, aFileBytes, 0600)
	require.NoError(t, err)
	t.Run("empty file", func(t *testing.T) {
		result, err = Read([]string{aFile})
		require.NoError(t, err)
		require.Empty(t, result)
	})

	aFile = filepath.Join(aDir, "a.yaml")
	aFileBytes = []byte("I solemnly swear that I am up to no good.")
	err = os.WriteFile(aFile, aFileBytes, 0600)
	require.NoError(t, err)
	t.Run("single file", func(t *testing.T) {
		result, err = Read([]string{aFile})
		require.NoError(t, err)
		require.Equal(t, aFileBytes, result)
	})

	t.Run("single file in a directory", func(t *testing.T) {
		result, err = Read([]string{aDir})
		require.NoError(t, err)
		require.Equal(t, aFileBytes, result)
	})

	bFile := filepath.Join(aDir, "b.yaml")
	bFileBytes := []byte(
		"Mr Moony presents his compliments to Professor Snape and begs him to " +
			"keep his abnormally large nose out of other people's business.",
	)
	err = os.WriteFile(bFile, bFileBytes, 0600)
	require.NoError(t, err)
	t.Run("multiple files", func(t *testing.T) {
		result, err = Read([]string{aFile, bFile})
		require.NoError(t, err)
		require.Equal(
			t,
			bytes.Join([][]byte{aFileBytes, bFileBytes}, sep),
			result,
		)
	})

	t.Run("multiple files in a directory", func(t *testing.T) {
		result, err = Read([]string{aDir})
		require.NoError(t, err)
		require.Equal(
			t,
			bytes.Join(
				[][]byte{
					aFileBytes,
					bFileBytes,
				},
				sep,
			),
			result,
		)
	})

	cDir := filepath.Join(testRootPath, "c")
	err = os.Mkdir(cDir, 0755)
	require.NoError(t, err)
	cFile := filepath.Join(cDir, "c.yaml")
	cFileBytes := []byte(
		"Mr Prongs agrees with Mr Moony and would like to add that Professor " +
			"Snape is an ugly git.",
	)
	err = os.WriteFile(cFile, cFileBytes, 0600)
	require.NoError(t, err)
	t.Run("multiple directories", func(t *testing.T) {
		result, err = Read([]string{aDir, cDir})
		require.NoError(t, err)
		require.Equal(
			t,
			bytes.Join(
				[][]byte{
					aFileBytes,
					bFileBytes,
					cFileBytes,
				},
				sep,
			),
			result,
		)
	})

	dDir := filepath.Join(testRootPath, "d")
	err = os.Mkdir(dDir, 0755)
	require.NoError(t, err)
	dFile := filepath.Join(dDir, "d.yaml")
	dFileBytes := []byte(
		"Mr Padfoot would like to register his astonishment that an idiot like " +
			"that ever became a Professor.",
	)
	err = os.WriteFile(dFile, dFileBytes, 0600)
	require.NoError(t, err)
	err = os.Symlink(dFile, filepath.Join(cDir, "d.yaml"))
	require.NoError(t, err)
	t.Run("symlink to a file", func(t *testing.T) {
		result, err = Read([]string{aDir, cDir})
		require.NoError(t, err)
		require.Equal(
			t,
			bytes.Join(
				[][]byte{
					aFileBytes,
					bFileBytes,
					cFileBytes,
					dFileBytes,
				},
				sep,
			),
			result,
		)
	})

	eDir := filepath.Join(testRootPath, "e")
	err = os.Mkdir(eDir, 0755)
	require.NoError(t, err)
	eFile := filepath.Join(eDir, "e.yaml")
	eFileBytes := []byte("Mischief managed.")
	err = os.WriteFile(eFile, eFileBytes, 0600)
	require.NoError(t, err)
	err = os.Symlink(eDir, filepath.Join(cDir, "e"))
	require.NoError(t, err)
	t.Run("symlink to a directory", func(t *testing.T) {
		result, err = Read([]string{aDir, cDir})
		require.NoError(t, err)
		require.Equal(
			t,
			bytes.Join(
				[][]byte{
					aFileBytes,
					bFileBytes,
					cFileBytes,
					dFileBytes,
					eFileBytes,
				},
				sep,
			),
			result,
		)
	})

	err = os.Symlink(cDir, filepath.Join(cDir, "f"))
	require.NoError(t, err)
	t.Run("symlink to a directory already being read", func(t *testing.T) {
		// This tests that symlinks don't cause infinite recursion.
		result, err = Read([]string{aDir, cDir})
		require.NoError(t, err)
		require.Equal(
			t,
			bytes.Join(
				[][]byte{
					aFileBytes,
					bFileBytes,
					cFileBytes,
					dFileBytes,
					eFileBytes,
				},
				sep,
			),
			result,
		)
	})

	gLink := filepath.Join(cDir, "g")
	err = os.Symlink(testRootPath, gLink)
	t.Run("symlink to a directory above files being read", func(t *testing.T) {
		// This tests that symlinks don't ever cause duplicate reads.
		result, err = Read([]string{aDir, cDir})
		require.NoError(t, err)
		require.Equal(
			t,
			bytes.Join(
				[][]byte{
					aFileBytes,
					bFileBytes,
					cFileBytes,
					dFileBytes,
					eFileBytes,
				},
				sep,
			),
			result,
		)
	})

	err = os.Remove(gLink)
	require.NoError(t, err)
	t.Run("deliberate attempt at duplicate read doesn't actually duplicate resources", func(t *testing.T) {
		result, err = Read([]string{aDir, cDir, aDir, cDir})
		require.NoError(t, err)
		require.Equal(
			t,
			bytes.Join(
				[][]byte{
					aFileBytes,
					bFileBytes,
					cFileBytes,
					dFileBytes,
					eFileBytes,
				},
				sep,
			),
			result,
		)
	})
}

func TestPathInPaths(t *testing.T) {
	testCases := []struct {
		name     string
		path     string
		paths    []string
		expected bool
	}{
		{
			name: "path matches one of paths",
			// This can happen if path was a symlink, which after being evaluated,
			// exactly matches one of the paths.
			path:     "/a/b/c",
			paths:    []string{"/a/b/c"},
			expected: true,
		},
		{
			name: "path has one of paths as a prefix",
			// This can happen if path was a symlink, which after being evaluated, is
			// a path in the file system within any of the paths.
			path:     "/a/b/c/d",
			paths:    []string{"/a/b/c"},
			expected: true,
		},
		{
			name:     "path has none of paths as a prefix",
			path:     "/a/b/c",
			paths:    []string{"/x/y/z"},
			expected: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.expected,
				isPathInPaths(testCase.path, testCase.paths),
			)
		})
	}
}

func TestSetStringsInBytes(t *testing.T) {
	testCases := []struct {
		name       string
		inBytes    []byte
		changes    map[string]string
		assertions func([]byte, error)
	}{
		{
			name: "invalid YAML",
			// Note: This YAML is invalid because one line is indented with a tab
			inBytes: []byte(`
characters:
- name: Anakin
	affiliation: Light side
`),
			assertions: func(bytes []byte, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error unmarshaling input")
				require.Nil(t, bytes)
			},
		},
		{
			name: "success",
			inBytes: []byte(`
characters:
- name: Anakin
  affiliation: Light side
`),
			changes: map[string]string{
				"characters.0.affiliation": "Dark side",
			},
			assertions: func(bytes []byte, err error) {
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
			testCase.assertions(
				SetStringsInBytes(testCase.inBytes, testCase.changes),
			)
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
		assertions func(found bool, line, col int)
	}{
		{
			name:    "node not found",
			keyPath: "characters.imperials",
			assertions: func(found bool, line, col int) {
				require.False(t, found)
				require.Zero(t, line)
				require.Zero(t, col)
			},
		},
		{
			name: "node not found due to error parsing int",
			// Really, this is a special case of a key that doesn't address a node,
			// because there is alpha input where numeric input would be expected.
			keyPath: "characters.rebels.first.name",
			assertions: func(found bool, line, col int) {
				require.False(t, found)
				require.Zero(t, line)
				require.Zero(t, col)
			},
		},
		{
			name:    "node found, but isn't a scalar node",
			keyPath: "characters.rebels",
			assertions: func(found bool, line, col int) {
				require.False(t, found)
				require.Zero(t, line)
				require.Zero(t, col)
			},
		},
		{
			name:    "success",
			keyPath: "characters.rebels.0.name",
			assertions: func(found bool, line, col int) {
				require.True(t, found)
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
			testCase.assertions(
				findScalarNode(doc, strings.Split(testCase.keyPath, ".")),
			)
		})
	}
}
