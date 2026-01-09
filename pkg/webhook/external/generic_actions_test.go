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

func TestHandleAction(t *testing.T) {
	testScheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(testScheme))
	env := make(map[string]any)

	testCases := []struct {
		name       string
		client     client.Client
		project    string
		action     kargoapi.GenericWebhookAction
		assertions func(*testing.T, actionResult)
	}{
		{
			name:   "whenExpression empty",
			client: fake.NewClientBuilder().WithScheme(testScheme).Build(),
			action: kargoapi.GenericWebhookAction{
				WhenExpression: "",
				ActionType:     kargoapi.GenericWebhookActionTypeRefresh,
				TargetSelectionCriteria: []kargoapi.GenericWebhookTargetSelectionCriteria{{
					Kind: kargoapi.GenericWebhookTargetKindWarehouse,
				}},
			},
			assertions: func(t *testing.T, ar actionResult) {
				require.Equal(t, ar.ActionType, kargoapi.GenericWebhookActionTypeRefresh)
				require.Equal(t, ar.TargetSelectionCriteria,
					[]kargoapi.GenericWebhookTargetSelectionCriteria{{
						Kind: kargoapi.GenericWebhookTargetKindWarehouse,
					}},
				)
				require.Equal(t, ar.WhenExpression, "")
				require.True(t, ar.MatchedWhenExpression)
				require.Empty(t, ar.SelectedTargets)
				require.Equal(t, ar.Result, resultSuccess)
				require.Equal(t, ar.Summary, "Refreshed 0 of 0 selected resources")
			},
		},
		{
			name:   "whenExpression not satisfied",
			client: fake.NewClientBuilder().WithScheme(testScheme).Build(),
			action: kargoapi.GenericWebhookAction{
				WhenExpression: "false",
				ActionType:     kargoapi.GenericWebhookActionTypeRefresh,
				TargetSelectionCriteria: []kargoapi.GenericWebhookTargetSelectionCriteria{{
					Kind: kargoapi.GenericWebhookTargetKindWarehouse,
				}},
			},
			assertions: func(t *testing.T, ar actionResult) {
				require.Equal(t, ar.ActionType, kargoapi.GenericWebhookActionTypeRefresh)
				require.Equal(t, ar.TargetSelectionCriteria,
					[]kargoapi.GenericWebhookTargetSelectionCriteria{{
						Kind: kargoapi.GenericWebhookTargetKindWarehouse,
					}},
				)
				require.Equal(t, ar.WhenExpression, "false")
				require.False(t, ar.MatchedWhenExpression)
				require.Empty(t, ar.SelectedTargets)
				require.Equal(t, ar.Result, resultNotApplicable)
				require.Equal(t, ar.Summary, summaryRequestNotMatched)
			},
		},
		{
			name:   "error evaluating whenExpression",
			client: fake.NewClientBuilder().WithScheme(testScheme).Build(),
			action: kargoapi.GenericWebhookAction{
				WhenExpression: "invalid expression",
				ActionType:     kargoapi.GenericWebhookActionTypeRefresh,
				TargetSelectionCriteria: []kargoapi.GenericWebhookTargetSelectionCriteria{{
					Kind: kargoapi.GenericWebhookTargetKindWarehouse,
				}},
			},
			assertions: func(t *testing.T, ar actionResult) {
				require.Equal(t, ar.ActionType, kargoapi.GenericWebhookActionTypeRefresh)
				require.Equal(t, ar.TargetSelectionCriteria,
					[]kargoapi.GenericWebhookTargetSelectionCriteria{{
						Kind: kargoapi.GenericWebhookTargetKindWarehouse,
					}},
				)
				require.Equal(t, ar.WhenExpression, "invalid expression")
				require.False(t, ar.MatchedWhenExpression)
				require.Empty(t, ar.SelectedTargets)
				require.Equal(t, ar.Result, resultError)
				require.Equal(t, ar.Summary, summaryRequestMatchingError)
			},
		},
		{
			name:   "error building list options - invalid operator",
			client: fake.NewClientBuilder().WithScheme(testScheme).Build(),
			action: kargoapi.GenericWebhookAction{
				WhenExpression: "true",
				ActionType:     kargoapi.GenericWebhookActionTypeRefresh,
				TargetSelectionCriteria: []kargoapi.GenericWebhookTargetSelectionCriteria{{
					Kind: kargoapi.GenericWebhookTargetKindWarehouse,
					IndexSelector: kargoapi.IndexSelector{
						MatchIndices: []kargoapi.IndexSelectorRequirement{{
							Key:      "",
							Operator: "InvalidOperator",
							Value:    "some-value",
						}},
					},
				}},
			},
			assertions: func(t *testing.T, ar actionResult) {
				require.Equal(t, ar.ActionType, kargoapi.GenericWebhookActionTypeRefresh)
				require.Equal(t, ar.TargetSelectionCriteria,
					[]kargoapi.GenericWebhookTargetSelectionCriteria{{
						Kind: kargoapi.GenericWebhookTargetKindWarehouse,
						IndexSelector: kargoapi.IndexSelector{
							MatchIndices: []kargoapi.IndexSelectorRequirement{{
								Key:      "",
								Operator: "InvalidOperator",
								Value:    "some-value",
							}},
						},
					}},
				)
				require.Equal(t, ar.WhenExpression, "true")
				require.True(t, ar.MatchedWhenExpression)
				require.Empty(t, ar.SelectedTargets)
				require.Equal(t, ar.Result, resultError)
				require.Equal(t, ar.Summary, summaryResourceSelectionError)
			},
		},
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
			action: kargoapi.GenericWebhookAction{
				WhenExpression: "true",
				ActionType:     kargoapi.GenericWebhookActionTypeRefresh,
				TargetSelectionCriteria: []kargoapi.GenericWebhookTargetSelectionCriteria{{
					Kind: kargoapi.GenericWebhookTargetKindWarehouse,
				}},
			},
			assertions: func(t *testing.T, ar actionResult) {
				require.Equal(t, ar.ActionType, kargoapi.GenericWebhookActionTypeRefresh)
				require.Equal(t, ar.TargetSelectionCriteria,
					[]kargoapi.GenericWebhookTargetSelectionCriteria{{
						Kind: kargoapi.GenericWebhookTargetKindWarehouse,
					}},
				)
				require.Equal(t, ar.WhenExpression, "true")
				require.True(t, ar.MatchedWhenExpression)
				require.Empty(t, ar.SelectedTargets)
				require.Equal(t, ar.Result, resultError)
				require.Equal(t, ar.Summary, summaryResourceSelectionError)
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
						InternalSubscriptions: []kargoapi.RepoSubscription{{
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
						InternalSubscriptions: []kargoapi.RepoSubscription{{
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
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{
								RepoURL: "https://github.com/example/wrong-repo.git",
							},
						}},
					},
				},
			).Build(),
			project: "test-project",
			action: kargoapi.GenericWebhookAction{
				WhenExpression: "true",
				ActionType:     kargoapi.GenericWebhookActionTypeRefresh,
				TargetSelectionCriteria: []kargoapi.GenericWebhookTargetSelectionCriteria{{
					Kind: kargoapi.GenericWebhookTargetKindWarehouse,
					IndexSelector: kargoapi.IndexSelector{
						MatchIndices: []kargoapi.IndexSelectorRequirement{{
							Key:      indexer.WarehousesBySubscribedURLsField,
							Operator: kargoapi.IndexSelectorOperatorEqual,
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
			},
			assertions: func(t *testing.T, ar actionResult) {
				require.Equal(t, ar.ActionType, kargoapi.GenericWebhookActionTypeRefresh)
				require.Equal(t, ar.TargetSelectionCriteria,
					[]kargoapi.GenericWebhookTargetSelectionCriteria{{
						Kind: kargoapi.GenericWebhookTargetKindWarehouse,
						IndexSelector: kargoapi.IndexSelector{
							MatchIndices: []kargoapi.IndexSelectorRequirement{{
								Key:      indexer.WarehousesBySubscribedURLsField,
								Operator: kargoapi.IndexSelectorOperatorEqual,
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
				)
				require.Equal(t, ar.WhenExpression, "true")
				require.True(t, ar.MatchedWhenExpression)
				require.Equal(
					t,
					[]selectedTarget{
						{
							Namespace: "test-project",
							Name:      "warehouse-1",
							Success:   true,
						},
					},
					ar.SelectedTargets,
				)
				require.Equal(t, ar.Result, resultSuccess)
				require.Equal(t, ar.Summary, "Refreshed 1 of 1 selected resources")
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
						InternalSubscriptions: []kargoapi.RepoSubscription{{
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
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{
								RepoURL: "https://github.com/example/backend-repo.git",
							},
						}},
					},
				},
			).WithInterceptorFuncs(interceptor.Funcs{
				Patch: func(
					ctx context.Context,
					client client.WithWatch,
					obj client.Object,
					patch client.Patch,
					opts ...client.PatchOption,
				) error {
					if obj.GetName() == "backend-warehouse" {
						return client.Patch(ctx, obj, patch, opts...)
					}
					return errors.New("something went wrong")
				},
			}).WithIndex(
				&kargoapi.Warehouse{},
				indexer.WarehousesBySubscribedURLsField,
				indexer.WarehousesBySubscribedURLs,
			).Build(),
			action: kargoapi.GenericWebhookAction{
				WhenExpression: "true",
				ActionType:     kargoapi.GenericWebhookActionTypeRefresh,
				TargetSelectionCriteria: []kargoapi.GenericWebhookTargetSelectionCriteria{{
					Kind: kargoapi.GenericWebhookTargetKindWarehouse,
					LabelSelector: metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{{
							Key:      "tier",
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{"frontend", "backend"},
						}},
					},
				}},
			},
			assertions: func(t *testing.T, ar actionResult) {
				require.Equal(t, ar.ActionType, kargoapi.GenericWebhookActionTypeRefresh)
				require.Equal(t, ar.TargetSelectionCriteria,
					[]kargoapi.GenericWebhookTargetSelectionCriteria{{
						Kind: kargoapi.GenericWebhookTargetKindWarehouse,
						LabelSelector: metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{{
								Key:      "tier",
								Operator: metav1.LabelSelectorOpIn,
								Values:   []string{"frontend", "backend"},
							}},
						},
					}},
				)
				require.Equal(t, ar.WhenExpression, "true")
				require.True(t, ar.MatchedWhenExpression)
				require.Equal(
					t,
					[]selectedTarget{
						{
							Namespace: "test-namespace",
							Name:      "backend-warehouse",
							Success:   true,
						},
						{
							Namespace: "test-namespace",
							Name:      "frontend-warehouse",
							Success:   false,
						},
					},
					ar.SelectedTargets,
				)
				require.Equal(t, ar.Result, resultPartialSuccess)
				require.Equal(t, ar.Summary, "Refreshed 1 of 2 selected resources")
			},
		},
		{
			name: "successful refresh using static name only",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				// incorrect name
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
			action: kargoapi.GenericWebhookAction{
				WhenExpression: "true",
				ActionType:     kargoapi.GenericWebhookActionTypeRefresh,
				TargetSelectionCriteria: []kargoapi.GenericWebhookTargetSelectionCriteria{{
					Kind: kargoapi.GenericWebhookTargetKindWarehouse,
					Name: "backend-warehouse",
				}},
			},
			assertions: func(t *testing.T, ar actionResult) {
				require.Equal(t, ar.ActionType, kargoapi.GenericWebhookActionTypeRefresh)
				require.Equal(t, ar.TargetSelectionCriteria,
					[]kargoapi.GenericWebhookTargetSelectionCriteria{{
						Kind: kargoapi.GenericWebhookTargetKindWarehouse,
						Name: "backend-warehouse",
					}},
				)
				require.Equal(t, ar.WhenExpression, "true")
				require.True(t, ar.MatchedWhenExpression)
				require.Equal(
					t,
					[]selectedTarget{
						{
							Namespace: "test-namespace",
							Name:      "backend-warehouse",
							Success:   true,
						},
					},
					ar.SelectedTargets,
				)
				require.Equal(t, ar.Result, resultSuccess)
				require.Equal(t, ar.Summary, "Refreshed 1 of 1 selected resources")
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
						InternalSubscriptions: []kargoapi.RepoSubscription{{
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
						InternalSubscriptions: []kargoapi.RepoSubscription{{
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
			action: kargoapi.GenericWebhookAction{
				WhenExpression: "true",
				ActionType:     kargoapi.GenericWebhookActionTypeRefresh,
				TargetSelectionCriteria: []kargoapi.GenericWebhookTargetSelectionCriteria{{
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
			},
			assertions: func(t *testing.T, ar actionResult) {
				require.Equal(t, ar.ActionType, kargoapi.GenericWebhookActionTypeRefresh)
				require.Equal(t, ar.TargetSelectionCriteria,
					[]kargoapi.GenericWebhookTargetSelectionCriteria{{
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
				)
				require.Equal(t, ar.WhenExpression, "true")
				require.True(t, ar.MatchedWhenExpression)
				require.Equal(
					t,
					[]selectedTarget{
						{
							Namespace: "test-namespace",
							Name:      "backend-warehouse",
							Success:   true,
						},
					},
					ar.SelectedTargets,
				)
				require.Equal(t, ar.Result, resultSuccess)
				require.Equal(t, ar.Summary, "Refreshed 1 of 1 selected resources")
			},
		},
		{
			name:   "unsupported target kind",
			client: fake.NewClientBuilder().WithScheme(testScheme).Build(),
			action: kargoapi.GenericWebhookAction{
				WhenExpression: "true",
				ActionType:     kargoapi.GenericWebhookActionTypeRefresh,
				TargetSelectionCriteria: []kargoapi.GenericWebhookTargetSelectionCriteria{{
					Kind: "UnsupportedKind",
				}},
			},
			assertions: func(t *testing.T, ar actionResult) {
				require.Equal(t, ar.ActionType, kargoapi.GenericWebhookActionTypeRefresh)
				require.Equal(t, ar.TargetSelectionCriteria,
					[]kargoapi.GenericWebhookTargetSelectionCriteria{{
						Kind: "UnsupportedKind",
					}},
				)
				require.Equal(t, ar.WhenExpression, "true")
				require.True(t, ar.MatchedWhenExpression)
				require.Empty(t, ar.SelectedTargets)
				require.Equal(t, ar.Result, resultError)
				require.Equal(t, ar.Summary, summaryResourceSelectionError)
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			tt.assertions(t, (&genericWebhookReceiver{
				baseWebhookReceiver: &baseWebhookReceiver{
					client:  tt.client,
					project: tt.project,
				},
			}).handleAction(t.Context(), tt.action, env))
		})
	}
}
