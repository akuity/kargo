package project

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestNewWebhook(t *testing.T) {
	w := newWebhook(fake.NewClientBuilder().Build())
	require.NotNil(t, w)
	require.NotNil(t, w.validateSpecFn)
	require.NotNil(t, w.getNamespaceFn)
	require.NotNil(t, w.createNamespaceFn)
	require.NotNil(t, w.updateNamespaceFn)
}

func TestValidateCreate(t *testing.T) {
	testCases := []struct {
		name       string
		webhook    *webhook
		assertions func(error)
	}{

		{
			name: "error validating spec",
			webhook: &webhook{
				validateSpecFn: func(f *field.Path, promotionPolicies *kargoapi.ProjectSpec) field.ErrorList {
					return field.ErrorList{
						field.Invalid(
							f,
							promotionPolicies,
							"something was invalid",
						),
					}
				},
			},
			assertions: func(err error) {
				require.Error(t, err)
				statusErr, ok := err.(*apierrors.StatusError)
				require.True(t, ok)
				require.Equal(t, int32(http.StatusUnprocessableEntity), statusErr.ErrStatus.Code)
			},
		},

		{
			name: "error getting namespace",
			webhook: &webhook{
				validateSpecFn: func(*field.Path, *kargoapi.ProjectSpec) field.ErrorList {
					return nil
				},
				getNamespaceFn: func(
					context.Context,
					types.NamespacedName,
					client.Object,
					...client.GetOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(err error) {
				require.Error(t, err)
				statusErr, ok := err.(*apierrors.StatusError)
				require.True(t, ok)
				require.Equal(t, int32(http.StatusInternalServerError), statusErr.ErrStatus.Code)
			},
		},

		{
			name: "namespace exists, is not owned by project, but is labeled " +
				"as a project; error updating namespace",
			webhook: &webhook{
				validateSpecFn: func(*field.Path, *kargoapi.ProjectSpec) field.ErrorList {
					return nil
				},
				getNamespaceFn: func(
					_ context.Context,
					_ types.NamespacedName,
					obj client.Object,
					_ ...client.GetOption,
				) error {
					ns, ok := obj.(*corev1.Namespace)
					require.True(t, ok)
					ns.Labels = map[string]string{
						kargoapi.ProjectLabelKey: kargoapi.LabelTrueValue,
					}
					return nil
				},
				updateNamespaceFn: func(
					context.Context,
					client.Object,
					...client.UpdateOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(err error) {
				require.Error(t, err)
				statusErr, ok := err.(*apierrors.StatusError)
				require.True(t, ok)
				require.Equal(
					t,
					int32(http.StatusInternalServerError),
					statusErr.ErrStatus.Code,
				)
			},
		},

		{
			name: "namespace exists, is not owned by project, but is labeled " +
				"as a project; success updating namespace",
			webhook: &webhook{
				validateSpecFn: func(*field.Path, *kargoapi.ProjectSpec) field.ErrorList {
					return nil
				},
				getNamespaceFn: func(
					_ context.Context,
					_ types.NamespacedName,
					obj client.Object,
					_ ...client.GetOption,
				) error {
					ns, ok := obj.(*corev1.Namespace)
					require.True(t, ok)
					ns.Labels = map[string]string{
						kargoapi.ProjectLabelKey: kargoapi.LabelTrueValue,
					}
					return nil
				},
				updateNamespaceFn: func(
					context.Context,
					client.Object,
					...client.UpdateOption,
				) error {
					return nil
				},
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},

		{
			name: "namespace exists, is not owned by project, and is not labeled as a project",
			webhook: &webhook{
				validateSpecFn: func(*field.Path, *kargoapi.ProjectSpec) field.ErrorList {
					return nil
				},
				getNamespaceFn: func(
					context.Context,
					types.NamespacedName,
					client.Object,
					...client.GetOption,
				) error {
					return nil
				},
			},
			assertions: func(err error) {
				require.Error(t, err)
				statusErr, ok := err.(*apierrors.StatusError)
				require.True(t, ok)
				require.Equal(t, int32(http.StatusConflict), statusErr.ErrStatus.Code)
			},
		},

		{
			name: "namespace exists and is owned by project",
			webhook: &webhook{
				validateSpecFn: func(*field.Path, *kargoapi.ProjectSpec) field.ErrorList {
					return nil
				},
				getNamespaceFn: func(
					_ context.Context,
					_ types.NamespacedName,
					obj client.Object,
					_ ...client.GetOption,
				) error {
					ns := obj.(*corev1.Namespace) // nolint: forcetypeassert
					ns.OwnerReferences = []metav1.OwnerReference{
						{
							UID: types.UID("fake-uid"),
						},
					}
					return nil
				},
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},

		{
			name: "namespace does not exist; error creating it",
			webhook: &webhook{
				validateSpecFn: func(*field.Path, *kargoapi.ProjectSpec) field.ErrorList {
					return nil
				},
				getNamespaceFn: func(
					context.Context,
					types.NamespacedName,
					client.Object,
					...client.GetOption,
				) error {
					return apierrors.NewNotFound(schema.GroupResource{}, "")
				},
				createNamespaceFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(err error) {
				require.Error(t, err)
				statusErr, ok := err.(*apierrors.StatusError)
				require.True(t, ok)
				require.Equal(
					t,
					int32(http.StatusInternalServerError),
					statusErr.ErrStatus.Code,
				)
			},
		},

		{
			name: "namespace does not exist; success creating it",
			webhook: &webhook{
				validateSpecFn: func(*field.Path, *kargoapi.ProjectSpec) field.ErrorList {
					return nil
				},
				getNamespaceFn: func(
					context.Context,
					types.NamespacedName,
					client.Object,
					...client.GetOption,
				) error {
					return apierrors.NewNotFound(schema.GroupResource{}, "")
				},
				createNamespaceFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return nil
				},
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		ctx := admission.NewContextWithRequest(
			context.Background(),
			admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					DryRun: ptr.To(false),
				},
			},
		)
		t.Run(testCase.name, func(t *testing.T) {
			_, err := testCase.webhook.ValidateCreate(
				ctx,
				&kargoapi.Project{
					ObjectMeta: metav1.ObjectMeta{
						UID: types.UID("fake-uid"),
					},
				},
			)
			testCase.assertions(err)
		})
	}
}

func TestValidateSpec(t *testing.T) {
	testCases := []struct {
		name       string
		spec       *kargoapi.ProjectSpec
		assertions func(*kargoapi.ProjectSpec, field.ErrorList)
	}{
		{
			name: "nil",
			assertions: func(_ *kargoapi.ProjectSpec, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
		{
			name: "invalid",
			spec: &kargoapi.ProjectSpec{
				// Has two conflicting PromotionPolicies...
				PromotionPolicies: []kargoapi.PromotionPolicy{
					{Stage: "fake-stage"},
					{Stage: "fake-stage"},
				},
			},
			assertions: func(spec *kargoapi.ProjectSpec, errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "spec.promotionPolicies",
							BadValue: spec.PromotionPolicies,
							Detail:   "multiple spec.promotionPolicies reference stage fake-stage",
						},
					},
					errs,
				)
			},
		},
		{
			name: "valid",
			spec: &kargoapi.ProjectSpec{
				PromotionPolicies: []kargoapi.PromotionPolicy{
					{Stage: "fake-stage"},
				},
			},
			assertions: func(_ *kargoapi.ProjectSpec, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.spec,
				w.validateSpec(field.NewPath("spec"), testCase.spec),
			)
		})
	}
}
