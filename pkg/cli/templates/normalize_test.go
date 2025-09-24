package templates

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExample(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Single line",
			input:    "Hello, World!",
			expected: fmt.Sprintf(`%[1]sHello, World!`, indentation),
		},
		{
			name: "Multiple lines",
			input: `Hello,
World!
Go!`,
			expected: fmt.Sprintf(`%[1]sHello,
%[1]sWorld!
%[1]sGo!`, indentation),
		},
		{
			name: "Leading and trailing spaces",
			input: `  Hello  

World  `,
			expected: fmt.Sprintf(`%[1]sHello

%[1]sWorld`, indentation),
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name: "Single newline",
			input: `
`,
			expected: "",
		},
		{
			name: "Multiple consecutive newlines",
			input: `Line 1


Line 2`,
			expected: fmt.Sprintf(`%[1]sLine 1


%[1]sLine 2`, indentation),
		},
		{
			name: "Spaces and newlines",
			input: ` 
   
  `,
			expected: "",
		},
		{
			name: "Mixed cases",
			input: ` Hello 
World
 Go! `,
			expected: fmt.Sprintf(`%[1]sHello
%[1]sWorld
%[1]sGo!`, indentation),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := Example(testCase.input)
			require.Equal(t, testCase.expected, actual)
		})
	}
}
