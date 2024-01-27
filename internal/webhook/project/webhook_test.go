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
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestNewWebhook(t *testing.T) {
	w := newWebhook(fake.NewClientBuilder().Build())
	require.NotNil(t, w)
	require.NotNil(t, w.getNamespaceFn)
	require.NotNil(t, w.createNamespaceFn)
}

func TestValidateCreate(t *testing.T) {
	testCases := []struct {
		name       string
		webhook    *webhook
		assertions func(error)
	}{

		{
			name: "error getting namespace",
			webhook: &webhook{
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
			name: "namespace exists and is not owned by project",
			webhook: &webhook{
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
