package governance

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/go-github/v76/github"
	"github.com/stretchr/testify/require"
)

func Test_removeLabelsRunner_run(t *testing.T) {
	// notFoundErr mimics what go-github returns when a label isn't on the
	// resource — the runner is expected to swallow these.
	notFoundErr := &github.ErrorResponse{
		Response: &http.Response{StatusCode: http.StatusNotFound},
	}
	otherErr := &github.ErrorResponse{
		Response: &http.Response{StatusCode: http.StatusInternalServerError},
		Message:  "boom",
	}

	testCases := []struct {
		name   string
		config []byte
		// errByLabel chooses what the fake returns for each label, in
		// order of the call. nil means success.
		errByLabel        []error
		expectedAttempts  []string
		expectErrContains string
	}{
		{
			name:              "decode error",
			config:            []byte(`true`),
			expectErrContains: "decoding removeLabels config",
		},
		{
			name:             "empty list — no API calls",
			config:           []byte(`[]`),
			expectedAttempts: nil,
		},
		{
			name:             "happy path — all labels removed",
			config:           []byte("- foo\n- bar\n"),
			errByLabel:       []error{nil, nil},
			expectedAttempts: []string{"foo", "bar"},
		},
		{
			name:             "404 on one label — others continue, no error",
			config:           []byte("- foo\n- bar\n- baz\n"),
			errByLabel:       []error{nil, notFoundErr, nil},
			expectedAttempts: []string{"foo", "bar", "baz"},
		},
		{
			name:              "non-404 error halts and propagates with label name",
			config:            []byte("- foo\n- bar\n- baz\n"),
			errByLabel:        []error{nil, otherErr, nil},
			expectedAttempts:  []string{"foo", "bar"},
			expectErrContains: `error removing label "bar"`,
		},
		{
			name:              "non-github error halts and propagates",
			config:            []byte("- foo\n"),
			errByLabel:        []error{errors.New("network")},
			expectedAttempts:  []string{"foo"},
			expectErrContains: `error removing label "foo": network`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var attempts []string
			fake := &fakeIssuesClient{
				RemoveLabelForIssueFn: func(
					_ context.Context,
					_, _ string,
					_ int,
					label string,
				) (*github.Response, error) {
					i := len(attempts)
					attempts = append(attempts, label)
					if i < len(testCase.errByLabel) {
						return nil, testCase.errByLabel[i]
					}
					return nil, nil
				},
			}
			err := removeLabelsRunner{}.run(
				t.Context(),
				&actionContext{
					issuesClient: fake,
					owner:        "akuity",
					repo:         "kargo",
					number:       1,
				},
				testCase.config,
			)
			if testCase.expectErrContains != "" {
				require.ErrorContains(t, err, testCase.expectErrContains)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, testCase.expectedAttempts, attempts)
		})
	}
}
