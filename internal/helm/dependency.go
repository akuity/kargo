package helm

import (
	"fmt"
	"os"

	"sigs.k8s.io/yaml"
)

// ChartDependency represents a dependency of a Helm chart.
//
// It contains the repository URL, chart name, and version of the dependency
// as specified in the chart's Chart.yaml or Chart.lock file.
type ChartDependency struct {
	Repository string `json:"repository,omitempty"`
	Name       string `json:"name,omitempty"`
	Version    string `json:"version,omitempty"`
}

// chartMetadata is a minimal representation of a Helm chart's metadata
// that includes only the dependencies. This is used to parse the Chart.yaml
// or Chart.lock file to extract the dependencies of a Helm chart.
type chartMetadata struct {
	Dependencies []ChartDependency `json:"dependencies,omitempty"`
}

// GetChartDependencies reads a Helm chart's Chart.yaml or Chart.lock file
// and returns a slice of ChartDependency structs representing the chart's
// dependencies. If the file cannot be read or parsed, an error is returned.
func GetChartDependencies(p string) ([]ChartDependency, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("read file %q: %w", p, err)
	}

	var meta chartMetadata
	if err = yaml.Unmarshal(b, &meta); err != nil {
		return nil, fmt.Errorf("unmarshal %q: %w", p, err)
	}
	return meta.Dependencies, nil
}
