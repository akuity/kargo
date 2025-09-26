package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"sigs.k8s.io/yaml"
)

const dataMask = "*** REDACTED ***"

var xdgConfigPath string

func init() {
	// If the XDG_CONFIG_HOME env var isn't set, we want to set it ourselves
	// because we disagree with both Go and the xdg package's interpretation of
	// what the default config home directory should be on non-*nix systems.
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			panic(fmt.Errorf("error determining user home directory: %w", err))
		}
		// This is what the spec says the default should be.
		//
		// See https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html
		if err := os.Setenv("XDG_CONFIG_HOME", filepath.Join(userHome, ".config")); err != nil {
			panic(fmt.Errorf("set XDG_CONFIG_HOME environment variable: %w", err))
		}
		xdg.Reload()
	}
	var err error
	if xdgConfigPath, err =
		xdg.ConfigFile(filepath.Join("kargo", "config")); err != nil {
		panic(fmt.Errorf("error determining XDG config path: %w", err))
	}
}

// CLIConfig represents CLI configuration.
type CLIConfig struct {
	// AuthMethod is the method used to authenticate with the Kargo API server.
	AuthMethod string `json:"authMethod,omitempty"`
	// APIAddress is the address of the Kargo API server.
	APIAddress string `json:"apiAddress,omitempty"`
	// BearerToken is used to authenticate with the Kargo API server. This could
	// be any of the following:
	//   1. An identity token issued by an OIDC identity provider
	//   2. An identity token issued by the Kargo API server itself
	//   3. An opaque token for the Kubernetes API server that the Kargo API
	//      server will communicate with
	// This token will be sent in the Authorization header of all requests to the
	// Kargo API server. The Kargo API server will ascertain which of the three
	// cases above applies and will act accordingly.
	BearerToken string `json:"bearerToken,omitempty"`
	// RefreshToken, if set, is used to refresh the Token, which must, in such a
	// case, have been issued by an OIDC identity provider.
	RefreshToken string `json:"refreshToken,omitempty"`
	// InsecureSkipTLSVerify indicates whether the user indicated during login
	// that certificate warnings should be ignored. When true, this option will be
	// applied to all subsequent Kargo commands until the user logs out or
	// re-authenticates. When true, refresh tokens will not be used, thereby
	// forcing users to periodically re-assess this choice.
	InsecureSkipTLSVerify bool `json:"insecureSkipTLSVerify,omitempty"`
	// Project is the default Project for the command.
	Project string `json:"project,omitempty"`
}

// NewDefaultCLIConfig returns a new default CLI configuration.
func NewDefaultCLIConfig() CLIConfig {
	return CLIConfig{}
}

// LoadCLIConfig loads Kargo CLI configuration from a file in the Kargo home
// directory.
func LoadCLIConfig() (CLIConfig, error) {
	return loadCLIConfig(xdgConfigPath)
}

func loadCLIConfig(configPath string) (CLIConfig, error) {
	var cfg CLIConfig
	_, err := os.Stat(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, fmt.Errorf("please use `kargo login` to continue: %w", NewConfigNotFoundErr(configPath))
		}
		return cfg, fmt.Errorf("os.Stat: %w", err)
	}
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		return cfg, fmt.Errorf(
			"error reading configuration file at %s: %w",
			configPath,
			err,
		)
	}
	if err := yaml.Unmarshal(configBytes, &cfg); err != nil {
		return cfg, fmt.Errorf(
			"error parsing configuration file at %s: %w",
			configPath,
			err,
		)
	}
	return cfg, nil
}

// SaveCLIConfig saves Kargo CLI configuration to a file in the Kargo home
// directory.
func SaveCLIConfig(config CLIConfig) error {
	return saveCLIConfig(config, xdgConfigPath)
}

func saveCLIConfig(config CLIConfig, configPath string) error {
	configBytes, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("error marshaling config: %w", err)
	}
	if err :=
		os.WriteFile(configPath, configBytes, 0600); err != nil {
		return fmt.Errorf("error writing to %s: %w", configPath, err)
	}
	return nil
}

// DeleteCLIConfig deletes the Kargo CLI configuration file from the Kargo home
// directory.
func DeleteCLIConfig() error {
	return deleteCLIConfig(xdgConfigPath)
}

func deleteCLIConfig(configPath string) error {
	if err := os.RemoveAll(configPath); err != nil {
		return fmt.Errorf("error deleting configuration: %w", err)
	}
	return nil
}

func MaskedConfig(config CLIConfig) CLIConfig {
	// We reconstruct the config to avoid accidentally exposing new fields.
	return CLIConfig{
		APIAddress:            config.APIAddress,
		BearerToken:           dataMask,
		RefreshToken:          dataMask,
		InsecureSkipTLSVerify: config.InsecureSkipTLSVerify,
		Project:               config.Project,
	}
}
