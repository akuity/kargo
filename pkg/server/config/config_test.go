package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeArgoCDURLs(t *testing.T) {

	testCases := []struct {
		name     string
		input    string
		expected ArgoCDURLMap
		errMsg   string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: ArgoCDURLMap(map[string]string{}),
		},
		{
			name:  "empty shard",
			input: `=https://argocd.com`,
			expected: ArgoCDURLMap(map[string]string{
				"": "https://argocd.com",
			}),
		},
		{
			name:   "invalid input",
			input:  `https://argocd.com`,
			errMsg: "expected <shard>=<URL>",
		},
		{
			name:  "multiple shards",
			input: `foo=https://argocd.com,bar=https://argocd.org`,
			expected: ArgoCDURLMap(map[string]string{
				"foo": "https://argocd.com",
				"bar": "https://argocd.org",
			}),
		},
		{
			name:  "ignore empty items",
			input: `foo=https://argocd.com,,bar=https://argocd.org,`,
			expected: ArgoCDURLMap(map[string]string{
				"foo": "https://argocd.com",
				"bar": "https://argocd.org",
			}),
		},
		{
			name:  "trim leading and trailing whitespace",
			input: ` = https://argocd.com , , bar = https://argocd.org`,
			expected: ArgoCDURLMap(map[string]string{
				"":    "https://argocd.com",
				"bar": "https://argocd.org",
			}),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var acdMap ArgoCDURLMap
			err := acdMap.Decode(tc.input)
			if tc.errMsg != "" {
				require.ErrorContains(t, err, tc.errMsg)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, acdMap)
		})
	}
}

func TestNormalizeBasePath(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty", input: "", expected: ""},
		{name: "whitespace only", input: "  ", expected: ""},
		{name: "already canonical", input: "/kargo", expected: "/kargo"},
		{name: "missing leading slash", input: "kargo", expected: "/kargo"},
		{name: "trailing slash", input: "/kargo/", expected: "/kargo"},
		{name: "neither leading nor trailing slash", input: "kargo", expected: "/kargo"},
		{name: "multi-segment", input: "/foo/bar", expected: "/foo/bar"},
		{name: "multi-segment trailing slash", input: "/foo/bar/", expected: "/foo/bar"},
		{name: "leading and trailing whitespace", input: "  /kargo  ", expected: "/kargo"},
		{name: "root slash", input: "/", expected: ""},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, NormalizeBasePath(tc.input))
		})
	}
}
