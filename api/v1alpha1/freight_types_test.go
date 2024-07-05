package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
