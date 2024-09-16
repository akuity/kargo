package directives

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGitCloneDirective_Run(t *testing.T) {
	ctx := context.Background()

	d := newGitCloneDirective()

	t.Run("validations", func(t *testing.T) {
		testCases := []struct {
			name             string
			config           Config
			expectedProblems []string
		}{
			{
				name:   "repoURL not specified",
				config: Config{},
				expectedProblems: []string{
					"(root): repoURL is required",
				},
			},
			{
				name: "repoURL is empty string",
				config: Config{
					"repoURL": "",
				},
				expectedProblems: []string{
					"repoURL: String length must be greater than or equal to 1",
					"repoURL: Does not match format 'uri'",
				},
			},
			{
				name:   "no checkout specified",
				config: Config{},
				expectedProblems: []string{
					"(root): checkout is required",
				},
			},
			{
				name: "checkout is an empty array",
				config: Config{
					"checkout": []Config{},
				},
				expectedProblems: []string{
					"checkout: Array must have at least 1 items",
				},
			},
			{
				name: "checkout path is not specified",
				config: Config{
					"checkout": []Config{{}},
				},
				expectedProblems: []string{
					"checkout.0: path is required",
				},
			},
			{
				name: "checkout path is empty string",
				config: Config{
					"checkout": []Config{{
						"path": "",
					}},
				},
				expectedProblems: []string{
					"checkout.0.path: String length must be greater than or equal to 1",
				},
			},
			{
				name: "neither branch nor fromFreight nor tag specified",
				// This is ok. The behavior should be to clone the default branch.
				config: Config{ // Should be completely valid
					"repoURL": "https://github.com/example/repo.git",
					"checkout": []Config{{
						"path": "/fake/path",
					}},
				},
			},
			{
				name: "branch is empty string, fromFreight is explicitly false, and tag is empty string",
				// This is ok. The behavior should be to clone the default branch.
				config: Config{ // Should be completely valid
					"repoURL": "https://github.com/example/repo.git",
					"checkout": []Config{{
						"branch":      "",
						"fromFreight": false,
						"tag":         "",
						"path":        "/fake/path",
					}},
				},
			},
			{
				name: "just branch is specified",
				config: Config{ // Should be completely valid
					"repoURL": "https://github.com/example/repo.git",
					"checkout": []Config{{
						"branch": "fake-branch",
						"path":   "/fake/path",
					}},
				},
			},
			{
				name: "branch is specified and fromFreight is true",
				// These are meant to be mutually exclusive.
				config: Config{
					"checkout": []Config{{
						"branch":      "fake-branch",
						"fromFreight": true,
					}},
				},
				expectedProblems: []string{
					"checkout.0: Must validate one and only one schema",
				},
			},
			{
				name: "branch and fromOrigin are both specified",
				// These are not meant to be used together.
				config: Config{
					"checkout": []Config{{
						"branch":     "fake-branch",
						"fromOrigin": Config{},
					}},
				},
				expectedProblems: []string{
					"checkout.0: Must validate one and only one schema",
				},
			},
			{
				name: "branch and tag are both specified",
				// These are meant to be mutually exclusive.
				config: Config{
					"checkout": []Config{{
						"branch": "fake-branch",
						"tag":    "fake-tag",
					}},
				},
				expectedProblems: []string{
					"checkout.0: Must validate one and only one schema",
				},
			},
			{
				name: "just fromFreight is true",
				config: Config{ // Should be completely valid
					"repoURL": "https://github.com/example/repo.git",
					"checkout": []Config{{
						"fromFreight": true,
						"path":        "/fake/path",
					}},
				},
			},
			{
				name: "fromFreight is true and fromOrigin is specified",
				config: Config{ // Should be completely valid
					"repoURL": "https://github.com/example/repo.git",
					"checkout": []Config{{
						"fromFreight": true,
						"fromOrigin": Config{
							"kind": "Warehouse",
							"name": "fake-warehouse",
						},
						"path": "/fake/path",
					}},
				},
			},
			{
				name: "fromFreight is true and tag is specified",
				// These are meant to be mutually exclusive.
				config: Config{
					"checkout": []Config{{
						"fromFreight": true,
						"tag":         "fake-tag",
					}},
				},
				expectedProblems: []string{
					"checkout.0: Must validate one and only one schema",
				},
			},
			{
				name: "just fromOrigin is specified",
				// This is not meant to be used without fromFreight=true.
				config: Config{
					"checkout": []Config{{
						"fromOrigin": Config{},
					}},
				},
				expectedProblems: []string{
					"checkout.0: Must validate one and only one schema",
				},
			},
			{
				name: "fromOrigin and tag are both specified",
				// These are not meant to be used together.
				config: Config{
					"checkout": []Config{{
						"fromOrigin": Config{},
						"tag":        "fake-tag",
					}},
				},
				expectedProblems: []string{
					"checkout.0: Must validate one and only one schema",
				},
			},
			{
				name: "just tag is specified",
				config: Config{ // Should be completely valid
					"repoURL": "https://github.com/example/repo.git",
					"checkout": []Config{{
						"tag":  "fake-tag",
						"path": "/fake/path",
					}},
				},
			},
			{
				name: "valid kitchen sink",
				config: Config{
					"repoURL": "https://github.com/example/repo.git",
					"checkout": []Config{
						{
							"path": "/fake/path/0",
						},
						{
							"branch": "fake-branch",
							"path":   "/fake/path/1",
						},
						{
							"fromFreight": true,
							"path":        "/fake/path/2",
						},
						{
							"fromFreight": true,
							"fromOrigin": Config{
								"kind": "Warehouse",
								"name": "fake-warehouse",
							},
							"path": "/fake/path/3",
						},
						{
							"tag":  "fake-tag",
							"path": "/fake/path/4",
						},
					},
				},
			},
		}
		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				stepCtx := &StepContext{
					Config: testCase.config,
				}
				_, err := d.Run(ctx, stepCtx)
				if len(testCase.expectedProblems) == 0 {
					require.NoError(t, err)
				} else {
					for _, problem := range testCase.expectedProblems {
						require.ErrorContains(t, err, problem)
					}
				}
			})
		}
	})
}
