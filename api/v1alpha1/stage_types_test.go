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

func TestStageStateUpdateID(t *testing.T) {
	state := StageState{
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
	state.UpdateStateID()
	result := state.ID
	// Doing this any number of times should yield the same ID
	for i := 0; i < 100; i++ {
		state.UpdateStateID()
		require.Equal(t, result, state.ID)
	}
	// Changing anything should change the result
	state.Commits[0].ID = "a-different-fake-commit"
	state.UpdateStateID()
	require.NotEqual(t, result, state.ID)
}

func TestStageStateStackEmpty(t *testing.T) {
	testCases := []struct {
		name           string
		stack          StageStateStack
		expectedResult bool
	}{
		{
			name:           "stack is nil",
			stack:          nil,
			expectedResult: true,
		},
		{
			name:           "stack is empty",
			stack:          StageStateStack{},
			expectedResult: true,
		},
		{
			name:           "stack has items",
			stack:          StageStateStack{{ID: "foo"}},
			expectedResult: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.expectedResult, testCase.stack.Empty())
		})
	}
}

func TestStageStateStackPop(t *testing.T) {
	testCases := []struct {
		name          string
		stack         StageStateStack
		expectedStack StageStateStack
		expectedState StageState
		expectedOK    bool
	}{
		{
			name:          "stack is nil",
			stack:         nil,
			expectedStack: nil,
			expectedState: StageState{},
			expectedOK:    false,
		},
		{
			name:          "stack is empty",
			stack:         StageStateStack{},
			expectedStack: StageStateStack{},
			expectedState: StageState{},
			expectedOK:    false,
		},
		{
			name:          "stack has items",
			stack:         StageStateStack{{ID: "foo"}, {ID: "bar"}},
			expectedStack: StageStateStack{{ID: "bar"}},
			expectedState: StageState{ID: "foo"},
			expectedOK:    true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			state, ok := testCase.stack.Pop()
			require.Equal(t, testCase.expectedStack, testCase.stack)
			require.Equal(t, testCase.expectedState, state)
			require.Equal(t, testCase.expectedOK, ok)
		})
	}
}

func TestStageStateStackTop(t *testing.T) {
	testCases := []struct {
		name          string
		stack         StageStateStack
		expectedState StageState
		expectedOK    bool
	}{
		{
			name:          "stack is nil",
			stack:         nil,
			expectedState: StageState{},
			expectedOK:    false,
		},
		{
			name:          "stack is empty",
			stack:         StageStateStack{},
			expectedState: StageState{},
			expectedOK:    false,
		},
		{
			name:          "stack has items",
			stack:         StageStateStack{{ID: "foo"}, {ID: "bar"}},
			expectedState: StageState{ID: "foo"},
			expectedOK:    true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			initialLen := len(testCase.stack)
			state, ok := testCase.stack.Top()
			require.Len(t, testCase.stack, initialLen)
			require.Equal(t, testCase.expectedState, state)
			require.Equal(t, testCase.expectedOK, ok)
		})
	}
}

func TestStageStateStackPush(t *testing.T) {
	testCases := []struct {
		name          string
		stack         StageStateStack
		newStates     []StageState
		expectedStack StageStateStack
	}{
		{
			name:          "initial stack is nil",
			stack:         nil,
			newStates:     []StageState{{ID: "foo"}, {ID: "bar"}},
			expectedStack: StageStateStack{{ID: "foo"}, {ID: "bar"}},
		},
		{
			name:          "initial stack is not nil",
			stack:         StageStateStack{{ID: "foo"}},
			newStates:     []StageState{{ID: "bar"}},
			expectedStack: StageStateStack{{ID: "bar"}, {ID: "foo"}},
		},
		{
			name: "initial stack is full",
			stack: StageStateStack{
				{}, {}, {}, {}, {}, {}, {}, {}, {}, {},
			},
			newStates: []StageState{{ID: "foo"}},
			expectedStack: StageStateStack{
				{ID: "foo"}, {}, {}, {}, {}, {}, {}, {}, {}, {},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.stack.Push(testCase.newStates...)
			require.Equal(t, testCase.expectedStack, testCase.stack)
		})
	}
}
