package warehouse

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/git"
)

func TestNewWebhook(t *testing.T) {
	kubeClient := fake.NewClientBuilder().Build()
	w := newWebhook(kubeClient)
	// Assert that all overridable behaviors were initialized to a default:
	require.NotNil(t, w.validateProjectFn)
	require.NotNil(t, w.validateCreateOrUpdateFn)
	require.NotNil(t, w.validateSpecFn)
}

func TestDefault(t *testing.T) {
	const testShardName = "fake-shard"

	w := &webhook{}

	t.Run("shard stays default when not specified at all", func(t *testing.T) {
		warehouse := &kargoapi.Warehouse{}
		err := w.Default(context.Background(), warehouse)
		require.NoError(t, err)
		require.Empty(t, warehouse.Labels)
		require.Empty(t, warehouse.Spec.Shard)
	})

	t.Run("sync shard label to non-empty shard field", func(t *testing.T) {
		warehouse := &kargoapi.Warehouse{
			Spec: kargoapi.WarehouseSpec{
				Shard: testShardName,
			},
		}
		err := w.Default(context.Background(), warehouse)
		require.NoError(t, err)
		require.Equal(t, testShardName, warehouse.Spec.Shard)
		require.Equal(t, testShardName, warehouse.Labels[kargoapi.LabelKeyShard])
	})

	t.Run("sync shard label to empty shard field", func(t *testing.T) {
		warehouse := &kargoapi.Warehouse{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					kargoapi.LabelKeyShard: testShardName,
				},
			},
		}
		err := w.Default(context.Background(), warehouse)
		require.NoError(t, err)
		require.Empty(t, warehouse.Spec.Shard)
		_, ok := warehouse.Labels[kargoapi.LabelKeyShard]
		require.False(t, ok)
	})
}

func TestValidateCreate(t *testing.T) {
	testCases := []struct {
		name       string
		webhook    *webhook
		assertions func(*testing.T, error)
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
			assertions: func(t *testing.T, err error) {
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
			assertions: func(t *testing.T, err error) {
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
			assertions: func(t *testing.T, err error) {
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
			testCase.assertions(t, err)
		})
	}
}

func TestValidateUpdate(t *testing.T) {
	testCases := []struct {
		name       string
		webhook    *webhook
		assertions func(*testing.T, error)
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
			assertions: func(t *testing.T, err error) {
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
			assertions: func(t *testing.T, err error) {
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
			testCase.assertions(t, err)
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
		assertions func(*testing.T, error)
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
			assertions: func(t *testing.T, err error) {
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
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := testCase.webhook.validateCreateOrUpdate(&kargoapi.Warehouse{})
			testCase.assertions(t, err)
		})
	}
}

func TestValidateSpec(t *testing.T) {
	testCases := []struct {
		name       string
		spec       kargoapi.WarehouseSpec
		assertions func(*testing.T, *kargoapi.WarehouseSpec, field.ErrorList)
	}{
		{
			name: "nil",
			assertions: func(t *testing.T, _ *kargoapi.WarehouseSpec, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
		{
			name: "invalid",
			spec: kargoapi.WarehouseSpec{
				Subscriptions: []kargoapi.RepoSubscription{
					{
						Git: &kargoapi.GitSubscription{
							RepoURL: "bogus",
						},
						Image: &kargoapi.ImageSubscription{
							SemverConstraint: "bogus",
							Platform:         "bogus",
						},
						Chart: &kargoapi.ChartSubscription{
							SemverConstraint: "bogus",
						},
					},
					{
						Git: &kargoapi.GitSubscription{
							RepoURL: "bogus",
						},
					},
				},
			},
			assertions: func(t *testing.T, spec *kargoapi.WarehouseSpec, errs field.ErrorList) {
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
								"spec.subscriptions[0].image, or spec.subscriptions[0].chart " +
								"must be non-empty",
						},
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "spec.subscriptions[1].git",
							BadValue: "bogus",
							Detail:   "subscription for Git repository already exists at \"spec.subscriptions[0].git\"",
						},
					},
					errs,
				)
			},
		},
		{
			name: "valid",
			spec: kargoapi.WarehouseSpec{
				// Nil subs are caught by declarative validation, so for the purposes of
				// this test, leaving that completely undefined should surface no
				// errors.
			},
			assertions: func(t *testing.T, _ *kargoapi.WarehouseSpec, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				&testCase.spec,
				w.validateSpec(
					field.NewPath("spec"),
					&testCase.spec,
				),
			)
		})
	}
}

func TestValidateSubs(t *testing.T) {
	testCases := []struct {
		name       string
		subs       []kargoapi.RepoSubscription
		assertions func(*testing.T, []kargoapi.RepoSubscription, field.ErrorList)
	}{
		{
			name: "empty",
			assertions: func(t *testing.T, _ []kargoapi.RepoSubscription, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
		{
			name: "invalid subscriptions",
			subs: []kargoapi.RepoSubscription{
				{
					Git: &kargoapi.GitSubscription{
						RepoURL: "bogus",
					},
					Image: &kargoapi.ImageSubscription{
						SemverConstraint: "bogus",
						Platform:         "bogus",
					},
					Chart: &kargoapi.ChartSubscription{
						SemverConstraint: "bogus",
					},
				},
				{
					Git: &kargoapi.GitSubscription{
						RepoURL: "bogus",
					},
				},
			},
			assertions: func(t *testing.T, subs []kargoapi.RepoSubscription, errs field.ErrorList) {
				require.Len(t, errs, 5)
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
							Detail: "exactly one of subs[0].git, subs[0].image, or " +
								"subs[0].chart must be non-empty",
						},
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "subs[1].git",
							BadValue: "bogus",
							Detail:   "subscription for Git repository already exists at \"subs[0].git\"",
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
			assertions: func(t *testing.T, _ []kargoapi.RepoSubscription, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
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
		seen       uniqueSubSet
		assertions func(*testing.T, kargoapi.RepoSubscription, field.ErrorList)
	}{
		{
			name: "invalid subscription",
			sub: kargoapi.RepoSubscription{
				Git: &kargoapi.GitSubscription{
					RepoURL: "bogus",
				},
				Image: &kargoapi.ImageSubscription{
					SemverConstraint: "bogus",
					Platform:         "bogus",
				},
				Chart: &kargoapi.ChartSubscription{
					SemverConstraint: "bogus",
				},
			},
			seen: uniqueSubSet{
				subscriptionKey{
					kind: "git",
					id:   git.NormalizeURL("bogus"),
				}: field.NewPath("spec.subscriptions[0].git"),
			},
			assertions: func(t *testing.T, sub kargoapi.RepoSubscription, errs field.ErrorList) {
				require.Len(t, errs, 5)
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "sub.git",
							BadValue: "bogus",
							Detail:   "subscription for Git repository already exists at \"spec.subscriptions[0].git\"",
						},
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
							Detail:   "exactly one of sub.git, sub.image, or sub.chart must be non-empty",
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
			seen: uniqueSubSet{},
			assertions: func(t *testing.T, _ kargoapi.RepoSubscription, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				testCase.sub,
				w.validateSub(field.NewPath("sub"), testCase.sub, testCase.seen),
			)
		})
	}
}

func TestValidateGitSub(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.GitSubscription
		seen       uniqueSubSet
		assertions func(*testing.T, field.ErrorList)
	}{
		{
			name: "invalid",
			sub: kargoapi.GitSubscription{
				RepoURL:          "bogus",
				SemverConstraint: "bogus",
			},
			seen: uniqueSubSet{
				subscriptionKey{
					kind: "git",
					id:   git.NormalizeURL("bogus"),
				}: field.NewPath("spec.subscriptions[0].git"),
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "git.semverConstraint",
							BadValue: "bogus",
						},
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "git",
							BadValue: "bogus",
							Detail:   "subscription for Git repository already exists at \"spec.subscriptions[0].git\"",
						},
					},
					errs,
				)
			},
		},

		{
			name: "valid",
			seen: uniqueSubSet{},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				w.validateGitSub(
					field.NewPath("git"),
					testCase.sub,
					testCase.seen,
				),
			)
		})
	}
}

func TestValidateImageSub(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.ImageSubscription
		seen       uniqueSubSet
		assertions func(*testing.T, field.ErrorList)
	}{
		{
			name: "invalid",
			sub: kargoapi.ImageSubscription{
				RepoURL:          "bogus",
				SemverConstraint: "bogus",
				Platform:         "bogus",
			},
			seen: uniqueSubSet{
				subscriptionKey{
					kind: "image",
					id:   "bogus",
				}: field.NewPath("spec.subscriptions[0].image"),
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
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
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "image",
							BadValue: "bogus",
							Detail:   "subscription for image repository already exists at \"spec.subscriptions[0].image\"",
						},
					},
					errs,
				)
			},
		},

		{
			name: "valid",
			seen: uniqueSubSet{},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				w.validateImageSub(
					field.NewPath("image"),
					testCase.sub,
					testCase.seen,
				),
			)
		})
	}
}

func TestValidateChartSub(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.ChartSubscription
		seen       uniqueSubSet
		assertions func(*testing.T, field.ErrorList)
	}{
		{
			name: "invalid semverConstraint and oci repoURL with name",
			sub: kargoapi.ChartSubscription{
				RepoURL:          "oci://fake-url",
				Name:             "should-not-be-here",
				SemverConstraint: "bogus",
			},
			seen: uniqueSubSet{},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "chart.semverConstraint",
							BadValue: "bogus",
						},
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "chart.name",
							BadValue: "should-not-be-here",
							Detail:   "must be empty if repoURL starts with oci://",
						},
					},
					errs,
				)
			},
		},

		{
			name: "https repoURL without name",
			sub: kargoapi.ChartSubscription{
				RepoURL: "https://fake-url",
			},
			seen: uniqueSubSet{},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "chart.name",
							BadValue: "",
							Detail:   "must be non-empty if repoURL starts with http:// or https://",
						},
					},
					errs,
				)
			},
		},

		{
			name: "duplicate HTTP/S chart",
			sub: kargoapi.ChartSubscription{
				RepoURL: "https://fake-url",
				Name:    "bogus",
			},
			seen: uniqueSubSet{
				subscriptionKey{
					kind: "chart",
					id:   "https://fake-url:bogus",
				}: field.NewPath("spec.subscriptions[0].chart"),
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "chart",
							BadValue: "https://fake-url",
							Detail:   "subscription for chart \"bogus\" already exists at \"spec.subscriptions[0].chart\"",
						},
					},
					errs,
				)
			},
		},

		{
			name: "duplicate OCI chart",
			sub: kargoapi.ChartSubscription{
				RepoURL: "oci://fake-url",
			},
			seen: uniqueSubSet{
				subscriptionKey{
					kind: "chart",
					id:   "fake-url",
				}: field.NewPath("spec.subscriptions[0].chart"),
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "chart",
							BadValue: "oci://fake-url",
							Detail:   "subscription for chart already exists at \"spec.subscriptions[0].chart\"",
						},
					},
					errs,
				)
			},
		},

		{
			name: "valid",
			sub:  kargoapi.ChartSubscription{},
			seen: uniqueSubSet{},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				w.validateChartSub(
					field.NewPath("chart"),
					testCase.sub,
					testCase.seen,
				),
			)
		})
	}
}

func TestValidateSemverConstraint(t *testing.T) {
	testCases := []struct {
		name             string
		semverConstraint string
		assertions       func(*testing.T, error)
	}{
		{
			name: "empty string",
			assertions: func(t *testing.T, err error) {
				require.Nil(t, err)
			},
		},

		{
			name:             "invalid",
			semverConstraint: "bogus",
			assertions: func(t *testing.T, err error) {
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
			assertions: func(t *testing.T, err error) {
				require.Nil(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				validateSemverConstraint(
					field.NewPath("semverConstraint"),
					testCase.semverConstraint,
				),
			)
		})
	}
}
