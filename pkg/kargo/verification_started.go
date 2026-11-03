package kargo

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// VerificationStarted requeues a Stage once its new AnalysisRun reference is
// visible in the Stage cache. A fast AnalysisRun can finish before that reference
// is persisted, when the AnalysisRun-to-Stage index cannot yet find its Stage.
type VerificationStarted struct {
	predicate.Funcs
}

func (VerificationStarted) Update(e event.UpdateEvent) bool {
	oldStage, oldOK := e.ObjectOld.(*kargoapi.Stage)
	newStage, newOK := e.ObjectNew.(*kargoapi.Stage)
	if !oldOK || !newOK || oldStage == nil || newStage == nil {
		return false
	}
	newFreight := newStage.Status.FreightHistory.Current()
	if newFreight == nil {
		return false
	}
	newVI := newFreight.VerificationHistory.Current()
	if newVI == nil || newVI.Phase.IsTerminal() || newVI.AnalysisRun == nil || newVI.AnalysisRun.Name == "" {
		return false
	}
	oldFreight := oldStage.Status.FreightHistory.Current()
	if oldFreight == nil {
		return true
	}
	oldVI := oldFreight.VerificationHistory.Current()
	return oldVI == nil || oldVI.ID != newVI.ID || oldVI.AnalysisRun == nil ||
		oldVI.AnalysisRun.Name != newVI.AnalysisRun.Name ||
		oldVI.AnalysisRun.Namespace != newVI.AnalysisRun.Namespace
}
