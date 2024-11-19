package directives

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
)

func TestSimpleEngine_Promote(t *testing.T) {
	tests := []struct {
		name        string
		promoCtx    PromotionContext
		steps       []PromotionStep
		interceptor interceptor.Funcs
		assertions  func(*testing.T, PromotionResult, error)
	}{
		{
			name: "successful promotion",
			promoCtx: PromotionContext{
				Project: "test-project",
				State:   State{"existing": "state"},
			},
			steps: []PromotionStep{
				{Kind: "success-step"},
			},
			assertions: func(t *testing.T, result PromotionResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseSucceeded, result.Status)
				assert.NotNil(t, result.State)
				assert.Equal(t, "state", result.State["existing"])
				assert.Equal(t, int64(0), result.CurrentStep)
			},
		},
		{
			name: "failed promotion",
			promoCtx: PromotionContext{
				Project: "test-project",
			},
			steps: []PromotionStep{
				{Kind: "error-step"},
			},
			assertions: func(t *testing.T, result PromotionResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseErrored, result.Status)
				assert.ErrorContains(t, err, "step execution failed")
			},
		},
		{
			name: "context cancellation",
			promoCtx: PromotionContext{
				Project: "test-project",
			},
			steps: []PromotionStep{
				{Kind: "context-waiter"},
			},
			assertions: func(t *testing.T, result PromotionResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseErrored, result.Status)
				assert.ErrorIs(t, err, context.Canceled)
			},
		},
		{
			name: "secrets retrieval failure",
			promoCtx: PromotionContext{
				Project: "test-project",
			},
			steps: []PromotionStep{
				{Kind: "success-step"},
			},
			interceptor: interceptor.Funcs{
				List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, result PromotionResult, err error) {
				assert.ErrorContains(t, err, "error listing Secrets for Project")
				assert.ErrorContains(t, err, "something went wrong")
				assert.Equal(t, kargoapi.PromotionPhaseErrored, result.Status)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			testRegistry := NewStepRunnerRegistry()
			testRegistry.RegisterPromotionStepRunner(
				&mockPromotionStepRunner{
					name:      "success-step",
					runResult: PromotionStepResult{Status: kargoapi.PromotionPhaseSucceeded},
				},
				&StepRunnerPermissions{},
			)
			testRegistry.RegisterPromotionStepRunner(
				&mockPromotionStepRunner{
					name:      "error-step",
					runResult: PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
					runErr:    errors.New("something went wrong"),
				},
				&StepRunnerPermissions{},
			)
			testRegistry.RegisterPromotionStepRunner(
				&mockPromotionStepRunner{
					name: "context-waiter",
					runFunc: func(ctx context.Context, _ *PromotionStepContext) (PromotionStepResult, error) {
						cancel() // Cancel context immediately
						<-ctx.Done()
						return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, ctx.Err()
					},
				},
				&StepRunnerPermissions{},
			)

			engine := &SimpleEngine{
				registry: testRegistry,
				kargoClient: fake.NewClientBuilder().
					WithInterceptorFuncs(tt.interceptor).
					Build(),
			}

			result, err := engine.Promote(ctx, tt.promoCtx, tt.steps)
			tt.assertions(t, result, err)
		})
	}
}

func TestSimpleEngine_executeSteps(t *testing.T) {
	tests := []struct {
		name       string
		promoCtx   PromotionContext
		steps      []PromotionStep
		assertions func(*testing.T, PromotionResult, error)
	}{
		{
			name: "execute all steps successfully",
			steps: []PromotionStep{
				{Kind: "success-step", Alias: "step1"},
				{Kind: "success-step", Alias: "step2"},
			},
			assertions: func(t *testing.T, result PromotionResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseSucceeded, result.Status)
				assert.Equal(t, int64(1), result.CurrentStep)

				// Verify state contains outputs from both steps
				assert.Equal(t, State{
					"step1": map[string]any{
						"attempts": int64(1),
						"key":      "value",
					},
					"step2": map[string]any{
						"attempts": int64(1),
						"key":      "value",
					},
				}, result.State)
			},
		},
		{
			name: "start from middle step",
			promoCtx: PromotionContext{
				StartFromStep: 1,
			},
			steps: []PromotionStep{
				{Kind: "error-step", Alias: "step1"}, // This step should be skipped
				{Kind: "success-step", Alias: "step2"},
			},
			assertions: func(t *testing.T, result PromotionResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseSucceeded, result.Status)
				assert.Equal(t, int64(1), result.CurrentStep)

				// Verify only second step output is in state
				assert.Equal(t, State{
					"step2": map[string]any{
						"attempts": int64(1),
						"key":      "value",
					},
				}, result.State)
			},
		},
		{
			name: "fail on step execution",
			steps: []PromotionStep{
				{Kind: "success-step", Alias: "step1"},
				{Kind: "error-step", Alias: "step2"},
			},
			assertions: func(t *testing.T, result PromotionResult, err error) {
				assert.ErrorContains(t, err, "something went wrong")
				assert.Equal(t, kargoapi.PromotionPhaseErrored, result.Status)
				assert.Equal(t, int64(1), result.CurrentStep)

				// Verify first step output is preserved in state
				assert.Equal(t, State{
					"step1": map[string]any{
						"attempts": int64(1),
						"key":      "value",
					},
					"step2": map[string]any{
						"attempts": int64(1),
					},
				}, result.State)
			},
		},
		{
			name: "fail on invalid step alias",
			steps: []PromotionStep{
				{Kind: "success-step", Alias: "step-1"},
			},
			assertions: func(t *testing.T, result PromotionResult, err error) {
				assert.ErrorContains(t, err, "is forbidden")
				assert.Equal(t, kargoapi.PromotionPhaseErrored, result.Status)
				assert.Equal(t, int64(0), result.CurrentStep)
			},
		},
		{
			name: "context cancellation",
			steps: []PromotionStep{
				{Kind: "context-waiter", Alias: "step1"},
			},
			assertions: func(t *testing.T, result PromotionResult, err error) {
				assert.ErrorIs(t, err, context.Canceled)
				assert.Equal(t, kargoapi.PromotionPhaseErrored, result.Status)
				assert.Equal(t, int64(0), result.CurrentStep)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			testRegistry := NewStepRunnerRegistry()
			testRegistry.RegisterPromotionStepRunner(
				&mockPromotionStepRunner{
					name: "success-step",
					runResult: PromotionStepResult{
						Status: kargoapi.PromotionPhaseSucceeded,
						Output: map[string]any{"key": "value"},
					},
				},
				&StepRunnerPermissions{},
			)
			testRegistry.RegisterPromotionStepRunner(
				&mockPromotionStepRunner{
					name:      "error-step",
					runResult: PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
					runErr:    errors.New("something went wrong"),
				},
				&StepRunnerPermissions{},
			)
			testRegistry.RegisterPromotionStepRunner(
				&mockPromotionStepRunner{
					name: "context-waiter",
					runFunc: func(ctx context.Context, _ *PromotionStepContext) (PromotionStepResult, error) {
						cancel()
						<-ctx.Done()
						return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, ctx.Err()
					},
				},
				&StepRunnerPermissions{},
			)

			engine := &SimpleEngine{
				registry:    testRegistry,
				kargoClient: fake.NewClientBuilder().Build(),
			}

			result, err := engine.executeSteps(ctx, tt.promoCtx, tt.steps, t.TempDir())
			tt.assertions(t, result, err)
		})
	}
}

func TestSimpleEngine_executeStep(t *testing.T) {
	tests := []struct {
		name       string
		promoCtx   PromotionContext
		step       PromotionStep
		assertions func(*testing.T, PromotionStepResult, error)
	}{
		{
			name: "successful step execution",
			step: PromotionStep{Kind: "success-step"},
			assertions: func(t *testing.T, result PromotionStepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseSucceeded, result.Status)

				// Verify attempts counter is added to output
				assert.Equal(t, int64(1), result.Output["attempts"])
				assert.Equal(t, "test-value", result.Output["key"])
			},
		},
		{
			name: "step execution with existing attempts",
			promoCtx: PromotionContext{
				State: State{"test-step": map[string]any{"attempts": 0}},
			},
			step: PromotionStep{Kind: "success-step", Alias: "test-step"},
			assertions: func(t *testing.T, result PromotionStepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseSucceeded, result.Status)

				// Verify attempts counter is incremented in output
				assert.Equal(t, int64(1), result.Output["attempts"])
				assert.Equal(t, "test-value", result.Output["key"])
			},
		},
		{
			name: "step execution failure",
			step: PromotionStep{Kind: "error-step"},
			assertions: func(t *testing.T, result PromotionStepResult, err error) {
				assert.ErrorContains(t, err, "failed to run step \"error-step\"")
				assert.ErrorContains(t, err, "something went wrong")
				assert.Equal(t, kargoapi.PromotionPhaseErrored, result.Status)

				// Verify attempts counter is still added to output even on failure
				assert.Equal(t, int64(1), result.Output["attempts"])
			},
		},
		{
			name: "unregistered step kind",
			step: PromotionStep{Kind: "unknown"},
			assertions: func(t *testing.T, result PromotionStepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseErrored, result.Status)
			},
		},
		{
			name: "max attempts exceeded",
			step: PromotionStep{Kind: "success-step", Alias: "test"},
			promoCtx: PromotionContext{
				State: State{"test": map[string]any{"attempts": 3}},
			},
			assertions: func(t *testing.T, result PromotionStepResult, err error) {
				assert.ErrorContains(t, err, "exceeded max attempts")
				assert.Equal(t, kargoapi.PromotionPhaseErrored, result.Status)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRegistry := NewStepRunnerRegistry()
			testRegistry.RegisterPromotionStepRunner(
				&mockPromotionStepRunner{
					name: "success-step",
					runResult: PromotionStepResult{
						Status: kargoapi.PromotionPhaseSucceeded,
						Output: map[string]any{"key": "test-value"},
					},
				},
				&StepRunnerPermissions{},
			)
			testRegistry.RegisterPromotionStepRunner(
				&mockPromotionStepRunner{
					name:      "error-step",
					runResult: PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
					runErr:    errors.New("something went wrong"),
				},
				&StepRunnerPermissions{},
			)

			engine := &SimpleEngine{
				registry:    testRegistry,
				kargoClient: fake.NewClientBuilder().Build(),
			}

			result, err := engine.executeStep(context.Background(), tt.promoCtx, tt.step, t.TempDir(), make(State))
			tt.assertions(t, result, err)
		})
	}
}

func TestSimpleEngine_preparePromotionStepContext(t *testing.T) {
	tests := []struct {
		name        string
		promoCtx    PromotionContext
		step        PromotionStep
		permissions StepRunnerPermissions
		assertions  func(*testing.T, *PromotionStepContext, error)
	}{
		{
			name: "successful context preparation",
			promoCtx: PromotionContext{
				Project:   "test-project",
				Stage:     "test-stage",
				UIBaseURL: "http://test",
			},
			step: PromotionStep{Kind: "test-step"},
			permissions: StepRunnerPermissions{
				AllowCredentialsDB: true,
				AllowKargoClient:   true,
				AllowArgoCDClient:  true,
			},
			assertions: func(t *testing.T, ctx *PromotionStepContext, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "test-project", ctx.Project)
				assert.Equal(t, "test-stage", ctx.Stage)
				assert.Equal(t, "http://test", ctx.UIBaseURL)
				assert.NotNil(t, ctx.CredentialsDB)
				assert.NotNil(t, ctx.KargoClient)
				assert.NotNil(t, ctx.ArgoCDClient)
			},
		},
		{
			name:        "permissions control client access",
			promoCtx:    PromotionContext{},
			step:        PromotionStep{Kind: "test-step"},
			permissions: StepRunnerPermissions{
				// Nothing allowed
			},
			assertions: func(t *testing.T, ctx *PromotionStepContext, err error) {
				assert.NoError(t, err)
				assert.Nil(t, ctx.CredentialsDB)
				assert.Nil(t, ctx.KargoClient)
				assert.Nil(t, ctx.ArgoCDClient)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRegistry := NewStepRunnerRegistry()
			testRegistry.RegisterPromotionStepRunner(
				&mockPromotionStepRunner{name: "test-step"},
				&tt.permissions,
			)

			engine := &SimpleEngine{
				registry:      testRegistry,
				kargoClient:   fake.NewClientBuilder().Build(),
				argoCDClient:  fake.NewClientBuilder().Build(),
				credentialsDB: &credentials.FakeDB{},
			}

			reg, _ := testRegistry.GetPromotionStepRunnerRegistration("test-step")
			stepCtx, err := engine.preparePromotionStepContext(
				context.Background(),
				tt.promoCtx,
				tt.step,
				t.TempDir(),
				make(State),
				reg,
			)
			tt.assertions(t, stepCtx, err)
		})
	}
}

func TestSimpleEngine_stepAlias(t *testing.T) {
	tests := []struct {
		name       string
		alias      string
		index      int64
		assertions func(*testing.T, string, error)
	}{
		{
			name:  "use provided alias",
			alias: "custom-step",
			assertions: func(t *testing.T, alias string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "custom-step", alias)
			},
		},
		{
			name:  "generate default alias",
			index: 42,
			assertions: func(t *testing.T, alias string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "step-42", alias)
			},
		},
		{
			name:  "reject reserved alias",
			alias: "step-1",
			assertions: func(t *testing.T, _ string, err error) {
				assert.ErrorContains(t, err, "forbidden")
			},
		},
		{
			name:  "trim whitespace",
			alias: "  step  ",
			assertions: func(t *testing.T, alias string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "step", alias)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &SimpleEngine{}
			alias, err := engine.stepAlias(tt.alias, tt.index)
			tt.assertions(t, alias, err)
		})
	}
}

func TestSimpleEngine_setupWorkDir(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		existingDir string
		assertions  func(*testing.T, string, error)
	}{
		{
			name:        "use existing directory",
			existingDir: tmpDir,
			assertions: func(t *testing.T, dir string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, tmpDir, dir)
			},
		},
		{
			name: "create new directory",
			assertions: func(t *testing.T, dir string, err error) {
				assert.NoError(t, err)
				assert.DirExists(t, dir)
				t.Cleanup(func() {
					_ = os.RemoveAll(dir)
				})
				assert.Contains(t, dir, "run-")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &SimpleEngine{}
			dir, err := engine.setupWorkDir(tt.existingDir)
			tt.assertions(t, dir, err)
		})
	}
}

func TestSimpleEngine_getProjectSecrets(t *testing.T) {
	tests := []struct {
		name        string
		project     string
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, map[string]map[string]string, error)
	}{
		{
			name:    "successful retrieval",
			project: "test-project",
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-project",
					},
					Data: map[string][]byte{
						"key1": []byte("value1"),
						"key2": []byte("value2"),
					},
				},
			},
			assertions: func(t *testing.T, secrets map[string]map[string]string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "value1", secrets["test-secret"]["key1"])
				assert.Equal(t, "value2", secrets["test-secret"]["key2"])
			},
		},
		{
			name:    "list error",
			project: "test-project",
			interceptor: interceptor.Funcs{
				List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
					return errors.New("list error")
				},
			},
			assertions: func(t *testing.T, _ map[string]map[string]string, err error) {
				assert.ErrorContains(t, err, "error listing Secrets")
			},
		},
		{
			name:    "no secrets",
			project: "empty-project",
			assertions: func(t *testing.T, secrets map[string]map[string]string, err error) {
				assert.NoError(t, err)
				assert.Empty(t, secrets)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &SimpleEngine{
				kargoClient: fake.NewClientBuilder().
					WithObjects(tt.objects...).
					WithInterceptorFuncs(tt.interceptor).
					Build(),
			}

			secrets, err := engine.getProjectSecrets(context.Background(), tt.project)
			tt.assertions(t, secrets, err)
		})
	}
}
