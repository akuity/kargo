package builtin

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	argocd "github.com/akuity/kargo/pkg/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

// argocdScheme is a runtime scheme with Argo CD types registered, used for
// fake clients that need to store Argo CD Application objects.
var argocdScheme = func() *runtime.Scheme {
	s := runtime.NewScheme()
	if err := argocd.AddToScheme(s); err != nil {
		panic(err)
	}
	return s
}()

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
			name: "app entry has neither name nor selector",
			config: promotion.Config{
				"apps": []promotion.Config{
					{},
				},
			},
			expectedProblems: []string{
				"apps.0: Must validate one and only one schema",
			},
		},
		{
			name: "app entry has both name and selector",
			config: promotion.Config{
				"apps": []promotion.Config{
					{
						"name": "my-app",
						"selector": promotion.Config{
							"matchLabels": map[string]string{"env": "prod"},
						},
					},
				},
			},
			expectedProblems: []string{
				"apps.0: Must validate one and only one schema",
			},
		},
		{
			name: "waitFor has invalid value",
			config: promotion.Config{
				"apps": []promotion.Config{
					{
						"name":    "my-app",
						"waitFor": []string{"invalid"},
					},
				},
			},
			expectedProblems: []string{
				"apps.0.waitFor.0: apps.0.waitFor.0 must be one of the following",
			},
		},
		{
			name: "valid config with name",
			config: promotion.Config{
				"apps": []promotion.Config{
					{"name": "my-app"},
				},
			},
			expectedProblems: nil,
		},
		{
			name: "valid config with selector",
			config: promotion.Config{
				"apps": []promotion.Config{
					{
						"selector": promotion.Config{
							"matchLabels": map[string]string{"env": "prod"},
						},
					},
				},
			},
			expectedProblems: nil,
		},
		{
			name: "valid config with waitFor",
			config: promotion.Config{
				"apps": []promotion.Config{
					{
						"name":    "my-app",
						"waitFor": []string{"health", "sync"},
					},
				},
			},
			expectedProblems: nil,
		},
		{
			name: "degradedGracePeriod is not a duration",
			config: promotion.Config{
				"apps":                []promotion.Config{{"name": "my-app"}},
				"degradedGracePeriod": "soon",
			},
			expectedProblems: []string{
				"degradedGracePeriod: Does not match pattern",
			},
		},
		{
			name: "valid config with degradedGracePeriod",
			config: promotion.Config{
				"apps":                []promotion.Config{{"name": "my-app"}},
				"degradedGracePeriod": "1m30s",
			},
			expectedProblems: nil,
		},
	}
	runner := &argocdWaiter{}
	runner.schemaLoader = getConfigSchemaLoader(stepKindArgoCDWait)
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
					context.Context, client.Client,
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
			name: "checkAppReadiness terminal error stops loop",
			runner: &argocdWaiter{
				argocdClient: fake.NewFakeClient(),
				getApplicationsFn: appsFn(&argocd.Application{
					ObjectMeta: metav1.ObjectMeta{Name: "my-app", Namespace: "argocd"},
				}),
				checkAppReadinessFn: func(
					context.Context, *argocd.Application,
					[]builtin.WaitFor, string,
				) (bool, string, error) {
					return false, "", &promotion.TerminalError{
						Err: errors.New("bad condition"),
					}
				},
			},
			stepCtx: &promotion.StepContext{},
			stepCfg: builtin.ArgoCDWaitConfig{
				Apps: []builtin.ArgoCDAppWait{{Name: "my-app"}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, res.Status)
				require.True(t, promotion.IsTerminal(err))
				require.ErrorContains(t, err, "bad condition")
				// Applications resolved before bailing are still reported.
				assert.Equal(
					t,
					api.ArgoCDAppRefsToOutput(
						[]api.ArgoCDAppRef{{Name: "my-app", Namespace: "argocd"}},
					),
					res.Output[api.ArgoCDAppsOutputKey],
				)
			},
		},
		{
			name: "health status tracked in output",
			runner: &argocdWaiter{
				argocdClient: fake.NewFakeClient(),
				getApplicationsFn: appsFn(&argocd.Application{
					ObjectMeta: metav1.ObjectMeta{Name: "my-app", Namespace: "argocd"},
				}),
				checkAppReadinessFn: func(
					context.Context, *argocd.Application,
					[]builtin.WaitFor, string,
				) (bool, string, error) {
					return false, "Progressing", nil
				},
			},
			stepCtx: &promotion.StepContext{},
			stepCfg: builtin.ArgoCDWaitConfig{
				Apps: []builtin.ArgoCDAppWait{{Name: "my-app"}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusRunning, res.Status)
				statuses, ok := res.Output[healthStatusKey].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "Progressing", statuses["argocd/my-app"])
			},
		},
		{
			name: "multiple apps, one not ready returns Running",
			runner: &argocdWaiter{
				argocdClient: fake.NewFakeClient(),
				getApplicationsFn: func(
					_ context.Context, _ client.Client,
					name, _ string, _ *builtin.ArgoCDAppSelector,
				) ([]*argocd.Application, error) {
					return []*argocd.Application{{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "argocd"}}}, nil
				},
				checkAppReadinessFn: func(
					_ context.Context, app *argocd.Application,
					_ []builtin.WaitFor, _ string,
				) (bool, string, error) {
					return app.Name == "app-a", "", nil
				},
			},
			stepCtx: &promotion.StepContext{},
			stepCfg: builtin.ArgoCDWaitConfig{
				Apps: []builtin.ArgoCDAppWait{{Name: "app-a"}, {Name: "app-b"}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusRunning, res.Status)
				assert.Nil(t, res.RetryAfter)
				assert.Equal(
					t,
					api.ArgoCDAppRefsToOutput([]api.ArgoCDAppRef{
						{Name: "app-a", Namespace: "argocd"},
						{Name: "app-b", Namespace: "argocd"},
					}),
					res.Output[api.ArgoCDAppsOutputKey],
				)
			},
		},
		{
			name: "all apps ready returns Succeeded",
			runner: &argocdWaiter{
				argocdClient: fake.NewFakeClient(),
				getApplicationsFn: appsFn(&argocd.Application{
					ObjectMeta: metav1.ObjectMeta{Name: "my-app", Namespace: "argocd"},
				}),
				checkAppReadinessFn: func(
					context.Context, *argocd.Application,
					[]builtin.WaitFor, string,
				) (bool, string, error) {
					return true, "Healthy", nil
				},
			},
			stepCtx: &promotion.StepContext{},
			stepCfg: builtin.ArgoCDWaitConfig{
				Apps: []builtin.ArgoCDAppWait{{Name: "my-app"}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
				assert.Equal(
					t,
					api.ArgoCDAppRefsToOutput(
						[]api.ArgoCDAppRef{{Name: "my-app", Namespace: "argocd"}},
					),
					res.Output[api.ArgoCDAppsOutputKey],
				)
			},
		},
		{
			name: "previous health statuses passed to checkAppReadiness",
			runner: &argocdWaiter{
				argocdClient: fake.NewFakeClient(),
				getApplicationsFn: appsFn(&argocd.Application{
					ObjectMeta: metav1.ObjectMeta{Name: "my-app", Namespace: "argocd"},
				}),
				checkAppReadinessFn: func(
					_ context.Context, _ *argocd.Application,
					_ []builtin.WaitFor, prevStatus string,
				) (bool, string, error) {
					assert.Equal(t, "Progressing", prevStatus)
					return true, "Healthy", nil
				},
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
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
			},
		},
		{
			name: "invalid degradedGracePeriod fails",
			runner: &argocdWaiter{
				argocdClient: fake.NewFakeClient(),
			},
			stepCtx: &promotion.StepContext{},
			stepCfg: builtin.ArgoCDWaitConfig{
				Apps:                []builtin.ArgoCDAppWait{{Name: "my-app"}},
				DegradedGracePeriod: "1x",
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				assert.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)
				require.True(t, promotion.IsTerminal(err))
				require.ErrorContains(t, err, "error parsing degradedGracePeriod")
			},
		},
		{
			name: "health regression without grace period is terminal",
			runner: &argocdWaiter{
				argocdClient:        fake.NewFakeClient(),
				getApplicationsFn:   appsFn(testWaitApp()),
				checkAppReadinessFn: regressesFromProgressing(t),
			},
			stepCtx: waitStepCtx(map[string]any{
				healthStatusKey: map[string]any{"argocd/my-app": "Progressing"},
			}),
			stepCfg: builtin.ArgoCDWaitConfig{
				Apps: []builtin.ArgoCDAppWait{{Name: "my-app"}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, res.Status)
				require.True(t, promotion.IsTerminal(err))
				require.ErrorContains(t, err, "regressed from Progressing to Degraded")
				assert.Empty(t, res.Output[degradedSinceKey])
			},
		},
		{
			name: "health regression within grace period keeps waiting",
			runner: &argocdWaiter{
				argocdClient:        fake.NewFakeClient(),
				getApplicationsFn:   appsFn(testWaitApp()),
				checkAppReadinessFn: regressesFromProgressing(t),
			},
			stepCtx: waitStepCtx(map[string]any{
				healthStatusKey: map[string]any{"argocd/my-app": "Progressing"},
			}),
			stepCfg: builtin.ArgoCDWaitConfig{
				Apps:                []builtin.ArgoCDAppWait{{Name: "my-app"}},
				DegradedGracePeriod: "1m",
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusRunning, res.Status)
				require.NotNil(t, res.RetryAfter)
				assert.InDelta(t, time.Minute, *res.RetryAfter, float64(5*time.Second))
				statuses, ok := res.Output[healthStatusKey].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "Degraded", statuses["argocd/my-app"])
				regressions, ok := res.Output[degradedSinceKey].(map[string]any)
				require.True(t, ok)
				regression, ok := regressions["argocd/my-app"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "Progressing", regression[degradedSinceFromKey])
			},
		},
		{
			name: "health regression still within grace period keeps original start",
			runner: &argocdWaiter{
				argocdClient:        fake.NewFakeClient(),
				getApplicationsFn:   appsFn(testWaitApp()),
				checkAppReadinessFn: regressesFromProgressing(t),
			},
			stepCtx: waitStepCtx(map[string]any{
				healthStatusKey: map[string]any{"argocd/my-app": "Degraded"},
				degradedSinceKey: map[string]any{
					"argocd/my-app": healthRegression{
						since: time.Now().Add(-40 * time.Second),
						from:  "Progressing",
					}.toOutput(),
				},
			}),
			stepCfg: builtin.ArgoCDWaitConfig{
				Apps:                []builtin.ArgoCDAppWait{{Name: "my-app"}},
				DegradedGracePeriod: "1m",
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusRunning, res.Status)
				require.NotNil(t, res.RetryAfter)
				assert.InDelta(t, 20*time.Second, *res.RetryAfter, float64(5*time.Second))
			},
		},
		{
			name: "health regression past grace period is terminal",
			runner: &argocdWaiter{
				argocdClient:        fake.NewFakeClient(),
				getApplicationsFn:   appsFn(testWaitApp()),
				checkAppReadinessFn: regressesFromProgressing(t),
			},
			stepCtx: waitStepCtx(map[string]any{
				healthStatusKey: map[string]any{"argocd/my-app": "Degraded"},
				degradedSinceKey: map[string]any{
					"argocd/my-app": healthRegression{
						since: time.Now().Add(-2 * time.Minute),
						from:  "Progressing",
					}.toOutput(),
				},
			}),
			stepCfg: builtin.ArgoCDWaitConfig{
				Apps:                []builtin.ArgoCDAppWait{{Name: "my-app"}},
				DegradedGracePeriod: "1m",
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, res.Status)
				require.True(t, promotion.IsTerminal(err))
				require.ErrorContains(
					t,
					err,
					"regressed from Progressing to Degraded and did not recover "+
						"within its degraded grace period of 1m0s",
				)
			},
		},
		{
			name: "grace period is timed separately for each app",
			runner: &argocdWaiter{
				argocdClient: fake.NewFakeClient(),
				getApplicationsFn: appsFn(
					&argocd.Application{
						ObjectMeta: metav1.ObjectMeta{Name: "app-a", Namespace: "argocd"},
					},
					&argocd.Application{
						ObjectMeta: metav1.ObjectMeta{Name: "app-b", Namespace: "argocd"},
					},
				),
				checkAppReadinessFn: regressesFromProgressing(t),
			},
			stepCtx: waitStepCtx(map[string]any{
				healthStatusKey: map[string]any{
					"argocd/app-a": "Degraded",
					"argocd/app-b": "Progressing",
				},
				degradedSinceKey: map[string]any{
					"argocd/app-a": healthRegression{
						since: time.Now().Add(-50 * time.Second),
						from:  "Progressing",
					}.toOutput(),
				},
			}),
			stepCfg: builtin.ArgoCDWaitConfig{
				Apps:                []builtin.ArgoCDAppWait{{Selector: &builtin.ArgoCDAppSelector{}}},
				DegradedGracePeriod: "1m",
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusRunning, res.Status)
				require.NotNil(t, res.RetryAfter)
				assert.InDelta(t, 10*time.Second, *res.RetryAfter, float64(5*time.Second))
				regressions, ok := res.Output[degradedSinceKey].(map[string]any)
				require.True(t, ok)
				regressionA, ok := regressions["argocd/app-a"].(map[string]any)
				require.True(t, ok)
				regressionB, ok := regressions["argocd/app-b"].(map[string]any)
				require.True(t, ok)
				assert.NotEqual(
					t,
					regressionA[degradedSinceTimeKey],
					regressionB[degradedSinceTimeKey],
				)
			},
		},
		{
			name: "recovery within grace period clears the regression",
			runner: &argocdWaiter{
				argocdClient:      fake.NewFakeClient(),
				getApplicationsFn: appsFn(testWaitApp()),
				checkAppReadinessFn: func(
					context.Context, *argocd.Application,
					[]builtin.WaitFor, string,
				) (bool, string, error) {
					return true, "Healthy", nil
				},
			},
			stepCtx: waitStepCtx(map[string]any{
				healthStatusKey: map[string]any{"argocd/my-app": "Degraded"},
				degradedSinceKey: map[string]any{
					"argocd/my-app": healthRegression{
						since: time.Now().Add(-10 * time.Second),
						from:  "Progressing",
					}.toOutput(),
				},
			}),
			stepCfg: builtin.ArgoCDWaitConfig{
				Apps:                []builtin.ArgoCDAppWait{{Name: "my-app"}},
				DegradedGracePeriod: "1m",
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
				assert.Empty(t, res.Output[degradedSinceKey])
			},
		},
		{
			name: "grace period does not apply to other terminal errors",
			runner: &argocdWaiter{
				argocdClient:      fake.NewFakeClient(),
				getApplicationsFn: appsFn(testWaitApp()),
				checkAppReadinessFn: func(
					context.Context, *argocd.Application,
					[]builtin.WaitFor, string,
				) (bool, string, error) {
					return false, "Degraded", &promotion.TerminalError{
						Err: errors.New("bad condition"),
					}
				},
			},
			stepCtx: &promotion.StepContext{},
			stepCfg: builtin.ArgoCDWaitConfig{
				Apps:                []builtin.ArgoCDAppWait{{Name: "my-app"}},
				DegradedGracePeriod: "1m",
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, res.Status)
				require.True(t, promotion.IsTerminal(err))
				require.ErrorContains(t, err, "bad condition")
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := tc.runner.run(context.Background(), tc.stepCtx, tc.stepCfg)
			requireJSONNativeOutput(t, res)
			tc.assertions(t, res, err)
		})
	}
}

// requireJSONNativeOutput asserts that a step's output can serve as shared
// state. The engine deep-copies shared state with runtime.DeepCopyJSON before
// every subsequent step, which panics on any value that is not one of the
// types JSON decodes to -- taking the Promotion controller down with it.
func requireJSONNativeOutput(t *testing.T, res promotion.StepResult) {
	t.Helper()
	require.NotPanics(t, func() {
		runtime.DeepCopyJSON(map[string]any{"output": res.Output})
	})
}

func Test_argocdWaiter_checkAppReadiness(t *testing.T) {
	t.Parallel()

	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	testCases := []struct {
		name             string
		app              *argocd.Application
		storedApp        *argocd.Application // pre-loaded into fake client for refresh patch
		waitFor          []builtin.WaitFor
		prevHealthStatus string
		assertions       func(*testing.T, bool, string, error)
	}{
		{
			name: "error condition is terminal",
			app: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Conditions: []argocd.ApplicationCondition{{
						Type:    argocd.ApplicationConditionInvalidSpecError,
						Message: "bad spec",
					}},
				},
			},
			waitFor: defaultWaitFor,
			assertions: func(t *testing.T, ready bool, _ string, err error) {
				assert.False(t, ready)
				require.True(t, promotion.IsTerminal(err))
				require.ErrorContains(t, err, "bad spec")
			},
		},
		{
			name: "operation in progress returns not ready",
			app: &argocd.Application{
				Operation: &argocd.Operation{},
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{Status: argocd.HealthStatusHealthy},
					Sync:   argocd.SyncStatus{Status: argocd.SyncStatusCodeSynced},
				},
			},
			waitFor: defaultWaitFor,
			assertions: func(t *testing.T, ready bool, _ string, err error) {
				assert.False(t, ready)
				require.NoError(t, err)
			},
		},
		{
			name: "reconciledAt nil after operation requests refresh and returns not ready",
			app: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{Name: "my-app", Namespace: "argocd"},
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{Status: argocd.HealthStatusHealthy},
					Sync:   argocd.SyncStatus{Status: argocd.SyncStatusCodeSynced},
					OperationState: &argocd.OperationState{
						Phase:      argocd.OperationSucceeded,
						FinishedAt: &metav1.Time{Time: now},
					},
					// ReconciledAt is nil
				},
			},
			storedApp: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{Name: "my-app", Namespace: "argocd"},
			},
			waitFor: defaultWaitFor,
			assertions: func(t *testing.T, ready bool, _ string, err error) {
				assert.False(t, ready)
				require.NoError(t, err)
			},
		},
		{
			name: "reconciledAt before finishedAt requests refresh and returns not ready",
			app: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{Name: "my-app", Namespace: "argocd"},
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{Status: argocd.HealthStatusHealthy},
					Sync:   argocd.SyncStatus{Status: argocd.SyncStatusCodeSynced},
					OperationState: &argocd.OperationState{
						Phase:      argocd.OperationSucceeded,
						FinishedAt: &metav1.Time{Time: now.Add(10 * time.Second)},
					},
					ReconciledAt: &metav1.Time{Time: now.Add(5 * time.Second)},
				},
			},
			storedApp: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{Name: "my-app", Namespace: "argocd"},
			},
			waitFor: defaultWaitFor,
			assertions: func(t *testing.T, ready bool, _ string, err error) {
				assert.False(t, ready)
				require.NoError(t, err)
			},
		},
		{
			name: "reconciledAt equal to finishedAt trusts health",
			app: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{Status: argocd.HealthStatusHealthy},
					Sync:   argocd.SyncStatus{Status: argocd.SyncStatusCodeSynced},
					OperationState: &argocd.OperationState{
						Phase:      argocd.OperationSucceeded,
						FinishedAt: &metav1.Time{Time: now},
					},
					ReconciledAt: &metav1.Time{Time: now},
				},
			},
			waitFor: defaultWaitFor,
			assertions: func(t *testing.T, ready bool, _ string, err error) {
				assert.True(t, ready)
				require.NoError(t, err)
			},
		},
		{
			name: "healthy and synced returns ready",
			app: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{Status: argocd.HealthStatusHealthy},
					Sync:   argocd.SyncStatus{Status: argocd.SyncStatusCodeSynced},
				},
			},
			waitFor: defaultWaitFor,
			assertions: func(t *testing.T, ready bool, healthStatus string, err error) {
				assert.True(t, ready)
				assert.Equal(t, string(argocd.HealthStatusHealthy), healthStatus)
				require.NoError(t, err)
			},
		},
		{
			name: "progressing returns not ready",
			app: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{Status: argocd.HealthStatusProgressing},
					Sync:   argocd.SyncStatus{Status: argocd.SyncStatusCodeSynced},
				},
			},
			waitFor: defaultWaitFor,
			assertions: func(t *testing.T, ready bool, _ string, err error) {
				assert.False(t, ready)
				require.NoError(t, err)
			},
		},
		{
			name: "out of sync returns not ready",
			app: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{Status: argocd.HealthStatusHealthy},
					Sync:   argocd.SyncStatus{Status: argocd.SyncStatusCodeOutOfSync},
				},
			},
			waitFor: defaultWaitFor,
			assertions: func(t *testing.T, ready bool, _ string, err error) {
				assert.False(t, ready)
				require.NoError(t, err)
			},
		},
		{
			name: "waitFor sync only — degraded health ignored",
			app: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{Status: argocd.HealthStatusDegraded},
					Sync:   argocd.SyncStatus{Status: argocd.SyncStatusCodeSynced},
				},
			},
			waitFor: []builtin.WaitFor{builtin.Sync},
			assertions: func(t *testing.T, ready bool, _ string, err error) {
				assert.True(t, ready)
				require.NoError(t, err)
			},
		},
		{
			name: "waitFor health and suspended — suspended satisfies health check",
			app: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{Status: argocd.HealthStatusSuspended},
					Sync:   argocd.SyncStatus{Status: argocd.SyncStatusCodeSynced},
				},
			},
			waitFor: []builtin.WaitFor{builtin.Health, builtin.Suspended},
			assertions: func(t *testing.T, ready bool, _ string, err error) {
				assert.True(t, ready)
				require.NoError(t, err)
			},
		},
		{
			name: "health regression to Degraded from prior non-Degraded is terminal",
			app: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{Status: argocd.HealthStatusDegraded},
					Sync:   argocd.SyncStatus{Status: argocd.SyncStatusCodeSynced},
				},
			},
			waitFor:          defaultWaitFor,
			prevHealthStatus: string(argocd.HealthStatusProgressing),
			assertions: func(t *testing.T, ready bool, _ string, err error) {
				assert.False(t, ready)
				require.True(t, promotion.IsTerminal(err))
				require.ErrorContains(t, err, "regressed")
				regressionErr, ok := asHealthRegressionError(err)
				require.True(t, ok)
				assert.Equal(t, string(argocd.HealthStatusProgressing), regressionErr.from)
			},
		},
		{
			name: "degraded with no prior health status is not terminal",
			app: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{Status: argocd.HealthStatusDegraded},
					Sync:   argocd.SyncStatus{Status: argocd.SyncStatusCodeSynced},
				},
			},
			waitFor: defaultWaitFor,
			assertions: func(t *testing.T, ready bool, _ string, err error) {
				assert.False(t, ready)
				require.NoError(t, err)
			},
		},
		{
			name: "waitFor operation only — operation completed, health/sync ignored",
			app: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{Status: argocd.HealthStatusDegraded},
					Sync:   argocd.SyncStatus{Status: argocd.SyncStatusCodeOutOfSync},
					OperationState: &argocd.OperationState{
						Phase:      argocd.OperationSucceeded,
						FinishedAt: &metav1.Time{Time: now.Add(-time.Minute)},
					},
				},
			},
			waitFor: []builtin.WaitFor{builtin.Operation},
			assertions: func(t *testing.T, ready bool, _ string, err error) {
				assert.True(t, ready)
				require.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var argocdClient client.Client
			if tc.storedApp != nil {
				argocdClient = fake.NewClientBuilder().
					WithScheme(argocdScheme).
					WithObjects(tc.storedApp).
					Build()
			} else {
				argocdClient = fake.NewFakeClient()
			}

			w := &argocdWaiter{argocdClient: argocdClient}
			ready, healthStatus, err := w.checkAppReadiness(
				context.Background(), tc.app, tc.waitFor, tc.prevHealthStatus,
			)
			tc.assertions(t, ready, healthStatus, err)
		})
	}
}

// testWaitApp returns the Application used by argocd-wait run tests.
func testWaitApp() *argocd.Application {
	return &argocd.Application{
		ObjectMeta: metav1.ObjectMeta{Name: "my-app", Namespace: "argocd"},
	}
}

// waitStepCtx returns a StepContext whose shared state holds the given output
// from a previous invocation of the step.
func waitStepCtx(prevOutput map[string]any) *promotion.StepContext {
	return &promotion.StepContext{
		Alias:       "wait-step",
		SharedState: promotion.State{"wait-step": prevOutput},
	}
}

// regressesFromProgressing returns a checkAppReadinessFn that reports a health
// regression from Progressing to Degraded, after asserting that it was called
// with Progressing as the previous health status.
func regressesFromProgressing(t *testing.T) func(
	context.Context, *argocd.Application,
	[]builtin.WaitFor, string,
) (bool, string, error) {
	return func(
		_ context.Context, app *argocd.Application,
		_ []builtin.WaitFor, prevHealthStatus string,
	) (bool, string, error) {
		assert.Equal(t, string(argocd.HealthStatusProgressing), prevHealthStatus)
		return false, "Degraded", &promotion.TerminalError{
			Err: &healthRegressionError{
				app:       app.Name,
				namespace: app.Namespace,
				from:      prevHealthStatus,
				to:        "Degraded",
			},
		}
	}
}

// appsFn is a helper that returns a getApplicationsFn that always returns the
// given applications.
func appsFn(apps ...*argocd.Application) func(
	context.Context, client.Client,
	string, string, *builtin.ArgoCDAppSelector,
) ([]*argocd.Application, error) {
	return func(
		context.Context, client.Client,
		string, string, *builtin.ArgoCDAppSelector,
	) ([]*argocd.Application, error) {
		return apps, nil
	}
}
