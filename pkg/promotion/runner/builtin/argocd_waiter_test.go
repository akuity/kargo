package builtin

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	argocd "github.com/akuity/kargo/pkg/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_argocdWaiter_convert(t *testing.T) {
	tests := []validationTestCase{
		{
			name:   "apps not specified",
			config: promotion.Config{},
			expectedProblems: []string{
				"(root): apps is required",
			},
		},
		{
			name: "apps is empty",
			config: promotion.Config{
				"apps": []promotion.Config{},
			},
			expectedProblems: []string{
				"apps: Array must have at least 1 items",
			},
		},
		{
			name: "app with neither name nor selector",
			config: promotion.Config{
				"apps": []promotion.Config{{}},
			},
			expectedProblems: []string{
				"Must validate one and only one schema (oneOf)",
			},
		},
		{
			name: "app with both name and selector",
			config: promotion.Config{
				"apps": []promotion.Config{
					{
						"name": "my-app",
						"selector": promotion.Config{
							"matchLabels": promotion.Config{"app": "test"},
						},
					},
				},
			},
			expectedProblems: []string{
				"Must validate one and only one schema (oneOf)",
			},
		},
		{
			name: "invalid waitFor value",
			config: promotion.Config{
				"apps": []promotion.Config{
					{
						"name":    "my-app",
						"waitFor": []string{"invalid"},
					},
				},
			},
			expectedProblems: []string{
				"apps.0.waitFor.0",
			},
		},
		{
			name: "valid: app with name, no waitFor",
			config: promotion.Config{
				"apps": []promotion.Config{
					{"name": "my-app"},
				},
			},
		},
		{
			name: "valid: app with name and waitFor",
			config: promotion.Config{
				"apps": []promotion.Config{
					{
						"name":    "my-app",
						"waitFor": []string{"health", "sync"},
					},
				},
			},
		},
		{
			name: "valid: app with selector",
			config: promotion.Config{
				"apps": []promotion.Config{
					{
						"selector": promotion.Config{
							"matchLabels": promotion.Config{"app": "test"},
						},
						"waitFor": []string{"health"},
					},
				},
			},
		},
	}

	r := newArgocdWaiter(promotion.StepRunnerCapabilities{})
	runner, ok := r.(*argocdWaiter)
	require.True(t, ok)
	runValidationTests(t, runner.convert, tests)
}

func Test_argocdWaiter_run(t *testing.T) {
	testCases := []struct {
		name       string
		runner     *argocdWaiter
		stepCtx    *promotion.StepContext
		stepCfg    builtin.ArgoCDWaitConfig
		assertions func(*testing.T, promotion.StepResult, error)
	}{
		{
			name: "argocd client is nil",
			runner: &argocdWaiter{
				argocdClient: nil,
			},
			stepCtx: &promotion.StepContext{},
			stepCfg: builtin.ArgoCDWaitConfig{
				Apps: []builtin.ArgoCDAppWait{{Name: "my-app"}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, res.Status)
				require.ErrorContains(t, err, "Argo CD integration is disabled")
			},
		},
		{
			name: "error getting applications",
			runner: &argocdWaiter{
				argocdClient: fake.NewFakeClient(),
				getApplicationsFn: func(
					context.Context, client.Client, *promotion.StepContext,
					string, string, *builtin.ArgoCDAppSelector,
				) ([]*argocd.Application, error) {
					return nil, errors.New("something went wrong")
				},
			},
			stepCtx: &promotion.StepContext{},
			stepCfg: builtin.ArgoCDWaitConfig{
				Apps: []builtin.ArgoCDAppWait{{Name: "my-app"}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, res.Status)
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "operation in progress returns Running",
			runner: &argocdWaiter{
				argocdClient: fake.NewFakeClient(),
				getApplicationsFn: appsFn(&argocd.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-app", Namespace: "argocd",
					},
					Operation: &argocd.Operation{},
					Status: argocd.ApplicationStatus{
						Health: argocd.HealthStatus{
							Status: argocd.HealthStatusHealthy,
						},
						Sync: argocd.SyncStatus{
							Status: argocd.SyncStatusCodeSynced,
						},
					},
				}),
			},
			stepCtx: &promotion.StepContext{},
			stepCfg: builtin.ArgoCDWaitConfig{
				Apps: []builtin.ArgoCDAppWait{{Name: "my-app"}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusRunning, res.Status)
				assert.NotNil(t, res.RetryAfter)
			},
		},
		{
			name: "operation finished recently returns Running (cooldown)",
			runner: &argocdWaiter{
				argocdClient: fake.NewFakeClient(),
				getApplicationsFn: appsFn(&argocd.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-app", Namespace: "argocd",
					},
					Status: argocd.ApplicationStatus{
						Health: argocd.HealthStatus{
							Status: argocd.HealthStatusHealthy,
						},
						Sync: argocd.SyncStatus{
							Status: argocd.SyncStatusCodeSynced,
						},
						OperationState: &argocd.OperationState{
							Phase:      argocd.OperationSucceeded,
							FinishedAt: &metav1.Time{Time: time.Now()},
						},
					},
				}),
			},
			stepCtx: &promotion.StepContext{},
			stepCfg: builtin.ArgoCDWaitConfig{
				Apps: []builtin.ArgoCDAppWait{{Name: "my-app"}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusRunning, res.Status)
				require.NotNil(t, res.RetryAfter)
				assert.LessOrEqual(t, *res.RetryAfter, waitHealthCooldownDuration)
			},
		},
		{
			name: "cooldown bypassed when LastTransitionTime is after FinishedAt",
			runner: &argocdWaiter{
				argocdClient: fake.NewFakeClient(),
				getApplicationsFn: appsFn(&argocd.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-app", Namespace: "argocd",
					},
					Status: argocd.ApplicationStatus{
						Health: argocd.HealthStatus{
							Status:             argocd.HealthStatusHealthy,
							LastTransitionTime: &metav1.Time{Time: time.Now()},
						},
						Sync: argocd.SyncStatus{
							Status: argocd.SyncStatusCodeSynced,
						},
						OperationState: &argocd.OperationState{
							Phase:      argocd.OperationSucceeded,
							FinishedAt: &metav1.Time{Time: time.Now().Add(-5 * time.Second)},
						},
					},
				}),
			},
			stepCtx: &promotion.StepContext{},
			stepCfg: builtin.ArgoCDWaitConfig{
				Apps: []builtin.ArgoCDAppWait{{Name: "my-app"}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
			},
		},
		{
			name: "healthy and synced returns Succeeded",
			runner: &argocdWaiter{
				argocdClient: fake.NewFakeClient(),
				getApplicationsFn: appsFn(&argocd.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-app", Namespace: "argocd",
					},
					Status: argocd.ApplicationStatus{
						Health: argocd.HealthStatus{
							Status: argocd.HealthStatusHealthy,
						},
						Sync: argocd.SyncStatus{
							Status: argocd.SyncStatusCodeSynced,
						},
					},
				}),
			},
			stepCtx: &promotion.StepContext{},
			stepCfg: builtin.ArgoCDWaitConfig{
				Apps: []builtin.ArgoCDAppWait{{Name: "my-app"}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
			},
		},
		{
			name: "progressing returns Running",
			runner: &argocdWaiter{
				argocdClient: fake.NewFakeClient(),
				getApplicationsFn: appsFn(&argocd.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-app", Namespace: "argocd",
					},
					Status: argocd.ApplicationStatus{
						Health: argocd.HealthStatus{
							Status: argocd.HealthStatusProgressing,
						},
						Sync: argocd.SyncStatus{
							Status: argocd.SyncStatusCodeSynced,
						},
					},
				}),
			},
			stepCtx: &promotion.StepContext{},
			stepCfg: builtin.ArgoCDWaitConfig{
				Apps: []builtin.ArgoCDAppWait{{Name: "my-app"}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusRunning, res.Status)
			},
		},
		{
			name: "out of sync returns Running",
			runner: &argocdWaiter{
				argocdClient: fake.NewFakeClient(),
				getApplicationsFn: appsFn(&argocd.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-app", Namespace: "argocd",
					},
					Status: argocd.ApplicationStatus{
						Health: argocd.HealthStatus{
							Status: argocd.HealthStatusHealthy,
						},
						Sync: argocd.SyncStatus{
							Status: argocd.SyncStatusCodeOutOfSync,
						},
					},
				}),
			},
			stepCtx: &promotion.StepContext{},
			stepCfg: builtin.ArgoCDWaitConfig{
				Apps: []builtin.ArgoCDAppWait{{Name: "my-app"}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusRunning, res.Status)
			},
		},
		{
			name: "waitFor sync only ignores health",
			runner: &argocdWaiter{
				argocdClient: fake.NewFakeClient(),
				getApplicationsFn: appsFn(&argocd.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-app", Namespace: "argocd",
					},
					Status: argocd.ApplicationStatus{
						Health: argocd.HealthStatus{
							Status: argocd.HealthStatusDegraded,
						},
						Sync: argocd.SyncStatus{
							Status: argocd.SyncStatusCodeSynced,
						},
					},
				}),
			},
			stepCtx: &promotion.StepContext{},
			stepCfg: builtin.ArgoCDWaitConfig{
				Apps: []builtin.ArgoCDAppWait{
					{Name: "my-app", WaitFor: []builtin.WaitFor{builtin.Sync}},
				},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
			},
		},
		{
			name: "waitFor health and suspended, app is suspended",
			runner: &argocdWaiter{
				argocdClient: fake.NewFakeClient(),
				getApplicationsFn: appsFn(&argocd.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-app", Namespace: "argocd",
					},
					Status: argocd.ApplicationStatus{
						Health: argocd.HealthStatus{
							Status: argocd.HealthStatusSuspended,
						},
						Sync: argocd.SyncStatus{
							Status: argocd.SyncStatusCodeSynced,
						},
					},
				}),
			},
			stepCtx: &promotion.StepContext{},
			stepCfg: builtin.ArgoCDWaitConfig{
				Apps: []builtin.ArgoCDAppWait{
					{
						Name:    "my-app",
						WaitFor: []builtin.WaitFor{builtin.Health, builtin.Suspended},
					},
				},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
			},
		},
		{
			name: "health regression to degraded is terminal error",
			runner: &argocdWaiter{
				argocdClient: fake.NewFakeClient(),
				getApplicationsFn: appsFn(&argocd.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-app", Namespace: "argocd",
					},
					Status: argocd.ApplicationStatus{
						Health: argocd.HealthStatus{
							Status: argocd.HealthStatusDegraded,
						},
						Sync: argocd.SyncStatus{
							Status: argocd.SyncStatusCodeSynced,
						},
					},
				}),
			},
			stepCtx: &promotion.StepContext{
				Alias: "wait-step",
				SharedState: promotion.State{
					"wait-step": map[string]any{
						healthStatusKey: map[string]any{
							"argocd/my-app": "Progressing",
						},
					},
				},
			},
			stepCfg: builtin.ArgoCDWaitConfig{
				Apps: []builtin.ArgoCDAppWait{{Name: "my-app"}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, res.Status)
				require.Error(t, err)
				assert.True(t, promotion.IsTerminal(err))
				assert.Contains(t, err.Error(), "regressed")
			},
		},
		{
			name: "degraded without prior state is not terminal",
			runner: &argocdWaiter{
				argocdClient: fake.NewFakeClient(),
				getApplicationsFn: appsFn(&argocd.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-app", Namespace: "argocd",
					},
					Status: argocd.ApplicationStatus{
						Health: argocd.HealthStatus{
							Status: argocd.HealthStatusDegraded,
						},
						Sync: argocd.SyncStatus{
							Status: argocd.SyncStatusCodeSynced,
						},
					},
				}),
			},
			stepCtx: &promotion.StepContext{},
			stepCfg: builtin.ArgoCDWaitConfig{
				Apps: []builtin.ArgoCDAppWait{{Name: "my-app"}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusRunning, res.Status)
			},
		},
		{
			name: "error condition is terminal",
			runner: &argocdWaiter{
				argocdClient: fake.NewFakeClient(),
				getApplicationsFn: appsFn(&argocd.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-app", Namespace: "argocd",
					},
					Status: argocd.ApplicationStatus{
						Conditions: []argocd.ApplicationCondition{
							{
								Type:    argocd.ApplicationConditionInvalidSpecError,
								Message: "bad spec",
							},
						},
					},
				}),
			},
			stepCtx: &promotion.StepContext{},
			stepCfg: builtin.ArgoCDWaitConfig{
				Apps: []builtin.ArgoCDAppWait{{Name: "my-app"}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, res.Status)
				require.Error(t, err)
				assert.True(t, promotion.IsTerminal(err))
				assert.Contains(t, err.Error(), "bad spec")
			},
		},
		{
			name: "multiple apps, one not ready",
			runner: &argocdWaiter{
				argocdClient: fake.NewFakeClient(),
				getApplicationsFn: func(
					_ context.Context, _ client.Client,
					_ *promotion.StepContext,
					name, _ string,
					_ *builtin.ArgoCDAppSelector,
				) ([]*argocd.Application, error) {
					if name == "app-a" {
						return []*argocd.Application{{
							ObjectMeta: metav1.ObjectMeta{
								Name: "app-a", Namespace: "argocd",
							},
							Status: argocd.ApplicationStatus{
								Health: argocd.HealthStatus{
									Status: argocd.HealthStatusHealthy,
								},
								Sync: argocd.SyncStatus{
									Status: argocd.SyncStatusCodeSynced,
								},
							},
						}}, nil
					}
					return []*argocd.Application{{
						ObjectMeta: metav1.ObjectMeta{
							Name: "app-b", Namespace: "argocd",
						},
						Status: argocd.ApplicationStatus{
							Health: argocd.HealthStatus{
								Status: argocd.HealthStatusProgressing,
							},
							Sync: argocd.SyncStatus{
								Status: argocd.SyncStatusCodeSynced,
							},
						},
					}}, nil
				},
			},
			stepCtx: &promotion.StepContext{},
			stepCfg: builtin.ArgoCDWaitConfig{
				Apps: []builtin.ArgoCDAppWait{
					{Name: "app-a"},
					{Name: "app-b"},
				},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusRunning, res.Status)
			},
		},
		{
			name: "multiple apps, all ready",
			runner: &argocdWaiter{
				argocdClient: fake.NewFakeClient(),
				getApplicationsFn: func(
					_ context.Context, _ client.Client,
					_ *promotion.StepContext,
					_ string, _ string,
					_ *builtin.ArgoCDAppSelector,
				) ([]*argocd.Application, error) {
					return []*argocd.Application{{
						ObjectMeta: metav1.ObjectMeta{
							Name: "my-app", Namespace: "argocd",
						},
						Status: argocd.ApplicationStatus{
							Health: argocd.HealthStatus{
								Status: argocd.HealthStatusHealthy,
							},
							Sync: argocd.SyncStatus{
								Status: argocd.SyncStatusCodeSynced,
							},
						},
					}}, nil
				},
			},
			stepCtx: &promotion.StepContext{},
			stepCfg: builtin.ArgoCDWaitConfig{
				Apps: []builtin.ArgoCDAppWait{
					{Name: "app-a"},
					{Name: "app-b"},
				},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
			},
		},
		{
			name: "waitFor operation only, operation completed",
			runner: &argocdWaiter{
				argocdClient: fake.NewFakeClient(),
				getApplicationsFn: appsFn(&argocd.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-app", Namespace: "argocd",
					},
					Status: argocd.ApplicationStatus{
						Health: argocd.HealthStatus{
							Status: argocd.HealthStatusDegraded,
						},
						Sync: argocd.SyncStatus{
							Status: argocd.SyncStatusCodeOutOfSync,
						},
						OperationState: &argocd.OperationState{
							Phase:      argocd.OperationSucceeded,
							FinishedAt: &metav1.Time{Time: time.Now().Add(-time.Minute)},
						},
					},
				}),
			},
			stepCtx: &promotion.StepContext{},
			stepCfg: builtin.ArgoCDWaitConfig{
				Apps: []builtin.ArgoCDAppWait{
					{Name: "my-app", WaitFor: []builtin.WaitFor{builtin.Operation}},
				},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := tc.runner.run(context.Background(), tc.stepCtx, tc.stepCfg)
			tc.assertions(t, res, err)
		})
	}
}

// appsFn is a helper that returns a getApplicationsFn that always returns the
// given applications.
func appsFn(apps ...*argocd.Application) func(
	context.Context, client.Client, *promotion.StepContext,
	string, string, *builtin.ArgoCDAppSelector,
) ([]*argocd.Application, error) {
	return func(
		context.Context, client.Client, *promotion.StepContext,
		string, string, *builtin.ArgoCDAppSelector,
	) ([]*argocd.Application, error) {
		return apps, nil
	}
}
