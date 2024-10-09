package promotion

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
	authnv1 "k8s.io/api/authentication/v1"
	authzv1 "k8s.io/api/authorization/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apiserver/pkg/authentication/serviceaccount"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	fakeevent "github.com/akuity/kargo/internal/kubernetes/event/fake"
	libWebhook "github.com/akuity/kargo/internal/webhook"
)

func TestNewWebhook(t *testing.T) {
	kubeClient := fake.NewClientBuilder().Build()
	w := newWebhook(
		libWebhook.Config{},
		kubeClient,
		admission.NewDecoder(kubeClient.Scheme()),
		&fakeevent.EventRecorder{},
	)
	// Assert that all overridable behaviors were initialized to a default:
	require.NotNil(t, w.getFreightFn)
	require.NotNil(t, w.getStageFn)
	require.NotNil(t, w.validateProjectFn)
	require.NotNil(t, w.authorizeFn)
	require.NotNil(t, w.admissionRequestFromContextFn)
	require.NotNil(t, w.createSubjectAccessReviewFn)
	require.NotNil(t, w.isRequestFromKargoControlplaneFn)
}

func TestDefault(t *testing.T) {
	testCases := []struct {
		name       string
		promotion  *kargoapi.Promotion
		webhook    *webhook
		assertions func(*testing.T, *kargoapi.Promotion, error)
	}{
		{
			name: "error getting stage",
			webhook: &webhook{
				admissionRequestFromContextFn: func(context.Context) (admission.Request, error) {
					return admission.Request{}, nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.Promotion, err error) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "stage not found",
			webhook: &webhook{
				admissionRequestFromContextFn: func(context.Context) (admission.Request, error) {
					return admission.Request{}, nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return nil, nil
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.Promotion, err error) {
				require.ErrorContains(t, err, "could not find Stage")
			},
		},
		{
			name: "stage without promotion steps",
			webhook: &webhook{
				admissionRequestFromContextFn: func(context.Context) (admission.Request, error) {
					return admission.Request{}, nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{}, nil
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.Promotion, err error) {
				require.ErrorContains(t, err, "defines no promotion steps")
			},
		},
		{
			name: "success with PromotionTemplate",
			promotion: &kargoapi.Promotion{
				Spec: kargoapi.PromotionSpec{
					Stage: "fake-stage",
					Steps: []kargoapi.PromotionStep{
						{},
					},
				},
			},
			webhook: &webhook{
				admissionRequestFromContextFn: func(context.Context) (admission.Request, error) {
					return admission.Request{}, nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{
						Spec: kargoapi.StageSpec{
							Shard: "fake-shard",
						},
					}, nil
				},
			},
			assertions: func(t *testing.T, promo *kargoapi.Promotion, err error) {
				require.NoError(t, err)
				require.Equal(t, "fake-shard", promo.Labels[kargoapi.ShardLabelKey])
				require.NotEmpty(t, promo.OwnerReferences)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			promo := testCase.promotion
			if promo == nil {
				promo = &kargoapi.Promotion{
					Spec: kargoapi.PromotionSpec{
						Stage: "fake-stage",
					},
				}
			}
			err := testCase.webhook.Default(context.Background(), promo)
			testCase.assertions(t, promo, err)
		})
	}
}

func TestValidateCreate(t *testing.T) {
	testCases := []struct {
		name       string
		webhook    *webhook
		userInfo   *authnv1.UserInfo
		assertions func(*testing.T, *fakeevent.EventRecorder, error)
	}{
		{
			name: "error validating project",
			webhook: &webhook{
				validateProjectFn: func(
					context.Context,
					client.Client,
					schema.GroupKind,
					client.Object,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *fakeevent.EventRecorder, err error) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
			},
		},
		{
			name: "authorization error",
			webhook: &webhook{
				validateProjectFn: func(
					context.Context,
					client.Client,
					schema.GroupKind,
					client.Object,
				) error {
					return nil
				},
				authorizeFn: func(context.Context, *kargoapi.Promotion, string) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *fakeevent.EventRecorder, err error) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
			},
		},
		{
			name: "record promotion created event on non-controlplane request",
			webhook: &webhook{
				validateProjectFn: func(
					context.Context,
					client.Client,
					schema.GroupKind,
					client.Object,
				) error {
					return nil
				},
				authorizeFn: func(context.Context, *kargoapi.Promotion, string) error {
					return nil
				},
				admissionRequestFromContextFn: admission.RequestFromContext,
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return nil, nil
				},
				isRequestFromKargoControlplaneFn: libWebhook.IsRequestFromKargoControlplane(
					regexp.MustCompile("^system:serviceaccount:kargo:(kargo-api|kargo-controller)$"),
				),
			},
			userInfo: &authnv1.UserInfo{
				Username: "fake-user",
			},
			assertions: func(t *testing.T, r *fakeevent.EventRecorder, err error) {
				require.NoError(t, err)
				require.Len(t, r.Events, 1)
				event := <-r.Events
				require.Equal(t, kargoapi.EventReasonPromotionCreated, event.Reason)
			},
		},
		{
			name: "skip recording promotion created event on controlplane request",
			webhook: &webhook{
				validateProjectFn: func(
					context.Context,
					client.Client,
					schema.GroupKind,
					client.Object,
				) error {
					return nil
				},
				authorizeFn: func(context.Context, *kargoapi.Promotion, string) error {
					return nil
				},
				admissionRequestFromContextFn: admission.RequestFromContext,
				isRequestFromKargoControlplaneFn: libWebhook.IsRequestFromKargoControlplane(
					regexp.MustCompile("^system:serviceaccount:kargo:(kargo-api|kargo-controller)$"),
				),
			},
			userInfo: &authnv1.UserInfo{
				Username: serviceaccount.ServiceAccountUsernamePrefix + "kargo:kargo-api",
			},
			assertions: func(t *testing.T, r *fakeevent.EventRecorder, err error) {
				require.NoError(t, err)
				require.Empty(t, r.Events)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := fakeevent.NewEventRecorder(1)
			testCase.webhook.recorder = recorder

			var req admission.Request
			if testCase.userInfo != nil {
				req.UserInfo = *testCase.userInfo
			}
			ctx := admission.NewContextWithRequest(context.Background(), req)

			_, err := testCase.webhook.ValidateCreate(
				ctx,
				&kargoapi.Promotion{
					Spec: kargoapi.PromotionSpec{
						Freight: "fake-freight",
					},
				},
			)
			testCase.assertions(t, recorder, err)
		})
	}
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
		assertions func(*testing.T, error)
	}{
		{
			name: "authorization error",
			setup: func() (*kargoapi.Promotion, *kargoapi.Promotion) {
				return &kargoapi.Promotion{}, &kargoapi.Promotion{}
			},
			authorizeFn: func(context.Context, *kargoapi.Promotion, string) error {
				return errors.New("something went wrong")
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "something went wrong")
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
					Spec: kargoapi.PromotionSpec{
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
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, `"fake-name" is invalid`)
				require.ErrorContains(t, err, "spec is immutable")
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
					Spec: kargoapi.PromotionSpec{
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
			assertions: func(t *testing.T, err error) {
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
			_, err := w.ValidateUpdate(context.Background(), oldPromo, newPromo)
			testCase.assertions(t, err)
		})
	}
}

func TestValidateDelete(t *testing.T) {
	testCases := []struct {
		name       string
		webhook    *webhook
		assertions func(*testing.T, error)
	}{
		{
			name: "authorization error",
			webhook: &webhook{
				authorizeFn: func(context.Context, *kargoapi.Promotion, string) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "success",
			webhook: &webhook{
				authorizeFn: func(context.Context, *kargoapi.Promotion, string) error {
					return nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := testCase.webhook.ValidateDelete(
				context.Background(),
				&kargoapi.Promotion{},
			)
			testCase.assertions(t, err)
		})
	}
}

func TestAuthorize(t *testing.T) {
	testCases := []struct {
		name                          string
		admissionRequestFromContextFn func(
			context.Context,
		) (admission.Request, error)
		createSubjectAccessReviewFn func(
			context.Context,
			client.Object,
			...client.CreateOption,
		) error
		assertions func(*testing.T, error)
	}{
		{
			name: "error getting admission request bound to context",
			admissionRequestFromContextFn: func(
				context.Context,
			) (admission.Request, error) {
				return admission.Request{}, errors.New("something went wrong")
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(
					t, err, "error retrieving admission request from context; refusing to",
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
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error creating SubjectAccessReview")
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
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "is not permitted")
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
			assertions: func(t *testing.T, err error) {
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
				t,
				w.authorize(
					context.Background(),
					&kargoapi.Promotion{
						ObjectMeta: v1.ObjectMeta{
							Name:      "fake-promotion",
							Namespace: "fake-namespace",
						},
						Spec: kargoapi.PromotionSpec{
							Stage: "fake-stage",
						},
					},
					"create",
				),
			)
		})
	}
}
