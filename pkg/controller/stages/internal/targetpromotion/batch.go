package targetpromotion

import (
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// Batch is the collection of target-specific Promotions created for one
// target-aware Stage promotion request.
type Batch struct {
	id         string
	promotions []*kargoapi.Promotion
}

// State summarizes the state of every Promotion in a Batch.
type State string

const (
	// StateActive indicates that at least one child Promotion is not terminal.
	StateActive State = "Active"
	// StateFailed indicates that every child is terminal and at least one did
	// not succeed.
	StateFailed State = "Failed"
	// StateSucceeded indicates that every child Promotion succeeded.
	StateSucceeded State = "Succeeded"
)

// Batches groups target-specific Promotions by their batch label.
func Batches(promotions []kargoapi.Promotion) []Batch {
	batchesByID := map[string][]*kargoapi.Promotion{}
	for i := range promotions {
		promotion := &promotions[i]
		if promotion.Spec.Target == "" {
			continue
		}
		if batchID := promotion.Labels[kargoapi.LabelKeyPromotionBatch]; batchID != "" {
			batchesByID[batchID] = append(batchesByID[batchID], promotion)
		}
	}

	batches := make([]Batch, 0, len(batchesByID))
	for batchID, promotions := range batchesByID {
		batches = append(batches, Batch{id: batchID, promotions: promotions})
	}
	return batches
}

// ID returns the label identifying the Batch.
func (b Batch) ID() string {
	return b.id
}

// MostRecent returns the Batch containing the most recently created Promotion.
func MostRecent(batches []Batch) *Batch {
	var mostRecent *Batch
	for i := range batches {
		if mostRecent == nil || batches[i].latestPromotion().CreationTimestamp.After(
			mostRecent.latestPromotion().CreationTimestamp.Time,
		) {
			mostRecent = &batches[i]
		}
	}
	return mostRecent
}

// State returns the aggregate state of the Batch.
func (b Batch) State() State {
	for _, promotion := range b.promotions {
		if !promotion.Status.Phase.IsTerminal() {
			return StateActive
		}
		if promotion.Status.Phase != kargoapi.PromotionPhaseSucceeded {
			return StateFailed
		}
	}
	return StateSucceeded
}

// FreightCollection returns the collection produced by a child Promotion.
// All child Promotions in a Batch promote the same Freight.
func (b Batch) FreightCollection() *kargoapi.FreightCollection {
	return b.promotions[0].Status.FreightCollection
}

// FreightName returns the Freight promoted by the Batch.
func (b Batch) FreightName() string {
	return b.promotions[0].Spec.Freight
}

func (b Batch) latestPromotion() *kargoapi.Promotion {
	latest := b.promotions[0]
	for _, promotion := range b.promotions[1:] {
		if promotion.CreationTimestamp.After(latest.CreationTimestamp.Time) {
			latest = promotion
		}
	}
	return latest
}
