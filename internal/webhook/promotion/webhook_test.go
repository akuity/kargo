package promotion

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	authzv1 "k8s.io/api/authorization/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestValidateCreate(t *testing.T) {
	w := &webhook{
		authorizeFn: func(context.Context, *kargoapi.Promotion, string) error {
			return nil // Always authorize
		},
		validateProjectFn: func(context.Context, *kargoapi.Promotion) error {
			return nil // Skip validation
		},
	}
	require.NoError(t, w.ValidateCreate(context.Background(), &kargoapi.Promotion{}))
}

func TestValidateUpdate(t *testing.T) {
	testCases := []struct {
		name        string
		setup       func() (*kargoapi.Promotion, *kargoapi.Promotion)
		authorizeFn func(
			ctx context.Context,
			promo *kargoapi.Promotion,
			action string,
		) error
		assertions func(error)
	}{
		{
			name: "authorization error",
			setup: func() (*kargoapi.Promotion, *kargoapi.Promotion) {
				return &kargoapi.Promotion{}, &kargoapi.Promotion{}
			},
			authorizeFn: func(context.Context, *kargoapi.Promotion, string) error {
				return errors.New("something went wrong")
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "attempt to mutate",
			setup: func() (*kargoapi.Promotion, *kargoapi.Promotion) {
				oldPromo := &kargoapi.Promotion{
					ObjectMeta: v1.ObjectMeta{
						Name:      "fake-name",
						Namespace: "fake-namespace",
					},
					Spec: &kargoapi.PromotionSpec{
						Stage:   "fake-stage",
						Freight: "fake-freight",
					},
				}
				newPromo := oldPromo.DeepCopy()
				newPromo.Spec.Freight = "another-fake-freight"
				return oldPromo, newPromo
			},
			authorizeFn: func(context.Context, *kargoapi.Promotion, string) error {
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
			setup: func() (*kargoapi.Promotion, *kargoapi.Promotion) {
				oldPromo := &kargoapi.Promotion{
					ObjectMeta: v1.ObjectMeta{
						Name:      "fake-name",
						Namespace: "fake-namespace",
					},
					Spec: &kargoapi.PromotionSpec{
						Stage:   "fake-stage",
						Freight: "fake-freight",
					},
				}
				newPromo := oldPromo.DeepCopy()
				return oldPromo, newPromo
			},
			authorizeFn: func(context.Context, *kargoapi.Promotion, string) error {
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
				validateProjectFn: func(context.Context, *kargoapi.Promotion) error {
					return nil // Skip validation
				},
			}
			oldPromo, newPromo := testCase.setup()
			testCase.assertions(
				w.ValidateUpdate(context.Background(), oldPromo, newPromo),
			)
		})
	}
}

func TestValidateDelete(t *testing.T) {
	w := &webhook{
		authorizeFn: func(context.Context, *kargoapi.Promotion, string) error {
			return nil // Always authorize
		},
		validateProjectFn: func(context.Context, *kargoapi.Promotion) error {
			return nil // Skip validation
		},
	}
	require.NoError(t, w.ValidateDelete(context.Background(), &kargoapi.Promotion{}))
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
					&kargoapi.Promotion{
						ObjectMeta: v1.ObjectMeta{
							Name:      "fake-promotion",
							Namespace: "fake-namespace",
						},
						Spec: &kargoapi.PromotionSpec{
							Stage: "fake-stage",
						},
					},
					"create",
				),
			)
		})
	}
}
