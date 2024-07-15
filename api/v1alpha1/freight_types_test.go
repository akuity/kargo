package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGitCommitDeepEquals(t *testing.T) {
	testCases := []struct {
		name           string
		a              *GitCommit
		b              *GitCommit
		expectedResult bool
	}{
		{
			name:           "a and b both nil",
			expectedResult: true,
		},
		{
			name:           "only a is nil",
			b:              &GitCommit{},
			expectedResult: false,
		},
		{
			name:           "only b is nil",
			a:              &GitCommit{},
			expectedResult: false,
		},
		{
			name: "repoURLs differ",
			a: &GitCommit{
				RepoURL: "foo",
			},
			b: &GitCommit{
				RepoURL: "bar",
			},
			expectedResult: false,
		},
		{
			name: "commit IDs differ",
			a: &GitCommit{
				RepoURL: "fake-url",
				ID:      "foo",
			},
			b: &GitCommit{
				RepoURL: "fake-url",
				ID:      "bar",
			},
			expectedResult: false,
		},
		{
			name: "branch names differ",
			a: &GitCommit{
				RepoURL: "fake-url",
				ID:      "fake-commit-id",
				Branch:  "foo",
			},
			b: &GitCommit{
				RepoURL: "fake-url",
				ID:      "fake-commit-id",
				Branch:  "bar",
			},
			expectedResult: false,
		},
		{
			name: "tags differ",
			a: &GitCommit{
				RepoURL: "fake-url",
				ID:      "fake-commit-id",
				Tag:     "foo",
			},
			b: &GitCommit{
				RepoURL: "fake-url",
				ID:      "fake-commit-id",
				Tag:     "bar",
			},
			expectedResult: false,
		},
		{
			name: "health check commits differ",
			a: &GitCommit{
				RepoURL:           "fake-url",
				ID:                "fake-commit-id",
				HealthCheckCommit: "foo",
			},
			b: &GitCommit{
				RepoURL:           "fake-url",
				ID:                "fake-commit-id",
				HealthCheckCommit: "bar",
			},
			expectedResult: false,
		},
		{
			name: "messages differ",
			a: &GitCommit{
				RepoURL: "fake-url",
				ID:      "fake-commit-id",
				Message: "foo",
			},
			b: &GitCommit{
				RepoURL: "fake-url",
				ID:      "fake-commit-id",
				Message: "bar",
			},
			expectedResult: false,
		},
		{
			name: "authors differ",
			a: &GitCommit{
				RepoURL: "fake-url",
				ID:      "fake-commit-id",
				Author:  "foo",
			},
			b: &GitCommit{
				RepoURL: "fake-url",
				ID:      "fake-commit-id",
				Author:  "bar",
			},
			expectedResult: false,
		},
		{
			name: "committers differ",
			a: &GitCommit{
				RepoURL:   "fake-url",
				ID:        "fake-commit-id",
				Committer: "foo",
			},
			b: &GitCommit{
				RepoURL:   "fake-url",
				ID:        "fake-commit-id",
				Committer: "bar",
			},
			expectedResult: false,
		},
		{
			name: "perfect match",
			a: &GitCommit{
				RepoURL:           "fake-url",
				ID:                "fake-commit-id",
				Branch:            "fake-branch",
				Tag:               "fake-tag",
				HealthCheckCommit: "fake-health-id",
				Message:           "fake-message",
				Author:            "fake-author",
				Committer:         "fake-committer",
			},
			b: &GitCommit{
				RepoURL:           "fake-url",
				ID:                "fake-commit-id",
				Branch:            "fake-branch",
				Tag:               "fake-tag",
				HealthCheckCommit: "fake-health-id",
				Message:           "fake-message",
				Author:            "fake-author",
				Committer:         "fake-committer",
			},
			expectedResult: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.expectedResult, testCase.a.DeepEquals(testCase.b))
			require.Equal(t, testCase.expectedResult, testCase.b.DeepEquals(testCase.a))
		})
	}
}

func TestGitCommitEquals(t *testing.T) {
	testCases := []struct {
		name           string
		lhs            *GitCommit
		rhs            *GitCommit
		expectedResult bool
	}{
		{
			name:           "lhs and rhs both nil",
			expectedResult: true,
		},
		{
			name:           "only lhs is nil",
			rhs:            &GitCommit{},
			expectedResult: false,
		},
		{
			name:           "only rhs is nil",
			lhs:            &GitCommit{},
			expectedResult: false,
		},
		{
			name: "repoUrls differ",
			lhs: &GitCommit{
				RepoURL: "foo",
				ID:      "fake-commit-id",
			},
			rhs: &GitCommit{
				RepoURL: "bar",
				ID:      "fake-commit-id",
			},
			expectedResult: false,
		},
		{
			name: "commit IDs differ",
			lhs: &GitCommit{
				RepoURL: "fake-url",
				ID:      "foo",
			},
			rhs: &GitCommit{
				RepoURL: "fake-url",
				ID:      "bar",
			},
			expectedResult: false,
		},
		{
			name: "perfect match",
			lhs: &GitCommit{
				RepoURL: "fake-url",
				ID:      "fake-commit-id",
			},
			rhs: &GitCommit{
				RepoURL: "fake-url",
				ID:      "fake-commit-id",
			},
			expectedResult: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.expectedResult,
				testCase.lhs.Equals(testCase.rhs),
			)
		})
	}
}

func TestFreightGenerateID(t *testing.T) {
	freight := Freight{
		Origin: FreightOrigin{
			Kind: "fake-kind",
			Name: "fake-name",
		},
		Commits: []GitCommit{
			{
				RepoURL: "fake-git-repo",
				ID:      "fake-commit-id",
			},
		},
		Images: []Image{
			{
				RepoURL: "fake-image-repo",
				Tag:     "fake-image-tag",
			},
		},
		Charts: []Chart{
			{
				RepoURL: "fake-chart-repo",
				Name:    "fake-chart",
				Version: "fake-chart-version",
			},
		},
	}
	id := freight.GenerateID()
	expected := id
	// Doing this any number of times should yield the same ID
	for i := 0; i < 100; i++ {
		require.Equal(t, expected, freight.GenerateID())
	}
	// Changing anything should change the result
	freight.Commits[0].ID = "a-different-fake-commit"
	require.NotEqual(t, expected, freight.GenerateID())
}
