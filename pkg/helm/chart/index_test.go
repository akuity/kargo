package chart

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
)

// unmarshalWholeIndex is the reference behavior that versionsFromIndex must
// match: unmarshal the entire index, then pick out one chart.
func unmarshalWholeIndex(t *testing.T, index, chartName string) []string {
	t.Helper()
	parsed := struct {
		Entries map[string][]indexEntry `yaml:"entries"`
	}{}
	require.NoError(t, yaml.Unmarshal([]byte(index), &parsed), index)
	entries, ok := parsed.Entries[chartName]
	if !ok {
		return nil
	}
	versions := make([]string, len(entries))
	for i, entry := range entries {
		versions[i] = entry.Version
	}
	return versions
}

func Test_versionsFromIndex(t *testing.T) {
	testCases := []struct {
		name     string
		index    string
		chart    string
		expected []string
	}{
		{
			name: "helm style, chart in the middle",
			index: `apiVersion: v1
entries:
  alpha:
  - name: alpha
    version: 9.9.9
  fake-chart:
  - apiVersion: v2
    name: fake-chart
    urls:
    - https://example.com/fake-chart-1.0.0.tgz
    version: 1.0.0
  - name: fake-chart
    version: 1.1.0
  zulu:
  - name: zulu
    version: 8.8.8
generated: "2025-01-01T00:00:00Z"
`,
			chart:    "fake-chart",
			expected: []string{"1.0.0", "1.1.0"},
		},
		{
			name: "chart is last and the index ends inside its entries",
			index: `entries:
  alpha:
  - version: 9.9.9
  fake-chart:
  - version: 1.0.0`,
			chart:    "fake-chart",
			expected: []string{"1.0.0"},
		},
		{
			name: "indented sequences",
			index: `entries:
    alpha:
        - version: 9.9.9
    fake-chart:
        - version: 1.0.0
        - version: 1.1.0
`,
			chart:    "fake-chart",
			expected: []string{"1.0.0", "1.1.0"},
		},
		{
			name: "entries after other top-level keys",
			index: `apiVersion: v1
generated: "2025-01-01T00:00:00Z"
serverInfo:
  contextPath: /charts
entries:
  fake-chart:
  - version: 1.0.0
`,
			chart:    "fake-chart",
			expected: []string{"1.0.0"},
		},
		{
			name: "quoted keys",
			index: `"entries":
  'alpha':
  - version: 9.9.9
  "fake-chart" :
  - version: 1.0.0
`,
			chart:    "fake-chart",
			expected: []string{"1.0.0"},
		},
		{
			name: "comments, blank lines, and CRLF line endings",
			index: "# generated\r\nentries: # all charts\r\n\r\n  # alpha\r\n  alpha:\r\n" +
				"  - version: 9.9.9\r\n  fake-chart:\r\n  # oldest first\r\n\r\n" +
				"  - version: 1.0.0\r\n\r\n  - version: 1.1.0\r\n",
			chart:    "fake-chart",
			expected: []string{"1.0.0", "1.1.0"},
		},
		{
			name: "block scalars that resemble keys",
			index: `entries:
  alpha:
  - description: |
      entries:
      fake-chart:
      - version: 6.6.6
    version: 9.9.9
  fake-chart:
  - description: >-
      not a key:
    version: 1.0.0
  omega:
  - version: 7.7.7
`,
			chart:    "fake-chart",
			expected: []string{"1.0.0"},
		},
		{
			name: "chart name is a prefix of another chart's name",
			index: `entries:
  fake-chart-extra:
  - version: 9.9.9
  fake-chart:
  - version: 1.0.0
`,
			chart:    "fake-chart",
			expected: []string{"1.0.0"},
		},
		{
			name: "flow-style chart entries",
			index: `entries:
  alpha: [{version: 9.9.9}]
  fake-chart: [{version: 1.0.0},
    {version: 1.1.0}]
`,
			chart:    "fake-chart",
			expected: []string{"1.0.0", "1.1.0"},
		},
		{
			name: "quoted versions",
			index: `entries:
  fake-chart:
  - version: "1.0.0"
  - version: '1.1.0'
`,
			chart:    "fake-chart",
			expected: []string{"1.0.0", "1.1.0"},
		},
		{
			name: "chart not listed",
			index: `entries:
  alpha:
  - version: 9.9.9
generated: "2025-01-01T00:00:00Z"
`,
			chart: "fake-chart",
		},
		{
			name:  "repository with no charts",
			index: "apiVersion: v1\nentries: {}\ngenerated: \"2025-01-01T00:00:00Z\"\n",
			chart: "fake-chart",
		},
		{
			name:  "null entries",
			index: "apiVersion: v1\nentries: ~ # none yet\n",
			chart: "fake-chart",
		},
		{
			name:     "tab after a chart key's colon",
			index:    "entries:\n  alpha:\t# first\n  - version: 9.9.9\n  fake-chart:\t\n  - version: 1.0.0\n",
			chart:    "fake-chart",
			expected: []string{"1.0.0"},
		},
		{
			name:  "no entries key",
			index: "apiVersion: v1\n",
			chart: "fake-chart",
		},
		{
			name:  "empty index",
			index: "",
			chart: "fake-chart",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			versions, err := versionsFromIndex(strings.NewReader(testCase.index), testCase.chart)
			require.NoError(t, err)
			require.Equal(t, testCase.expected, versions)
			require.Equal(t, unmarshalWholeIndex(t, testCase.index, testCase.chart), versions)
		})
	}
}

func Test_versionsFromIndex_json(t *testing.T) {
	testCases := []struct {
		name     string
		index    string
		expected []string
	}{
		{
			name: "pretty-printed",
			index: `{
  "apiVersion": "v1",
  "entries": {
    "alpha": [{"version": "9.9.9", "urls": ["https://example.com/a.tgz"]}],
    "fake-chart": [
      {"name": "fake-chart", "version": "1.0.0"},
      {"name": "fake-chart", "version": "1.1.0"}
    ]
  },
  "generated": "2025-01-01T00:00:00Z"
}`,
			expected: []string{"1.0.0", "1.1.0"},
		},
		{
			name:     "compact, entries last",
			index:    `{"apiVersion":"v1","serverInfo":{"x":[1,{"y":null}]},"entries":{"fake-chart":[{"version":"1.0.0"}]}}`,
			expected: []string{"1.0.0"},
		},
		{
			name:  "chart not listed",
			index: `{"entries":{"alpha":[{"version":"9.9.9"}]}}`,
		},
		{
			name:  "no entries key",
			index: `{"apiVersion":"v1"}`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			versions, err := versionsFromIndex(strings.NewReader(testCase.index), "fake-chart")
			require.NoError(t, err)
			require.Equal(t, testCase.expected, versions)
			require.Equal(t, unmarshalWholeIndex(t, testCase.index, "fake-chart"), versions)
		})
	}
}

func Test_versionsFromIndex_errors(t *testing.T) {
	testCases := []struct {
		name        string
		index       string
		errContains string
	}{
		{
			name:        "not a mapping",
			index:       "this isn't yaml",
			errContains: "response is not a repository index",
		},
		{
			name:        "HTML error page",
			index:       "<html>\n<body>Sign in</body>\n</html>\n",
			errContains: "response is not a repository index",
		},
		{
			name:        "malformed chart entries",
			index:       "entries:\n  fake-chart:\n  - version: [1.0.0\n",
			errContains: `error unmarshaling entries for chart "fake-chart"`,
		},
		{
			name:        "entries is a sequence",
			index:       "entries:\n  - version: 1.0.0\n",
			errContains: "repository index entries are malformed",
		},
		{
			name:        "flow-style entries mapping",
			index:       "entries: {fake-chart: [{version: 1.0.0}]}\n",
			errContains: "not in block style",
		},
		{
			name:        "truncated JSON",
			index:       `{"entries":{"alpha":[{"version":`,
			errContains: "unexpected EOF",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := versionsFromIndex(strings.NewReader(testCase.index), "fake-chart")
			require.ErrorContains(t, err, testCase.errContains)
		})
	}
}

func Test_versionsFromIndex_decodesOnlyTheNamedChart(t *testing.T) {
	// Entries for other charts are skipped without being parsed, so malformed
	// entries elsewhere in the index do not prevent reading the named chart.
	const index = `entries:
  alpha:
  - version: [9.9.9
  fake-chart:
  - version: 1.0.0
  zulu:
  - version: [8.8.8
`
	versions, err := versionsFromIndex(strings.NewReader(index), "fake-chart")
	require.NoError(t, err)
	require.Equal(t, []string{"1.0.0"}, versions)

	const jsonIndex = `{"entries":{"alpha":[{"version":9}],"fake-chart":[{"version":"1.0.0"}],` +
		`"zulu":[{"version":8}]}}`
	versions, err = versionsFromIndex(strings.NewReader(jsonIndex), "fake-chart")
	require.NoError(t, err)
	require.Equal(t, []string{"1.0.0"}, versions)
}

func Test_versionsFromIndex_helmWrittenIndices(t *testing.T) {
	// Metadata that a line-oriented scanner could mistake for index structure.
	trickyText := []string{
		"plain", "", " leading space", "trailing space ", "multi\nline\ntext",
		"entries:", "  other-chart:", "- version: 6.6.6", "\n  fake:\n  - version: 6.6.6\n",
		"key: value", "# not a comment", "'single'", `"double"`, "tab\there", "ünïcødé",
		"{flow: map}", "[a, b]", "---", "...", "&anchor *alias", "!!str tagged", "| block",
		"> folded", "with\n\n\nblank lines", "\r\nwindows\r\n", ":", "a:b", "x # y",
	}
	r := rand.New(rand.NewSource(1)) // #nosec G404 -- deterministic test data
	tricky := func() string { return trickyText[r.Intn(len(trickyText))] + trickyText[r.Intn(len(trickyText))] }
	dir := t.TempDir()
	for i := range 40 {
		index := repo.NewIndexFile()
		var names []string
		for range r.Intn(12) + 1 {
			name := fmt.Sprintf("chart-%d", r.Intn(15))
			if r.Intn(4) == 0 {
				name = "fake"
			}
			for range r.Intn(6) + 1 {
				md := &chart.Metadata{
					APIVersion:  "v2",
					Name:        name,
					Version:     fmt.Sprintf("%d.%d.%d", r.Intn(3), r.Intn(10), r.Intn(10)),
					Description: tricky(),
					Home:        tricky(),
					Keywords:    []string{tricky(), tricky()},
					Annotations: map[string]string{tricky() + "k": tricky()},
					AppVersion:  tricky(),
				}
				if index.MustAdd(md, name+".tgz", "https://example.com", "digest") == nil {
					names = append(names, name)
				}
			}
		}
		index.SortEntries()
		yamlPath := filepath.Join(dir, fmt.Sprintf("%d.yaml", i))
		require.NoError(t, index.WriteFile(yamlPath, 0o600))
		jsonPath := filepath.Join(dir, fmt.Sprintf("%d.json", i))
		require.NoError(t, index.WriteJSONFile(jsonPath, 0o600))
		for _, path := range []string{yamlPath, jsonPath} {
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			whole := struct {
				Entries map[string][]indexEntry `yaml:"entries"`
			}{}
			require.NoError(t, yaml.Unmarshal(data, &whole))
			for _, name := range append(names, "chart-99", "fak") {
				var expected []string
				if entries, ok := whole.Entries[name]; ok {
					expected = make([]string, len(entries))
					for j, entry := range entries {
						expected[j] = entry.Version
					}
				}
				versions, err := versionsFromIndex(bytes.NewReader(data), name)
				require.NoError(t, err, "chart %q in\n%s", name, data)
				require.Equal(t, expected, versions, "chart %q in\n%s", name, data)
			}
		}
	}
}
