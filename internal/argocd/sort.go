package argocd

import (
	"slices"
	"sort"

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

func (a ByOperationPhase) Len() int { return len(a) }

func (a ByOperationPhase) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func (a ByOperationPhase) Less(i, j int) bool {
	orderI, existsI := operationPhaseOrder[a[i]]
	orderJ, existsJ := operationPhaseOrder[a[j]]

	// If both elements exist in the order map, compare their order values.
	if existsI && existsJ {
		return orderI < orderJ
	}

	// If neither element exists, sort them lexicographically.
	if !existsI && !existsJ {
		return a[i] < a[j]
	}

	// If only one element exists, prioritize the existing element.
	return existsI
}

func (a ByOperationPhase) Sort() { 
	slices.Sort(a)
}
