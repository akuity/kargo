package event

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestPromotionSucceeded(t *testing.T) {
	evt := &PromotionSucceeded{}
	require.Equal(t, kargoapi.EventTypePromotionSucceeded, evt.Type())
}

func TestPromotionFailed(t *testing.T) {
	evt := &PromotionFailed{}
	require.Equal(t, kargoapi.EventTypePromotionFailed, evt.Type())
}

func TestPromotionErrored(t *testing.T) {
	evt := &PromotionErrored{}
	require.Equal(t, kargoapi.EventTypePromotionErrored, evt.Type())
}

func TestPromotionAborted(t *testing.T) {
	evt := &PromotionAborted{}
	require.Equal(t, kargoapi.EventTypePromotionAborted, evt.Type())
}

func TestPromotionCreated(t *testing.T) {
	evt := &PromotionCreated{}
	require.Equal(t, kargoapi.EventTypePromotionCreated, evt.Type())
}

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
	// NOTE(thomastaylor312): I'm including these for now as we might need to test more edge cases
	// if we expand events. If this test isn't adding any value in the future, we can remove
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

	testCases := map[string]struct {
		constructor  func() Meta
		expectedType kargoapi.EventType
	}{
		"succeeded": {
			constructor: func() Meta {
				return NewPromotionSucceeded("Success message", "test-actor", promotion, freight)
			},
			expectedType: kargoapi.EventTypePromotionSucceeded,
		},
		"failed": {
			constructor: func() Meta {
				return NewPromotionFailed("Failed message", "test-actor", promotion, freight)
			},
			expectedType: kargoapi.EventTypePromotionFailed,
		},
		"errored": {
			constructor: func() Meta {
				return NewPromotionErrored("Error message", "test-actor", promotion, freight)
			},
			expectedType: kargoapi.EventTypePromotionErrored,
		},
		"aborted": {
			constructor: func() Meta {
				return NewPromotionAborted("Aborted message", "test-actor", promotion, freight)
			},
			expectedType: kargoapi.EventTypePromotionAborted,
		},
		"created": {
			constructor: func() Meta {
				return NewPromotionCreated("Created message", "test-actor", promotion, freight)
			},
			expectedType: kargoapi.EventTypePromotionCreated,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			event := tc.constructor()

			require.Equal(t, tc.expectedType, event.Type())
			require.Equal(t, "test-project", event.GetProject())
			require.Equal(t, "test-promotion", event.GetName())
			require.Equal(t, "Promotion", event.Kind())
		})
	}
}

func TestPromotionSucceeded_VerificationPending(t *testing.T) {
	promotion := &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-promotion",
			Namespace: "test-project",
		},
		Spec: kargoapi.PromotionSpec{
			Stage: "test-stage",
		},
	}

	event := NewPromotionSucceeded("Success message", "test-actor", promotion, nil)

	// Test that VerificationPending is initially nil
	require.Nil(t, event.VerificationPending)

	// Test marshaling without VerificationPending
	annotations := event.MarshalAnnotations()
	require.NotContains(t, annotations, kargoapi.AnnotationKeyEventVerificationPending)

	// Test setting VerificationPending to true
	event.VerificationPending = ptr.To(true)
	annotations = event.MarshalAnnotations()
	require.Equal(t, "true", annotations[kargoapi.AnnotationKeyEventVerificationPending])

	// Test setting VerificationPending to false
	event.VerificationPending = ptr.To(false)
	annotations = event.MarshalAnnotations()
	require.Equal(t, "false", annotations[kargoapi.AnnotationKeyEventVerificationPending])
}

func TestPromotionEventMarshalAnnotations(t *testing.T) {
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
		Alias: "v1.0.0",
	}

	testCases := map[string]struct {
		event    AnnotationMarshaler
		expected map[string]string
	}{
		"promotion succeeded": {
			event: NewPromotionSucceeded("Success message", "test-actor", promotion, freight),
			expected: map[string]string{
				kargoapi.AnnotationKeyEventProject:             "test-project",
				kargoapi.AnnotationKeyEventActor:               "test-actor",
				kargoapi.AnnotationKeyEventPromotionName:       "test-promotion",
				kargoapi.AnnotationKeyEventStageName:           "test-stage",
				kargoapi.AnnotationKeyEventPromotionCreateTime: "2024-01-01T12:00:00Z",
				kargoapi.AnnotationKeyEventFreightName:         "test-freight",
				kargoapi.AnnotationKeyEventFreightCreateTime:   "2024-01-01T00:00:00Z",
				kargoapi.AnnotationKeyEventFreightAlias:        "v1.0.0",
			},
		},
		"promotion failed": {
			event: NewPromotionFailed("Failed message", "test-actor", promotion, nil),
			expected: map[string]string{
				kargoapi.AnnotationKeyEventProject:             "test-project",
				kargoapi.AnnotationKeyEventActor:               "test-actor",
				kargoapi.AnnotationKeyEventPromotionName:       "test-promotion",
				kargoapi.AnnotationKeyEventStageName:           "test-stage",
				kargoapi.AnnotationKeyEventPromotionCreateTime: "2024-01-01T12:00:00Z",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			annotations := tc.event.MarshalAnnotations()
			require.Equal(t, tc.expected, annotations)
		})
	}
}

func TestPromotionEventUnmarshalAnnotations(t *testing.T) {
	testCases := map[string]struct {
		annotations   map[string]string
		unmarshalFunc func(map[string]string) (Meta, error)
		expectedType  Meta
		expectError   bool
		errorMessage  string
	}{
		"promotion succeeded": {
			annotations: map[string]string{
				kargoapi.AnnotationKeyEventProject:             "test-project",
				kargoapi.AnnotationKeyEventPromotionName:       "test-promotion",
				kargoapi.AnnotationKeyEventStageName:           "test-stage",
				kargoapi.AnnotationKeyEventPromotionCreateTime: "2024-01-01T12:00:00Z",
				kargoapi.AnnotationKeyEventVerificationPending: "true",
			},
			unmarshalFunc: func(annotations map[string]string) (Meta, error) {
				return UnmarshalPromotionSucceededAnnotations("event-id", annotations)
			},
			expectedType: &PromotionSucceeded{
				Common: Common{
					Project: "test-project",
					ID:      "event-id",
				},
				Promotion: Promotion{
					Name:       "test-promotion",
					StageName:  "test-stage",
					CreateTime: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				},
				VerificationPending: ptr.To(true),
			},
		},
		"promotion failed": {
			annotations: map[string]string{
				kargoapi.AnnotationKeyEventProject:             "test-project",
				kargoapi.AnnotationKeyEventPromotionName:       "test-promotion",
				kargoapi.AnnotationKeyEventStageName:           "test-stage",
				kargoapi.AnnotationKeyEventPromotionCreateTime: "2024-01-01T12:00:00Z",
			},
			unmarshalFunc: func(annotations map[string]string) (Meta, error) {
				return UnmarshalPromotionFailedAnnotations("event-id", annotations)
			},
			expectedType: &PromotionFailed{
				Common: Common{
					Project: "test-project",
					ID:      "event-id",
				},
				Promotion: Promotion{
					Name:       "test-promotion",
					StageName:  "test-stage",
					CreateTime: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				},
			},
		},
		"promotion errored": {
			annotations: map[string]string{
				kargoapi.AnnotationKeyEventProject:             "test-project",
				kargoapi.AnnotationKeyEventPromotionName:       "test-promotion",
				kargoapi.AnnotationKeyEventStageName:           "test-stage",
				kargoapi.AnnotationKeyEventPromotionCreateTime: "2024-01-01T12:00:00Z",
			},
			unmarshalFunc: func(annotations map[string]string) (Meta, error) {
				return UnmarshalPromotionErroredAnnotations("event-id", annotations)
			},
			expectedType: &PromotionErrored{
				Common: Common{
					Project: "test-project",
					ID:      "event-id",
				},
				Promotion: Promotion{
					Name:       "test-promotion",
					StageName:  "test-stage",
					CreateTime: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				},
			},
		},
		"promotion aborted": {
			annotations: map[string]string{
				kargoapi.AnnotationKeyEventProject:             "test-project",
				kargoapi.AnnotationKeyEventPromotionName:       "test-promotion",
				kargoapi.AnnotationKeyEventStageName:           "test-stage",
				kargoapi.AnnotationKeyEventPromotionCreateTime: "2024-01-01T12:00:00Z",
			},
			unmarshalFunc: func(annotations map[string]string) (Meta, error) {
				return UnmarshalPromotionAbortedAnnotations("event-id", annotations)
			},
			expectedType: &PromotionAborted{
				Common: Common{
					Project: "test-project",
					ID:      "event-id",
				},
				Promotion: Promotion{
					Name:       "test-promotion",
					StageName:  "test-stage",
					CreateTime: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				},
			},
		},
		"promotion created": {
			annotations: map[string]string{
				kargoapi.AnnotationKeyEventProject:             "test-project",
				kargoapi.AnnotationKeyEventPromotionName:       "test-promotion",
				kargoapi.AnnotationKeyEventStageName:           "test-stage",
				kargoapi.AnnotationKeyEventPromotionCreateTime: "2024-01-01T12:00:00Z",
			},
			unmarshalFunc: func(annotations map[string]string) (Meta, error) {
				return UnmarshalPromotionCreatedAnnotations("event-id", annotations)
			},
			expectedType: &PromotionCreated{
				Common: Common{
					Project: "test-project",
					ID:      "event-id",
				},
				Promotion: Promotion{
					Name:       "test-promotion",
					StageName:  "test-stage",
					CreateTime: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				},
			},
		},
		"invalid promotion annotations": {
			annotations: map[string]string{
				kargoapi.AnnotationKeyEventPromotionCreateTime: "invalid-time",
			},
			unmarshalFunc: func(annotations map[string]string) (Meta, error) {
				return UnmarshalPromotionSucceededAnnotations("event-id", annotations)
			},
			expectError:  true,
			errorMessage: "failed to parse promotion create time",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result, err := tc.unmarshalFunc(tc.annotations)

			if tc.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errorMessage)
				return
			}

			require.NoError(t, err)

			require.Equal(t, tc.expectedType, result, "oh noes, types don't match!")
		})
	}
}

func TestPromotionSucceeded_UnmarshalVerificationPending(t *testing.T) {
	testCases := map[string]struct {
		annotations map[string]string
		expected    *bool
	}{
		"verification pending true": {
			annotations: map[string]string{
				kargoapi.AnnotationKeyEventProject:             "test-project",
				kargoapi.AnnotationKeyEventPromotionName:       "test-promotion",
				kargoapi.AnnotationKeyEventStageName:           "test-stage",
				kargoapi.AnnotationKeyEventPromotionCreateTime: "2024-01-01T12:00:00Z",
				kargoapi.AnnotationKeyEventVerificationPending: "true",
			},
			expected: ptr.To(true),
		},
		"verification pending false": {
			annotations: map[string]string{
				kargoapi.AnnotationKeyEventProject:             "test-project",
				kargoapi.AnnotationKeyEventPromotionName:       "test-promotion",
				kargoapi.AnnotationKeyEventStageName:           "test-stage",
				kargoapi.AnnotationKeyEventPromotionCreateTime: "2024-01-01T12:00:00Z",
				kargoapi.AnnotationKeyEventVerificationPending: "false",
			},
			expected: ptr.To(false),
		},
		"verification pending missing": {
			annotations: map[string]string{
				kargoapi.AnnotationKeyEventProject:             "test-project",
				kargoapi.AnnotationKeyEventPromotionName:       "test-promotion",
				kargoapi.AnnotationKeyEventStageName:           "test-stage",
				kargoapi.AnnotationKeyEventPromotionCreateTime: "2024-01-01T12:00:00Z",
			},
			expected: nil,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result, err := UnmarshalPromotionSucceededAnnotations("event-id", tc.annotations)
			require.NoError(t, err)
			require.Equal(t, tc.expected, result.VerificationPending)
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
