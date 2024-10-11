package argocd

import (
	"testing"

	"github.com/stretchr/testify/require"

	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
)

func TestByOperationPhaseSort(t *testing.T) {
	testCases := []struct {
		name     string
		input    []argocd.OperationPhase
		expected []argocd.OperationPhase
	}{
		{
			name:     "Empty slice",
			input:    []argocd.OperationPhase{},
			expected: []argocd.OperationPhase{},
		},
		{
			name:     "Single element",
			input:    []argocd.OperationPhase{argocd.OperationRunning},
			expected: []argocd.OperationPhase{argocd.OperationRunning},
		},
		{
			name: "Sorted slice",
			input: []argocd.OperationPhase{
				argocd.OperationFailed,
				argocd.OperationError,
				argocd.OperationRunning,
				argocd.OperationTerminating,
				argocd.OperationSucceeded,
			},
			expected: []argocd.OperationPhase{
				argocd.OperationFailed,
				argocd.OperationError,
				argocd.OperationRunning,
				argocd.OperationTerminating,
				argocd.OperationSucceeded,
			},
		},
		{
			name: "Unsorted slice",
			input: []argocd.OperationPhase{
				argocd.OperationRunning,
				argocd.OperationFailed,
				argocd.OperationSucceeded,
				argocd.OperationError,
				argocd.OperationTerminating,
			},
			expected: []argocd.OperationPhase{
				argocd.OperationFailed,
				argocd.OperationError,
				argocd.OperationRunning,
				argocd.OperationTerminating,
				argocd.OperationSucceeded,
			},
		},
		{
			name: "Duplicate elements",
			input: []argocd.OperationPhase{
				argocd.OperationRunning,
				argocd.OperationFailed,
				argocd.OperationRunning,
				argocd.OperationSucceeded,
			},
			expected: []argocd.OperationPhase{
				argocd.OperationFailed,
				argocd.OperationRunning,
				argocd.OperationRunning,
				argocd.OperationSucceeded,
			},
		},
		{
			name: "Unrecognized elements",
			input: []argocd.OperationPhase{
				argocd.OperationSucceeded,
				"Foo",
				"Bar",
				argocd.OperationRunning,
			},
			expected: []argocd.OperationPhase{
				argocd.OperationRunning,
				argocd.OperationSucceeded,
				"Bar",
				"Foo",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sorted := ByOperationPhase(tc.input)
			sorted.Sort()
			require.Equal(t, ByOperationPhase(tc.expected), sorted)
		})
	}
}
