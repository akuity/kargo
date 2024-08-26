package argocd

import (
	"slices"
	"strings"

	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
)

// operationPhaseOrder is a map of operation phases to their order of
// precedence. The order can be used to determine the most important
// operation phase in a list of operation phases.
// The lower the number, the more important the operation phase.
var operationPhaseOrder = map[argocd.OperationPhase]uint{
	argocd.OperationFailed:      0,
	argocd.OperationError:       1,
	argocd.OperationRunning:     2,
	argocd.OperationTerminating: 3,
	argocd.OperationSucceeded:   4,
}

type ByOperationPhase []argocd.OperationPhase

func (a ByOperationPhase) Sort() {
	slices.SortFunc(a, func(lhs, rhs argocd.OperationPhase) int {
		orderLhs, existsLhs := operationPhaseOrder[lhs]
		orderRhs, existsRhs := operationPhaseOrder[rhs]

		// If both elements exist in the order map, compare their order values.
		if existsLhs && existsRhs {
			// The max value is 4, so we can safely cast to int without worrying about
			// overflow.
			return int(orderLhs) - int(orderRhs) // nolint: gosec
		}

		// If neither element exists, sort them lexicographically.
		if !existsLhs && !existsRhs {
			return strings.Compare(string(lhs), string(rhs))
		}

		// If only one element exists, prioritize the existing element.
		if existsLhs {
			return -1
		}
		return 1
	})
}
