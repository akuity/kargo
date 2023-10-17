package warehouse

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestNewWebhook(t *testing.T) {
	kubeClient := fake.NewClientBuilder().Build()
	w := newWebhook(kubeClient)
	// Assert that all overridable behaviors were initialized to a default:
	require.NotNil(t, w.validateProjectFn)
	require.NotNil(t, w.validateCreateOrUpdateFn)
	require.NotNil(t, w.validateSpecFn)
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
			name: "error validating warehouse",
			webhook: &webhook{
				validateProjectFn: func(
					context.Context,
					client.Client,
					schema.GroupKind,
					client.Object,
				) error {
					return nil
				},
				validateCreateOrUpdateFn: func(*kargoapi.Warehouse) error {
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
				validateCreateOrUpdateFn: func(*kargoapi.Warehouse) error {
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
					&kargoapi.Warehouse{},
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
			name: "error validating warehouse",
			webhook: &webhook{
				validateCreateOrUpdateFn: func(*kargoapi.Warehouse) error {
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
				validateCreateOrUpdateFn: func(*kargoapi.Warehouse) error {
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
					nil,
					&kargoapi.Warehouse{},
				),
			)
		})
	}
}

func TestValidateDelete(t *testing.T) {
	w := &webhook{}
	require.NoError(t, w.ValidateDelete(context.Background(), nil))
}

func TestValidateCreateOrUpdate(t *testing.T) {
	testCases := []struct {
		name       string
		webhook    *webhook
		assertions func(error)
	}{
		{
			name: "error validating spec",
			webhook: &webhook{
				validateSpecFn: func(
					*field.Path,
					*kargoapi.WarehouseSpec,
				) field.ErrorList {
					return field.ErrorList{{}}
				},
			},
			assertions: func(err error) {
				require.Error(t, err)
			},
		},
		{
			name: "success",
			webhook: &webhook{
				validateSpecFn: func(
					*field.Path,
					*kargoapi.WarehouseSpec,
				) field.ErrorList {
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
				testCase.webhook.validateCreateOrUpdate(&kargoapi.Warehouse{}),
			)
		})
	}
}

func TestValidateSpec(t *testing.T) {
	testCases := []struct {
		name       string
		spec       *kargoapi.WarehouseSpec
		assertions func(*kargoapi.WarehouseSpec, field.ErrorList)
	}{
		{
			name: "nil",
			assertions: func(_ *kargoapi.WarehouseSpec, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},

		{
			name: "invalid",
			spec: &kargoapi.WarehouseSpec{
				// Doesn't describe any subscriptions
				Subscriptions: &kargoapi.RepoSubscriptions{},
			},
			assertions: func(spec *kargoapi.WarehouseSpec, errs field.ErrorList) {
				// We really want to see that all underlying errors have been bubbled up
				// to this level and been aggregated.
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "spec.subscriptions",
							BadValue: spec.Subscriptions,
							Detail: "at least one of spec.subscriptions.git, " +
								"spec.subscriptions.images, or spec.subscriptions.charts " +
								"must be non-empty",
						},
					},
					errs,
				)
			},
		},

		{
			name: "valid",
			spec: &kargoapi.WarehouseSpec{
				// Nil subs are caught by declarative validation, so for the purposes of
				// this test, leaving that completely undefined should surface no
				// errors.
			},
			assertions: func(_ *kargoapi.WarehouseSpec, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.spec,
				w.validateSpec(
					field.NewPath("spec"),
					testCase.spec,
				),
			)
		})
	}
}

func TestValidateSubs(t *testing.T) {
	testCases := []struct {
		name       string
		subs       *kargoapi.RepoSubscriptions
		assertions func(*kargoapi.RepoSubscriptions, field.ErrorList)
	}{
		{
			name: "nil",
			assertions: func(_ *kargoapi.RepoSubscriptions, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},

		{
			name: "no subscriptions",
			subs: &kargoapi.RepoSubscriptions{}, // Has no subs
			assertions: func(subs *kargoapi.RepoSubscriptions, errs field.ErrorList) {
				require.Len(t, errs, 1)
				require.Equal(
					t,
					&field.Error{
						Type:     field.ErrorTypeInvalid,
						Field:    "repos",
						BadValue: subs,
						Detail: "at least one of repos.git, repos.images, or " +
							"repos.charts must be non-empty",
					},
					errs[0],
				)
			},
		},

		{
			name: "invalid subscriptions",
			subs: &kargoapi.RepoSubscriptions{
				Images: []kargoapi.ImageSubscription{
					{
						SemverConstraint: "bogus",
						Platform:         "bogus",
					},
				},
				Charts: []kargoapi.ChartSubscription{
					{
						SemverConstraint: "bogus",
					},
				},
			},
			assertions: func(subs *kargoapi.RepoSubscriptions, errs field.ErrorList) {
				require.Len(t, errs, 3)
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "repos.images[0].semverConstraint",
							BadValue: "bogus",
						},
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "repos.images[0].platform",
							BadValue: "bogus",
						},
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "repos.charts[0].semverConstraint",
							BadValue: "bogus",
						},
					},
					errs,
				)
			},
		},

		{
			name: "valid",
			subs: &kargoapi.RepoSubscriptions{
				Images: []kargoapi.ImageSubscription{
					{},
				},
			},
			assertions: func(subs *kargoapi.RepoSubscriptions, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.subs,
				w.validateSubs(field.NewPath("repos"), testCase.subs),
			)
		})
	}
}

func TestValidateImageSubs(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.ImageSubscription
		assertions func(field.ErrorList)
	}{
		{
			name: "invalid",
			sub: kargoapi.ImageSubscription{
				SemverConstraint: "bogus",
				Platform:         "bogus",
			},
			assertions: func(errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "images[0].semverConstraint",
							BadValue: "bogus",
						},
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "images[0].platform",
							BadValue: "bogus",
						},
					},
					errs,
				)
			},
		},

		{
			name: "valid",
			assertions: func(errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				w.validateImageSubs(
					field.NewPath("images"),
					[]kargoapi.ImageSubscription{
						testCase.sub,
					},
				),
			)
		})
	}
}

func TestValidateImageSub(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.ImageSubscription
		assertions func(field.ErrorList)
	}{
		{
			name: "invalid",
			sub: kargoapi.ImageSubscription{
				SemverConstraint: "bogus",
				Platform:         "bogus",
			},
			assertions: func(errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "image.semverConstraint",
							BadValue: "bogus",
						},
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "image.platform",
							BadValue: "bogus",
						},
					},
					errs,
				)
			},
		},

		{
			name: "valid",
			assertions: func(errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				w.validateImageSub(
					field.NewPath("image"),
					testCase.sub,
				),
			)
		})
	}
}

func TestValidateChartSubs(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.ChartSubscription
		assertions func(field.ErrorList)
	}{
		{
			name: "invalid",
			sub: kargoapi.ChartSubscription{
				SemverConstraint: "bogus",
			},
			assertions: func(errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "charts[0].semverConstraint",
							BadValue: "bogus",
						},
					},
					errs,
				)
			},
		},

		{
			name: "valid",
			sub:  kargoapi.ChartSubscription{},
			assertions: func(errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				w.validateChartSubs(
					field.NewPath("charts"),
					[]kargoapi.ChartSubscription{
						testCase.sub,
					},
				),
			)
		})
	}
}

func TestValidateChartSub(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.ChartSubscription
		assertions func(field.ErrorList)
	}{
		{
			name: "invalid",
			sub: kargoapi.ChartSubscription{
				SemverConstraint: "bogus",
			},
			assertions: func(errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "chart.semverConstraint",
							BadValue: "bogus",
						},
					},
					errs,
				)
			},
		},

		{
			name: "valid",
			sub:  kargoapi.ChartSubscription{},
			assertions: func(errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				w.validateChartSub(
					field.NewPath("chart"),
					testCase.sub,
				),
			)
		})
	}
}

func TestValidateSemverConstraint(t *testing.T) {
	testCases := []struct {
		name             string
		semverConstraint string
		assertions       func(error)
	}{
		{
			name: "empty string",
			assertions: func(err error) {
				require.Nil(t, err)
			},
		},

		{
			name:             "invalid",
			semverConstraint: "bogus",
			assertions: func(err error) {
				require.NotNil(t, err)
				require.Equal(
					t,
					&field.Error{
						Type:     field.ErrorTypeInvalid,
						Field:    "semverConstraint",
						BadValue: "bogus",
					},
					err,
				)
			},
		},

		{
			name:             "valid",
			semverConstraint: "^1.0.0",
			assertions: func(err error) {
				require.Nil(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				validateSemverConstraint(
					field.NewPath("semverConstraint"),
					testCase.semverConstraint,
				),
			)
		})
	}
}
