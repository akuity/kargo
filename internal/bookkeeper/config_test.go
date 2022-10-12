package bookkeeper

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadRepoConfig(t *testing.T) {
	testCases := []struct {
		name       string
		setup      func() (string, error)
		assertions func(error)
	}{
		{
			name: "invalid JSON",
			setup: func() (string, error) {
				dir, err := os.MkdirTemp("", "")
				if err != nil {
					return "", err
				}
				return dir, os.WriteFile(
					filepath.Join(dir, "Bookfile.json"),
					[]byte("bogus"),
					0600,
				)
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error unmarshaling JSON config file")
			},
		},
		{
			name: "invalid YAML",
			setup: func() (string, error) {
				dir, err := os.MkdirTemp("", "")
				if err != nil {
					return "", err
				}
				return dir, os.WriteFile(
					filepath.Join(dir, "Bookfile.yaml"),
					[]byte("bogus"),
					0600,
				)
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error unmarshaling YAML config file")
			},
		},
		{
			name: "valid JSON",
			setup: func() (string, error) {
				dir, err := os.MkdirTemp("", "")
				if err != nil {
					return "", err
				}
				return dir, os.WriteFile(
					filepath.Join(dir, "Bookfile.json"),
					[]byte("{}"),
					0600,
				)
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "valid YAML",
			setup: func() (string, error) {
				dir, err := os.MkdirTemp("", "")
				if err != nil {
					return "", err
				}
				return dir, os.WriteFile(
					filepath.Join(dir, "Bookfile.yaml"),
					[]byte(""), // An empty file should actually be valid
					0600,
				)
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.setup != nil {
				path, err := testCase.setup()
				require.NoError(t, err)
				_, err = LoadRepoConfig(path)
				testCase.assertions(err)
			}
		})
	}
}

func TestGetBranchConfig(t *testing.T) {
	const testBranchName = "foo"
	testCases := []struct {
		name       string
		repoConfig RepoConfig
		assertions func(BranchConfig)
	}{
		{
			name: "branch config explicitly specified",
			repoConfig: &repoConfig{
				BranchConfigs: []BranchConfig{
					{
						Name: testBranchName,
						ConfigManagement: ConfigManagementConfig{
							Ytt: &YttConfig{},
						},
					},
				},
			},
			assertions: func(cfg BranchConfig) {
				require.Equal(t, testBranchName, cfg.Name)
				require.Nil(t, cfg.ConfigManagement.Helm)
				require.Nil(t, cfg.ConfigManagement.Kustomize)
				require.NotNil(t, cfg.ConfigManagement.Ytt)
			},
		},
		{
			name: "default branch config explicitly specified",
			repoConfig: &repoConfig{
				DefaultBranchConfig: &BranchConfig{
					ConfigManagement: ConfigManagementConfig{
						Ytt: &YttConfig{},
					},
				},
			},
			assertions: func(cfg BranchConfig) {
				require.Equal(t, testBranchName, cfg.Name)
				require.Nil(t, cfg.ConfigManagement.Helm)
				require.Nil(t, cfg.ConfigManagement.Kustomize)
				require.NotNil(t, cfg.ConfigManagement.Ytt)
			},
		},
		{
			name:       "nothing explicitly specified",
			repoConfig: &repoConfig{},
			assertions: func(cfg BranchConfig) {
				require.Equal(t, testBranchName, cfg.Name)
				require.Nil(t, cfg.ConfigManagement.Helm)
				require.NotNil(t, cfg.ConfigManagement.Kustomize)
				require.Nil(t, cfg.ConfigManagement.Ytt)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.repoConfig.GetBranchConfig(testBranchName),
			)
		})
	}
}
