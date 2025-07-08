package promotiontask

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func Test_webhook_ValidateCreate(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, kargoapi.AddToScheme(scheme))

	tests := []struct {
		name       string
		objects    []client.Object
		task       *kargoapi.PromotionTask
		assertions func(*testing.T, admission.Warnings, error)
	}{
		{
			name: "project does not exist",
			objects: []client.Object{
				&corev1.Namespace{
					ObjectMeta: v1.ObjectMeta{
						Name: "fake-project",
					},
				},
			},
			task: &kargoapi.PromotionTask{
				ObjectMeta: v1.ObjectMeta{
					Name:      "fake-template",
					Namespace: "fake-project",
				},
			},
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings)
				assert.ErrorContains(t, err, "namespace \"fake-project\" is not a project")
			},
		},
		{
			name: "project exists",
			objects: []client.Object{
				&corev1.Namespace{
					ObjectMeta: v1.ObjectMeta{
						Name: "fake-project",
						Labels: map[string]string{
							kargoapi.LabelKeyProject: kargoapi.LabelValueTrue,
						},
					},
				},
				&kargoapi.Project{
					ObjectMeta: v1.ObjectMeta{
						Name: "fake-project",
					},
				},
			},
			task: &kargoapi.PromotionTask{
				ObjectMeta: v1.ObjectMeta{
					Name:      "fake-template",
					Namespace: "fake-project",
				},
			},
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings)
				assert.NoError(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithObjects(tt.objects...).
				WithScheme(scheme).
				Build()

			w := &webhook{
				client: c,
			}

			got, err := w.ValidateCreate(context.Background(), tt.task)
			tt.assertions(t, got, err)
		})
	}
}

func Test_webhook_ValidateSpec(t *testing.T) {
	testCases := []struct {
		name       string
		spec       kargoapi.PromotionTaskSpec
		assertions func(*testing.T, field.ErrorList)
	}{
		{
			name: "invalid",
			spec: kargoapi.PromotionTaskSpec{
				Steps: []kargoapi.PromotionStep{
					{As: "step-42"}, // This step alias matches a reserved pattern
					{As: "commit"},
					{As: "commit"}, // Duplicate!
				},
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				// We really want to see that all underlying errors have been bubbled up
				// to this level and been aggregated.
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "spec.steps[0].as",
							BadValue: "step-42",
							Detail:   "step alias is reserved",
						},
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "spec.steps[2].as",
							BadValue: "commit",
							Detail:   "step alias duplicates that of spec.steps[1]",
						},
					},
					errs,
				)
			},
		},
		{
			name: "valid",
			spec: kargoapi.PromotionTaskSpec{
				Steps: []kargoapi.PromotionStep{
					{As: "foo"},
					{As: "bar"},
					{As: "baz"},
					{As: ""},
					{As: ""}, // optional not dup
				},
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Empty(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				w.validateSpec(field.NewPath("spec"), testCase.spec),
			)
		})
	}
}
