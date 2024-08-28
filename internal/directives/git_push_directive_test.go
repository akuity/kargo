package directives

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGitPushDirective_Run(t *testing.T) {
	ctx := context.Background()

	d := newGitPushDirective()

	t.Run("validations", func(t *testing.T) {
		testCases := []struct {
			name             string
			config           Config
			expectedProblems []string
		}{
			{
				name:   "path not specified",
				config: Config{},
				expectedProblems: []string{
					"(root): path is required",
				},
			},
			{
				name: "path is empty string",
				config: Config{
					"path": "",
				},
				expectedProblems: []string{
					"path: String length must be greater than or equal to 1",
				},
			},
			{
				name: "just generateTargetBranch is true",
				config: Config{ // Should be completely valid
					"generateTargetBranch": true,
					"path":                 "/fake/path",
				},
			},
			{
				name: "generateTargetBranch is true and targetBranch is empty string",
				config: Config{ // Should be completely valid
					"generateTargetBranch": true,
					"path":                 "/fake/path",
					"targetBranch":         "",
				},
			},
			{
				name: "generateTargetBranch is true and targetBranch is specified",
				// These are meant to be mutually exclusive.
				config: Config{
					"generateTargetBranch": true,
					"targetBranch":         "fake-branch",
				},
				expectedProblems: []string{
					"(root): Must validate one and only one schema",
				},
			},
			{
				name:   "generateTargetBranch not specified and targetBranch not specified",
				config: Config{},
				expectedProblems: []string{
					"(root): Must validate one and only one schema",
				},
			},
			{
				name: "generateTargetBranch not specified and targetBranch is empty string",
				config: Config{
					"targetBranch": "",
				},
				expectedProblems: []string{
					"(root): Must validate one and only one schema",
				},
			},
			{
				name: "generateTargetBranch not specified and targetBranch is specified",
				config: Config{ // Should be completely valid
					"path":         "/fake/path",
					"targetBranch": "fake-branch",
				},
			},
			{
				name: "just generateTargetBranch is false",
				config: Config{
					"generateTargetBranch": false,
				},
				expectedProblems: []string{
					"(root): Must validate one and only one schema",
				},
			},
			{
				name: "generateTargetBranch is false and targetBranch is empty string",
				config: Config{
					"generateTargetBranch": false,
					"targetBranch":         "",
				},
				expectedProblems: []string{
					"(root): Must validate one and only one schema",
				},
			},
			{
				name: "generateTargetBranch is false and targetBranch is specified",
				config: Config{ // Should be completely valid
					"path":         "/fake/path",
					"targetBranch": "fake-branch",
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
