package helm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetChartVersionsFromClassicRepo(t *testing.T) {
	// This is a mock registry. Depending on the request path, it returns a 404,
	// invalid YAML, or valid YAML.
	testServer := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				defer r.Body.Close()
				switch r.URL.Path {
				case "/bad-repo/index.yaml":
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte("this isn't yaml"))
					require.NoError(t, err)
				case "/fake-repo/index.yaml":
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(`entries:
  fake-chart:
    - version: 1.0.0
    - version: 1.1.0
    - version: 1.2.0
`))
					require.NoError(t, err)
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			},
		),
	)
	defer testServer.Close()
	testCases := []struct {
		name       string
		repoURL    string
		chart      string
		assertions func(t *testing.T, versions []string, err error)
	}{
		{
			name:    "request for repo index returns non-200 status",
			repoURL: fmt.Sprintf("%s/non-existent-repo", testServer.URL),
			chart:   "fake-chart",
			assertions: func(t *testing.T, _ []string, err error) {
				require.ErrorContains(t, err, "received unexpected HTTP 404")
			},
		},
		{
			name:    "index isn't valid YAML",
			repoURL: fmt.Sprintf("%s/bad-repo", testServer.URL),
			chart:   "fake-chart",
			assertions: func(t *testing.T, _ []string, err error) {
				require.ErrorContains(t, err, "error unmarshaling repository index")
			},
		},
		{
			name:    "no versions found",
			repoURL: fmt.Sprintf("%s/fake-repo", testServer.URL),
			chart:   "non-existent-chart",
			assertions: func(t *testing.T, _ []string, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:    "success",
			repoURL: fmt.Sprintf("%s/fake-repo", testServer.URL),
			chart:   "fake-chart",
			assertions: func(t *testing.T, versions []string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"1.0.0", "1.1.0", "1.2.0"}, versions)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			versions, err := getChartVersionsFromClassicRepo(
				testCase.repoURL,
				testCase.chart,
				nil,
			)
			testCase.assertions(t, versions, err)
		})
	}
}

func TestGetChartVersionsFromOCIRepo(t *testing.T) {
	// Instead of mocking out an OCI registry, it's more expedient to use Kargo's
	// own chart repo on ghcr.io to test this.
	versions, err := getChartVersionsFromOCIRepo(
		context.Background(),
		"oci://ghcr.io/akuity/kargo-charts/kargo",
		nil,
	)
	require.NoError(t, err)
	require.NotEmpty(t, versions)
}

func TestVersionsToSemVerCollection(t *testing.T) {
	testCases := []struct {
		name     string
		input    []string
		isOCI    bool
		expected semver.Collection
	}{
		{
			name:     "empty input",
			input:    []string{},
			expected: semver.Collection{},
		},
		{
			name:  "valid semvers",
			input: []string{"1.2.3", "4.5.6", "7.8.9"},
			expected: semver.Collection{
				semver.MustParse("1.2.3"),
				semver.MustParse("4.5.6"),
				semver.MustParse("7.8.9"),
			},
		},
		{
			name:  "mixed valid and invalid semvers",
			input: []string{"1.2.3", "invalid", "4.5.6", "also-invalid", "7.8.9"},
			expected: semver.Collection{
				semver.MustParse("1.2.3"),
				semver.MustParse("4.5.6"),
				semver.MustParse("7.8.9"),
			},
		},
		{
			name:  "prerelease versions",
			input: []string{"1.2.3-alpha.1", "4.5.6-beta.2", "7.8.9-rc.3"},
			expected: semver.Collection{
				semver.MustParse("1.2.3-alpha.1"),
				semver.MustParse("4.5.6-beta.2"),
				semver.MustParse("7.8.9-rc.3"),
			},
		},
		{
			name:  "metadata versions",
			input: []string{"1.2.3+build.1", "4.5.6+build.2", "7.8.9+build.3"},
			expected: semver.Collection{
				semver.MustParse("1.2.3+build.1"),
				semver.MustParse("4.5.6+build.2"),
				semver.MustParse("7.8.9+build.3"),
			},
		},
		{
			name:  "metadata versions from OCI origin",
			input: []string{"1.2.3_build.1", "4.5.6_build.2", "7.8.9_build.3"},
			isOCI: true,
			expected: semver.Collection{
				semver.MustParse("1.2.3+build.1"),
				semver.MustParse("4.5.6+build.2"),
				semver.MustParse("7.8.9+build.3"),
			},
		},
		{
			name:     "loose versions from OCI origin",
			input:    []string{"v1.2.3", "v4.5.6", "v7.8.9"},
			isOCI:    true,
			expected: semver.Collection{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := versionsToSemVerCollection(tc.input, tc.isOCI)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestSemVerCollectionToVersions(t *testing.T) {
	testCases := []struct {
		name     string
		input    semver.Collection
		expected []string
	}{
		{
			name:     "empty collection",
			input:    semver.Collection{},
			expected: []string{},
		},
		{
			name: "valid semvers",
			input: semver.Collection{
				semver.MustParse("1.2.3"),
				semver.MustParse("4.5.6"),
				semver.MustParse("7.8.9"),
			},
			expected: []string{"1.2.3", "4.5.6", "7.8.9"},
		},
		{
			name: "prerelease versions",
			input: semver.Collection{
				semver.MustParse("1.2.3-alpha.1"),
				semver.MustParse("4.5.6-beta.2"),
				semver.MustParse("7.8.9-rc.3"),
			},
			expected: []string{"1.2.3-alpha.1", "4.5.6-beta.2", "7.8.9-rc.3"},
		},
		{
			name: "metadata versions",
			input: semver.Collection{
				semver.MustParse("1.2.3+build.1"),
				semver.MustParse("4.5.6+build.2"),
				semver.MustParse("7.8.9+build.3"),
			},
			expected: []string{"1.2.3+build.1", "4.5.6+build.2", "7.8.9+build.3"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := semVerCollectionToVersions(tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestFilterSemVers(t *testing.T) {
	testCases := []struct {
		name             string
		input            semver.Collection
		constraint       string
		expectedFiltered semver.Collection
		expectedError    error
	}{
		{
			name:             "empty collection",
			input:            semver.Collection{},
			constraint:       "^1.2.3",
			expectedFiltered: semver.Collection{},
		},
		{
			name: "exact version constraint",
			input: semver.Collection{
				semver.MustParse("1.2.3"),
				semver.MustParse("4.5.6"),
				semver.MustParse("7.8.9"),
			},
			constraint: "=4.5.6",
			expectedFiltered: semver.Collection{
				semver.MustParse("4.5.6"),
			},
		},
		{
			name: "range constraint",
			input: semver.Collection{
				semver.MustParse("1.2.3"),
				semver.MustParse("4.5.6"),
				semver.MustParse("7.8.9"),
			},
			constraint: ">=4.5.6 <7.8.9",
			expectedFiltered: semver.Collection{
				semver.MustParse("4.5.6"),
			},
		},
		{
			name: "prerelease constraint",
			input: semver.Collection{
				semver.MustParse("1.2.3-alpha.1"),
				semver.MustParse("1.2.3-beta.2"),
				semver.MustParse("1.3.0"),
			},
			constraint: "1.2.x-0",
			expectedFiltered: semver.Collection{
				semver.MustParse("1.2.3-alpha.1"),
				semver.MustParse("1.2.3-beta.2"),
			},
		},
		{
			name: "multiple matches",
			input: semver.Collection{
				semver.MustParse("1.2.3"),
				semver.MustParse("1.2.4"),
				semver.MustParse("1.3.0"),
			},
			constraint: "1.2.x",
			expectedFiltered: semver.Collection{
				semver.MustParse("1.2.3"),
				semver.MustParse("1.2.4"),
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filtered, err := filterSemVers(tc.input, tc.constraint)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedFiltered, filtered)
		})
	}

	t.Run("invalid constraint", func(t *testing.T) {
		_, err := filterSemVers(semver.Collection{}, "invalid")
		assert.ErrorContains(t, err, "error parsing constraint")
	})
}

func TestNormalizeChartRepositoryURL(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single word",
			input:    "repo",
			expected: "repo",
		},
		{
			name:     "leading and trailing whitespace",
			input:    "  repo  ",
			expected: "repo",
		},
		{
			name:     "mixed case",
			input:    "REpo",
			expected: "repo",
		},
		{
			name:     "oci prefix",
			input:    "oci://repo",
			expected: "repo",
		},
		{
			name:     "oci prefix with whitespace",
			input:    "  oci://repo  ",
			expected: "repo",
		},
		{
			name:     "oci prefix with mixed case",
			input:    "OCI://Repo",
			expected: "repo",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := NormalizeChartRepositoryURL(tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
