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

func TestStageFreightUpdateID(t *testing.T) {
	freight := Freight{
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
				RegistryURL: "fake-chart-registry",
				Name:        "fake-chart",
				Version:     "fake-chart-version",
			},
		},
	}
	freight.UpdateFreightID()
	result := freight.ID
	// Doing this any number of times should yield the same ID
	for i := 0; i < 100; i++ {
		freight.UpdateFreightID()
		require.Equal(t, result, freight.ID)
	}
	// Changing anything should change the result
	freight.Commits[0].ID = "a-different-fake-commit"
	freight.UpdateFreightID()
	require.NotEqual(t, result, freight.ID)
}

func TestStageFreightStackEmpty(t *testing.T) {
	testCases := []struct {
		name           string
		stack          FreightStack
		expectedResult bool
	}{
		{
			name:           "stack is nil",
			stack:          nil,
			expectedResult: true,
		},
		{
			name:           "stack is empty",
			stack:          FreightStack{},
			expectedResult: true,
		},
		{
			name:           "stack has items",
			stack:          FreightStack{{ID: "foo"}},
			expectedResult: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.expectedResult, testCase.stack.Empty())
		})
	}
}

func TestStageFreightStackPop(t *testing.T) {
	testCases := []struct {
		name            string
		stack           FreightStack
		expectedStack   FreightStack
		expectedFreight Freight
		expectedOK      bool
	}{
		{
			name:            "stack is nil",
			stack:           nil,
			expectedStack:   nil,
			expectedFreight: Freight{},
			expectedOK:      false,
		},
		{
			name:            "stack is empty",
			stack:           FreightStack{},
			expectedStack:   FreightStack{},
			expectedFreight: Freight{},
			expectedOK:      false,
		},
		{
			name:            "stack has items",
			stack:           FreightStack{{ID: "foo"}, {ID: "bar"}},
			expectedStack:   FreightStack{{ID: "bar"}},
			expectedFreight: Freight{ID: "foo"},
			expectedOK:      true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			freight, ok := testCase.stack.Pop()
			require.Equal(t, testCase.expectedStack, testCase.stack)
			require.Equal(t, testCase.expectedFreight, freight)
			require.Equal(t, testCase.expectedOK, ok)
		})
	}
}

func TestStageFreightStackTop(t *testing.T) {
	testCases := []struct {
		name            string
		stack           FreightStack
		expectedFreight Freight
		expectedOK      bool
	}{
		{
			name:            "stack is nil",
			stack:           nil,
			expectedFreight: Freight{},
			expectedOK:      false,
		},
		{
			name:            "stack is empty",
			stack:           FreightStack{},
			expectedFreight: Freight{},
			expectedOK:      false,
		},
		{
			name:            "stack has items",
			stack:           FreightStack{{ID: "foo"}, {ID: "bar"}},
			expectedFreight: Freight{ID: "foo"},
			expectedOK:      true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			initialLen := len(testCase.stack)
			freight, ok := testCase.stack.Top()
			require.Len(t, testCase.stack, initialLen)
			require.Equal(t, testCase.expectedFreight, freight)
			require.Equal(t, testCase.expectedOK, ok)
		})
	}
}

func TestStageFreightStackPush(t *testing.T) {
	testCases := []struct {
		name          string
		stack         FreightStack
		newFreight    []Freight
		expectedStack FreightStack
	}{
		{
			name:          "initial stack is nil",
			stack:         nil,
			newFreight:    []Freight{{ID: "foo"}, {ID: "bar"}},
			expectedStack: FreightStack{{ID: "foo"}, {ID: "bar"}},
		},
		{
			name:          "initial stack is not nil",
			stack:         FreightStack{{ID: "foo"}},
			newFreight:    []Freight{{ID: "bar"}},
			expectedStack: FreightStack{{ID: "bar"}, {ID: "foo"}},
		},
		{
			name: "initial stack is full",
			stack: FreightStack{
				{}, {}, {}, {}, {}, {}, {}, {}, {}, {},
			},
			newFreight: []Freight{{ID: "foo"}},
			expectedStack: FreightStack{
				{ID: "foo"}, {}, {}, {}, {}, {}, {}, {}, {}, {},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.stack.Push(testCase.newFreight...)
			require.Equal(t, testCase.expectedStack, testCase.stack)
		})
	}
}
