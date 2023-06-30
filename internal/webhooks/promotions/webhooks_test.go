package promotions

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	authzv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	api "github.com/akuity/kargo/api/v1alpha1"
)

func TestValidateCreate(t *testing.T) {
	w := &webhook{
		authorizeFn: func(context.Context, *api.Promotion, string) error {
			return nil // Always authorize
		},
	}
	require.NoError(t, w.ValidateCreate(context.Background(), &api.Promotion{}))
}

func TestValidateUpdate(t *testing.T) {
	testCases := []struct {
		name        string
		setup       func() (*api.Promotion, *api.Promotion)
		authorizeFn func(
			ctx context.Context,
			promo *api.Promotion,
			action string,
		) error
		assertions func(error)
	}{
		{
			name: "authorization error",
			setup: func() (*api.Promotion, *api.Promotion) {
				return &api.Promotion{}, &api.Promotion{}
			},
			authorizeFn: func(context.Context, *api.Promotion, string) error {
				return errors.New("something went wrong")
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "attempt to mutate",
			setup: func() (*api.Promotion, *api.Promotion) {
				oldPromo := &api.Promotion{
					ObjectMeta: v1.ObjectMeta{
						Name:      "fake-name",
						Namespace: "fake-namespace",
					},
					Spec: &api.PromotionSpec{
						Environment: "fake-environment",
						State:       "fake-state",
					},
				}
				newPromo := oldPromo.DeepCopy()
				newPromo.Spec.State = "another-fake-state"
				return oldPromo, newPromo
			},
			authorizeFn: func(context.Context, *api.Promotion, string) error {
				return nil
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "\"fake-name\" is invalid")
				require.Contains(t, err.Error(), "spec is immutable")
			},
		},

		{
			name: "update without mutation",
			setup: func() (*api.Promotion, *api.Promotion) {
				oldPromo := &api.Promotion{
					ObjectMeta: v1.ObjectMeta{
						Name:      "fake-name",
						Namespace: "fake-namespace",
					},
					Spec: &api.PromotionSpec{
						Environment: "fake-environment",
						State:       "fake-state",
					},
				}
				newPromo := oldPromo.DeepCopy()
				return oldPromo, newPromo
			},
			authorizeFn: func(context.Context, *api.Promotion, string) error {
				return nil
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			w := &webhook{
				authorizeFn: testCase.authorizeFn,
			}
			oldPromo, newPromo := testCase.setup()
			testCase.assertions(
				w.ValidateUpdate(context.Background(), oldPromo, newPromo),
			)
		})
	}
}

func TestValidateDelete(t *testing.T) {
	testCases := []struct {
		name                          string
		admissionRequestFromContextFn func(
			context.Context,
		) (admission.Request, error)
		authorizeFn func(
			context.Context,
			*api.Promotion,
			string,
		) error
		assertions func(error)
	}{
		{
			name: "error getting admission request bound to context",
			admissionRequestFromContextFn: func(
				context.Context,
			) (admission.Request, error) {
				return admission.Request{}, errors.New("something went wrong")
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.True(t, apierrors.IsForbidden(err))
				require.Contains(
					t,
					err.Error(),
					"error retrieving admission request from context",
				)
			},
		},
		{
			name: "user is namespace controller service account",
			admissionRequestFromContextFn: func(
				context.Context,
			) (admission.Request, error) {
				return admission.Request{
					AdmissionRequest: admissionv1.AdmissionRequest{
						UserInfo: authenticationv1.UserInfo{
							Username: "system:serviceaccount:kube-system:namespace-controller", // nolint: lll
						},
					},
				}, nil
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "user is not authorized",
			admissionRequestFromContextFn: func(
				context.Context,
			) (admission.Request, error) {
				return admission.Request{}, nil
			},
			authorizeFn: func(context.Context, *api.Promotion, string) error {
				return errors.Errorf("not authorized")
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Equal(t, "not authorized", err.Error())
			},
		},
		{
			name: "user is authorized",
			admissionRequestFromContextFn: func(
				context.Context,
			) (admission.Request, error) {
				return admission.Request{}, nil
			},
			authorizeFn: func(context.Context, *api.Promotion, string) error {
				return nil
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			w := &webhook{
				admissionRequestFromContextFn: testCase.admissionRequestFromContextFn,
				authorizeFn:                   testCase.authorizeFn,
			}
			testCase.assertions(
				w.ValidateDelete(context.Background(), &api.Promotion{}),
			)
		})
	}
}

func TestAuthorize(t *testing.T) {
	testCases := []struct {
		name                          string
		admissionRequestFromContextFn func(context.Context) (admission.Request, error)
		createSubjectAccessReviewFn   func(
			context.Context,
			client.Object,
			...client.CreateOption,
		) error
		assertions func(err error)
	}{
		{
			name: "error getting admission request bound to context",
			admissionRequestFromContextFn: func(
				context.Context,
			) (admission.Request, error) {
				return admission.Request{}, errors.New("something went wrong")
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error retrieving admission request from context; refusing to",
				)
			},
		},
		{
			name: "error creating subject access review",
			admissionRequestFromContextFn: func(
				context.Context,
			) (admission.Request, error) {
				return admission.Request{}, nil
			},
			createSubjectAccessReviewFn: func(
				context.Context,
				client.Object,
				...client.CreateOption,
			) error {
				return errors.New("something went wrong")
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error creating SubjectAccessReview")
			},
		},
		{
			name: "subject is not authorized",
			admissionRequestFromContextFn: func(
				context.Context,
			) (admission.Request, error) {
				return admission.Request{}, nil
			},
			createSubjectAccessReviewFn: func(
				_ context.Context,
				obj client.Object,
				_ ...client.CreateOption,
			) error {
				obj.(*authzv1.SubjectAccessReview).Status.Allowed = false // nolint: forcetypeassert
				return nil
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "is not permitted")
			},
		},
		{
			name: "subject is authorized",
			admissionRequestFromContextFn: func(
				context.Context,
			) (admission.Request, error) {
				return admission.Request{}, nil
			},
			createSubjectAccessReviewFn: func(
				_ context.Context,
				obj client.Object,
				_ ...client.CreateOption,
			) error {
				obj.(*authzv1.SubjectAccessReview).Status.Allowed = true // nolint: forcetypeassert
				return nil
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			w := &webhook{
				admissionRequestFromContextFn: testCase.admissionRequestFromContextFn,
				createSubjectAccessReviewFn:   testCase.createSubjectAccessReviewFn,
			}
			testCase.assertions(
				w.authorize(
					context.Background(),
					&api.Promotion{
						ObjectMeta: v1.ObjectMeta{
							Name:      "fake-promotion",
							Namespace: "fake-namespace",
						},
						Spec: &api.PromotionSpec{
							Environment: "fake-environment",
						},
					},
					"create",
				),
			)
		})
	}
}
