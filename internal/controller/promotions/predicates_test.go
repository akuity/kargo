package promotions

import (
	"io"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/event"

	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
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
			logger := logrus.New()
			logger.Out = io.Discard

			p := ArgoCDAppOperationCompleted[*argocd.Application]{
				logger: logger,
			}

			require.Equal(t, testCase.want, p.Update(testCase.e))
		})
	}
}
