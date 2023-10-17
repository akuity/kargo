package promotionpolicy

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestNewWebhook(t *testing.T) {
	kubeClient := fake.NewClientBuilder().Build()
	w := newWebhook(kubeClient)
	// Assert that all overridable behaviors were initialized to a default:
	require.NotNil(t, w.validateProjectFn)
	require.NotNil(t, w.validateStageUniquenessFn)
}

func TestValidateCreate(t *testing.T) {
	testCases := []struct {
		name       string
		webhook    *webhook
		assertions func(error)
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
			assertions: func(err error) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
			},
		},
		{
			name: "error validating stage uniqueness",
			webhook: &webhook{
				validateProjectFn: func(
					context.Context,
					client.Client,
					schema.GroupKind,
					client.Object,
				) error {
					return nil
				},
				validateStageUniquenessFn: func(
					context.Context,
					*kargoapi.PromotionPolicy,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
			},
		},
		{
			name: "success",
			webhook: &webhook{
				validateProjectFn: func(
					context.Context,
					client.Client,
					schema.GroupKind,
					client.Object,
				) error {
					return nil
				},
				validateStageUniquenessFn: func(
					context.Context,
					*kargoapi.PromotionPolicy,
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
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.webhook.ValidateCreate(
					context.Background(),
					&kargoapi.PromotionPolicy{},
				),
			)
		})
	}
}

func TestValidateUpdate(t *testing.T) {
	testCases := []struct {
		name       string
		webhook    *webhook
		assertions func(error)
	}{
		{
			name: "error validating stage uniqueness",
			webhook: &webhook{
				validateStageUniquenessFn: func(
					context.Context,
					*kargoapi.PromotionPolicy,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
			},
		},
		{
			name: "success",
			webhook: &webhook{
				validateStageUniquenessFn: func(
					context.Context,
					*kargoapi.PromotionPolicy,
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
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.webhook.ValidateUpdate(
					context.Background(),
					&kargoapi.PromotionPolicy{},
					&kargoapi.PromotionPolicy{},
				),
			)
		})
	}
}

func TestValidateDelete(t *testing.T) {
	w := &webhook{}
	require.NoError(
		t,
		w.ValidateDelete(context.Background(),
			&kargoapi.Promotion{}),
	)
}

func TestValidateStageUniqueness(t *testing.T) {
	testCases := []struct {
		name       string
		webhook    *webhook
		assertions func(error)
	}{
		{
			name: "error listing promotion policies",
			webhook: &webhook{
				listPromotionPoliciesFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name: "another promotion policy with the same stage exists",
			webhook: &webhook{
				listPromotionPoliciesFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					policies := objList.(*kargoapi.PromotionPolicyList) // nolint: forcetypeassert
					policies.Items = []kargoapi.PromotionPolicy{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "another-policy",
							},
						},
					}
					return nil
				},
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "policy for stage")
				require.Contains(t, err.Error(), "already exists")
			},
		},
		{
			name: "success",
			webhook: &webhook{
				listPromotionPoliciesFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
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
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.webhook.validateStageUniqueness(
					context.Background(),
					&kargoapi.PromotionPolicy{},
				),
			)
		})
	}
}
