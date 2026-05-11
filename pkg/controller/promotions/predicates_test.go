package promotions

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestArgoCDAppHealthChanged_Update(t *testing.T) {
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
			name: "Health unchanged",
			e: event.TypedUpdateEvent[*argocd.Application]{
				ObjectOld: &argocd.Application{
					Status: argocd.ApplicationStatus{
						Health: argocd.HealthStatus{
							Status: argocd.HealthStatusHealthy,
						},
					},
				},
				ObjectNew: &argocd.Application{
					Status: argocd.ApplicationStatus{
						Health: argocd.HealthStatus{
							Status: argocd.HealthStatusHealthy,
						},
					},
				},
			},
			want: false,
		},
		{
			name: "Health changed",
			e: event.TypedUpdateEvent[*argocd.Application]{
				ObjectOld: &argocd.Application{
					Status: argocd.ApplicationStatus{
						Health: argocd.HealthStatus{
							Status: argocd.HealthStatusProgressing,
						},
					},
				},
				ObjectNew: &argocd.Application{
					Status: argocd.ApplicationStatus{
						Health: argocd.HealthStatus{
							Status: argocd.HealthStatusHealthy,
						},
					},
				},
			},
			want: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			p := ArgoCDAppHealthChanged[*argocd.Application]{
				logger: logging.NewDiscardLoggerOrDie(),
			}
			require.Equal(t, testCase.want, p.Update(testCase.e))
		})
	}
}

func TestArgoCDAppSyncChanged_Update(t *testing.T) {
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
			name: "Sync unchanged",
			e: event.TypedUpdateEvent[*argocd.Application]{
				ObjectOld: &argocd.Application{
					Status: argocd.ApplicationStatus{
						Sync: argocd.SyncStatus{
							Status: argocd.SyncStatusCodeSynced,
						},
					},
				},
				ObjectNew: &argocd.Application{
					Status: argocd.ApplicationStatus{
						Sync: argocd.SyncStatus{
							Status: argocd.SyncStatusCodeSynced,
						},
					},
				},
			},
			want: false,
		},
		{
			name: "Sync changed",
			e: event.TypedUpdateEvent[*argocd.Application]{
				ObjectOld: &argocd.Application{
					Status: argocd.ApplicationStatus{
						Sync: argocd.SyncStatus{
							Status: argocd.SyncStatusCodeOutOfSync,
						},
					},
				},
				ObjectNew: &argocd.Application{
					Status: argocd.ApplicationStatus{
						Sync: argocd.SyncStatus{
							Status: argocd.SyncStatusCodeSynced,
						},
					},
				},
			},
			want: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			p := ArgoCDAppSyncChanged[*argocd.Application]{
				logger: logging.NewDiscardLoggerOrDie(),
			}
			require.Equal(t, testCase.want, p.Update(testCase.e))
		})
	}
}

func TestArgoCDAppReconciledAfterOperation_Update(t *testing.T) {
	t0 := metav1.Now()
	t1 := metav1.NewTime(t0.Add(5 * time.Second))
	t2 := metav1.NewTime(t0.Add(10 * time.Second))

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
			name: "Operation state has no finishedAt",
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
		{
			name: "reconciledAt unchanged",
			e: event.TypedUpdateEvent[*argocd.Application]{
				ObjectOld: &argocd.Application{
					Status: argocd.ApplicationStatus{
						ReconciledAt: &t1,
					},
				},
				ObjectNew: &argocd.Application{
					Status: argocd.ApplicationStatus{
						ReconciledAt: &t1,
						OperationState: &argocd.OperationState{
							Phase:      argocd.OperationSucceeded,
							FinishedAt: &t2,
						},
					},
				},
			},
			want: false,
		},
		{
			name: "reconciledAt was nil, now advanced past finishedAt",
			e: event.TypedUpdateEvent[*argocd.Application]{
				ObjectOld: &argocd.Application{},
				ObjectNew: &argocd.Application{
					Status: argocd.ApplicationStatus{
						ReconciledAt: &t2,
						OperationState: &argocd.OperationState{
							Phase:      argocd.OperationSucceeded,
							FinishedAt: &t1,
						},
					},
				},
			},
			want: true,
		},
		{
			name: "reconciledAt advanced from before finishedAt to after",
			e: event.TypedUpdateEvent[*argocd.Application]{
				ObjectOld: &argocd.Application{
					Status: argocd.ApplicationStatus{
						ReconciledAt: &t0,
					},
				},
				ObjectNew: &argocd.Application{
					Status: argocd.ApplicationStatus{
						ReconciledAt: &t2,
						OperationState: &argocd.OperationState{
							Phase:      argocd.OperationSucceeded,
							FinishedAt: &t1,
						},
					},
				},
			},
			want: true,
		},
		{
			name: "reconciledAt already past finishedAt (trusted), advances further",
			e: event.TypedUpdateEvent[*argocd.Application]{
				ObjectOld: &argocd.Application{
					Status: argocd.ApplicationStatus{
						ReconciledAt: &t2,
					},
				},
				ObjectNew: &argocd.Application{
					Status: argocd.ApplicationStatus{
						ReconciledAt: func() *metav1.Time { t := metav1.NewTime(t2.Add(time.Second)); return &t }(),
						OperationState: &argocd.OperationState{
							Phase:      argocd.OperationSucceeded,
							FinishedAt: &t1,
						},
					},
				},
			},
			want: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			p := ArgoCDAppReconciledAfterOperation[*argocd.Application]{
				logger: logging.NewDiscardLoggerOrDie(),
			}
			require.Equal(t, testCase.want, p.Update(testCase.e))
		})
	}
}
