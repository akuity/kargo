package argocd

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	argocdapi "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
)

func TestGetDesiredRevisions(t *testing.T) {
	testOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse",
	}
	testOrigin2 := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "another-warehouse",
	}

	testCases := []struct {
		name           string
		app            *argocdapi.Application
		stage          *kargoapi.Stage
		freightHistory kargoapi.FreightHistory
		assertions     func(*testing.T, []string, error)
	}{
		{
			name: "no application",
			assertions: func(t *testing.T, result []string, err error) {
				require.NoError(t, err)
				require.Empty(t, result)
			},
		},
		{
			name: "no application source",
			app:  &argocdapi.Application{},
			assertions: func(t *testing.T, result []string, err error) {
				require.NoError(t, err)
				require.Empty(t, result)
			},
		},
		{
			name: "no source repo URL",
			app: &argocdapi.Application{
				Spec: argocdapi.ApplicationSpec{
					Source: &argocdapi.ApplicationSource{},
				},
			},
			assertions: func(t *testing.T, result []string, err error) {
				require.NoError(t, err)
				require.Equal(t, result, []string{""})
			},
		},
		{
			name: "chart source",
			app: &argocdapi.Application{
				Spec: argocdapi.ApplicationSpec{
					Source: &argocdapi.ApplicationSource{
						RepoURL: "https://example.com",
						Chart:   "fake-chart",
					},
				},
			},
			freightHistory: kargoapi.FreightHistory{
				&kargoapi.FreightCollection{
					Freight: map[string]kargoapi.FreightReference{
						testOrigin.String(): {
							Origin: testOrigin,
							Charts: []kargoapi.Chart{
								{
									RepoURL: "https://example.com",
									Name:    "other-fake-chart",
									Version: "v1.0.0",
								},
								{
									RepoURL: "https://example.com",
									Name:    "fake-chart",
									Version: "v2.0.0",
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, result []string, err error) {
				require.NoError(t, err)
				require.Equal(t, result, []string{"v2.0.0"})
			},
		},
		{
			name: "git source",
			app: &argocdapi.Application{
				Spec: argocdapi.ApplicationSpec{
					Source: &argocdapi.ApplicationSource{
						RepoURL: "https://github.com/universe/42",
					},
				},
			},
			freightHistory: kargoapi.FreightHistory{
				&kargoapi.FreightCollection{
					Freight: map[string]kargoapi.FreightReference{
						testOrigin.String(): {
							Origin: testOrigin,
							Commits: []kargoapi.GitCommit{
								{
									RepoURL: "https://github.com/bad/41",
									ID:      "bad-revision",
								},
								{
									RepoURL: "https://github.com/universe/42",
									ID:      "fake-revision",
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, result []string, err error) {
				require.NoError(t, err)
				require.Equal(t, result, []string{"fake-revision"})
			},
		},
		{
			name: "git multisource with chart",
			app: &argocdapi.Application{
				Spec: argocdapi.ApplicationSpec{
					Sources: []argocdapi.ApplicationSource{
						{
							RepoURL: "https://example.com",
							Chart:   "fake-chart",
						},
						{
							RepoURL:        "https://github.com/universe/42",
							TargetRevision: "fake-revision",
						},
					},
				},
				Status: argocdapi.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Status:    argocd.SyncStatusCodeSynced,
						Revisions: []string{"chart-revision", "fake-revision"},
					},
					OperationState: &argocd.OperationState{
						FinishedAt: ptr.To(metav1.Now()),
					},
				},
			},
			freightHistory: kargoapi.FreightHistory{
				&kargoapi.FreightCollection{
					Freight: map[string]kargoapi.FreightReference{
						testOrigin.String(): {
							Origin: testOrigin,
							Commits: []kargoapi.GitCommit{
								{
									RepoURL: "https://github.com/universe/42",
									ID:      "fake-revision",
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, result []string, err error) {
				require.NoError(t, err)
				require.Equal(t, result, []string{"", "fake-revision"})
			},
		},
		{
			name: "git multisource with chart without synced revisions",
			app: &argocdapi.Application{
				Spec: argocdapi.ApplicationSpec{
					Sources: []argocdapi.ApplicationSource{
						{
							RepoURL: "https://example.com",
							Chart:   "fake-chart",
						},
						{
							RepoURL:        "https://github.com/universe/42",
							TargetRevision: "fake-revision",
						},
					},
				},
				Status: argocdapi.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Status: argocd.SyncStatusCodeSynced,
					},
					OperationState: &argocd.OperationState{
						FinishedAt: ptr.To(metav1.Now()),
					},
				},
			},
			freightHistory: kargoapi.FreightHistory{
				&kargoapi.FreightCollection{
					Freight: map[string]kargoapi.FreightReference{
						testOrigin.String(): {
							Origin: testOrigin,
							Commits: []kargoapi.GitCommit{
								{
									RepoURL: "https://github.com/universe/42",
									ID:      "fake-revision",
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, result []string, err error) {
				require.NoError(t, err)
				require.Equal(t, result, []string{"", "fake-revision"})
			},
		},
		{
			name: "git multisource with multiple freight references",
			app: &argocdapi.Application{
				Spec: argocdapi.ApplicationSpec{
					Sources: []argocdapi.ApplicationSource{
						{
							RepoURL: "https://example.com",
							Chart:   "fake-chart",
						},
						{
							RepoURL:        "https://github.com/universe/42",
							TargetRevision: "fake-revision",
						},
						{
							RepoURL:        "https://github.com/another-universe/42",
							TargetRevision: "another-revision",
						},
					},
				},
				Status: argocdapi.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Status:    argocd.SyncStatusCodeSynced,
						Revisions: []string{"", "fake-revision", "another-revision"},
					},
					OperationState: &argocd.OperationState{
						FinishedAt: ptr.To(metav1.Now()),
					},
				},
			},
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{
						ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{
							{
								SourceUpdates: []kargoapi.ArgoCDSourceUpdate{
									{Origin: &testOrigin, RepoURL: "https://github.com/universe/42"},
									{Origin: &testOrigin2, RepoURL: "https://github.com/another-universe/42"},
								},
							},
						},
					},
				},
				Status: kargoapi.StageStatus{
					FreightHistory: kargoapi.FreightHistory{
						&kargoapi.FreightCollection{
							Freight: map[string]kargoapi.FreightReference{
								testOrigin.String(): {
									Origin: testOrigin,
									Commits: []kargoapi.GitCommit{
										{
											RepoURL: "https://github.com/universe/42",
											ID:      "fake-revision",
										},
									},
								},
								testOrigin2.String(): {
									Origin: testOrigin2,
									Commits: []kargoapi.GitCommit{
										{
											RepoURL: "https://github.com/another-universe/42",
											ID:      "another-revision",
										},
									},
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, result []string, err error) {
				require.NoError(t, err)
				require.Equal(t, result, []string{"", "fake-revision", "another-revision"})
			},
		},
		{
			name: "git source with health check commit",
			app: &argocdapi.Application{
				Spec: argocdapi.ApplicationSpec{
					Source: &argocdapi.ApplicationSource{
						RepoURL: "https://github.com/universe/42",
					},
				},
			},
			freightHistory: kargoapi.FreightHistory{
				&kargoapi.FreightCollection{
					Freight: map[string]kargoapi.FreightReference{
						testOrigin.String(): {
							Origin: testOrigin,
							Commits: []kargoapi.GitCommit{
								{
									RepoURL:           "https://github.com/universe/42",
									HealthCheckCommit: "fake-revision",
									ID:                "bad-revision",
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, result []string, err error) {
				require.NoError(t, err)
				require.Equal(t, result, []string{"fake-revision"})
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {

			stage := testCase.stage
			if stage == nil {
				stage = &kargoapi.Stage{
					Spec: kargoapi.StageSpec{
						PromotionMechanisms: &kargoapi.PromotionMechanisms{
							ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{
								{
									Origin: &testOrigin,
								},
							},
						},
					},
					Status: kargoapi.StageStatus{
						FreightHistory: testCase.freightHistory,
					},
				}
			}

			revisions, err := GetDesiredRevisions(
				context.Background(),
				nil, // No client is needed as long as we're always explicit about origins
				stage,
				&stage.Spec.PromotionMechanisms.ArgoCDAppUpdates[0],
				testCase.app,
				stage.Status.FreightHistory.Current().References(),
			)
			require.NoError(t, err)
			testCase.assertions(t, revisions, err)
		})
	}
}
