package promotions

import (
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/event"

	argocd "github.com/akuity/kargo/pkg/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

func TestArgoCDAppOperationCompleted_Update(t *testing.T) {
	testCases := []struct {
		name string
		e    event.TypedUpdateEvent[*argocd.Application]
		want bool
	}{
		{
			name: "ObjectOld is nil",
			e: event.TypedUpdateEvent[*argocd.Application]{
				ObjectNew: &argocd.Application{},
			},
			want: false,
		},
		{
			name: "ObjectNew is nil",
			e: event.TypedUpdateEvent[*argocd.Application]{
				ObjectOld: &argocd.Application{},
			},
			want: false,
		},
		{
			name: "No operation state",
			e: event.TypedUpdateEvent[*argocd.Application]{
				ObjectOld: &argocd.Application{},
				ObjectNew: &argocd.Application{},
			},
			want: false,
		},
		{
			name: "Operation completed",
			e: event.TypedUpdateEvent[*argocd.Application]{
				ObjectOld: &argocd.Application{},
				ObjectNew: &argocd.Application{
					Status: argocd.ApplicationStatus{
						OperationState: &argocd.OperationState{
							Phase: argocd.OperationSucceeded,
						},
					},
				},
			},
			want: true,
		},
		{
			name: "Completed operation unchanged",
			e: event.TypedUpdateEvent[*argocd.Application]{
				ObjectOld: &argocd.Application{
					Status: argocd.ApplicationStatus{
						OperationState: &argocd.OperationState{
							Phase: argocd.OperationSucceeded,
						},
					},
				},
				ObjectNew: &argocd.Application{
					Status: argocd.ApplicationStatus{
						OperationState: &argocd.OperationState{
							Phase: argocd.OperationSucceeded,
						},
					},
				},
			},
			want: false,
		},
		{
			name: "Operation running",
			e: event.TypedUpdateEvent[*argocd.Application]{
				ObjectOld: &argocd.Application{},
				ObjectNew: &argocd.Application{
					Status: argocd.ApplicationStatus{
						OperationState: &argocd.OperationState{
							Phase: argocd.OperationRunning,
						},
					},
				},
			},
			want: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			p := ArgoCDAppOperationCompleted[*argocd.Application]{
				logger: logging.NewDiscardLoggerOrDie(),
			}

			require.Equal(t, testCase.want, p.Update(testCase.e))
		})
	}
}
