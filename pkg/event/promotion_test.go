package event

import (
	"strconv"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestNewPromotionCommon(t *testing.T) {
	testCases := map[string]struct {
		message         string
		actor           string
		promotion       *kargoapi.Promotion
		freight         *kargoapi.Freight
		verifyCommon    func(t *testing.T, common Common)
		verifyPromotion func(t *testing.T, promotion Promotion)
	}{
		"complete promotion with freight": {
			message: "test message",
			actor:   "test-actor",
			promotion: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-promotion",
					Namespace: "test-project",
					CreationTimestamp: metav1.Time{
						Time: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
					},
				},
				Spec: kargoapi.PromotionSpec{
					Stage: "test-stage",
				},
			},
			freight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-freight",
					CreationTimestamp: metav1.Time{
						Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					},
				},
				Alias: "v1.0.0",
			},
			verifyCommon: func(t *testing.T, common Common) {
				require.Equal(t, "test-project", common.Project)
				require.Equal(t, "test message", common.Message)
				require.Equal(t, ptr.To("test-actor"), common.Actor)
			},
			verifyPromotion: func(t *testing.T, promotion Promotion) {
				require.Equal(t, "test-promotion", promotion.Name)
				require.Equal(t, "test-stage", promotion.StageName)
				require.Equal(t, time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), promotion.CreateTime)
				require.NotNil(t, promotion.Freight)
				require.Equal(t, "test-freight", promotion.Freight.Name)
			},
		},
		"promotion with actor annotation": {
			message: "test message",
			actor:   "external-actor",
			promotion: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-promotion",
					Namespace: "test-project",
					CreationTimestamp: metav1.Time{
						Time: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
					},
					Annotations: map[string]string{
						kargoapi.AnnotationKeyCreateActor: "promotion-actor",
					},
				},
				Spec: kargoapi.PromotionSpec{
					Stage: "test-stage",
				},
			},
			freight: nil,
			verifyCommon: func(t *testing.T, common Common) {
				require.Equal(t, "test-project", common.Project)
				require.Equal(t, "test message", common.Message)
				require.Equal(t, ptr.To("promotion-actor"), common.Actor) // annotation takes precedence
			},
			verifyPromotion: func(t *testing.T, promotion Promotion) {
				require.Equal(t, "test-promotion", promotion.Name)
				require.Equal(t, "test-stage", promotion.StageName)
				require.Nil(t, promotion.Freight)
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			common, promotion := NewPromotionCommon(tc.message, tc.actor, tc.promotion, tc.freight)
			tc.verifyCommon(t, common)
			tc.verifyPromotion(t, promotion)
		})
	}
}

func TestPromotionConstructors(t *testing.T) {
	promotion := &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-promotion",
			Namespace: "test-project",
			CreationTimestamp: metav1.Time{
				Time: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			},
		},
		Spec: kargoapi.PromotionSpec{
			Stage: "test-stage",
		},
	}
	freight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-freight",
			CreationTimestamp: metav1.Time{
				Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	}
	expectedCommon := Common{
		Project: "test-project",
		Actor:   ptr.To("test-actor"),
		Message: "test message",
	}
	expectedPromotion := newPromotion(promotion, freight)

	testCases := []struct {
		name         string
		constructor  func() (cloudevents.Event, error)
		expectedType kargoapi.EventType
		assert       func(*testing.T, cloudevents.Event)
	}{
		{
			name: "succeeded",
			constructor: func() (cloudevents.Event, error) {
				return NewPromotionSucceeded("test message", "test-actor", promotion, freight, true)
			},
			expectedType: kargoapi.EventTypePromotionSucceeded,
			assert: func(t *testing.T, evt cloudevents.Event) {
				require.Equal(
					t,
					&PromotionSucceeded{
						Common:              expectedCommon,
						Promotion:           expectedPromotion,
						VerificationPending: ptr.To(true),
					},
					dataAs[PromotionSucceeded](t, evt),
				)
			},
		},
		{
			name: "failed",
			constructor: func() (cloudevents.Event, error) {
				return NewPromotionFailed("test message", "test-actor", promotion, freight)
			},
			expectedType: kargoapi.EventTypePromotionFailed,
			assert: func(t *testing.T, evt cloudevents.Event) {
				require.Equal(
					t,
					&PromotionFailed{Common: expectedCommon, Promotion: expectedPromotion},
					dataAs[PromotionFailed](t, evt),
				)
			},
		},
		{
			name: "errored",
			constructor: func() (cloudevents.Event, error) {
				return NewPromotionErrored("test message", "test-actor", promotion, freight)
			},
			expectedType: kargoapi.EventTypePromotionErrored,
			assert: func(t *testing.T, evt cloudevents.Event) {
				require.Equal(
					t,
					&PromotionErrored{Common: expectedCommon, Promotion: expectedPromotion},
					dataAs[PromotionErrored](t, evt),
				)
			},
		},
		{
			name: "aborted",
			constructor: func() (cloudevents.Event, error) {
				return NewPromotionAborted("test message", "test-actor", promotion, freight)
			},
			expectedType: kargoapi.EventTypePromotionAborted,
			assert: func(t *testing.T, evt cloudevents.Event) {
				require.Equal(
					t,
					&PromotionAborted{Common: expectedCommon, Promotion: expectedPromotion},
					dataAs[PromotionAborted](t, evt),
				)
			},
		},
		{
			name: "discarded",
			constructor: func() (cloudevents.Event, error) {
				return NewPromotionDiscarded("test message", "test-actor", promotion, freight)
			},
			expectedType: kargoapi.EventTypePromotionDiscarded,
			assert: func(t *testing.T, evt cloudevents.Event) {
				require.Equal(
					t,
					&PromotionDiscarded{Common: expectedCommon, Promotion: expectedPromotion},
					dataAs[PromotionDiscarded](t, evt),
				)
			},
		},
		{
			name: "created",
			constructor: func() (cloudevents.Event, error) {
				return NewPromotionCreated("test message", "test-actor", promotion, freight)
			},
			expectedType: kargoapi.EventTypePromotionCreated,
			assert: func(t *testing.T, evt cloudevents.Event) {
				require.Equal(
					t,
					&PromotionCreated{Common: expectedCommon, Promotion: expectedPromotion},
					dataAs[PromotionCreated](t, evt),
				)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			evt, err := testCase.constructor()
			require.NoError(t, err)
			requireCloudEvent(
				t,
				evt,
				testCase.expectedType,
				"Promotion",
			)
			testCase.assert(t, evt)
		})
	}
}

func TestNewPromotionSucceeded_VerificationPending(t *testing.T) {
	promotion := &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-promotion",
			Namespace: "test-project",
		},
		Spec: kargoapi.PromotionSpec{
			Stage: "test-stage",
		},
	}
	for _, pending := range []bool{true, false} {
		t.Run(strconv.FormatBool(pending), func(t *testing.T) {
			evt, err := NewPromotionSucceeded("test message", "test-actor", promotion, nil, pending)
			require.NoError(t, err)
			// The field must be present even when false so consumers can tell
			// "no verification" apart from "unknown"
			require.Contains(t, string(evt.Data()), `"verificationPending":`+strconv.FormatBool(pending))
			require.Equal(
				t,
				&pending,
				dataAs[PromotionSucceeded](t, evt).VerificationPending,
			)
		})
	}
}

func TestNewPromotion_Rollback(t *testing.T) {
	testCases := map[string]struct {
		annotations map[string]string
		expected    bool
	}{
		"rollback annotation true": {
			annotations: map[string]string{
				kargoapi.AnnotationKeyRollback: "true",
			},
			expected: true,
		},
		"rollback annotation false": {
			annotations: map[string]string{
				kargoapi.AnnotationKeyRollback: "false",
			},
			expected: false,
		},
		"rollback annotation absent": {
			expected: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			promotion := &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-promotion",
					Namespace:   "test-project",
					Annotations: tc.annotations,
				},
				Spec: kargoapi.PromotionSpec{
					Stage: "test-stage",
				},
			}
			evt := newPromotion(promotion, nil)
			require.Equal(t, tc.expected, evt.Rollback)
		})
	}
}

func TestCalculatePromotionVars(t *testing.T) {
	testCases := map[string]struct {
		promotion *kargoapi.Promotion
		baseEnv   map[string]any
		expected  map[string]any
	}{
		"simple string variables": {
			promotion: &kargoapi.Promotion{
				Spec: kargoapi.PromotionSpec{
					Vars: []kargoapi.ExpressionVariable{
						{Name: "simpleVar", Value: "simpleValue"},
						{Name: "numberVar", Value: "123"},
					},
				},
			},
			baseEnv: map[string]any{
				"ctx": map[string]any{"project": "test"},
			},
			expected: map[string]any{
				"simpleVar": "simpleValue",
				"numberVar": "123",
			},
		},
		"template expressions": {
			promotion: &kargoapi.Promotion{
				Spec: kargoapi.PromotionSpec{
					Vars: []kargoapi.ExpressionVariable{
						{Name: "projectVar", Value: "${{ ctx.project }}"},
						{Name: "combinedVar", Value: "${{ ctx.project }}-suffix"},
					},
				},
			},
			baseEnv: map[string]any{
				"ctx": map[string]any{"project": "test-project"},
			},
			expected: map[string]any{
				"projectVar":  "test-project",
				"combinedVar": "test-project-suffix",
			},
		},
		"variable dependencies": {
			promotion: &kargoapi.Promotion{
				Spec: kargoapi.PromotionSpec{
					Vars: []kargoapi.ExpressionVariable{
						{Name: "baseVar", Value: "base"},
						{Name: "derivedVar", Value: "${{ vars.baseVar }}-derived"},
					},
				},
			},
			baseEnv: map[string]any{},
			expected: map[string]any{
				"baseVar":    "base",
				"derivedVar": "base-derived",
			},
		},
		"invalid expressions gracefully ignored": {
			promotion: &kargoapi.Promotion{
				Spec: kargoapi.PromotionSpec{
					Vars: []kargoapi.ExpressionVariable{
						{Name: "validVar", Value: "valid"},
						{Name: "invalidVar", Value: "${{ invalid.syntax }}"},
						{Name: "anotherValidVar", Value: "also-valid"},
					},
				},
			},
			baseEnv: map[string]any{},
			expected: map[string]any{
				"validVar":        "valid",
				"anotherValidVar": "also-valid",
				// invalidVar should be skipped
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result := calculatePromotionVars(tc.promotion, tc.baseEnv)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestCalculateStepVars(t *testing.T) {
	testCases := map[string]struct {
		step     kargoapi.PromotionStep
		baseEnv  map[string]any
		expected map[string]any
	}{
		"step variables with base environment": {
			step: kargoapi.PromotionStep{
				Vars: []kargoapi.ExpressionVariable{
					{Name: "stepVar", Value: "stepValue"},
					{Name: "contextVar", Value: "${{ ctx.stage }}"},
				},
			},
			baseEnv: map[string]any{
				"ctx":  map[string]any{"stage": "production"},
				"vars": map[string]any{"existingVar": "existing"},
			},
			expected: map[string]any{
				"stepVar":    "stepValue",
				"contextVar": "production",
			},
		},
		"step variables override base variables": {
			step: kargoapi.PromotionStep{
				Vars: []kargoapi.ExpressionVariable{
					{Name: "overrideVar", Value: "step-value"},
					{Name: "newVar", Value: "${{ vars.overrideVar }}-new"},
				},
			},
			baseEnv: map[string]any{
				"vars": map[string]any{"overrideVar": "base-value"},
			},
			expected: map[string]any{
				"overrideVar": "step-value",
				"newVar":      "step-value-new",
			},
		},
		"invalid expressions gracefully ignored": {
			step: kargoapi.PromotionStep{
				Vars: []kargoapi.ExpressionVariable{
					{Name: "validVar", Value: "valid"},
					{Name: "invalidVar", Value: "${{ invalid.syntax }}"},
				},
			},
			baseEnv: map[string]any{},
			expected: map[string]any{
				"validVar": "valid",
				// invalidVar should be skipped
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result := calculateStepVars(tc.step, tc.baseEnv)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestNewPromotionWithArgoCDApps(t *testing.T) {
	testCases := map[string]struct {
		promotion  *kargoapi.Promotion
		freight    *kargoapi.Freight
		verifyApps func(t *testing.T, promotion Promotion)
	}{
		"promotion with static argocd apps": {
			promotion: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-promotion",
					Namespace: "test-namespace",
					CreationTimestamp: metav1.Time{
						Time: time.Date(2024, 10, 22, 0, 0, 0, 0, time.UTC),
					},
				},
				Spec: kargoapi.PromotionSpec{
					Freight: "test-freight",
					Stage:   "test-stage",
					Steps: []kargoapi.PromotionStep{
						{
							Uses: "argocd-update",
							Config: &v1.JSON{Raw: []byte(`{
								"apps": [
									{
										"name": "test-app-1"
									},
									{
										"name": "test-app-2",
										"namespace": "test-namespace"
									}
								]
							}`)},
						},
					},
				},
			},
			freight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Time{
						Time: time.Date(2024, 10, 22, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			verifyApps: func(t *testing.T, promotion Promotion) {
				require.Len(t, promotion.Applications, 2)

				expectedApps := []types.NamespacedName{
					{Namespace: "argocd", Name: "test-app-1"},
					{Namespace: "test-namespace", Name: "test-app-2"},
				}
				require.Equal(t, expectedApps, promotion.Applications)
			},
		},
		"promotion with template variables": {
			promotion: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-promotion",
					Namespace: "kargo-demo",
					CreationTimestamp: metav1.Time{
						Time: time.Date(2024, 10, 22, 0, 0, 0, 0, time.UTC),
					},
					Annotations: map[string]string{
						kargoapi.AnnotationKeyCreateActor: "admin",
					},
				},
				Spec: kargoapi.PromotionSpec{
					Freight: "test-freight",
					Stage:   "test",
					Vars: []kargoapi.ExpressionVariable{
						{Name: "argocdApp", Value: "my-application"},
						{Name: "appNamespace", Value: "test-namespace"},
					},
					Steps: []kargoapi.PromotionStep{
						{
							Uses: "argocd-update",
							Config: &v1.JSON{Raw: []byte(`{
								"apps": [
									{
										"name": "kargo-demo-${{ ctx.stage }}"
									},
									{
										"name": "${{ vars.argocdApp }}",
										"namespace": "argocd"
									},
									{
										"name": "${{ vars.argocdApp }}-${{ ctx.stage }}",
										"namespace": "${{ vars.appNamespace }}"
									}
								]
							}`)},
						},
					},
				},
			},
			freight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-freight",
					CreationTimestamp: metav1.Time{
						Time: time.Date(2024, 10, 22, 0, 0, 0, 0, time.UTC),
					},
				},
				Origin: kargoapi.FreightOrigin{
					Name: "test-warehouse",
				},
			},
			verifyApps: func(t *testing.T, promotion Promotion) {
				require.Len(t, promotion.Applications, 3)

				expectedApps := []types.NamespacedName{
					{Namespace: "argocd", Name: "kargo-demo-test"},
					{Namespace: "argocd", Name: "my-application"},
					{Namespace: "test-namespace", Name: "my-application-test"},
				}
				require.Equal(t, expectedApps, promotion.Applications)
			},
		},
		"promotion with invalid template expressions": {
			promotion: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-promotion",
					Namespace: "test-namespace",
					CreationTimestamp: metav1.Time{
						Time: time.Date(2024, 10, 22, 0, 0, 0, 0, time.UTC),
					},
				},
				Spec: kargoapi.PromotionSpec{
					Freight: "test-freight",
					Stage:   "test-stage",
					Steps: []kargoapi.PromotionStep{
						{
							Uses: "argocd-update",
							Config: &v1.JSON{Raw: []byte(`{
								"apps": [{"name": "${{ invalid.template.syntax }}"}]
							}`)},
						},
					},
				},
			},
			freight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-freight",
					CreationTimestamp: metav1.Time{
						Time: time.Date(2024, 10, 22, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			verifyApps: func(t *testing.T, promotion Promotion) {
				// Invalid templates should be gracefully ignored
				require.Empty(t, promotion.Applications)
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			promotion := newPromotion(tc.promotion, tc.freight)
			tc.verifyApps(t, promotion)
		})
	}
}
