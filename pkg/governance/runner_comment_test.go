package governance

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-github/v76/github"
	"github.com/stretchr/testify/require"
)

func Test_commentRunner_run(t *testing.T) {
	testCases := []struct {
		name              string
		config            []byte
		templateData      map[string]string
		createCommentErr  error
		expectedBody      string
		expectErrContains string
	}{
		{
			name:              "decode error — config is not a string",
			config:            []byte(`[1, 2]`),
			expectErrContains: "decoding comment config",
		},
		{
			name:         "empty template — no API call",
			config:       []byte(`""`),
			expectedBody: "",
		},
		{
			name:              "template parse error",
			config:            []byte(`"hello {{ .Bad"`),
			templateData:      map[string]string{"Foo": "bar"},
			expectErrContains: "rendering comment template",
		},
		{
			name:         "happy path — no template data passes through verbatim",
			config:       []byte(`"hello world"`),
			expectedBody: "hello world",
		},
		{
			name:         "happy path — template variables substituted",
			config:       []byte(`"Hello {{.Name}}!"`),
			templateData: map[string]string{"Name": "world"},
			expectedBody: "Hello world!",
		},
		{
			name:              "API error propagates",
			config:            []byte(`"hi"`),
			createCommentErr:  errors.New("upstream"),
			expectedBody:      "hi",
			expectErrContains: "error posting comment",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var sentBody string
			fake := &fakeIssuesClient{
				CreateCommentFn: func(
					_ context.Context,
					_, _ string,
					_ int,
					comment *github.IssueComment,
				) (*github.IssueComment, *github.Response, error) {
					sentBody = comment.GetBody()
					return comment, nil, testCase.createCommentErr
				},
			}
			err := commentRunner{}.run(
				t.Context(),
				&actionContext{
					issuesClient: fake,
					owner:        "akuity",
					repo:         "kargo",
					number:       1,
					templateData: testCase.templateData,
				},
				testCase.config,
			)
			if testCase.expectErrContains != "" {
				require.ErrorContains(t, err, testCase.expectErrContains)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, testCase.expectedBody, sentBody)
		})
	}
}

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
