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

	api "github.com/akuity/kargo/api/v1alpha1"
)

func TestValidateCreate(t *testing.T) {
	w := &webhook{
		authorizeFn: func(context.Context, *api.Promotion, string) error {
			return nil // Always authorize
		},
		validateProjectFn: func(context.Context, *api.Promotion) error {
			return nil // Skip validation
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
						Stage: "fake-stage",
						State: "fake-state",
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
						Stage: "fake-stage",
						State: "fake-state",
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
				validateProjectFn: func(context.Context, *api.Promotion) error {
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
		authorizeFn: func(context.Context, *api.Promotion, string) error {
			return nil // Always authorize
		},
		validateProjectFn: func(context.Context, *api.Promotion) error {
			return nil // Skip validation
		},
	}
	require.NoError(t, w.ValidateDelete(context.Background(), &api.Promotion{}))
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
							Stage: "fake-stage",
						},
					},
					"create",
				),
			)
		})
	}
}
