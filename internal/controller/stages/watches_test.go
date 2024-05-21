package stages

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/event"

	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	rollouts "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
)

func TestAppHealthOrSyncStatusChanged(t *testing.T) {
	testCases := []struct {
		name    string
		old     *argocd.Application
		new     *argocd.Application
		updated bool
	}{
		{
			name: "health changed",
			old: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: "Healthy",
					},
				},
			},
			new: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: "Degraded",
					},
				},
			},
			updated: true,
		},
		{
			name: "health did not change",
			old: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: "Healthy",
					},
				},
			},
			new: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: "Healthy",
					},
				},
			},
			updated: false,
		},
		{
			name: "sync status changed",
			old: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Status: "",
					},
				},
			},
			new: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Status: "Synced",
					},
				},
			},
			updated: true,
		},
		{
			name: "sync status did not change",
			old: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Status: "Synced",
					},
				},
			},
			new: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Status: "Synced",
					},
				},
			},
			updated: false,
		},
		{
			name: "revision changed",
			old: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Revision: "fake-revision",
					},
				},
			},
			new: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Revision: "different-fake-revision",
					},
				},
			},
			updated: true,
		},
		{
			name: "revision did not change",
			old: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Revision: "fake-revision",
					},
				},
			},
			new: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Revision: "fake-revision",
					},
				},
			},
			updated: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			e := event.UpdateEvent{
				ObjectOld: testCase.old,
				ObjectNew: testCase.new,
			}
			require.Equal(
				t,
				testCase.updated,
				appHealthOrSyncStatusChanged(context.Background(), e),
			)
		})
	}
}

func TestAnalysisRunPhaseChanged(t *testing.T) {
	testCases := []struct {
		name    string
		old     *rollouts.AnalysisRun
		new     *rollouts.AnalysisRun
		updated bool
	}{
		{
			name: "phase changed",
			old: &rollouts.AnalysisRun{
				Status: rollouts.AnalysisRunStatus{
					Phase: "old-phase",
				},
			},
			new: &rollouts.AnalysisRun{
				Status: rollouts.AnalysisRunStatus{
					Phase: "new-phase",
				},
			},
			updated: true,
		},
		{
			name: "phase did not change",
			old: &rollouts.AnalysisRun{
				Status: rollouts.AnalysisRunStatus{
					Phase: "old-phase",
				},
			},
			new: &rollouts.AnalysisRun{
				Status: rollouts.AnalysisRunStatus{
					Phase: "old-phase",
				},
			},
			updated: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			e := event.UpdateEvent{
				ObjectOld: testCase.old,
				ObjectNew: testCase.new,
			}
			require.Equal(
				t,
				testCase.updated,
				analysisRunPhaseChanged(context.Background(), e),
			)
		})
	}
}
