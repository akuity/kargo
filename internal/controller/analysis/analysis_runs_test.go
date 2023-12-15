package analysis

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestAnalysisRunPhaseChangePredicate(t *testing.T) {
	testCases := []struct {
		name    string
		old     map[string]any
		new     map[string]any
		updated bool
	}{
		{
			name: "phase changed",
			old: map[string]any{
				"status": map[string]any{
					"phase": "old-phase",
				},
			},
			new: map[string]any{
				"status": map[string]any{
					"phase": "new-phase",
				},
			},
			updated: true,
		},
		{
			name: "phase did not change",
			old: map[string]any{
				"status": map[string]any{
					"phase": "old-phase",
				},
			},
			new: map[string]any{
				"status": map[string]any{
					"phase": "old-phase",
				},
			},
			updated: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			p := analysisRunPhaseChangePredicate{}
			newUn := &unstructured.Unstructured{Object: testCase.new}
			oldUn := &unstructured.Unstructured{Object: testCase.old}
			updated := p.Update(event.UpdateEvent{
				ObjectNew: newUn,
				ObjectOld: oldUn,
			})
			require.Equal(t, testCase.updated, updated)
		})
	}
}
