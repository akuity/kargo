package external

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/indexer"
	"github.com/akuity/kargo/pkg/urls"
)

func TestHandleRefreshAction(t *testing.T) {
	testScheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(testScheme))

	testCases := []struct {
		name       string
		client     client.Client
		project    string
		targets    []kargoapi.GenericWebhookTarget
		actionEnv  map[string]any
		assertions func(*testing.T, []targetResult)
	}{
		{
			name: "error listing Warehouses",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithInterceptorFuncs(
				interceptor.Funcs{
					List: func(
						context.Context,
						client.WithWatch,
						client.ObjectList,
						...client.ListOption,
					) error {
						return errors.New("something went wrong")
					},
				},
			).Build(),
			targets: []kargoapi.GenericWebhookTarget{{
				Kind: kargoapi.GenericWebhookTargetKindWarehouse,
			}},
			assertions: func(t *testing.T, results []targetResult) {
				require.Len(t, results, 1)
				require.Equal(t, kargoapi.GenericWebhookTargetKindWarehouse, results[0].Kind)
				require.Error(t, results[0].ListError)
				require.Contains(t, results[0].ListError.Error(), "error listing warehouse targets")
			},
		},
		{
			name: "list warehouses with complex mixed index and label selector combo",
			client: fake.NewClientBuilder().WithScheme(testScheme).
				WithIndex(
					&kargoapi.Warehouse{},
					indexer.WarehousesBySubscribedURLsField,
					indexer.WarehousesBySubscribedURLs,
				).WithObjects(
				// this warehouse satisifies both index and label selectors
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-project",
						Name:      "warehouse-1",
						Labels: map[string]string{
							"env":  "prod",
							"tier": "backend",
						},
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{
								RepoURL: "https://github.com/example/repo.git",
							},
						}},
					},
				},
				// label selector should not match this Warehouse
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-project",
						Name:      "warehouse-with-mismatching-labels",
						Labels:    map[string]string{"doesnt": "match"},
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{
								RepoURL: "https://github.com/example/repo.git",
							},
						}},
					},
				},
				// index selector should not match this Warehouse
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-project",
						Name:      "warehouse-with-mismatching-index",
						Labels: map[string]string{
							"env":  "prod",
							"tier": "frontend",
						},
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{
								RepoURL: "https://github.com/example/wrong-repo.git",
							},
						}},
					},
				},
			).Build(),
			project: "test-project",
			targets: []kargoapi.GenericWebhookTarget{{
				Kind: kargoapi.GenericWebhookTargetKindWarehouse,
				IndexSelector: kargoapi.IndexSelector{
					MatchExpressions: []kargoapi.IndexSelectorRequirement{{
						Key:      indexer.WarehousesBySubscribedURLsField,
						Operator: kargoapi.IndexSelectorRequirementOperatorEqual,
						Value:    "https://github.com/example/repo",
					}},
				},
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{"env": "prod"},
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "tier",
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{"backend", "frontend"},
						},
					},
				},
			}},
			assertions: func(t *testing.T, results []targetResult) {
				require.Len(t, results, 1)
				require.Equal(t, kargoapi.GenericWebhookTargetKindWarehouse, results[0].Kind)
				require.NoError(t, results[0].ListError)
				require.Len(t, results[0].RefreshResults, 1)
				require.Equal(t, "test-project/warehouse-1", results[0].RefreshResults[0].Success)
			},
		},
		{
			name: "partial success refreshing warehouses",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				// both warehouses satisfy the label selector
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-namespace",
						Name:      "frontend-warehouse",
						Labels:    map[string]string{"tier": "frontend"},
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{
								RepoURL: "https://github.com/example/frontend-repo.git",
							},
						}},
					},
				},
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-namespace",
						Name:      "backend-warehouse",
						Labels:    map[string]string{"tier": "backend"},
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{
								RepoURL: "https://github.com/example/backend-repo.git",
							},
						}},
					},
				},
			).WithInterceptorFuncs(interceptor.Funcs{
				Patch: func(
					_ context.Context,
					_ client.WithWatch,
					obj client.Object,
					_ client.Patch,
					_ ...client.PatchOption,
				) error {
					if obj.GetName() == "backend-warehouse" {
						return nil
					}
					return errors.New("something went wrong")
				},
			}).WithIndex(
				&kargoapi.Warehouse{},
				indexer.WarehousesBySubscribedURLsField,
				indexer.WarehousesBySubscribedURLs,
			).Build(),
			targets: []kargoapi.GenericWebhookTarget{{
				Kind: kargoapi.GenericWebhookTargetKindWarehouse,
				LabelSelector: metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{{
						Key:      "tier",
						Operator: metav1.LabelSelectorOpIn,
						Values:   []string{"frontend", "backend"},
					}},
				},
			}},
			assertions: func(t *testing.T, results []targetResult) {
				require.Len(t, results, 1)
				require.Len(t, results[0].RefreshResults, 2)
				firstWhResult := results[0].RefreshResults[0]
				require.Empty(t, firstWhResult.Failure)
				require.Equal(t, firstWhResult.Success, "test-namespace/backend-warehouse")
				secondResult := results[0].RefreshResults[1]
				require.NotEmpty(t, secondResult.Failure)
				require.Contains(t, secondResult.Failure, "test-namespace/frontend-warehouse")
			},
		},
		{
			name:   "unsupported target kind",
			client: fake.NewClientBuilder().WithScheme(testScheme).Build(),
			targets: []kargoapi.GenericWebhookTarget{{
				Kind: "UnsupportedKind",
			}},
			assertions: func(t *testing.T, results []targetResult) {
				require.Len(t, results, 1)
				require.Equal(t, "UnsupportedKind", string(results[0].Kind))
				require.Error(t, results[0].ListError)
				require.ErrorContains(t, results[0].ListError, "skipped listing of unsupported target type")
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			tt.assertions(t, refreshTargets(
				t.Context(),
				tt.client,
				tt.project,
				tt.actionEnv,
				tt.targets,
			))
		})
	}
}

func TestRefreshWarehouses(t *testing.T) {
	// Callers are responsible for normalizing the repository URL.
	testRepoURL := urls.NormalizeGit("https://github.com/example/repo.git")
	testRepoURLs := []string{testRepoURL}

	const testProject = "fake-project"

	testScheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(testScheme))

	testCases := []struct {
		name       string
		client     client.Client
		project    string
		assertions func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "error listing Warehouses",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithInterceptorFuncs(
				interceptor.Funcs{
					List: func(
						context.Context,
						client.WithWatch,
						client.ObjectList,
						...client.ListOption,
					) error {
						return errors.New("something went wrong")
					},
				},
			).Build(),
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rr.Code)
				require.JSONEq(t, "{}", rr.Body.String())
			},
		},
		{
			name: "partial success -- Project not specified",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "some-namespace",
						Name:      "some-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{RepoURL: testRepoURL},
						}},
					},
				},
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "some-other-namespace",
						Name:      "some-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{RepoURL: testRepoURL},
						}},
					},
				},
			).WithInterceptorFuncs(interceptor.Funcs{
				Patch: func(
					_ context.Context,
					_ client.WithWatch,
					obj client.Object,
					_ client.Patch,
					_ ...client.PatchOption,
				) error {
					if obj.GetNamespace() == "some-namespace" {
						return nil
					}
					return errors.New("something went wrong")
				},
			}).WithIndex(
				&kargoapi.Warehouse{},
				indexer.WarehousesBySubscribedURLsField,
				indexer.WarehousesBySubscribedURLs,
			).Build(),
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rr.Code)
				require.JSONEq(t, `{"error":"failed to refresh 1 of 2 warehouses"}`, rr.Body.String())
			},
		},
		{
			name: "complete success -- Project not specified",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "some-namespace",
						Name:      "some-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{RepoURL: testRepoURL},
						}},
					},
				},
			).WithIndex(
				&kargoapi.Warehouse{},
				indexer.WarehousesBySubscribedURLsField,
				indexer.WarehousesBySubscribedURLs,
			).Build(),
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 1 warehouse(s)"}`, rr.Body.String())
			},
		},
		{
			name:    "partial success -- Project specified",
			project: testProject,
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "some-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{RepoURL: testRepoURL},
						}},
					},
				},
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "some-other-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{RepoURL: testRepoURL},
						}},
					},
				},
			).WithInterceptorFuncs(interceptor.Funcs{
				Patch: func(
					_ context.Context,
					_ client.WithWatch,
					obj client.Object,
					_ client.Patch,
					_ ...client.PatchOption,
				) error {
					if obj.GetName() == "some-warehouse" {
						return nil
					}
					return errors.New("something went wrong")
				},
			}).WithIndex(
				&kargoapi.Warehouse{},
				indexer.WarehousesBySubscribedURLsField,
				indexer.WarehousesBySubscribedURLs,
			).Build(),
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rr.Code)
				require.JSONEq(t, `{"error":"failed to refresh 1 of 2 warehouses"}`, rr.Body.String())
			},
		},
		{
			name:    "complete success -- Project specified",
			project: testProject,
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "some-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{RepoURL: testRepoURL},
						}},
					},
				},
			).WithIndex(
				&kargoapi.Warehouse{},
				indexer.WarehousesBySubscribedURLsField,
				indexer.WarehousesBySubscribedURLs,
			).Build(),
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 1 warehouse(s)"}`, rr.Body.String())
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			refreshWarehouses(
				t.Context(),
				w,
				tt.client,
				tt.project,
				testRepoURLs,
			)
			tt.assertions(t, w)
		})
	}
}

func TestShouldRefresh(t *testing.T) {
	testCases := []struct {
		name       string
		wh         kargoapi.Warehouse
		qualifiers []string
		repoURL    string
		expect     bool
	}{
		{
			name: "Git subscription with matching qualifier",
			wh: kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					Subscriptions: []kargoapi.RepoSubscription{{
						Git: &kargoapi.GitSubscription{
							CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestFromBranch,
							RepoURL:                 "https://github.com/username/repo",
							Branch:                  "main",
						},
					}},
				},
			},
			repoURL:    "https://github.com/username/repo",
			qualifiers: []string{"refs/heads/main"},
			expect:     true,
		},
		{
			name: "Git subscription with non-matching qualifier",
			wh: kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					Subscriptions: []kargoapi.RepoSubscription{{
						Git: &kargoapi.GitSubscription{
							CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestFromBranch,
							RepoURL:                 "https://github.com/username/repo.git",
							Branch:                  "main",
						},
					}},
				},
			},
			repoURL:    "https://github.com/username/repo",
			qualifiers: []string{"release"},
			expect:     false,
		},
		{
			name: "Image subscription with matching qualifier",
			wh: kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					Subscriptions: []kargoapi.RepoSubscription{{
						Image: &kargoapi.ImageSubscription{
							RepoURL:                "docker.io/example/repo",
							ImageSelectionStrategy: kargoapi.ImageSelectionStrategySemVer,
							Constraint:             "^1.0.0",
							StrictSemvers:          true,
						},
					}},
				},
			},
			repoURL:    "example/repo",
			qualifiers: []string{"v1.0.0"},
			expect:     true,
		},
		{
			name: "Image subscription with non-matching qualifier",
			wh: kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					Subscriptions: []kargoapi.RepoSubscription{{
						Image: &kargoapi.ImageSubscription{
							RepoURL:                "docker.io/example/repo",
							ImageSelectionStrategy: kargoapi.ImageSelectionStrategySemVer,
							Constraint:             "^1.0.0",
							StrictSemvers:          true,
						},
					}},
				},
			},
			repoURL:    "docker.io/example/repo",
			qualifiers: []string{"invalid-tag"},
			expect:     false,
		},
		{
			name: "Chart subscription with matching qualifier",
			wh: kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					Subscriptions: []kargoapi.RepoSubscription{{
						Chart: &kargoapi.ChartSubscription{
							RepoURL:          "oci://example.com/charts",
							SemverConstraint: "^1.0.0",
						},
					}},
				},
			},
			repoURL:    "example.com/charts",
			qualifiers: []string{"v1.0.0"},
			expect:     true,
		},
		{
			name: "Chart subscription with non-matching qualifier",
			wh: kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					Subscriptions: []kargoapi.RepoSubscription{{
						Chart: &kargoapi.ChartSubscription{
							RepoURL:          "oci://example.com/charts",
							SemverConstraint: "^2.0.0",
						},
					}},
				},
			},
			repoURL:    "example.com/charts",
			qualifiers: []string{"1.0.0"},
			expect:     false,
		},
		{
			name:       "No subscriptions",
			wh:         kargoapi.Warehouse{},
			qualifiers: []string{"main"},
			expect:     false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := shouldRefresh(tc.wh, tc.repoURL, tc.qualifiers...)
			require.NoError(t, err)
			require.Equal(t, tc.expect, result)
		})
	}
}
