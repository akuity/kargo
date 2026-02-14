package releases

import (
	"encoding/json"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func v(s string) *semver.Version {
	return semver.MustParse(s)
}

func TestGithubReleases_UnmarshalJSON(t *testing.T) {
	t.Run("invalid JSON returns error", func(t *testing.T) {
		var releases githubReleases
		require.Error(t, json.Unmarshal([]byte(`not json`), &releases))
	})

	t.Run("invalid entry in array returns error", func(t *testing.T) {
		var releases githubReleases
		require.Error(t, json.Unmarshal([]byte(`[42]`), &releases))
	})

	t.Run("filters drafts, prereleases, and non-semver tags", func(t *testing.T) {
		input := `[
			{"tag_name": "v1.0.0", "draft": false, "prerelease": false},
			{"tag_name": "v1.1.0-rc.1", "draft": false, "prerelease": true},
			{"tag_name": "v1.1.0", "draft": true, "prerelease": false},
			{"tag_name": "nightly", "draft": false, "prerelease": false},
			{"tag_name": "v2.0.0", "draft": false, "prerelease": false}
		]`

		var releases githubReleases
		require.NoError(t, json.Unmarshal([]byte(input), &releases))

		require.Len(t, releases, 2)
		assert.True(t, v("1.0.0").Equal(releases[0].Version))
		assert.True(t, v("2.0.0").Equal(releases[1].Version))
	})
}

func TestRelease_MarshalJSON(t *testing.T) {
	r := Release{
		Version: v("1.2.3"),
		CLIBinaries: CLIBinaries{
			"linux": {"amd64": "https://example.com/dl"},
		},
	}
	data, err := json.Marshal(r)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	assert.Equal(t, "1.2.3", m["version"])
	assert.NotNil(t, m["cliBinaries"])
}

func TestCLIAssetPattern(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantOS   string
		wantArch string
	}{
		{name: "linux amd64", input: "kargo-linux-amd64", wantOS: "linux", wantArch: "amd64"},
		{name: "linux arm64", input: "kargo-linux-arm64", wantOS: "linux", wantArch: "arm64"},
		{name: "darwin arm64", input: "kargo-darwin-arm64", wantOS: "darwin", wantArch: "arm64"},
		{name: "darwin amd64", input: "kargo-darwin-amd64", wantOS: "darwin", wantArch: "amd64"},
		{name: "windows exe", input: "kargo-windows-amd64.exe", wantOS: "windows", wantArch: "amd64"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := cliAssetPattern.FindStringSubmatch(tt.input)
			require.Len(t, m, 3, "expected match for %q", tt.input)
			assert.Equal(t, tt.wantOS, m[1])
			assert.Equal(t, tt.wantArch, m[2])
		})
	}

	nonMatches := []string{
		"checksums.txt",
		"akuity-kargo_v1.spdx.json",
		"kargo-cli.intoto.jsonl",
		"Source code (zip)",
		"Source code (tar.gz)",
		"kargo-linux",        // missing arch
		"kargo-.exe",         // missing os and arch
		"KARGO-linux-amd64",  // uppercase
		"kargo-linux-amd64~", // trailing character
	}
	for _, name := range nonMatches {
		t.Run("no match: "+name, func(t *testing.T) {
			assert.Nil(t, cliAssetPattern.FindStringSubmatch(name))
		})
	}
}

func TestRelease_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		assertions  func(*testing.T, Release)
		expectedErr bool
	}{
		{
			name: "parses version and CLI assets across platforms",
			input: `{
				"tag_name": "v1.2.3",
				"assets": [
					{"name": "kargo-linux-amd64", "browser_download_url": "https://example.com/kargo-linux-amd64"},
					{"name": "kargo-linux-arm64", "browser_download_url": "https://example.com/kargo-linux-arm64"},
					{"name": "kargo-darwin-arm64", "browser_download_url": "https://example.com/kargo-darwin-arm64"},
					{"name": "kargo-windows-amd64.exe", "browser_download_url": "https://example.com/kargo-windows-amd64.exe"}
				]
			}`,
			assertions: func(t *testing.T, r Release) {
				assert.True(t, v("1.2.3").Equal(r.Version))
				require.Len(t, r.CLIBinaries, 3)
				assert.Equal(t, "https://example.com/kargo-linux-amd64", r.CLIBinaries["linux"]["amd64"])
				assert.Equal(t, "https://example.com/kargo-linux-arm64", r.CLIBinaries["linux"]["arm64"])
				assert.Equal(t, "https://example.com/kargo-darwin-arm64", r.CLIBinaries["darwin"]["arm64"])
				assert.Equal(t, "https://example.com/kargo-windows-amd64.exe", r.CLIBinaries["windows"]["amd64"])
			},
		},
		{
			name: "filters out non-CLI assets",
			input: `{
				"tag_name": "v1.0.0",
				"assets": [
					{"name": "kargo-linux-amd64", "browser_download_url": "https://example.com/kargo-linux-amd64"},
					{"name": "checksums.txt", "browser_download_url": "https://example.com/checksums.txt"},
					{"name": "akuity-kargo_v1.spdx.json", "browser_download_url": "https://example.com/sbom"},
					{"name": "kargo-cli.intoto.jsonl", "browser_download_url": "https://example.com/provenance"},
					{"name": "Source code (zip)", "browser_download_url": "https://example.com/source.zip"},
					{"name": "Source code (tar.gz)", "browser_download_url": "https://example.com/source.tar.gz"}
				]
			}`,
			assertions: func(t *testing.T, r Release) {
				require.Len(t, r.CLIBinaries, 1)
				assert.Equal(t, "https://example.com/kargo-linux-amd64", r.CLIBinaries["linux"]["amd64"])
			},
		},
		{
			name: "no assets produces empty CLIBinaries",
			input: `{
				"tag_name": "v1.0.0",
				"assets": []
			}`,
			assertions: func(t *testing.T, r Release) {
				assert.True(t, v("1.0.0").Equal(r.Version))
				assert.Empty(t, r.CLIBinaries)
			},
		},
		{
			name:        "invalid semver tag returns error",
			input:       `{"tag_name": "nightly", "assets": []}`,
			expectedErr: true,
		},
		{
			name:        "invalid JSON returns error",
			input:       `not json`,
			expectedErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r Release
			err := json.Unmarshal([]byte(tt.input), &r)
			if tt.expectedErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			tt.assertions(t, r)
		})
	}
}
