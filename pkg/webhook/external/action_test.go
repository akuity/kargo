package external

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/indexer"
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
				require.Contains(t, results[0].ListError.Error(), "something went wrong")
			},
		},
		{
			name: "full success refreshing warehouses with complex mixed index and label selector combo",
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
					MatchIndices: []kargoapi.IndexSelectorRequirement{{
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
				// this one will fail to refresh per the interceptor logic below
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
			name: "successful refresh using static name only",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				// does not have the specified name
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-namespace",
						Name:      "frontend-warehouse",
					},
				},
				// has the specified name
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-namespace",
						Name:      "backend-warehouse",
					},
				},
			).WithIndex(
				&kargoapi.Warehouse{},
				indexer.WarehousesBySubscribedURLsField,
				indexer.WarehousesBySubscribedURLs,
			).Build(),
			targets: []kargoapi.GenericWebhookTarget{{
				Kind: kargoapi.GenericWebhookTargetKindWarehouse,
				Name: "backend-warehouse",
			}},
			assertions: func(t *testing.T, results []targetResult) {
				require.Len(t, results, 1)
				require.Len(t, results[0].RefreshResults, 1)
				firstWhResult := results[0].RefreshResults[0]
				require.Equal(t, firstWhResult.Success, "test-namespace/backend-warehouse")
			},
		},
		{
			name: "successful refresh using static name with label selector",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				// this warehouse satisfies the label selector
				// but does not have the specified name
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
				// this warehouse has the specified name and satisfies the label selector
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
			).WithIndex(
				&kargoapi.Warehouse{},
				indexer.WarehousesBySubscribedURLsField,
				indexer.WarehousesBySubscribedURLs,
			).Build(),
			targets: []kargoapi.GenericWebhookTarget{{
				Kind: kargoapi.GenericWebhookTargetKindWarehouse,
				Name: "backend-warehouse",
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
				require.Len(t, results[0].RefreshResults, 1)
				firstWhResult := results[0].RefreshResults[0]
				require.Equal(t, firstWhResult.Success, "test-namespace/backend-warehouse")
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
				require.ErrorContains(t, results[0].ListError, "unsupported target kind: \"UnsupportedKind\"")
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			tt.assertions(t, handleAction(
				t.Context(),
				tt.client,
				tt.project,
				tt.actionEnv,
				kargoapi.GenericWebhookAction{
					Name:    kargoapi.GenericWebhookActionNameRefresh,
					Targets: tt.targets,
				},
			))
		})
	}
}
