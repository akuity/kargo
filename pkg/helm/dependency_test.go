package helm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetChartDependencies(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*testing.T) string
		assertions func(*testing.T, []ChartDependency, error)
	}{
		{
			name: "valid chart.yaml",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()

				const chartYAML = `---
apiVersion: v2
name: test-chart
version: 0.1.0
dependencies:
- name: dep1
  version: 1.0.0
  repository: https://charts.example.com
- name: dep2
  version: 2.0.0
  repository: oci://registry.example.com/charts
`
				chartPath := filepath.Join(tmpDir, "Chart.yaml")
				require.NoError(t, os.WriteFile(chartPath, []byte(chartYAML), 0o600))

				return chartPath
			},
			assertions: func(t *testing.T, dependencies []ChartDependency, err error) {
				require.NoError(t, err)
				assert.Len(t, dependencies, 2)

				assert.Equal(t, []ChartDependency{
					{
						Name:       "dep1",
						Version:    "1.0.0",
						Repository: "https://charts.example.com",
					},
					{
						Name:       "dep2",
						Version:    "2.0.0",
						Repository: "oci://registry.example.com/charts",
					},
				}, dependencies)
			},
		},
		{
			name: "valid Chart.lock",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()

				const chartLock = `---
dependencies:
- name: dep1
  version: 1.0.0
  repository: https://charts.example.com
- name: dep2
  version: 2.0.0
  repository: oci://registry.example.com/charts
`
				lockPath := filepath.Join(tmpDir, "Chart.lock")
				require.NoError(t, os.WriteFile(lockPath, []byte(chartLock), 0o600))
				return lockPath
			},
			assertions: func(t *testing.T, dependencies []ChartDependency, err error) {
				require.NoError(t, err)
				assert.Len(t, dependencies, 2)

				assert.Equal(t, []ChartDependency{
					{
						Name:       "dep1",
						Version:    "1.0.0",
						Repository: "https://charts.example.com",
					},
					{
						Name:       "dep2",
						Version:    "2.0.0",
						Repository: "oci://registry.example.com/charts",
					},
				}, dependencies)
			},
		},
		{
			name: "invalid Chart.yaml",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()

				const chartYAML = `---
this is not a valid chart.yaml
`
				chartPath := filepath.Join(tmpDir, "Chart.yaml")
				require.NoError(t, os.WriteFile(chartPath, []byte(chartYAML), 0o600))

				return chartPath
			},
			assertions: func(t *testing.T, dependencies []ChartDependency, err error) {
				require.ErrorContains(t, err, "unmarshal")
				assert.Nil(t, dependencies)
			},
		},
		{
			name: "missing Chart.yaml",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "Chart.yaml")
			},
			assertions: func(t *testing.T, dependencies []ChartDependency, err error) {
				require.ErrorContains(t, err, "read file")
				assert.Nil(t, dependencies)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chartPath := tt.setup(t)
			dependencies, err := GetChartDependencies(chartPath)
			tt.assertions(t, dependencies, err)
		})
	}
}
