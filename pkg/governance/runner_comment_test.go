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
		name             string
		config           []byte
		templateData     map[string]string
		createCommentErr error
		assert           func(t *testing.T, sentBody string, err error)
	}{
		{
			name:   "decode error — config is not a string",
			config: []byte(`[1, 2]`),
			assert: func(t *testing.T, sentBody string, err error) {
				require.ErrorContains(t, err, "decoding comment config")
				require.Empty(t, sentBody)
			},
		},
		{
			name:   "empty template — no API call",
			config: []byte(`""`),
			assert: func(t *testing.T, sentBody string, err error) {
				require.NoError(t, err)
				require.Empty(t, sentBody)
			},
		},
		{
			name:         "template parse error",
			config:       []byte(`"hello {{ .Bad"`),
			templateData: map[string]string{"Foo": "bar"},
			assert: func(t *testing.T, sentBody string, err error) {
				require.ErrorContains(t, err, "rendering comment template")
				require.Empty(t, sentBody)
			},
		},
		{
			name:   "happy path — no template data passes through verbatim",
			config: []byte(`"hello world"`),
			assert: func(t *testing.T, sentBody string, err error) {
				require.NoError(t, err)
				require.Equal(t, "hello world", sentBody)
			},
		},
		{
			name:         "happy path — template variables substituted",
			config:       []byte(`"Hello {{.Name}}!"`),
			templateData: map[string]string{"Name": "world"},
			assert: func(t *testing.T, sentBody string, err error) {
				require.NoError(t, err)
				require.Equal(t, "Hello world!", sentBody)
			},
		},
		{
			name:             "API error propagates",
			config:           []byte(`"hi"`),
			createCommentErr: errors.New("upstream"),
			assert: func(t *testing.T, sentBody string, err error) {
				require.ErrorContains(t, err, "error posting comment")
				require.Equal(t, "hi", sentBody)
			},
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
					repoContext: repoContext{
						issuesClient: fake,
						owner:        "akuity",
						repo:         "kargo",
					},
					number:       1,
					templateData: testCase.templateData,
				},
				testCase.config,
			)
			testCase.assert(t, sentBody, err)
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
