package kargo

import (
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestVerificationStarted_Update(t *testing.T) {
	t.Parallel()
	stage := func(id, run string, phase kargoapi.VerificationPhase) *kargoapi.Stage {
		return &kargoapi.Stage{Status: kargoapi.StageStatus{
			FreightHistory: kargoapi.FreightHistory{{
				ID: "freight",
				VerificationHistory: kargoapi.VerificationInfoStack{{
					ID: id, Phase: phase,
					AnalysisRun: &kargoapi.AnalysisRunReference{Namespace: "project", Name: run},
				}},
			}},
		}}
	}
	pending := stage("first", "run-1", kargoapi.VerificationPhasePending)
	tests := []struct {
		name string
		old  client.Object
		new  client.Object
		want bool
	}{
		{"missing old object", nil, pending, false},
		{"missing new object", pending, nil, false},
		{"non-Stage object", &kargoapi.Warehouse{}, pending, false},
		{"no verification", &kargoapi.Stage{}, &kargoapi.Stage{}, false},
		{"first reference persisted after fast completion event", &kargoapi.Stage{}, pending, true},
		{"unchanged reference does not loop", pending, pending.DeepCopy(), false},
		{"running status does not loop", pending, stage("first", "run-1", kargoapi.VerificationPhaseRunning), false},
		{"terminal status does not loop", pending, stage("first", "run-1", kargoapi.VerificationPhaseFailed), false},
		{"reverification", pending, stage("second", "run-2", kargoapi.VerificationPhasePending), true},
		{"missing run name", &kargoapi.Stage{}, stage("first", "", kargoapi.VerificationPhasePending), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, (VerificationStarted{}).Update(event.UpdateEvent{
				ObjectOld: tt.old, ObjectNew: tt.new,
			}))
		})
	}
}
