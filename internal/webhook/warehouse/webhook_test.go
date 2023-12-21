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
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

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
				validateCreateOrUpdateFn: func(
					*kargoapi.Warehouse,
				) (admission.Warnings, error) {
					return nil, errors.New("something went wrong")
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
				validateCreateOrUpdateFn: func(
					*kargoapi.Warehouse,
				) (admission.Warnings, error) {
					return nil, nil
				},
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := testCase.webhook.ValidateCreate(
				context.Background(),
				&kargoapi.Warehouse{},
			)
			testCase.assertions(err)
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
				validateCreateOrUpdateFn: func(
					*kargoapi.Warehouse,
				) (admission.Warnings, error) {
					return nil, errors.New("something went wrong")
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
				validateCreateOrUpdateFn: func(
					*kargoapi.Warehouse,
				) (admission.Warnings, error) {
					return nil, nil
				},
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := testCase.webhook.ValidateUpdate(
				context.Background(),
				nil,
				&kargoapi.Warehouse{},
			)
			testCase.assertions(err)
		})
	}
}

func TestValidateDelete(t *testing.T) {
	w := &webhook{}
	_, err := w.ValidateDelete(context.Background(), nil)
	require.NoError(t, err, nil)
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
			_, err := testCase.webhook.validateCreateOrUpdate(&kargoapi.Warehouse{})
			testCase.assertions(err)
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
				Subscriptions: []kargoapi.RepoSubscription{
					{
						Image: &kargoapi.ImageSubscription{
							SemverConstraint: "bogus",
							Platform:         "bogus",
						},
						Chart: &kargoapi.ChartSubscription{
							SemverConstraint: "bogus",
						},
					},
				},
			},
			assertions: func(spec *kargoapi.WarehouseSpec, errs field.ErrorList) {
				// We really want to see that all underlying errors have been bubbled up
				// to this level and been aggregated.
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "spec.subscriptions[0].image.semverConstraint",
							BadValue: "bogus",
						},
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "spec.subscriptions[0].image.platform",
							BadValue: "bogus",
						},
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "spec.subscriptions[0].chart.semverConstraint",
							BadValue: "bogus",
						},
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "spec.subscriptions[0]",
							BadValue: spec.Subscriptions[0],
							Detail: "exactly one of spec.subscriptions[0].git, " +
								"spec.subscriptions[0].images, or spec.subscriptions[0].charts " +
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
		subs       []kargoapi.RepoSubscription
		assertions func([]kargoapi.RepoSubscription, field.ErrorList)
	}{
		{
			name: "empty",
			assertions: func(_ []kargoapi.RepoSubscription, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
		{
			name: "invalid subscriptions",
			subs: []kargoapi.RepoSubscription{
				{
					Image: &kargoapi.ImageSubscription{
						SemverConstraint: "bogus",
						Platform:         "bogus",
					},
					Chart: &kargoapi.ChartSubscription{
						SemverConstraint: "bogus",
					},
				},
			},
			assertions: func(subs []kargoapi.RepoSubscription, errs field.ErrorList) {
				require.Len(t, errs, 4)
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "subs[0].image.semverConstraint",
							BadValue: "bogus",
						},
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "subs[0].image.platform",
							BadValue: "bogus",
						},
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "subs[0].chart.semverConstraint",
							BadValue: "bogus",
						},
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "subs[0]",
							BadValue: subs[0],
							Detail: "exactly one of subs[0].git, subs[0].images, or " +
								"subs[0].charts must be non-empty",
						},
					},
					errs,
				)
			},
		},
		{
			name: "valid",
			subs: []kargoapi.RepoSubscription{
				{Image: &kargoapi.ImageSubscription{}},
			},
			assertions: func(_ []kargoapi.RepoSubscription, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.subs,
				w.validateSubs(field.NewPath("subs"), testCase.subs),
			)
		})
	}
}

func TestValidateSub(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.RepoSubscription
		assertions func(kargoapi.RepoSubscription, field.ErrorList)
	}{
		{
			name: "invalid subscription",
			sub: kargoapi.RepoSubscription{
				Image: &kargoapi.ImageSubscription{
					SemverConstraint: "bogus",
					Platform:         "bogus",
				},
				Chart: &kargoapi.ChartSubscription{
					SemverConstraint: "bogus",
				},
			},
			assertions: func(sub kargoapi.RepoSubscription, errs field.ErrorList) {
				require.Len(t, errs, 4)
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "sub.image.semverConstraint",
							BadValue: "bogus",
						},
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "sub.image.platform",
							BadValue: "bogus",
						},
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "sub.chart.semverConstraint",
							BadValue: "bogus",
						},
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "sub",
							BadValue: sub,
							Detail:   "exactly one of sub.git, sub.images, or sub.charts must be non-empty",
						},
					},
					errs,
				)
			},
		},
		{
			name: "valid",
			sub: kargoapi.RepoSubscription{
				Image: &kargoapi.ImageSubscription{},
			},
			assertions: func(_ kargoapi.RepoSubscription, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.sub,
				w.validateSub(field.NewPath("sub"), testCase.sub),
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
