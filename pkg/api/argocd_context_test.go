package api

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// The inputs to both extractors are untyped values decoded from a Promotion's
// status, which is to say they are whatever happens to be there. Every case
// below asserts on the extracted result, but the more important assertion is
// implicit: a type assertion that does not hold must yield nothing rather than
// panic, because a panic here takes down the Stage controller.

func TestArgoCDAppRefsToOutput(t *testing.T) {
	t.Parallel()

	refs := []ArgoCDAppRef{
		{Name: "app-a", Namespace: "argocd"},
		{Name: "app-b"},
	}
	output := ArgoCDAppRefsToOutput(refs)

	require.Equal(
		t,
		[]any{
			map[string]any{"name": "app-a", "namespace": "argocd"},
			map[string]any{"name": "app-b", "namespace": ""},
		},
		output,
	)

	// Step output becomes shared state, which is deep-copied between steps
	// with runtime.DeepCopyJSON. That panics on any value that is not one of
	// the types JSON decodes to, so the output must contain none.
	state := map[string]any{"step": map[string]any{ArgoCDAppsOutputKey: output}}
	require.NotPanics(t, func() { runtime.DeepCopyJSON(state) })

	// What a step reports must be what the annotation reads back out, both
	// before the state is serialized and after it round-trips through JSON.
	require.Equal(
		t,
		refs,
		argoCDAppRefsFromStepOutputs(
			[]kargoapi.PromotionStep{{Uses: "argocd-wait", As: "step"}},
			state,
		),
	)

	stateJSON, err := json.Marshal(state)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(stateJSON, &decoded))
	require.Equal(
		t,
		refs,
		argoCDAppRefsFromStepOutputs(
			[]kargoapi.PromotionStep{{Uses: "argocd-wait", As: "step"}},
			decoded,
		),
	)
}

func Test_argoCDAppRefsFromStepOutputs(t *testing.T) {
	t.Parallel()

	const waitStep = "argocd-wait"

	testCases := []struct {
		name     string
		steps    []kargoapi.PromotionStep
		state    map[string]any
		expected []ArgoCDAppRef
	}{
		{
			name: "nil steps and nil state",
		},
		{
			name:  "state is nil",
			steps: []kargoapi.PromotionStep{{Uses: waitStep, As: "wait"}},
		},
		{
			name:  "state has no entry for the step",
			steps: []kargoapi.PromotionStep{{Uses: waitStep, As: "wait"}},
			state: map[string]any{"other": map[string]any{}},
		},
		{
			name:  "step has no alias",
			steps: []kargoapi.PromotionStep{{Uses: waitStep}},
			state: map[string]any{
				"": map[string]any{
					ArgoCDAppsOutputKey: []any{map[string]any{"name": "app"}},
				},
			},
		},
		{
			// Any step may produce an "apps" output. Only Argo CD-aware steps
			// are permitted to contribute to the Argo CD context.
			name:  "step is not Argo CD-aware",
			steps: []kargoapi.PromotionStep{{Uses: "compose-output", As: "compose"}},
			state: map[string]any{
				"compose": map[string]any{
					ArgoCDAppsOutputKey: []any{map[string]any{"name": "app"}},
				},
			},
		},
		{
			name:  "step output is not an object",
			steps: []kargoapi.PromotionStep{{Uses: waitStep, As: "wait"}},
			state: map[string]any{"wait": "not-an-object"},
		},
		{
			name:  "step output is nil",
			steps: []kargoapi.PromotionStep{{Uses: waitStep, As: "wait"}},
			state: map[string]any{"wait": nil},
		},
		{
			name:  "step output is a list",
			steps: []kargoapi.PromotionStep{{Uses: waitStep, As: "wait"}},
			state: map[string]any{"wait": []any{"a", "b"}},
		},
		{
			name:  "step output has no apps key",
			steps: []kargoapi.PromotionStep{{Uses: waitStep, As: "wait"}},
			state: map[string]any{
				"wait": map[string]any{"healthStatus": map[string]any{}},
			},
		},
		{
			name:  "apps is not a list",
			steps: []kargoapi.PromotionStep{{Uses: waitStep, As: "wait"}},
			state: map[string]any{
				"wait": map[string]any{ArgoCDAppsOutputKey: "not-a-list"},
			},
		},
		{
			name:  "apps is nil",
			steps: []kargoapi.PromotionStep{{Uses: waitStep, As: "wait"}},
			state: map[string]any{
				"wait": map[string]any{ArgoCDAppsOutputKey: nil},
			},
		},
		{
			name:  "apps is an object",
			steps: []kargoapi.PromotionStep{{Uses: waitStep, As: "wait"}},
			state: map[string]any{
				"wait": map[string]any{
					ArgoCDAppsOutputKey: map[string]any{"name": "app"},
				},
			},
		},
		{
			name:  "apps is an empty list",
			steps: []kargoapi.PromotionStep{{Uses: waitStep, As: "wait"}},
			state: map[string]any{
				"wait": map[string]any{ArgoCDAppsOutputKey: []any{}},
			},
		},
		{
			name:  "apps entries are not objects",
			steps: []kargoapi.PromotionStep{{Uses: waitStep, As: "wait"}},
			state: map[string]any{
				"wait": map[string]any{
					ArgoCDAppsOutputKey: []any{nil, "app", 42, true, []any{"nested"}},
				},
			},
		},
		{
			// State is normally decoded from JSON, so it holds only JSON types.
			// Should a caller ever pass state that has not been round-tripped
			// through JSON, the Go types the steps actually emit must not be
			// mistaken for something extractable, nor cause a panic.
			name:  "apps is a list of concrete Go structs",
			steps: []kargoapi.PromotionStep{{Uses: waitStep, As: "wait"}},
			state: map[string]any{
				"wait": map[string]any{
					ArgoCDAppsOutputKey: []ArgoCDAppRef{{Name: "app", Namespace: "argocd"}},
				},
			},
		},
		{
			name:  "app fields are not strings",
			steps: []kargoapi.PromotionStep{{Uses: waitStep, As: "wait"}},
			state: map[string]any{
				"wait": map[string]any{
					ArgoCDAppsOutputKey: []any{
						map[string]any{"name": 42, "namespace": true},
						map[string]any{"name": nil, "namespace": []any{}},
					},
				},
			},
			// Unusable fields decode to their zero value rather than being
			// dropped here. Refs without a name are dropped by
			// dedupeArgoCDAppRefs.
			expected: []ArgoCDAppRef{{}, {}},
		},
		{
			name:  "app object is empty",
			steps: []kargoapi.PromotionStep{{Uses: waitStep, As: "wait"}},
			state: map[string]any{
				"wait": map[string]any{ArgoCDAppsOutputKey: []any{map[string]any{}}},
			},
			expected: []ArgoCDAppRef{{}},
		},
		{
			name:  "app has extra fields",
			steps: []kargoapi.PromotionStep{{Uses: waitStep, As: "wait"}},
			state: map[string]any{
				"wait": map[string]any{
					ArgoCDAppsOutputKey: []any{map[string]any{
						"name":      "app",
						"namespace": "argocd",
						"extra":     map[string]any{"deeply": []any{"nested"}},
					}},
				},
			},
			expected: []ArgoCDAppRef{{Name: "app", Namespace: "argocd"}},
		},
		{
			name:  "usable entries survive alongside unusable ones",
			steps: []kargoapi.PromotionStep{{Uses: waitStep, As: "wait"}},
			state: map[string]any{
				"wait": map[string]any{
					ArgoCDAppsOutputKey: []any{
						"junk",
						map[string]any{"name": "app", "namespace": "argocd"},
						nil,
					},
				},
			},
			expected: []ArgoCDAppRef{{Name: "app", Namespace: "argocd"}},
		},
		{
			name: "a broken step does not prevent extraction from others",
			steps: []kargoapi.PromotionStep{
				{Uses: "argocd-update", As: "update"},
				{Uses: waitStep, As: "wait"},
			},
			state: map[string]any{
				"update": "not-an-object",
				"wait": map[string]any{
					ArgoCDAppsOutputKey: []any{
						map[string]any{"name": "app", "namespace": "argocd"},
					},
				},
			},
			expected: []ArgoCDAppRef{{Name: "app", Namespace: "argocd"}},
		},
		{
			name: "apps from multiple steps, in step order",
			steps: []kargoapi.PromotionStep{
				{Uses: "argocd-update", As: "update"},
				{Uses: waitStep, As: "task-1::wait"},
			},
			state: map[string]any{
				"update": map[string]any{
					ArgoCDAppsOutputKey: []any{
						map[string]any{"name": "app-a", "namespace": "argocd"},
					},
				},
				"task-1::wait": map[string]any{
					ArgoCDAppsOutputKey: []any{
						map[string]any{"name": "app-b", "namespace": "other"},
					},
				},
			},
			expected: []ArgoCDAppRef{
				{Name: "app-a", Namespace: "argocd"},
				{Name: "app-b", Namespace: "other"},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(
				t,
				testCase.expected,
				argoCDAppRefsFromStepOutputs(testCase.steps, testCase.state),
			)
		})
	}
}

func Test_argoCDAppRefsFromHealthChecks(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		healthChecks []kargoapi.HealthCheckStep
		expected     []ArgoCDAppRef
	}{
		{
			name: "nil health checks",
		},
		{
			name:         "nil config",
			healthChecks: []kargoapi.HealthCheckStep{{Uses: "argocd-update"}},
		},
		{
			name: "empty raw config",
			healthChecks: []kargoapi.HealthCheckStep{{
				Uses:   "argocd-update",
				Config: &apiextensionsv1.JSON{},
			}},
		},
		{
			name: "unparsable raw config",
			healthChecks: []kargoapi.HealthCheckStep{{
				Uses:   "argocd-update",
				Config: &apiextensionsv1.JSON{Raw: []byte(`{"apps": [`)},
			}},
		},
		{
			name: "config is not an object",
			healthChecks: []kargoapi.HealthCheckStep{{
				Uses:   "argocd-update",
				Config: &apiextensionsv1.JSON{Raw: []byte(`["apps"]`)},
			}},
		},
		{
			name: "config has no apps key",
			healthChecks: []kargoapi.HealthCheckStep{{
				Uses:   "argocd-update",
				Config: &apiextensionsv1.JSON{Raw: []byte(`{"other":"value"}`)},
			}},
		},
		{
			name: "apps is not a list",
			healthChecks: []kargoapi.HealthCheckStep{{
				Uses:   "argocd-update",
				Config: &apiextensionsv1.JSON{Raw: []byte(`{"apps":"not-a-list"}`)},
			}},
		},
		{
			name: "apps entries are not objects",
			healthChecks: []kargoapi.HealthCheckStep{{
				Uses:   "argocd-update",
				Config: &apiextensionsv1.JSON{Raw: []byte(`{"apps":[null,"app",42,true,[]]}`)},
			}},
		},
		{
			name: "app fields are not strings",
			healthChecks: []kargoapi.HealthCheckStep{{
				Uses:   "argocd-update",
				Config: &apiextensionsv1.JSON{Raw: []byte(`{"apps":[{"name":42,"namespace":null}]}`)},
			}},
			expected: []ArgoCDAppRef{{}},
		},
		{
			// The shape an argocd-update step's health check criteria have
			// always had. Extra fields are ignored.
			name: "argocd-update health check criteria",
			healthChecks: []kargoapi.HealthCheckStep{{
				Uses: "argocd-update",
				Config: &apiextensionsv1.JSON{Raw: []byte(
					`{"apps":[{"name":"app","namespace":"argocd","desiredRevisions":["abc123"]}]}`,
				)},
			}},
			expected: []ArgoCDAppRef{{Name: "app", Namespace: "argocd"}},
		},
		{
			// Namespace is omitempty in the health check criteria, so criteria
			// recorded before this field was consistently populated may lack
			// it. The ref must still be usable.
			name: "argocd-update health check criteria without a namespace",
			healthChecks: []kargoapi.HealthCheckStep{{
				Uses:   "argocd-update",
				Config: &apiextensionsv1.JSON{Raw: []byte(`{"apps":[{"name":"app"}]}`)},
			}},
			expected: []ArgoCDAppRef{{Name: "app"}},
		},
		{
			name: "a broken health check does not prevent extraction from others",
			healthChecks: []kargoapi.HealthCheckStep{
				{
					Uses:   "argocd-update",
					Config: &apiextensionsv1.JSON{Raw: []byte(`{"apps":{}}`)},
				},
				{
					Uses: "argocd-update",
					Config: &apiextensionsv1.JSON{Raw: []byte(
						`{"apps":[{"name":"app","namespace":"argocd"}]}`,
					)},
				},
			},
			expected: []ArgoCDAppRef{{Name: "app", Namespace: "argocd"}},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(
				t,
				testCase.expected,
				argoCDAppRefsFromHealthChecks(testCase.healthChecks),
			)
		})
	}
}

func Test_dedupeArgoCDAppRefs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		refs     []ArgoCDAppRef
		expected []ArgoCDAppRef
	}{
		{
			name:     "nil refs",
			expected: []ArgoCDAppRef{},
		},
		{
			// A ref whose name could not be extracted would break the UI's
			// parsing of the annotation for the Stage as a whole.
			name:     "refs without a name are dropped",
			refs:     []ArgoCDAppRef{{}, {Namespace: "argocd"}},
			expected: []ArgoCDAppRef{},
		},
		{
			name:     "a ref without a namespace is kept",
			refs:     []ArgoCDAppRef{{Name: "app"}},
			expected: []ArgoCDAppRef{{Name: "app"}},
		},
		{
			name: "duplicates are dropped, order of first occurrence is kept",
			refs: []ArgoCDAppRef{
				{Name: "app-b", Namespace: "argocd"},
				{Name: "app-a", Namespace: "argocd"},
				{Name: "app-b", Namespace: "argocd"},
			},
			expected: []ArgoCDAppRef{
				{Name: "app-b", Namespace: "argocd"},
				{Name: "app-a", Namespace: "argocd"},
			},
		},
		{
			name: "same name in different namespaces are distinct",
			refs: []ArgoCDAppRef{
				{Name: "app", Namespace: "argocd"},
				{Name: "app", Namespace: "other"},
				{Name: "app"},
			},
			expected: []ArgoCDAppRef{
				{Name: "app", Namespace: "argocd"},
				{Name: "app", Namespace: "other"},
				{Name: "app"},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, testCase.expected, dedupeArgoCDAppRefs(testCase.refs))
		})
	}
}
