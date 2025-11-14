package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

func TestLoadCLIConfig(t *testing.T) {
	testConfig := CLIConfig{
		APIAddress:  "http://localhost:8080",
		BearerToken: "thisisafaketoken",
	}
	testCases := []struct {
		name       string
		setup      func() string
		assertions func(*testing.T, CLIConfig, error)
	}{
		{
			name: "file does not exist",
			setup: func() string {
				return getTestConfigPath()
			},
			assertions: func(t *testing.T, _ CLIConfig, err error) {
				require.ErrorContains(t, err, "no configuration file was found")
			},
		},
		{
			name: "file exists but is invalid",
			setup: func() string {
				configPath := getTestConfigPath()
				err := os.WriteFile(configPath, []byte("this is not yaml"), 0600)
				require.NoError(t, err)
				return configPath
			},
			assertions: func(t *testing.T, _ CLIConfig, err error) {
				require.ErrorContains(t, err, "error parsing configuration file")
			},
		},
		{
			name: "file exists and is valid",
			setup: func() string {
				configPath := getTestConfigPath()
				configBytes, err := yaml.Marshal(testConfig)
				require.NoError(t, err)
				err = os.WriteFile(configPath, configBytes, 0600)
				require.NoError(t, err)
				return configPath
			},
			assertions: func(t *testing.T, cfg CLIConfig, err error) {
				require.NoError(t, err)
				require.Equal(t, testConfig, cfg)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			configPath := testCase.setup()
			cfg, err := loadCLIConfig(configPath)
			testCase.assertions(t, cfg, err)
		})
	}
}

func TestSaveCLIConfig(t *testing.T) {
	testConfig := CLIConfig{
		APIAddress:  "http://localhost:8080",
		BearerToken: "thisisafaketoken",
	}

	configPath := getTestConfigPath()

	err := saveCLIConfig(testConfig, configPath)
	require.NoError(t, err)

	configBytes, err := os.ReadFile(configPath)
	require.NoError(t, err)
	cfg := CLIConfig{}
	err = yaml.Unmarshal(configBytes, &cfg)
	require.NoError(t, err)
	require.Equal(t, testConfig, cfg)
}

func TestDeleteCLIConfig(t *testing.T) {
	testCases := []struct {
		name  string
		setup func() string
	}{
		{
			name: "file does not exist",
			setup: func() string {
				return getTestConfigPath()
			},
		},
		{
			name: "file exists",
			setup: func() string {
				configPath := getTestConfigPath()
				err := os.WriteFile(configPath, []byte("nonsense"), 0600)
				require.NoError(t, err)
				return configPath
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.NoError(t, deleteCLIConfig(testCase.setup()))
		})
	}
}

func TestMaskedConfig(t *testing.T) {
	testConfig := CLIConfig{
		APIAddress:            "http://localhost:8080",
		BearerToken:           "secret",
		RefreshToken:          "secret",
		InsecureSkipTLSVerify: true,
		Project:               "project",
	}
	maskedConfig := MaskedConfig(testConfig)

	require.Equal(t, "http://localhost:8080", maskedConfig.APIAddress)
	require.Equal(t, dataMask, maskedConfig.BearerToken)
	require.Equal(t, dataMask, maskedConfig.RefreshToken)
	require.Equal(t, true, maskedConfig.InsecureSkipTLSVerify)
	require.Equal(t, "project", maskedConfig.Project)
}

func getTestConfigPath() string {
	return filepath.Join(os.TempDir(), "config")
}
