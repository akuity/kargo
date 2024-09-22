package directives

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/sosedoff/gitkit"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
)

func TestGitCloneDirective_Validate(t *testing.T) {
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

	d := newGitCloneDirective()
	dir, ok := d.(*gitCloneDirective)
	require.True(t, ok)

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := dir.validate(testCase.config)
			if len(testCase.expectedProblems) == 0 {
				require.NoError(t, err)
			} else {
				for _, problem := range testCase.expectedProblems {
					require.ErrorContains(t, err, problem)
				}
			}
		})
	}
}

func TestGitCloneDirective_runPromotionStep(t *testing.T) {
	// Set up a test Git server in-process
	service := gitkit.New(
		gitkit.Config{
			Dir:        t.TempDir(),
			AutoCreate: true,
		},
	)
	require.NoError(t, service.Setup())
	server := httptest.NewServer(service)
	defer server.Close()

	// This is the URL of the "remote" repository
	testRepoURL := fmt.Sprintf("%s/test.git", server.URL)

	// Create some content and push it to the remote repository's default branch
	repo, err := git.Clone(testRepoURL, nil, nil)
	require.NoError(t, err)
	defer repo.Close()
	err = os.WriteFile(filepath.Join(repo.Dir(), "test.txt"), []byte("foo"), 0600)
	require.NoError(t, err)
	err = repo.AddAllAndCommit("Initial commit")
	require.NoError(t, err)
	err = repo.Push(nil)
	require.NoError(t, err)

	// Now we can proceed to test the git-clone directive...

	d := newGitCloneDirective()
	dir, ok := d.(*gitCloneDirective)
	require.True(t, ok)

	stepCtx := &PromotionStepContext{
		CredentialsDB: &credentials.FakeDB{},
		WorkDir:       t.TempDir(),
	}

	res, err := dir.runPromotionStep(
		context.Background(),
		stepCtx,
		GitCloneConfig{
			RepoURL: fmt.Sprintf("%s/test.git", server.URL),
			Checkout: []Checkout{
				{
					// "master" is still the default branch name for a new repository
					// unless you configure it otherwise.
					Branch: "master",
					Path:   "master",
				},
				{
					Branch: "stage/dev",
					Path:   "dev",
				},
			},
		},
	)
	require.NoError(t, err)
	require.Equal(t, PromotionStatusSuccess, res.Status)
	require.DirExists(t, filepath.Join(stepCtx.WorkDir, "master"))
	// The checked out master branch should have the content we know is in the
	// test remote's master branch.
	require.FileExists(t, filepath.Join(stepCtx.WorkDir, "master", "test.txt"))
	require.DirExists(t, filepath.Join(stepCtx.WorkDir, "dev"))
	// The stage/dev branch is a new orphan branch with a single empty commit.
	// It should lack any content.
	dirEntries, err := os.ReadDir(filepath.Join(stepCtx.WorkDir, "dev"))
	require.NoError(t, err)
	require.Len(t, dirEntries, 1) // Just the .git file
	require.FileExists(t, filepath.Join(stepCtx.WorkDir, "dev", ".git"))
}
