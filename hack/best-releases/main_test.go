package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteBestReleasesJSON(t *testing.T) {
	testCases := []struct {
		name     string
		setup    func(*testing.T, string)
		releases []Release
		assert   func(*testing.T, string, error)
	}{
		{
			name: "error creating directory",
			setup: func(t *testing.T, dir string) {
				// A file where the site directory belongs
				require.NoError(t, os.WriteFile(filepath.Join(dir, "site"), nil, 0o600))
			},
			releases: []Release{{Version: v("v1.11.8")}},
			assert: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, "error creating directory")
			},
		},
		{
			name: "error writing file",
			setup: func(t *testing.T, dir string) {
				// A directory where the file belongs
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "site", "best-releases.json"), 0o700))
			},
			releases: []Release{{Version: v("v1.11.8")}},
			assert: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, "error writing")
			},
		},
		{
			name: "success",
			releases: []Release{
				{
					Version: v("v1.11.8"),
					Latest:  true,
				},
				{Version: v("v1.9.15")},
			},
			assert: func(t *testing.T, outDir string, err error) {
				require.NoError(t, err)
				data, readErr := os.ReadFile(filepath.Join(outDir, "best-releases.json"))
				require.NoError(t, readErr)
				var parsed struct {
					Releases []struct {
						Version string `json:"version"`
						Latest  bool   `json:"latest"`
					} `json:"releases"`
				}
				require.NoError(t, json.Unmarshal(data, &parsed))
				require.Len(t, parsed.Releases, 2)
				require.Equal(t, "v1.11.8", parsed.Releases[0].Version)
				require.True(t, parsed.Releases[0].Latest)
				require.Equal(t, "v1.9.15", parsed.Releases[1].Version)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if testCase.setup != nil {
				testCase.setup(t, dir)
			}
			outDir := filepath.Join(dir, "site")
			err := writeBestReleasesJSON(outDir, testCase.releases)
			testCase.assert(t, outDir, err)
		})
	}
}

func TestWriteVersionFiles(t *testing.T) {
	testCases := []struct {
		name     string
		setup    func(*testing.T, string)
		releases []Release
		assert   func(*testing.T, string, error)
	}{
		{
			name: "error creating line directory",
			setup: func(t *testing.T, dir string) {
				// A file where the line's directory belongs
				require.NoError(t, os.WriteFile(filepath.Join(dir, "1.11"), nil, 0o600))
			},
			releases: []Release{{Version: v("v1.11.8")}},
			assert: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, "error creating directory")
			},
		},
		{
			name: "error writing version file",
			setup: func(t *testing.T, dir string) {
				// A directory where the version file belongs
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "1.11", "version.txt"), 0o700))
			},
			releases: []Release{{Version: v("v1.11.8")}},
			assert: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, "error writing")
			},
		},
		{
			name: "success",
			releases: []Release{
				{Version: v("v1.11.8")},
				{Version: v("v1.9.15")},
			},
			assert: func(t *testing.T, dir string, err error) {
				require.NoError(t, err)
				for line, version := range map[string]string{
					"1.11": "v1.11.8",
					"1.9":  "v1.9.15",
				} {
					contents, readErr := os.ReadFile(filepath.Join(dir, line, "version.txt"))
					require.NoError(t, readErr)
					require.Equal(t, version, string(contents))
				}
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if testCase.setup != nil {
				testCase.setup(t, dir)
			}
			err := writeVersionFiles(dir, testCase.releases)
			testCase.assert(t, dir, err)
		})
	}
}
