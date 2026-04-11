package governance

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_renderTemplate(t *testing.T) {
	testCases := []struct {
		name     string
		tmpl     string
		data     map[string]string
		expected string
	}{
		{
			name:     "no data passthrough",
			tmpl:     "Hello world",
			data:     nil,
			expected: "Hello world",
		},
		{
			name:     "template with variables",
			tmpl:     "Issue #{{.IssueNumber}} blocked by {{.BlockingLabels}}",
			data:     map[string]string{"IssueNumber": "42", "BlockingLabels": "`kind/proposal`"},
			expected: "Issue #42 blocked by `kind/proposal`",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := renderTemplate(testCase.tmpl, testCase.data)
			require.NoError(t, err)
			require.Equal(t, testCase.expected, result)
		})
	}
}
