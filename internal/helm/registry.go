package helm

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"helm.sh/helm/v3/pkg/registry"
)

func NewRegistryClient(home string) (*registry.Client, error) {
	credentialsPath := filepath.Join(home, ".docker", "config.json")
	if _, err := os.Stat(credentialsPath); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("error checking for credentials file existence: %w", err)
		}

		// Credentials file does not exist, create a new one.
		if err = os.MkdirAll(filepath.Dir(credentialsPath), 0o700); err != nil {
			return nil, fmt.Errorf("error creating credentials directory: %w", err)
		}
		if err = os.WriteFile(credentialsPath, []byte("{}"), 0o600); err != nil {
			return nil, fmt.Errorf("error creating credentials file: %w", err)
		}
	}

	opts := []registry.ClientOption{
		registry.ClientOptWriter(io.Discard),
		registry.ClientOptCredentialsFile(credentialsPath),
		// NB: Disable the cache, preventing Helm from opting to use a global cache.
		registry.ClientOptEnableCache(false),
		// TODO(hidde): enable https://github.com/helm/helm/pull/12588 to further
		// isolate ourselves from the system global configuration.
	}

	return registry.NewClient(opts...)
}
