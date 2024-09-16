package directives

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGitCommitDirective_Run(t *testing.T) {
	ctx := context.Background()

	d := newGitCommitDirective()

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
				name: "author email is not specified",
				config: Config{ // Should be completely valid
					"author": Config{},
					"path":   "/tmp/foo",
				},
			},
			{
				name: "author email is empty string",
				config: Config{ // Should be completely valid
					"author": Config{
						"email": "",
					},
					"path": "/tmp/foo",
				},
			},
			{
				name: "author name is not specified",
				config: Config{ // Should be completely valid
					"author": Config{},
					"path":   "/tmp/foo",
				},
			},
			{
				name: "author name is empty string",
				config: Config{ // Should be completely valid
					"author": Config{
						"name": "",
					},
					"path": "/tmp/foo",
				},
			},
			{
				name: "valid kitchen sink",
				config: Config{
					"author": Config{
						"email": "tony@starkindustries.com",
						"name":  "Tony Stark",
					},
					"path": "/tmp/foo",
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
