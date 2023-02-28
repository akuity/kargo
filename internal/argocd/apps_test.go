package argocd

import (
	"testing"

	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	"github.com/stretchr/testify/require"
)

func TestIsApplicationHealthyAndSynced(t *testing.T) {
	testCases := []struct {
		name       string
		app        *argocd.Application
		revision   string
		assertions func(bool, string)
	}{
		{
			name: "app has non-healthy status",
			app: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: health.HealthStatusProgressing,
					},
				},
			},
			assertions: func(healthy bool, reason string) {
				require.False(t, healthy)
				require.Contains(t, reason, "has health state")
			},
		},

		{
			name: "source isn't synced",
			app: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: health.HealthStatusHealthy,
					},
					Sync: argocd.SyncStatus{
						Status: argocd.SyncStatusCodeOutOfSync,
					},
				},
			},
			assertions: func(healthy bool, reason string) {
				require.False(t, healthy)
				require.Contains(t, reason, "is not synced to revision")
			},
		},

		{
			name: "source isn't synced to revision",
			app: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: health.HealthStatusHealthy,
					},
					Sync: argocd.SyncStatus{
						Status:   argocd.SyncStatusCodeSynced,
						Revision: "different-fake-revision",
					},
				},
			},
			revision: "fake-revision",
			assertions: func(healthy bool, reason string) {
				require.False(t, healthy)
				require.Contains(t, reason, "is not synced to revision")
			},
		},

		{
			name: "app is healthy and synced",
			app: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: health.HealthStatusHealthy,
					},
					Sync: argocd.SyncStatus{
						Status:   argocd.SyncStatusCodeSynced,
						Revision: "fake-revision",
					},
				},
			},
			revision: "fake-revision",
			assertions: func(healthy bool, reason string) {
				require.True(t, healthy)
				require.Empty(t, reason)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				IsApplicationHealthyAndSynced(
					testCase.app,
					testCase.revision,
				),
			)
		})
	}
}
